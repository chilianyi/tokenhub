package model

import (
	"errors"
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// Organization —— 部门（组织）与部门额度池
//
// 额度语义（分配池）：
//   Quota      部门充值进来的总额度池
//   Allocated  已经分配给部门成员的额度总和
//   不变式：   Allocated <= Quota
//
// 成员调用模型时仍然只扣自己的 users.quota，不改扣费主流程；
// 部门池只在「给成员分配额度」时校验，超出即拒绝。
// ---------------------------------------------------------------------------

var (
	ErrOrgQuotaExceeded = errors.New("organization quota exceeded")
	ErrOrgNotFound      = errors.New("organization not found")
	ErrOrgNameDuplicated = errors.New("organization name duplicated")
)

const (
	OrgStatusEnabled  = 1
	OrgStatusDisabled = 0
)

type Organization struct {
	Id     int    `json:"id"`
	// 唯一索引用 name + deleted_at 组合（priority 1/2），而不是 partial index：
	// MySQL 不支持 `where: deleted_at IS NULL` 这种条件索引，AutoMigrate 会静默失败。
	// 组合索引实现「删除后允许重名」，见 model/vendor_meta.go 的 uk_vendor_name_delete_at。
	Name        string         `json:"name" gorm:"size:128;not null;uniqueIndex:uk_org_name_delete_at,priority:1"`
	Quota       int            `json:"quota" gorm:"type:bigint;not null;default:0"`     // 部门额度池
	Allocated   int            `json:"allocated" gorm:"type:bigint;not null;default:0"` // 已分配给成员的总额
	Status      int            `json:"status" gorm:"default:1"`                         // 1=正常 0=停用
	Remark      string         `json:"remark,omitempty" gorm:"type:varchar(255)"`
	CreatedTime int64          `json:"created_time" gorm:"bigint"`
	UpdatedTime int64          `json:"updated_time" gorm:"bigint"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index;uniqueIndex:uk_org_name_delete_at,priority:2"`
}

func (o *Organization) TableName() string {
	return "organizations"
}

// Remaining 返回部门池中尚未分配的额度。
func (o *Organization) Remaining() int {
	return o.Quota - o.Allocated
}

// Insert 新建部门。
func (o *Organization) Insert() error {
	dup, err := IsOrganizationNameDuplicated(0, o.Name)
	if err != nil {
		return err
	}
	if dup {
		return ErrOrgNameDuplicated
	}
	now := common.GetTimestamp()
	o.CreatedTime = now
	o.UpdatedTime = now
	if o.Status == 0 {
		o.Status = OrgStatusEnabled
	}
	return DB.Create(o).Error
}

// Update 更新部门（只更新可变字段）。
func (o *Organization) Update() error {
	dup, err := IsOrganizationNameDuplicated(o.Id, o.Name)
	if err != nil {
		return err
	}
	if dup {
		return ErrOrgNameDuplicated
	}
	o.UpdatedTime = common.GetTimestamp()
	return DB.Model(&Organization{}).Where("id = ?", o.Id).Updates(map[string]any{
		"name":         o.Name,
		"quota":        o.Quota,
		"status":       o.Status,
		"remark":       o.Remark,
		"updated_time": o.UpdatedTime,
	}).Error
}

// DeleteOrganizationByID 软删除部门，并解除成员归属。
func DeleteOrganizationByID(id int) error {
	if id <= 0 {
		return ErrOrgNotFound
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		var org Organization
		if err := lockForUpdate(tx).First(&org, id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrOrgNotFound
			}
			return err
		}
		if err := tx.Where("organization_id = ?", id).Model(&User{}).Update("organization_id", 0).Error; err != nil {
			return err
		}
		return tx.Delete(&Organization{}, id).Error
	})
}

func GetOrganizationById(id int) (*Organization, error) {
	var org Organization
	err := DB.First(&org, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrOrgNotFound
	}
	if err != nil {
		return nil, err
	}
	return &org, nil
}

func IsOrganizationNameDuplicated(id int, name string) (bool, error) {
	if name == "" {
		return false, nil
	}
	var cnt int64
	err := DB.Model(&Organization{}).Where("name = ? AND id <> ?", name, id).Count(&cnt).Error
	return cnt > 0, err
}

// GetOrganizations 分页查询部门，keyword 模糊匹配名称。
func GetOrganizations(keyword string, startIdx, num int) ([]*Organization, int64, error) {
	tx := DB.Model(&Organization{})
	if keyword != "" {
		tx = tx.Where("name LIKE ?", "%"+keyword+"%")
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var orgs []*Organization
	err := tx.Order("id desc").Limit(num).Offset(startIdx).Find(&orgs).Error
	return orgs, total, err
}

// RecalcAllocated 按成员当前余额重算 Allocated，用于首次启用或对账。
func RecalcAllocated(tx *gorm.DB, orgId int) (int, error) {
	var sum int64
	if err := tx.Model(&User{}).
		Where("organization_id = ? AND deleted_at IS NULL", orgId).
		Select("COALESCE(SUM(quota), 0)").Scan(&sum).Error; err != nil {
		return 0, err
	}
	if sum < 0 {
		// 成员余额可能被扣成负数（额度用尽），对账时按 0 计，不冲减部门池。
		sum = 0
	}
	if err := tx.Model(&Organization{}).Where("id = ?", orgId).
		Update("allocated", sum).Error; err != nil {
		return 0, err
	}
	return int(sum), nil
}

// ---------------------------------------------------------------------------
// 额度池原语
// ---------------------------------------------------------------------------

// AllocateQuota 从部门池分配 delta 给成员。
//
// 原子性：校验与更新在同一条 UPDATE 内完成，避免两个并发请求读到同一个
// allocated 值而双双通过。与 model/topup.go 的 creditTopUpQuota 同一模式。
//
// orgId <= 0 表示成员不属于任何部门，直接放行。
func AllocateQuota(tx *gorm.DB, orgId, delta int) error {
	if orgId <= 0 || delta <= 0 {
		return nil
	}
	result := tx.Model(&Organization{}).
		Where("id = ? AND allocated + ? <= quota", orgId, delta).
		Update("allocated", gorm.Expr("allocated + ?", delta))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 1 {
		return nil
	}
	// 区分「部门不存在」和「额度不足」，给出可读错误。
	var count int64
	if err := tx.Model(&Organization{}).Where("id = ?", orgId).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return ErrOrgNotFound
	}
	var org Organization
	if err := tx.First(&org, orgId).Error; err != nil {
		return err
	}
	return fmt.Errorf("%w: 部门额度不足，可用 %d，需要 %d", ErrOrgQuotaExceeded, org.Remaining(), delta)
}

// ReleaseQuota 成员额度减少时把 delta 归还到部门池。
//
// 用两条 UPDATE 而不是 GREATEST/NULLIF 等方言函数：不同数据库对 GREATEST 的
// 支持不一致（SQLite 有但语义依编译选项，MySQL 有，部分 PG 版本无），
// 这里先用条件更新处理常规情况，仅在 allocated < delta 的罕见情形下补一条钳位。
func ReleaseQuota(tx *gorm.DB, orgId, delta int) error {
	if orgId <= 0 || delta <= 0 {
		return nil
	}
	result := tx.Model(&Organization{}).
		Where("id = ? AND allocated >= ?", orgId, delta).
		Update("allocated", gorm.Expr("allocated - ?", delta))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 1 {
		return nil
	}
	// allocated < delta：直接钳到 0。
	clamp := tx.Model(&Organization{}).
		Where("id = ?", orgId).
		Update("allocated", 0)
	if clamp.Error != nil {
		return clamp.Error
	}
	if clamp.RowsAffected == 0 {
		var count int64
		if err := tx.Model(&Organization{}).Where("id = ?", orgId).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			return ErrOrgNotFound
		}
	}
	return nil
}
