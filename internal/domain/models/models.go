/**
 * Package models 定义模式识别引擎的领域模型
 *
 * 包含会话、模式、事件步骤等核心数据结构
 */

package models

import (
	"time"

	"github.com/chenyang-zz/flowmind/pkg/events"
)

/**
 * Session 用户会话
 *
 * 表示用户在一段时间内的连续操作序列
 * 用于将大量原始事件划分为有意义的会话单元
 */
type Session struct {
	// ID 会话唯一标识
	ID string

	// StartTime 会话开始时间
	StartTime time.Time

	// EndTime 会话结束时间（nil表示会话仍在进行）
	EndTime *time.Time

	// Application 应用名称
	Application string

	// BundleID 应用Bundle ID
	BundleID string

	// EventCount 会话包含的事件数量
	EventCount int

	// Events 会话包含的事件列表（按时间顺序）
	Events []events.Event
}

/**
 * IsCompleted 判断会话是否已完成
 *
 * Returns: bool - true表示会话已结束，false表示进行中
 */
func (s *Session) IsCompleted() bool {
	return s.EndTime != nil
}

/**
 * Duration 计算会话持续时间
 *
 * Returns: time.Duration - 会话持续时间，进行中的会话返回从开始到现在的时间
 */
func (s *Session) Duration() time.Duration {
	if s.EndTime != nil {
		return s.EndTime.Sub(s.StartTime)
	}
	return time.Since(s.StartTime)
}

/**
 * EventStep 事件步骤
 *
 * 将原始事件抽象为标准化的步骤，用于模式挖掘
 */
type EventStep struct {
	// Type 事件类型
	Type events.EventType

	// Action 抽象动作（如"keypress"、"clipboard_copy"）
	Action string

	// Context 步骤上下文信息
	Context *StepContext
}

/**
 * StepContext 步骤上下文
 *
 * 提取事件的关键上下文信息
 */
type StepContext struct {
	// Application 应用名称
	Application string

	// BundleID 应用Bundle ID
	BundleID string

	// PatternValue 模式值（用于泛化）
	// 例如：具体的按键码泛化为"字母键"、"功能键"等
	PatternValue string
}

/**
 * Pattern 重复操作模式
 *
 * 表示用户行为中发现的重复序列，可能适合自动化
 */
type Pattern struct {
	// ID 模式唯一标识
	ID string

	// Sequence 事件步骤序列（核心模式）
	Sequence []EventStep

	// SupportCount 支持计数（出现次数）
	SupportCount int

	// Confidence 置信度（0-1之间）
	Confidence float64

	// FirstSeen 首次发现时间
	FirstSeen time.Time

	// LastSeen 最后发现时间
	LastSeen time.Time

	// IsAutomated 是否已自动化
	IsAutomated bool

	// AIAnalysis AI分析结果
	AIAnalysis *AIAnalysis
}

/**
 * Length 获取模式长度（步骤数）
 *
 * Returns: int - 模式包含的步骤数
 */
func (p *Pattern) Length() int {
	return len(p.Sequence)
}

/**
 * Frequency 计算模式频率（每小时出现次数）
 *
 * Returns: float64 - 每小时出现次数
 */
func (p *Pattern) Frequency() float64 {
	duration := p.LastSeen.Sub(p.FirstSeen).Hours()
	if duration == 0 {
		return 0
	}
	return float64(p.SupportCount) / duration
}

/**
 * AIAnalysis AI分析结果
 *
 * 包含AI对模式的评估和建议
 */
type AIAnalysis struct {
	// ShouldAutomate 是否值得自动化
	ShouldAutomate bool

	// Reason 原因说明
	Reason string

	// EstimatedTimeSaving 预计节省时间（秒）
	EstimatedTimeSaving int64

	// Complexity 实现复杂度（low/medium/high）
	Complexity string

	// SuggestedName 建议的自动化名称
	SuggestedName string

	// SuggestedSteps 建议的自动化步骤
	SuggestedSteps []string

	// AnalyzedAt 分析时间
	AnalyzedAt time.Time
}

/**
 * SessionRepository 会话仓储接口
 *
 * 定义会话持久化的操作
 */
type SessionRepository interface {
	// Save 保存会话
	Save(session *Session) error

	// FindByID 根据ID查询会话
	FindByID(id string) (*Session, error)

	// FindByTimeRange 按时间范围查询会话
	FindByTimeRange(start, end time.Time) ([]*Session, error)

	// FindByApplication 按应用查询会话
	FindByApplication(application string, limit int) ([]*Session, error)

	// DeleteOlderThan 删除旧会话
	DeleteOlderThan(cutoff time.Time) (int64, error)
}

/**
 * PatternRepository 模式仓储接口
 *
 * 定义模式持久化的操作
 */
type PatternRepository interface {
	// Save 保存模式
	Save(pattern *Pattern) error

	// SaveBatch 批量保存模式
	SaveBatch(patterns []*Pattern) error

	// FindByID 根据ID查询模式
	FindByID(id string) (*Pattern, error)

	// FindAll 查询所有模式
	FindAll() ([]*Pattern, error)

	// FindUnanalyzed 查询未分析的模式
	FindUnanalyzed() ([]*Pattern, error)

	// Update 更新模式
	Update(pattern *Pattern) error

	// Delete 删除模式
	Delete(id string) error
}
