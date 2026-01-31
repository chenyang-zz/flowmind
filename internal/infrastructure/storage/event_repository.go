package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/chenyang-zz/flowmind/pkg/events"
	"github.com/chenyang-zz/flowmind/internal/infrastructure/logger"
	"go.uber.org/zap"
)

/**
 * EventRepository 事件存储接口
 *
 * 定义事件持久化的所有操作
 */
type EventRepository interface {
	// Save 保存单个事件
	Save(event events.Event) error

	// SaveBatch 批量保存事件（性能优化）
	SaveBatch(eventList []events.Event) error

	// FindByTimeRange 按时间范围查询
	FindByTimeRange(start, end time.Time) ([]events.Event, error)

	// FindRecent 查询最近的事件
	FindRecent(limit int) ([]events.Event, error)

	// FindByType 按类型查询
	FindByType(eventType events.EventType, limit int) ([]events.Event, error)

	// DeleteOlderThan 删除旧数据
	DeleteOlderThan(cutoff time.Time) (int64, error)

	// GetStats 获取统计信息
	GetStats() (*EventStats, error)
}

/**
 * EventStats 事件统计信息
 */
type EventStats struct {
	// TotalCount 总事件数
	TotalCount int64

	// CountByType 按类型统计
	CountByType map[string]int64

	// OldestEvent 最旧的事件时间
	OldestEvent *time.Time

	// NewestEvent 最新的事件时间
	NewestEvent *time.Time
}

/**
 * SQLiteEventRepository SQLite 事件仓储实现
 */
type SQLiteEventRepository struct {
	db *sql.DB
}

/**
 * NewSQLiteEventRepository 创建 SQLite 事件仓储
 *
 * Parameters:
 *   - db: 数据库连接
 *
 * Returns: *SQLiteEventRepository - 事件仓储实例
 */
func NewSQLiteEventRepository(db *sql.DB) *SQLiteEventRepository {
	return &SQLiteEventRepository{db: db}
}

/**
 * Save 保存单个事件
 *
 * Parameters:
 *   - event: 事件对象
 *
 * Returns: error - 错误信息
 */
func (r *SQLiteEventRepository) Save(event events.Event) error {
	query := `
		INSERT INTO events (uuid, type, timestamp, data, application, bundle_id, window_title, file_path, selection)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	// 序列化事件数据为 JSON
	dataJSON, err := json.Marshal(event.Data)
	if err != nil {
		return fmt.Errorf("序列化事件数据失败: %w", err)
	}

	// 提取上下文信息
	var application, bundleID, windowTitle, filePath, selection string
	if event.Context != nil {
		application = event.Context.Application
		bundleID = event.Context.BundleID
		windowTitle = event.Context.WindowTitle
		filePath = event.Context.FilePath
		selection = event.Context.Selection
	}

	_, err = r.db.Exec(
		query,
		event.ID,
		event.Type,
		event.Timestamp,
		string(dataJSON),
		application,
		bundleID,
		windowTitle,
		filePath,
		selection,
	)

	if err != nil {
		logger.Error("保存事件失败",
			zap.String("event_id", event.ID),
			zap.Error(err),
		)
		return fmt.Errorf("保存事件失败: %w", err)
	}

	return nil
}

/**
 * SaveBatch 批量保存事件
 *
 * 使用事务和预处理语句优化批量写入性能
 *
 * Parameters:
 *   - eventList: 事件数组
 *
 * Returns: error - 错误信息
 */
func (r *SQLiteEventRepository) SaveBatch(eventList []events.Event) error {
	if len(eventList) == 0 {
		return nil
	}

	// 开启事务
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}
	defer tx.Rollback()

	// 准备语句
	stmt, err := tx.Prepare(`
		INSERT INTO events (uuid, type, timestamp, data, application, bundle_id, window_title, file_path, selection)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("准备语句失败: %w", err)
	}
	defer stmt.Close()

	// 批量插入
	for _, event := range eventList {
		// 序列化事件数据
		dataJSON, err := json.Marshal(event.Data)
		if err != nil {
			logger.Error("序列化事件数据失败",
				zap.String("event_id", event.ID),
				zap.Error(err),
			)
			continue
		}

		// 提取上下文
		var application, bundleID, windowTitle, filePath, selection string
		if event.Context != nil {
			application = event.Context.Application
			bundleID = event.Context.BundleID
			windowTitle = event.Context.WindowTitle
			filePath = event.Context.FilePath
			selection = event.Context.Selection
		}

		_, err = stmt.Exec(
			event.ID,
			event.Type,
			event.Timestamp,
			string(dataJSON),
			application,
			bundleID,
			windowTitle,
			filePath,
			selection,
		)

		if err != nil {
			logger.Error("插入事件失败",
				zap.String("event_id", event.ID),
				zap.Error(err),
			)
			return fmt.Errorf("插入事件失败: %w", err)
		}
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}

	logger.Debug("批量保存事件成功",
		zap.Int("count", len(eventList)),
	)

	return nil
}

/**
 * FindByTimeRange 按时间范围查询事件
 *
 * Parameters:
 *   - start: 开始时间
 *   - end: 结束时间
 *
 * Returns: []events.Event - 事件列表, error - 错误信息
 */
func (r *SQLiteEventRepository) FindByTimeRange(start, end time.Time) ([]events.Event, error) {
	query := `
		SELECT uuid, type, timestamp, data, application, bundle_id, window_title, file_path, selection
		FROM events
		WHERE timestamp >= ? AND timestamp <= ?
		ORDER BY timestamp ASC
	`

	rows, err := r.db.Query(query, start, end)
	if err != nil {
		return nil, fmt.Errorf("查询事件失败: %w", err)
	}
	defer rows.Close()

	return r.scanEvents(rows)
}

/**
 * FindRecent 查询最近的事件
 *
 * Parameters:
 *   - limit: 返回数量限制
 *
 * Returns: []events.Event - 事件列表, error - 错误信息
 */
func (r *SQLiteEventRepository) FindRecent(limit int) ([]events.Event, error) {
	query := `
		SELECT uuid, type, timestamp, data, application, bundle_id, window_title, file_path, selection
		FROM events
		ORDER BY timestamp DESC
		LIMIT ?
	`

	rows, err := r.db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("查询最近事件失败: %w", err)
	}
	defer rows.Close()

	eventList, err := r.scanEvents(rows)
	if err != nil {
		return nil, err
	}

	// 反转顺序（从旧到新）
	for i, j := 0, len(eventList)-1; i < j; i, j = i+1, j-1 {
		eventList[i], eventList[j] = eventList[j], eventList[i]
	}

	return eventList, nil
}

/**
 * FindByType 按类型查询事件
 *
 * Parameters:
 *   - eventType: 事件类型
 *   - limit: 返回数量限制
 *
 * Returns: []events.Event - 事件列表, error - 错误信息
 */
func (r *SQLiteEventRepository) FindByType(eventType events.EventType, limit int) ([]events.Event, error) {
	query := `
		SELECT uuid, type, timestamp, data, application, bundle_id, window_title, file_path, selection
		FROM events
		WHERE type = ?
		ORDER BY timestamp DESC
		LIMIT ?
	`

	rows, err := r.db.Query(query, eventType, limit)
	if err != nil {
		return nil, fmt.Errorf("按类型查询事件失败: %w", err)
	}
	defer rows.Close()

	return r.scanEvents(rows)
}

/**
 * DeleteOlderThan 删除旧于指定时间的事件
 *
 * Parameters:
 *   - cutoff: 截止时间
 *
 * Returns: int64 - 删除的记录数, error - 错误信息
 */
func (r *SQLiteEventRepository) DeleteOlderThan(cutoff time.Time) (int64, error) {
	result, err := r.db.Exec("DELETE FROM events WHERE timestamp < ?", cutoff)
	if err != nil {
		return 0, fmt.Errorf("删除旧事件失败: %w", err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("获取删除行数失败: %w", err)
	}

	if count > 0 {
		logger.Info("删除旧事件",
			zap.Int64("count", count),
			zap.Time("cutoff", cutoff),
		)
	}

	return count, nil
}

/**
 * GetStats 获取事件统计信息
 *
 * Returns: *EventStats - 统计信息, error - 错误信息
 */
func (r *SQLiteEventRepository) GetStats() (*EventStats, error) {
	stats := &EventStats{
		CountByType: make(map[string]int64),
	}

	// 总数
	err := r.db.QueryRow("SELECT COUNT(*) FROM events").Scan(&stats.TotalCount)
	if err != nil {
		return nil, fmt.Errorf("查询总数失败: %w", err)
	}

	// 按类型统计
	rows, err := r.db.Query("SELECT type, COUNT(*) FROM events GROUP BY type")
	if err != nil {
		return nil, fmt.Errorf("按类型统计失败: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var eventType string
		var count int64
		if err := rows.Scan(&eventType, &count); err != nil {
			return nil, fmt.Errorf("扫描类型统计失败: %w", err)
		}
		stats.CountByType[eventType] = count
	}

	// 最旧和最新事件
	// 使用 SQLite 的 strftime 函数格式化为 RFC3339
	var oldestStr, newestStr sql.NullString
	err = r.db.QueryRow(`
		SELECT
			strftime('%Y-%m-%dT%H:%M:%f', MIN(timestamp)),
			strftime('%Y-%m-%dT%H:%M:%f', MAX(timestamp))
		FROM events
	`).Scan(&oldestStr, &newestStr)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("查询时间范围失败: %w", err)
	}

	// 解析时间字符串（手动构建 RFC3339 格式）
	if oldestStr.Valid {
		// SQLite 返回格式: 2026-01-30T14:20:56.729
		// 需要添加时区信息以符合 RFC3339
		oldestTime, err := time.Parse("2006-01-02T15:04:05.999", oldestStr.String)
		if err == nil {
			stats.OldestEvent = &oldestTime
		} else {
			// 如果毫秒格式解析失败，尝试不带毫秒的格式
			oldestTime, err = time.Parse("2006-01-02T15:04:05", oldestStr.String)
			if err == nil {
				stats.OldestEvent = &oldestTime
			}
		}
	}
	if newestStr.Valid {
		newestTime, err := time.Parse("2006-01-02T15:04:05.999", newestStr.String)
		if err == nil {
			stats.NewestEvent = &newestTime
		} else {
			// 如果毫秒格式解析失败，尝试不带毫秒的格式
			newestTime, err = time.Parse("2006-01-02T15:04:05", newestStr.String)
			if err == nil {
				stats.NewestEvent = &newestTime
			}
		}
	}

	return stats, nil
}

/**
 * scanEvents 扫描事件行并转换为事件对象
 *
 * Parameters:
 *   - rows: 查询结果集
 *
 * Returns: []events.Event - 事件列表, error - 错误信息
 */
func (r *SQLiteEventRepository) scanEvents(rows *sql.Rows) ([]events.Event, error) {
	var eventList []events.Event

	for rows.Next() {
		var event events.Event
		var dataJSON string
		var application, bundleID, windowTitle, filePath, selection sql.NullString

		err := rows.Scan(
			&event.ID,
			&event.Type,
			&event.Timestamp,
			&dataJSON,
			&application,
			&bundleID,
			&windowTitle,
			&filePath,
			&selection,
		)

		if err != nil {
			return nil, fmt.Errorf("扫描事件行失败: %w", err)
		}

		// 反序列化数据
		if err := json.Unmarshal([]byte(dataJSON), &event.Data); err != nil {
			logger.Error("反序列化事件数据失败",
				zap.String("event_id", event.ID),
				zap.Error(err),
			)
			event.Data = make(map[string]interface{})
		}

		// 构建上下文
		if application.Valid || bundleID.Valid || windowTitle.Valid ||
			filePath.Valid || selection.Valid {
			event.Context = &events.EventContext{}
			if application.Valid {
				event.Context.Application = application.String
			}
			if bundleID.Valid {
				event.Context.BundleID = bundleID.String
			}
			if windowTitle.Valid {
				event.Context.WindowTitle = windowTitle.String
			}
			if filePath.Valid {
				event.Context.FilePath = filePath.String
			}
			if selection.Valid {
				event.Context.Selection = selection.String
			}
		}

		eventList = append(eventList, event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历事件行失败: %w", err)
	}

	return eventList, nil
}
