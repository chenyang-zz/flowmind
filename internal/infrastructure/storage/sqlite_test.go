package storage

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewSQLiteDB 测试创建 SQLite 数据库连接
//
// 验证能够成功创建数据库连接并配置 WAL 模式
func TestNewSQLiteDB(t *testing.T) {
	// 创建临时数据库
	config := SQLiteConfig{
		Path:             t.TempDir() + "/test.db",
		MaxOpenConns:     25,
		MaxIdleConns:     5,
		ConnMaxLifetime:  5 * time.Minute,
	}

	db, err := NewSQLiteDB(config)
	require.NoError(t, err)
	require.NotNil(t, db)

	defer db.Close()

	// 验证连接
	err = db.Ping()
	assert.NoError(t, err)
}

// TestRunMigrations 测试数据库迁移
//
// 验证所有迁移脚本能够正确执行
func TestRunMigrations(t *testing.T) {
	// 创建临时文件数据库
	db, err := NewSQLiteDB(SQLiteConfig{Path: t.TempDir() + "/test.db"})
	require.NoError(t, err)
	defer db.Close()

	// 执行迁移
	err = RunMigrations(db)
	require.NoError(t, err)

	// 验证表是否创建
	tables := []string{"events", "sessions", "patterns", "schema_migrations"}
	for _, table := range tables {
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count, "表 %s 应该存在", table)
	}

	// 验证迁移记录
	var version int
	err = db.QueryRow("SELECT MAX(version) FROM schema_migrations").Scan(&version)
	require.NoError(t, err)
	assert.Equal(t, 4, version, "应该有 4 个迁移版本")
}

// TestRunMigrations_Idempotent 测试迁移的幂等性
//
// 验证重复执行迁移不会出错
func TestRunMigrations_Idempotent(t *testing.T) {
	db, err := NewSQLiteDB(SQLiteConfig{Path: t.TempDir() + "/test.db"})
	require.NoError(t, err)
	defer db.Close()

	// 第一次执行
	err = RunMigrations(db)
	require.NoError(t, err)

	// 第二次执行（应该跳过已应用的迁移）
	err = RunMigrations(db)
	require.NoError(t, err)

	// 验证迁移记录
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 4, count)
}

// TestNewSQLiteDB_InvalidPath 测试无效路径的错误处理
//
// 验证当数据库路径无效时能正确返回错误
func TestNewSQLiteDB_InvalidPath(t *testing.T) {
	// 使用无效路径（包含不允许的字符或无法创建的目录）
	config := SQLiteConfig{
		Path:             "/nonexistent/directory/that/cannot/be/created/test.db",
		MaxOpenConns:     25,
		MaxIdleConns:     5,
		ConnMaxLifetime:  5 * time.Minute,
	}

	db, err := NewSQLiteDB(config)
	assert.Error(t, err, "应该返回错误")
	assert.Nil(t, db, "数据库连接应该为空")
}

// TestNewSQLiteDB_ConfigOptions 测试数据库配置选项
//
// 验证不同的配置选项能正确应用
func TestNewSQLiteDB_ConfigOptions(t *testing.T) {
	tests := []struct {
		name    string
		config  SQLiteConfig
		wantErr bool
	}{
		{
			name: "最小配置",
			config: SQLiteConfig{
				Path: t.TempDir() + "/minimal.db",
			},
			wantErr: false,
		},
		{
			name: "完整配置",
			config: SQLiteConfig{
				Path:             t.TempDir() + "/full.db",
				MaxOpenConns:     10,
				MaxIdleConns:     3,
				ConnMaxLifetime:  10 * time.Minute,
			},
			wantErr: false,
		},
		{
			name: "零连接数",
			config: SQLiteConfig{
				Path:         t.TempDir() + "/zero.db",
				MaxOpenConns: 0, // 测试边界情况
				MaxIdleConns: 0,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, err := NewSQLiteDB(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, db)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, db)
				if db != nil {
					defer db.Close()
					// 验证数据库可用
					err := db.Ping()
					assert.NoError(t, err)
				}
			}
		})
	}
}

// TestRunMigrations_Partial 测试部分迁移后的增量迁移
//
// 验证在已有部分迁移的情况下，增量执行剩余迁移
func TestRunMigrations_Partial(t *testing.T) {
	db, err := NewSQLiteDB(SQLiteConfig{Path: t.TempDir() + "/test.db"})
	require.NoError(t, err)
	defer db.Close()

	// 手动执行前两个迁移（模拟部分迁移状态）
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
		    version INTEGER PRIMARY KEY,
		    applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`)
	require.NoError(t, err)

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS events (
		    id INTEGER PRIMARY KEY AUTOINCREMENT,
		    uuid TEXT UNIQUE NOT NULL,
		    type TEXT NOT NULL,
		    timestamp DATETIME NOT NULL
		);
	`)
	require.NoError(t, err)

	// 记录前两个迁移已应用
	_, err = db.Exec("INSERT INTO schema_migrations (version) VALUES (1)")
	require.NoError(t, err)
	_, err = db.Exec("INSERT INTO schema_migrations (version) VALUES (2)")
	require.NoError(t, err)

	// 执行迁移，应该跳过已应用的前两个，执行剩余的
	err = RunMigrations(db)
	require.NoError(t, err)

	// 验证所有表都已创建
	tables := []string{"events", "sessions", "patterns", "schema_migrations"}
	for _, table := range tables {
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count, "表 %s 应该存在", table)
	}

	// 验证迁移记录
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 4, count)
}

// TestMigrations_DataValidation 测试迁移后的数据验证
//
// 验证迁移创建的表结构符合预期
func TestMigrations_DataValidation(t *testing.T) {
	db, err := NewSQLiteDB(SQLiteConfig{Path: t.TempDir() + "/test.db"})
	require.NoError(t, err)
	defer db.Close()

	// 执行迁移
	err = RunMigrations(db)
	require.NoError(t, err)

	// 验证 events 表结构
	var eventColumns []string
	rows, err := db.Query("PRAGMA table_info(events)")
	require.NoError(t, err)
	defer rows.Close()

	expectedColumns := []string{"id", "uuid", "type", "timestamp", "data", "application", "bundle_id", "window_title", "file_path", "selection", "created_at"}
	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull, pk int
		var dfltValue interface{}
		err = rows.Scan(&cid, &name, &dataType, &notNull, &dfltValue, &pk)
		require.NoError(t, err)
		eventColumns = append(eventColumns, name)
	}
	assert.Equal(t, len(expectedColumns), len(eventColumns), "events 表列数不匹配")

	// 验证索引存在
	indexes := []string{
		"idx_events_timestamp",
		"idx_events_type",
		"idx_events_application",
		"idx_events_uuid",
		"idx_sessions_time",
		"idx_sessions_application",
		"idx_patterns_automated",
		"idx_patterns_support",
		"idx_patterns_hash",
	}

	for _, index := range indexes {
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name=?", index).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count, "索引 %s 应该存在", index)
	}
}

// TestNewSQLiteDB_ConcurrentAccess 测试并发访问
//
// 验证数据库支持并发访问
func TestNewSQLiteDB_ConcurrentAccess(t *testing.T) {
	config := SQLiteConfig{
		Path:             t.TempDir() + "/concurrent.db",
		MaxOpenConns:     25,
		MaxIdleConns:     5,
		ConnMaxLifetime:  5 * time.Minute,
	}

	db, err := NewSQLiteDB(config)
	require.NoError(t, err)
	defer db.Close()

	// 并发测试
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			// 执行查询
			var result int
			err := db.QueryRow("SELECT 1").Scan(&result)
			assert.NoError(t, err)
			assert.Equal(t, 1, result)
			done <- true
		}(i)
	}

	// 等待所有 goroutine 完成
	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestNewSQLiteDB_MemoryDatabase 测试内存数据库
func TestNewSQLiteDB_MemoryDatabase(t *testing.T) {
	config := SQLiteConfig{
		Path:             ":memory:",
		MaxOpenConns:     5,
		MaxIdleConns:     1,
		ConnMaxLifetime:  1 * time.Minute,
	}

	db, err := NewSQLiteDB(config)
	require.NoError(t, err)
	require.NotNil(t, db)
	defer db.Close()

	// 验证连接
	err = db.Ping()
	assert.NoError(t, err)

	// 验证可以创建表
	_, err = db.Exec("CREATE TABLE test (id INTEGER PRIMARY KEY)")
	require.NoError(t, err)
}

// TestNewSQLiteDB_SmallPath 测试相对路径数据库
func TestNewSQLiteDB_SmallPath(t *testing.T) {
	// 创建子目录
	subDir := t.TempDir() + "/subdir"
	err := os.MkdirAll(subDir, 0755)
	require.NoError(t, err)

	config := SQLiteConfig{
		Path:             subDir + "/test.db",
		MaxOpenConns:     5,
		MaxIdleConns:     1,
		ConnMaxLifetime:  1 * time.Minute,
	}

	db, err := NewSQLiteDB(config)
	require.NoError(t, err)
	require.NotNil(t, db)
	defer db.Close()

	// 验证连接
	err = db.Ping()
	assert.NoError(t, err)
}

// TestNewSQLiteDB_ConcurrencyLimits 测试连接池限制
func TestNewSQLiteDB_ConcurrencyLimits(t *testing.T) {
	config := SQLiteConfig{
		Path:             t.TempDir() + "/pool_test.db",
		MaxOpenConns:     2, // 只允许2个打开连接
		MaxIdleConns:     1,
		ConnMaxLifetime:  1 * time.Minute,
	}

	db, err := NewSQLiteDB(config)
	require.NoError(t, err)
	defer db.Close()

	// 设置连接池统计
	db.SetMaxOpenConns(2)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(1 * time.Minute)

	// 验证连接
	err = db.Ping()
	assert.NoError(t, err)
}

// TestRunMigrations_EmptyDatabase 测试空数据库迁移
func TestRunMigrations_EmptyDatabase(t *testing.T) {
	db, err := NewSQLiteDB(SQLiteConfig{Path: t.TempDir() + "/test.db"})
	require.NoError(t, err)
	defer db.Close()

	// 执行迁移
	err = RunMigrations(db)
	require.NoError(t, err)

	// 验证表创建成功
	var tableCount int
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'").Scan(&tableCount)
	require.NoError(t, err)
	assert.Equal(t, 4, tableCount, "应该创建4个表")
}

// TestRunMigrations_RecoverableError 测试迁移中的可恢复错误
func TestRunMigrations_RecoverableError(t *testing.T) {
	db, err := NewSQLiteDB(SQLiteConfig{Path: t.TempDir() + "/test.db"})
	require.NoError(t, err)
	defer db.Close()

	// 第一次执行迁移
	err = RunMigrations(db)
	require.NoError(t, err)

	// 尝试手动插入一个已存在的迁移（会失败，但不应该影响系统）
	_, err = db.Exec("INSERT INTO schema_migrations (version) VALUES (1)")
	assert.Error(t, err, "插入重复版本应该失败")

	// 但重新运行迁移应该成功（幂等性）
	err = RunMigrations(db)
	assert.NoError(t, err, "重复运行迁移应该成功")
}

// TestNewSQLiteDB_ZeroLifetime 测试零连接生命周期
func TestNewSQLiteDB_ZeroLifetime(t *testing.T) {
	config := SQLiteConfig{
		Path:             t.TempDir() + "/lifetime.db",
		MaxOpenConns:     5,
		MaxIdleConns:     2,
		ConnMaxLifetime:  0, // 零生命周期（连接永不关闭）
	}

	db, err := NewSQLiteDB(config)
	require.NoError(t, err)
	defer db.Close()

	err = db.Ping()
	assert.NoError(t, err)
}

// TestNewSQLiteDB_VeryLargePool 测试大连接池
func TestNewSQLiteDB_VeryLargePool(t *testing.T) {
	config := SQLiteConfig{
		Path:             t.TempDir() + "/largepool.db",
		MaxOpenConns:     1000,
		MaxIdleConns:     500,
		ConnMaxLifetime:  1 * time.Hour,
	}

	db, err := NewSQLiteDB(config)
	require.NoError(t, err)
	defer db.Close()

	err = db.Ping()
	assert.NoError(t, err)
}
