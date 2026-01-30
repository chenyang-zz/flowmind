
package storage

import (
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRunMigrations_DuplicateVersion 测试重复迁移版本
func TestRunMigrations_DuplicateVersion(t *testing.T) {
	db, err := NewSQLiteDB(SQLiteConfig{Path: t.TempDir() + "/test.db"})
	require.NoError(t, err)
	defer db.Close()

	// 执行第一次迁移
	err = RunMigrations(db)
	require.NoError(t, err)

	// 手动插入重复的迁移记录（模拟不完整的状态）
	_, err = db.Exec("INSERT INTO schema_migrations (version) VALUES (5)")
	assert.NoError(t, err)

	// 再次运行迁移应该跳过已应用的
	err = RunMigrations(db)
	assert.NoError(t, err)
}

// TestRunMigrations_AllMigrationsApplied 测试所有迁移已应用
func TestRunMigrations_AllMigrationsApplied(t *testing.T) {
	db, err := NewSQLiteDB(SQLiteConfig{Path: t.TempDir() + "/test.db"})
	require.NoError(t, err)
	defer db.Close()

	// 执行迁移
	err = RunMigrations(db)
	require.NoError(t, err)

	// 验证所有4个迁移都已应用
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 4, count)

	// 再次执行应该跳过所有迁移
	err = RunMigrations(db)
	assert.NoError(t, err)

	// 验证没有重复的迁移记录
	err = db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 4, count)
}

// TestRunMigrations_ConnectionClosed 测试数据库连接关闭的情况
func TestRunMigrations_ConnectionClosed(t *testing.T) {
	db, err := NewSQLiteDB(SQLiteConfig{Path: t.TempDir() + "/test.db"})
	require.NoError(t, err)

	// 关闭数据库连接
	db.Close()

	// 尝试运行迁移应该失败
	err = RunMigrations(db)
	assert.Error(t, err, "关闭的数据库连接应该返回错误")
}
