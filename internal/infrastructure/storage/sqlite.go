/**
 * Package storage 提供数据持久化功能
 *
 * 负责将监控事件和分析结果持久化到数据库
 */

package storage

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3" // SQLite 驱动
	"github.com/chenyang-zz/flowmind/internal/infrastructure/logger"
	"go.uber.org/zap"
)

/**
 * SQLiteConfig SQLite 配置
 */
type SQLiteConfig struct {
	// Path 数据库文件路径
	Path string

	// MaxOpenConns 最大打开连接数
	MaxOpenConns int

	// MaxIdleConns 最大空闲连接数
	MaxIdleConns int

	// ConnMaxLifetime 连接最大生命周期
	ConnMaxLifetime time.Duration
}

/**
 * NewSQLiteDB 创建 SQLite 数据库连接
 *
 * 配置 WAL 模式以提升并发性能，优化连接池参数
 *
 * Parameters:
 *   - config: SQLite 配置
 *
 * Returns: *sql.DB - 数据库连接实例, error - 错误信息
 */
func NewSQLiteDB(config SQLiteConfig) (*sql.DB, error) {
	logger.Info("创建 SQLite 数据库连接",
		zap.String("path", config.Path),
	)

	// 打开数据库连接
	// 对于内存数据库，使用 file::memory:?mode=memory&cache=shared 来启用共享缓存
	dataSourceName := config.Path
	if config.Path == ":memory:" {
		dataSourceName = "file::memory:?mode=memory&cache=shared"
	}

	db, err := sql.Open("sqlite3", dataSourceName)
	if err != nil {
		logger.Error("打开数据库失败", zap.Error(err))
		return nil, fmt.Errorf("打开数据库失败: %w", err)
	}

	// 配置连接池
	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)
	db.SetConnMaxLifetime(config.ConnMaxLifetime)

	// 只有非内存数据库才配置WAL模式
	if config.Path != ":memory:" {
		// 启用 WAL 模式 (Write-Ahead Logging)
		// WAL 模式允许并发读写，提升性能
		if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
			logger.Error("配置 WAL 模式失败", zap.Error(err))
			return nil, fmt.Errorf("配置 WAL 模式失败: %w", err)
		}

		// 配置同步模式为 NORMAL (在性能和安全性之间平衡)
		if _, err := db.Exec("PRAGMA synchronous=NORMAL"); err != nil {
			logger.Error("配置同步模式失败", zap.Error(err))
			return nil, fmt.Errorf("配置同步模式失败: %w", err)
		}

		// 配置缓存大小 (10MB 页缓存)
		if _, err := db.Exec("PRAGMA cache_size=10000"); err != nil {
			logger.Error("配置缓存大小失败", zap.Error(err))
			return nil, fmt.Errorf("配置缓存大小失败: %w", err)
		}
	}

	// 验证连接
	if err := db.Ping(); err != nil {
		logger.Error("数据库连接验证失败", zap.Error(err))
		return nil, fmt.Errorf("数据库连接验证失败: %w", err)
	}

	logger.Info("SQLite 数据库连接成功")
	return db, nil
}
