package model

import (
	"errors"
	"sync"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// useOrganizationTestDB 用内存 sqlite 建表并把全局 DB 指过去。
// 参照 model/frontend_option_migration_test.go 的既有模式。
func useOrganizationTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	previousDB := DB
	previousType := common.MainDatabaseType()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	// 单连接避免 sqlite :memory: 在多连接下各是独立库。
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(&Organization{}, &User{}, &Redemption{}))
	DB = db
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)
	t.Cleanup(func() {
		DB = previousDB
		common.SetMainDatabaseType(previousType)
		_ = sqlDB.Close()
	})
	return db
}

func mustCreateOrg(t *testing.T, db *gorm.DB, name string, quota int) *Organization {
	t.Helper()
	org := &Organization{Name: name, Quota: quota}
	require.NoError(t, org.Insert())
	return org
}

func mustCreateUser(t *testing.T, db *gorm.DB, username string, quota, orgId int) *User {
	t.Helper()
	u := &User{
		Username: username, Quota: quota, OrganizationId: orgId, Role: 1, Status: 1,
		// aff_code 有唯一索引，测试里必须给唯一值
		AffCode: "aff-" + username,
	}
	require.NoError(t, db.Create(u).Error)
	return u
}

func reloadOrg(t *testing.T, db *gorm.DB, id int) Organization {
	t.Helper()
	var o Organization
	require.NoError(t, db.First(&o, id).Error)
	return o
}

// ---------------------------------------------------------------------------
// AllocateQuota / ReleaseQuota 原语
// ---------------------------------------------------------------------------

func TestAllocateQuota_WithinLimit(t *testing.T) {
	db := useOrganizationTestDB(t)
	org := mustCreateOrg(t, db, "研究组", 1000)

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return AllocateQuota(tx, org.Id, 600)
	}))
	assert.Equal(t, 600, reloadOrg(t, db, org.Id).Allocated)

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return AllocateQuota(tx, org.Id, 400)
	}))
	got := reloadOrg(t, db, org.Id)
	assert.Equal(t, 1000, got.Allocated, "刚好用完池子应允许")
	assert.Equal(t, 0, got.Remaining())
}

func TestAllocateQuota_ExceedsLimit(t *testing.T) {
	db := useOrganizationTestDB(t)
	org := mustCreateOrg(t, db, "研究组", 1000)

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return AllocateQuota(tx, org.Id, 600)
	}))

	err := db.Transaction(func(tx *gorm.DB) error {
		return AllocateQuota(tx, org.Id, 600)
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrOrgQuotaExceeded), "应返回 ErrOrgQuotaExceeded，实际: %v", err)
	assert.Equal(t, 600, reloadOrg(t, db, org.Id).Allocated, "失败的分配不应改变池子")
}

func TestAllocateQuota_NoOrgIsNoop(t *testing.T) {
	db := useOrganizationTestDB(t)
	// organization_id = 0 表示未归属部门，不做任何校验。
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return AllocateQuota(tx, 0, 999999)
	}))
}

func TestAllocateQuota_OrgNotFound(t *testing.T) {
	db := useOrganizationTestDB(t)
	err := db.Transaction(func(tx *gorm.DB) error {
		return AllocateQuota(tx, 99999, 100)
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrOrgNotFound))
}

func TestReleaseQuota_ReturnsToPool(t *testing.T) {
	db := useOrganizationTestDB(t)
	org := mustCreateOrg(t, db, "研究组", 1000)

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return AllocateQuota(tx, org.Id, 800)
	}))
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return ReleaseQuota(tx, org.Id, 300)
	}))
	assert.Equal(t, 500, reloadOrg(t, db, org.Id).Allocated)

	// 释放后池子有空间，可以重新分配
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return AllocateQuota(tx, org.Id, 500)
	}))
	assert.Equal(t, 1000, reloadOrg(t, db, org.Id).Allocated)
}

func TestReleaseQuota_ClampsAtZero(t *testing.T) {
	db := useOrganizationTestDB(t)
	org := mustCreateOrg(t, db, "研究组", 1000)

	// 释放量大于已分配量时钳到 0，不应变成负数
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return ReleaseQuota(tx, org.Id, 500)
	}))
	assert.Equal(t, 0, reloadOrg(t, db, org.Id).Allocated)
}

// ---------------------------------------------------------------------------
// 并发：额度不能超分
// ---------------------------------------------------------------------------

// 部门池 1000，20 个 goroutine 各申请 100，只能有 10 个成功。
func TestAllocateQuota_ConcurrentNoOversell(t *testing.T) {
	db := useOrganizationTestDB(t)
	org := mustCreateOrg(t, db, "研究组", 1000)

	const workers = 20
	const each = 100
	var wg sync.WaitGroup
	var mu sync.Mutex
	success := 0
	failed := 0

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := DB.Transaction(func(tx *gorm.DB) error {
				return AllocateQuota(tx, org.Id, each)
			})
			mu.Lock()
			defer mu.Unlock()
			if err == nil {
				success++
			} else {
				failed++
			}
		}()
	}
	wg.Wait()

	assert.Equal(t, workers, success+failed)
	assert.Equal(t, 10, success, "池子 1000 / 每次 100，只应成功 10 次")
	assert.Equal(t, 10, failed)
	got := reloadOrg(t, db, org.Id)
	assert.Equal(t, 1000, got.Allocated)
	assert.LessOrEqual(t, got.Allocated, got.Quota, "不得超分")
}

// ---------------------------------------------------------------------------
// AdjustUserQuota 集成：部门额度阻止分配
// ---------------------------------------------------------------------------

func TestAdjustUserQuota_BlockedByOrgQuota(t *testing.T) {
	db := useOrganizationTestDB(t)
	org := mustCreateOrg(t, db, "研究组", 1000)
	userA := mustCreateUser(t, db, "alice", 0, org.Id)
	userB := mustCreateUser(t, db, "bob", 0, org.Id)

	// A 加 600 → 成功
	_, err := AdjustUserQuota(userA.Id, common.RoleRootUser, "add", 600)
	require.NoError(t, err)
	assert.Equal(t, 600, reloadOrg(t, db, org.Id).Allocated)

	// B 加 600 → 超池子，必须失败
	_, err = AdjustUserQuota(userB.Id, common.RoleRootUser, "add", 600)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrOrgQuotaExceeded))

	var b User
	require.NoError(t, db.First(&b, userB.Id).Error)
	assert.Equal(t, 0, b.Quota, "失败时成员余额不得改变")
	assert.Equal(t, 600, reloadOrg(t, db, org.Id).Allocated, "失败时部门池不得改变")

	// B 加 400 → 刚好用完
	_, err = AdjustUserQuota(userB.Id, common.RoleRootUser, "add", 400)
	require.NoError(t, err)
	assert.Equal(t, 1000, reloadOrg(t, db, org.Id).Allocated)
}

func TestAdjustUserQuota_NoOrgUnconstrained(t *testing.T) {
	db := useOrganizationTestDB(t)
	user := mustCreateUser(t, db, "solo", 0, 0)

	// 未归属部门，行为与改造前一致，不受任何部门额度限制
	_, err := AdjustUserQuota(user.Id, common.RoleRootUser, "add", 1_000_000)
	require.NoError(t, err)

	var u User
	require.NoError(t, db.First(&u, user.Id).Error)
	assert.Equal(t, 1_000_000, u.Quota)
}

func TestAdjustUserQuota_SubtractReleasesToPool(t *testing.T) {
	db := useOrganizationTestDB(t)
	org := mustCreateOrg(t, db, "研究组", 1000)
	user := mustCreateUser(t, db, "alice", 0, org.Id)

	_, err := AdjustUserQuota(user.Id, common.RoleRootUser, "add", 800)
	require.NoError(t, err)
	assert.Equal(t, 800, reloadOrg(t, db, org.Id).Allocated)

	_, err = AdjustUserQuota(user.Id, common.RoleRootUser, "subtract", 300)
	require.NoError(t, err)
	assert.Equal(t, 500, reloadOrg(t, db, org.Id).Allocated)

	var u User
	require.NoError(t, db.First(&u, user.Id).Error)
	assert.Equal(t, 500, u.Quota)
}

// override 到更高额度也要走部门校验
func TestAdjustUserQuota_OverrideAlsoChecked(t *testing.T) {
	db := useOrganizationTestDB(t)
	org := mustCreateOrg(t, db, "研究组", 1000)
	user := mustCreateUser(t, db, "alice", 0, org.Id)

	_, err := AdjustUserQuota(user.Id, common.RoleRootUser, "override", 1500)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrOrgQuotaExceeded))

	var u User
	require.NoError(t, db.First(&u, user.Id).Error)
	assert.Equal(t, 0, u.Quota)
}

// ---------------------------------------------------------------------------
// 兑换码集成：同样受部门额度约束
// ---------------------------------------------------------------------------

func TestRedeem_BlockedByOrgQuota(t *testing.T) {
	db := useOrganizationTestDB(t)
	org := mustCreateOrg(t, db, "研究组", 100)
	user := mustCreateUser(t, db, "alice", 0, org.Id)

	code := &Redemption{Key: "test-redeem-key-0123456789abcdef", Status: common.RedemptionCodeStatusEnabled, Quota: 500, Name: "test"}
	require.NoError(t, db.Create(code).Error)

	_, err := Redeem(code.Key, user.Id)
	require.Error(t, err)
	// Redeem 会把内部错误统一包装成 ErrRedeemFailed（对外不暴露细分原因，
	// 防止攻击者据此探测兑换码状态）。这里验证的是「兑换被拦住且余额未动」。
	assert.ErrorIs(t, err, ErrRedeemFailed)

	var u User
	require.NoError(t, db.First(&u, user.Id).Error)
	assert.Equal(t, 0, u.Quota, "兑换失败时余额不得改变")
	assert.Equal(t, 0, reloadOrg(t, db, org.Id).Allocated)
}

// ---------------------------------------------------------------------------
// 对账
// ---------------------------------------------------------------------------

func TestRecalcAllocated_FollowsMemberBalances(t *testing.T) {
	db := useOrganizationTestDB(t)
	org := mustCreateOrg(t, db, "研究组", 1000)
	mustCreateUser(t, db, "a", 300, org.Id)
	mustCreateUser(t, db, "b", 200, org.Id)
	mustCreateUser(t, db, "outsider", 5000, 0)

	var got int
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		var err error
		got, err = RecalcAllocated(tx, org.Id)
		return err
	}))
	assert.Equal(t, 500, got, "只统计本部门成员，且按当前余额")
	assert.Equal(t, 500, reloadOrg(t, db, org.Id).Allocated)
}

// ---------------------------------------------------------------------------
// 换部门：额度归属要在两个部门间迁移
// ---------------------------------------------------------------------------

func TestEditUser_MoveOrganizationMigratesQuota(t *testing.T) {
	db := useOrganizationTestDB(t)
	orgA := mustCreateOrg(t, db, "部门A", 1000)
	orgB := mustCreateOrg(t, db, "部门B", 1000)
	user := mustCreateUser(t, db, "alice", 0, orgA.Id)

	_, err := AdjustUserQuota(user.Id, common.RoleRootUser, "add", 600)
	require.NoError(t, err)
	assert.Equal(t, 600, reloadOrg(t, db, orgA.Id).Allocated)

	// 换到部门B
	user.OrganizationId = orgB.Id
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return user.EditWithTx(tx, false)
	}))

	assert.Equal(t, 0, reloadOrg(t, db, orgA.Id).Allocated, "旧部门应释放该成员额度")
	assert.Equal(t, 600, reloadOrg(t, db, orgB.Id).Allocated, "新部门应承接该成员额度")

	var u User
	require.NoError(t, db.First(&u, user.Id).Error)
	assert.Equal(t, orgB.Id, u.OrganizationId)
	assert.Equal(t, 600, u.Quota, "个人余额不变")
}

// 新部门池子不够时，换部门必须被拒绝且两边都不变
func TestEditUser_MoveOrganizationRejectedWhenTargetFull(t *testing.T) {
	db := useOrganizationTestDB(t)
	orgA := mustCreateOrg(t, db, "部门A", 1000)
	orgB := mustCreateOrg(t, db, "部门B", 100) // 装不下 600
	user := mustCreateUser(t, db, "alice", 0, orgA.Id)

	_, err := AdjustUserQuota(user.Id, common.RoleRootUser, "add", 600)
	require.NoError(t, err)

	user.OrganizationId = orgB.Id
	err = db.Transaction(func(tx *gorm.DB) error {
		return user.EditWithTx(tx, false)
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrOrgQuotaExceeded), "实际: %v", err)

	assert.Equal(t, 600, reloadOrg(t, db, orgA.Id).Allocated, "失败时旧部门不得释放")
	assert.Equal(t, 0, reloadOrg(t, db, orgB.Id).Allocated, "失败时新部门不得占用")

	var u User
	require.NoError(t, db.First(&u, user.Id).Error)
	assert.Equal(t, orgA.Id, u.OrganizationId, "失败时归属不变")
}

// 移出部门（到 0）应释放额度且不受任何限制
func TestEditUser_MoveOutOfOrganizationReleases(t *testing.T) {
	db := useOrganizationTestDB(t)
	org := mustCreateOrg(t, db, "部门A", 1000)
	user := mustCreateUser(t, db, "alice", 0, org.Id)

	_, err := AdjustUserQuota(user.Id, common.RoleRootUser, "add", 400)
	require.NoError(t, err)

	user.OrganizationId = 0
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return user.EditWithTx(tx, false)
	}))
	assert.Equal(t, 0, reloadOrg(t, db, org.Id).Allocated)

	var u User
	require.NoError(t, db.First(&u, user.Id).Error)
	assert.Equal(t, 400, u.Quota, "移出部门不扣个人余额")
}
