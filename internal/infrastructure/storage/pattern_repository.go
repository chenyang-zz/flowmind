package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/chenyang-zz/flowmind/internal/domain/models"
	"github.com/chenyang-zz/flowmind/internal/infrastructure/logger"
	"go.uber.org/zap"
)

/**
 * SQLitePatternRepository SQLite 模式仓储实现
 */
type SQLitePatternRepository struct {
	db *sql.DB
}

/**
 * NewSQLitePatternRepository 创建 SQLite 模式仓储
 *
 * Parameters:
 *   - db: 数据库连接
 *
 * Returns: *SQLitePatternRepository - 模式仓储实例
 */
func NewSQLitePatternRepository(db *sql.DB) *SQLitePatternRepository {
	return &SQLitePatternRepository{db: db}
}

/**
 * Save 保存模式
 *
 * Parameters:
 *   - pattern: 模式对象
 *
 * Returns: error - 错误信息
 */
func (r *SQLitePatternRepository) Save(pattern *models.Pattern) error {
	query := `
		INSERT INTO patterns (uuid, name, sequence_hash, sequence, support_count,
			confidence, first_seen, last_seen, is_automated, ai_analysis,
			estimated_time_saving)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	// 序列化模式序列
	sequenceJSON, err := json.Marshal(pattern.Sequence)
	if err != nil {
		return fmt.Errorf("序列化模式序列失败: %w", err)
	}

	// 序列化 AI 分析结果
	var aiAnalysisJSON []byte
	if pattern.AIAnalysis != nil {
		aiAnalysisJSON, err = json.Marshal(pattern.AIAnalysis)
		if err != nil {
			return fmt.Errorf("序列化 AI 分析结果失败: %w", err)
		}
	}

	var name sql.NullString
	if pattern.Description != "" {
		name.String = pattern.Description
		name.Valid = true
	}

	// 计算序列哈希
	sequenceHash := r.calculateSequenceHash(pattern.Sequence)

	_, err = r.db.Exec(
		query,
		pattern.ID,
		name,
		sequenceHash,
		string(sequenceJSON),
		pattern.SupportCount,
		pattern.Confidence,
		pattern.FirstSeen,
		pattern.LastSeen,
		pattern.IsAutomated,
		string(aiAnalysisJSON),
		func() int64 {
			if pattern.AIAnalysis != nil {
				return pattern.AIAnalysis.EstimatedTimeSaving
			}
			return 0
		}(),
	)

	if err != nil {
		logger.Error("保存模式失败",
			zap.String("pattern_id", pattern.ID),
			zap.Error(err))
		return fmt.Errorf("保存模式失败: %w", err)
	}

	logger.Debug("模式已保存",
		zap.String("pattern_id", pattern.ID),
		zap.Int("support_count", pattern.SupportCount))

	return nil
}

/**
 * SaveBatch 批量保存模式
 *
 * Parameters:
 *   - patterns: 模式数组
 *
 * Returns: error - 错误信息
 */
func (r *SQLitePatternRepository) SaveBatch(patterns []*models.Pattern) error {
	if len(patterns) == 0 {
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
		INSERT INTO patterns (uuid, name, sequence_hash, sequence, support_count,
			confidence, first_seen, last_seen, is_automated, ai_analysis,
			estimated_time_saving)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("准备语句失败: %w", err)
	}
	defer stmt.Close()

	// 批量插入
	for _, pattern := range patterns {
		// 序列化模式序列
		sequenceJSON, err := json.Marshal(pattern.Sequence)
		if err != nil {
			logger.Error("序列化模式序列失败",
				zap.String("pattern_id", pattern.ID),
				zap.Error(err))
			continue
		}

		// 序列化 AI 分析结果
		var aiAnalysisJSON []byte
		if pattern.AIAnalysis != nil {
			aiAnalysisJSON, err = json.Marshal(pattern.AIAnalysis)
			if err != nil {
				logger.Error("序列化 AI 分析结果失败",
					zap.String("pattern_id", pattern.ID),
					zap.Error(err))
				continue
			}
		}

		var name sql.NullString
		if pattern.Description != "" {
			name.String = pattern.Description
			name.Valid = true
		}

		// 计算序列哈希
		sequenceHash := r.calculateSequenceHash(pattern.Sequence)

		_, err = stmt.Exec(
			pattern.ID,
			name,
			sequenceHash,
			string(sequenceJSON),
			pattern.SupportCount,
			pattern.Confidence,
			pattern.FirstSeen,
			pattern.LastSeen,
			pattern.IsAutomated,
			string(aiAnalysisJSON),
			func() int64 {
				if pattern.AIAnalysis != nil {
					return pattern.AIAnalysis.EstimatedTimeSaving
				}
				return 0
			}(),
		)

		if err != nil {
			logger.Error("插入模式失败",
				zap.String("pattern_id", pattern.ID),
				zap.Error(err))
			return fmt.Errorf("插入模式失败: %w", err)
		}
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}

	logger.Debug("批量保存模式成功",
		zap.Int("count", len(patterns)))

	return nil
}

/**
 * FindByID 根据ID查询模式
 *
 * Parameters:
 *   - id: 模式ID
 *
 * Returns: *models.Pattern - 模式对象, error - 错误信息
 */
func (r *SQLitePatternRepository) FindByID(id string) (*models.Pattern, error) {
	query := `
		SELECT uuid, name, sequence, support_count, confidence, first_seen, last_seen,
			is_automated, ai_analysis, estimated_time_saving
		FROM patterns
		WHERE uuid = ?
	`

	var name sql.NullString
	var sequenceJSON, aiAnalysisJSON string
	var estimatedTimeSaving int64
	var pattern models.Pattern

	err := r.db.QueryRow(query, id).Scan(
		&pattern.ID,
		&name,
		&sequenceJSON,
		&pattern.SupportCount,
		&pattern.Confidence,
		&pattern.FirstSeen,
		&pattern.LastSeen,
		&pattern.IsAutomated,
		&aiAnalysisJSON,
		&estimatedTimeSaving,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("模式不存在: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("查询模式失败: %w", err)
	}

	// 反序列化序列
	if err := json.Unmarshal([]byte(sequenceJSON), &pattern.Sequence); err != nil {
		logger.Error("反序列化模式序列失败",
			zap.String("pattern_id", pattern.ID),
			zap.Error(err))
		return nil, fmt.Errorf("反序列化模式序列失败: %w", err)
	}

	// 设置描述
	if name.Valid {
		pattern.Description = name.String
	}

	// 反序列化 AI 分析结果
	if aiAnalysisJSON != "" {
		if err := json.Unmarshal([]byte(aiAnalysisJSON), &pattern.AIAnalysis); err != nil {
			logger.Warn("反序列化 AI 分析结果失败",
				zap.String("pattern_id", pattern.ID),
				zap.Error(err))
		}
	}

	return &pattern, nil
}

/**
 * FindAll 查询所有模式
 *
 * Returns: []*models.Pattern - 模式列表, error - 错误信息
 */
func (r *SQLitePatternRepository) FindAll() ([]*models.Pattern, error) {
	query := `
		SELECT uuid, name, sequence, support_count, confidence, first_seen, last_seen,
			is_automated, ai_analysis, estimated_time_saving
		FROM patterns
		ORDER BY support_count DESC
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("查询所有模式失败: %w", err)
	}
	defer rows.Close()

	return r.scanPatterns(rows)
}

/**
 * FindUnanalyzed 查询未分析的模式
 *
 * Returns: []*models.Pattern - 未分析的模式列表, error - 错误信息
 */
func (r *SQLitePatternRepository) FindUnanalyzed() ([]*models.Pattern, error) {
	query := `
		SELECT uuid, name, sequence, support_count, confidence, first_seen, last_seen,
			is_automated, ai_analysis, estimated_time_saving
		FROM patterns
		WHERE ai_analysis IS NULL OR ai_analysis = ''
		ORDER BY support_count DESC
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("查询未分析模式失败: %w", err)
	}
	defer rows.Close()

	return r.scanPatterns(rows)
}

/**
 * Update 更新模式
 *
 * Parameters:
 *   - pattern: 模式对象
 *
 * Returns: error - 错误信息
 */
func (r *SQLitePatternRepository) Update(pattern *models.Pattern) error {
	query := `
		UPDATE patterns
		SET is_automated = ?, ai_analysis = ?, estimated_time_saving = ?
		WHERE uuid = ?
	`

	// 序列化 AI 分析结果
	var aiAnalysisJSON []byte
	if pattern.AIAnalysis != nil {
		var err error
		aiAnalysisJSON, err = json.Marshal(pattern.AIAnalysis)
		if err != nil {
			return fmt.Errorf("序列化 AI 分析结果失败: %w", err)
		}
	}

	result, err := r.db.Exec(
		query,
		pattern.IsAutomated,
		string(aiAnalysisJSON),
		func() int64 {
			if pattern.AIAnalysis != nil {
				return pattern.AIAnalysis.EstimatedTimeSaving
			}
			return 0
		}(),
		pattern.ID,
	)

	if err != nil {
		logger.Error("更新模式失败",
			zap.String("pattern_id", pattern.ID),
			zap.Error(err))
		return fmt.Errorf("更新模式失败: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("模式不存在: %s", pattern.ID)
	}

	logger.Debug("模式已更新",
		zap.String("pattern_id", pattern.ID))

	return nil
}

/**
 * Delete 删除模式
 *
 * Parameters:
 *   - id: 模式ID
 *
 * Returns: error - 错误信息
 */
func (r *SQLitePatternRepository) Delete(id string) error {
	result, err := r.db.Exec("DELETE FROM patterns WHERE uuid = ?", id)
	if err != nil {
		return fmt.Errorf("删除模式失败: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("模式不存在: %s", id)
	}

	logger.Debug("模式已删除", zap.String("pattern_id", id))
	return nil
}

/**
 * scanPatterns 扫描模式行并转换为模式对象
 *
 * Parameters:
 *   - rows: 查询结果集
 *
 * Returns: []*models.Pattern - 模式列表, error - 错误信息
 */
func (r *SQLitePatternRepository) scanPatterns(rows *sql.Rows) ([]*models.Pattern, error) {
	var patterns []*models.Pattern

	for rows.Next() {
		var pattern models.Pattern
		var name sql.NullString
		var sequenceJSON, aiAnalysisJSON string
		var estimatedTimeSaving int64

		err := rows.Scan(
			&pattern.ID,
			&name,
			&sequenceJSON,
			&pattern.SupportCount,
			&pattern.Confidence,
			&pattern.FirstSeen,
			&pattern.LastSeen,
			&pattern.IsAutomated,
			&aiAnalysisJSON,
			&estimatedTimeSaving,
		)

		if err != nil {
			return nil, fmt.Errorf("扫描模式行失败: %w", err)
		}

		// 反序列化序列
		if err := json.Unmarshal([]byte(sequenceJSON), &pattern.Sequence); err != nil {
			logger.Error("反序列化模式序列失败",
				zap.String("pattern_id", pattern.ID),
				zap.Error(err))
			continue
		}

		// 设置描述
		if name.Valid {
			pattern.Description = name.String
		}

		// 反序列化 AI 分析结果
		if aiAnalysisJSON != "" {
			if err := json.Unmarshal([]byte(aiAnalysisJSON), &pattern.AIAnalysis); err != nil {
				logger.Warn("反序列化 AI 分析结果失败",
					zap.String("pattern_id", pattern.ID),
					zap.Error(err))
			}
		}

		patterns = append(patterns, &pattern)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历模式行失败: %w", err)
	}

	return patterns, nil
}

/**
 * calculateSequenceHash 计算模式序列的哈希值
 *
 * Parameters:
 *   - sequence: 事件步骤序列
 *
 * Returns: string - 哈希值
 */
func (r *SQLitePatternRepository) calculateSequenceHash(sequence []models.EventStep) string {
	var hashStr string
	for _, step := range sequence {
		hashStr += string(step.Type) + ":" + step.Action + "|"
	}
	return hashStr
}
