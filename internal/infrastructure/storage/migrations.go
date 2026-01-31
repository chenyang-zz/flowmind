package storage

import (
	"database/sql"
	"fmt"

	"github.com/chenyang-zz/flowmind/internal/infrastructure/logger"
	"go.uber.org/zap"
)

/**
 * Migration 数据库迁移
 */
type Migration struct {
	// Version 迁移版本号
	Version int

	// Name 迁移名称
	Name string

	// SQL 迁移 SQL 语句
	SQL string
}

// 所有迁移脚本（按版本号排序）
var migrations = []Migration{
	{
		Version: 1,
		Name:    "init_schema_migrations",
		SQL: `
CREATE TABLE IF NOT EXISTS schema_migrations (
    version INTEGER PRIMARY KEY,
    applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
`,
	},
	{
		Version: 2,
		Name:    "init_events_table",
		SQL: `
CREATE TABLE IF NOT EXISTS events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    uuid TEXT UNIQUE NOT NULL,
    type TEXT NOT NULL,
    timestamp DATETIME NOT NULL,
    data JSON,
    application TEXT,
    bundle_id TEXT,
    window_title TEXT,
    file_path TEXT,
    selection TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_events_timestamp ON events(timestamp);
CREATE INDEX IF NOT EXISTS idx_events_type ON events(type);
CREATE INDEX IF NOT EXISTS idx_events_application ON events(application);
CREATE INDEX IF NOT EXISTS idx_events_uuid ON events(uuid);
`,
	},
	{
		Version: 3,
		Name:    "init_sessions_table",
		SQL: `
CREATE TABLE IF NOT EXISTS sessions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    uuid TEXT UNIQUE NOT NULL,
    application TEXT NOT NULL,
    bundle_id TEXT,
    start_time DATETIME NOT NULL,
    end_time DATETIME,
    event_count INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_sessions_time ON sessions(start_time);
CREATE INDEX IF NOT EXISTS idx_sessions_application ON sessions(application);
`,
	},
	{
		Version: 4,
		Name:    "init_patterns_table",
		SQL: `
CREATE TABLE IF NOT EXISTS patterns (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    uuid TEXT UNIQUE NOT NULL,
    name TEXT,
    sequence_hash TEXT UNIQUE NOT NULL,
    sequence JSON NOT NULL,
    support_count INTEGER DEFAULT 1,
    confidence REAL DEFAULT 0.0,
    first_seen DATETIME NOT NULL,
    last_seen DATETIME NOT NULL,
    is_automated BOOLEAN DEFAULT FALSE,
    automation_id INTEGER,
    ai_analysis TEXT,
    estimated_time_saving INTEGER,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_patterns_automated ON patterns(is_automated);
CREATE INDEX IF NOT EXISTS idx_patterns_support ON patterns(support_count);
CREATE INDEX IF NOT EXISTS idx_patterns_hash ON patterns(sequence_hash);
`,
	},
}

/**
 * RunMigrations 执行数据库迁移
 *
 * Parameters:
 *   - db: 数据库连接
 *
 * Returns: error - 错误信息
 */
func RunMigrations(db *sql.DB) error {
	logger.Info("开始执行数据库迁移")

	// 开启事务
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}

	// 获取已应用的迁移版本
	appliedVersions := make(map[int]bool)

	// 尝试查询已应用的迁移
	// 如果表不存在（首次运行），会返回错误，但我们忽略它
	rows, _ := tx.Query("SELECT version FROM schema_migrations")
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var version int
			if err := rows.Scan(&version); err != nil {
				tx.Rollback()
				return fmt.Errorf("扫描迁移版本失败: %w", err)
			}
			appliedVersions[version] = true
		}
		// 检查rows迭代是否有错误
		if err := rows.Err(); err != nil {
			tx.Rollback()
			return fmt.Errorf("遍历迁移版本失败: %w", err)
		}
	}

	// 执行未应用的迁移
	for _, migration := range migrations {
		if appliedVersions[migration.Version] {
			logger.Debug("跳过已应用的迁移",
				zap.Int("version", migration.Version),
				zap.String("name", migration.Name),
			)
			continue
		}

		logger.Info("应用迁移",
			zap.Int("version", migration.Version),
			zap.String("name", migration.Name),
		)

		// 执行迁移 SQL
		if _, err := tx.Exec(migration.SQL); err != nil {
			tx.Rollback()
			return fmt.Errorf("执行迁移 %s 失败: %w", migration.Name, err)
		}

		// 记录迁移版本
		if _, err := tx.Exec(
			"INSERT INTO schema_migrations (version) VALUES (?)",
			migration.Version,
		); err != nil {
			tx.Rollback()
			return fmt.Errorf("记录迁移版本失败: %w", err)
		}

		logger.Info("迁移应用成功",
			zap.Int("version", migration.Version),
			zap.String("name", migration.Name),
		)
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交迁移事务失败: %w", err)
	}

	logger.Info("数据库迁移完成")
	return nil
}
