package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func useChannelPublicNameTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	previousDB := DB
	previousMemCache := common.MemoryCacheEnabled
	previousType := common.MainDatabaseType()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(&Channel{}, &Log{}))
	DB = db
	// 强制走 DB 分支（内存缓存需要 Redis，这里不做）
	common.MemoryCacheEnabled = false
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)
	t.Cleanup(func() {
		DB = previousDB
		common.MemoryCacheEnabled = previousMemCache
		common.SetMainDatabaseType(previousType)
		_ = sqlDB.Close()
	})
	return db
}

func strPtr(s string) *string { return &s }

// 配置了 public_name 的渠道，用户侧日志应显示公开名
func TestFormatUserLogs_ShowsPublicName(t *testing.T) {
	db := useChannelPublicNameTestDB(t)
	require.NoError(t, db.Create(&Channel{
		Id: 1, Name: "zhongyun-lingmiao", PublicName: strPtr("中云"),
		Key: "k", Type: 58, Status: 1, Group: "default",
	}).Error)

	logs := []*Log{{ChannelId: 1, ChannelName: "zhongyun-lingmiao"}}
	formatUserLogs(logs, 0)

	assert.Equal(t, "中云", logs[0].ChannelName, "应显示公开名，不是内部名")
}

// 未配置 public_name 的渠道，用户侧日志渠道名为空（前端据此不显示）
func TestFormatUserLogs_HidesChannelWithoutPublicName(t *testing.T) {
	db := useChannelPublicNameTestDB(t)
	require.NoError(t, db.Create(&Channel{
		Id: 2, Name: "experimental-canary", Key: "k", Type: 1, Status: 1, Group: "default",
	}).Error)

	logs := []*Log{{ChannelId: 2, ChannelName: "experimental-canary"}}
	formatUserLogs(logs, 0)

	assert.Equal(t, "", logs[0].ChannelName, "内部名不得泄露给用户")
}

// 批量混合场景
func TestFormatUserLogs_MixedChannels(t *testing.T) {
	db := useChannelPublicNameTestDB(t)
	require.NoError(t, db.Create(&Channel{Id: 1, Name: "cy", PublicName: strPtr("中云"), Key: "k", Type: 58, Status: 1}).Error)
	require.NoError(t, db.Create(&Channel{Id: 2, Name: "fl", PublicName: strPtr("FlagOS"), Key: "k", Type: 1, Status: 1}).Error)
	require.NoError(t, db.Create(&Channel{Id: 3, Name: "hidden", Key: "k", Type: 1, Status: 1}).Error)

	logs := []*Log{
		{ChannelId: 1}, {ChannelId: 2}, {ChannelId: 3}, {ChannelId: 0},
	}
	formatUserLogs(logs, 0)

	assert.Equal(t, "中云", logs[0].ChannelName)
	assert.Equal(t, "FlagOS", logs[1].ChannelName)
	assert.Equal(t, "", logs[2].ChannelName)
	assert.Equal(t, "", logs[3].ChannelName, "无渠道的日志不应 panic 或报错")
}
