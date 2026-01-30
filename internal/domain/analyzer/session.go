/**
 * Package analyzer 模式识别引擎的分析组件
 *
 * 负责会话划分、事件标准化、模式挖掘等核心功能
 */

package analyzer

import (
	"time"

	"github.com/chenyang-zz/flowmind/internal/domain/models"
	"github.com/chenyang-zz/flowmind/pkg/events"
	"github.com/google/uuid"
)

/**
 * SessionDividerConfig 会话划分器配置
 */
type SessionDividerConfig struct {
	// Timeout 会话超时时间（默认10分钟）
	Timeout time.Duration

	// MinEvents 最小事件数（默认5个）
	MinEvents int

	// SplitOnAppChange 是否在应用切换时分割会话
	SplitOnAppChange bool
}

/**
 * DefaultSessionDividerConfig 默认配置
 */
func DefaultSessionDividerConfig() SessionDividerConfig {
	return SessionDividerConfig{
		Timeout:          10 * time.Minute,
		MinEvents:        5,
		SplitOnAppChange: true,
	}
}

/**
 * SessionDivider 会话划分器
 *
 * 将原始事件流划分为有意义的会话单元
 */
type SessionDivider struct {
	config SessionDividerConfig
}

/**
 * NewSessionDivider 创建会话划分器
 *
 * Parameters:
 *   - config: 配置（使用 DefaultSessionDividerConfig() 获取默认配置）
 *
 * Returns: *SessionDivider - 会话划分器实例
 */
func NewSessionDivider(config SessionDividerConfig) *SessionDivider {
	return &SessionDivider{
		config: config,
	}
}

/**
 * Divide 划分会话
 *
 * 将事件列表划分为会话列表
 *
 * Parameters:
 *   - eventList: 事件列表（必须按时间顺序）
 *
 * Returns: []*models.Session - 会话列表
 */
func (sd *SessionDivider) Divide(eventList []events.Event) []*models.Session {
	if len(eventList) == 0 {
		return []*models.Session{}
	}

	var sessions []*models.Session
	var currentSession *models.Session
	var lastEventTime time.Time
	var lastApp string

	for _, event := range eventList {
		// 获取应用信息
		app := sd.getApplication(event)

		// 检查是否需要创建新会话
		if sd.shouldStartNewSession(currentSession, event, lastEventTime, lastApp) {
			// 保存当前会话
			if currentSession != nil && len(currentSession.Events) >= sd.config.MinEvents {
				sessions = append(sessions, currentSession)
			}

			// 创建新会话
			currentSession = &models.Session{
				ID:          uuid.New().String(),
				StartTime:   event.Timestamp,
				Application: app,
				BundleID:    sd.getBundleID(event),
				Events:      []events.Event{event},
				EventCount:  1,
			}
		} else {
			// 添加到当前会话
			currentSession.Events = append(currentSession.Events, event)
			currentSession.EventCount++
		}

		lastEventTime = event.Timestamp
		lastApp = app
	}

	// 保存最后一个会话
	if currentSession != nil && len(currentSession.Events) >= sd.config.MinEvents {
		sessions = append(sessions, currentSession)
	}

	return sessions
}

/**
 * shouldStartNewSession 判断是否应该开始新会话
 *
 * Parameters:
 *   - currentSession: 当前会话
 *   - event: 新事件
 *   - lastEventTime: 上一个事件时间
 *   - lastApp: 上一个应用
 *
 * Returns: bool - true表示需要新会话
 */
func (sd *SessionDivider) shouldStartNewSession(
	currentSession *models.Session,
	event events.Event,
	lastEventTime time.Time,
	lastApp string,
) bool {
	// 没有当前会话，需要创建
	if currentSession == nil {
		return true
	}

	// 检查超时
	if !lastEventTime.IsZero() {
		elapsed := event.Timestamp.Sub(lastEventTime)
		if elapsed > sd.config.Timeout {
			return true
		}
	}

	// 检查应用切换
	if sd.config.SplitOnAppChange && lastApp != "" {
		currentApp := sd.getApplication(event)
		if currentApp != "" && currentApp != lastApp {
			return true
		}
	}

	return false
}

/**
 * getApplication 获取事件的应用名称
 *
 * Parameters:
 *   - event: 事件对象
 *
 * Returns: string - 应用名称
 */
func (sd *SessionDivider) getApplication(event events.Event) string {
	if event.Context != nil {
		return event.Context.Application
	}
	return ""
}

/**
 * getBundleID 获取事件的Bundle ID
 *
 * Parameters:
 *   - event: 事件对象
 *
 * Returns: string - Bundle ID
 */
func (sd *SessionDivider) getBundleID(event events.Event) string {
	if event.Context != nil {
		return event.Context.BundleID
	}
	return ""
}

/**
 * FilterByEventCount 按事件数量过滤会话
 *
 * 过滤掉事件数过少的会话
 *
 * Parameters:
 *   - sessions: 会话列表
 *   - minEvents: 最小事件数（默认使用配置值）
 *
 * Returns: []*models.Session - 过滤后的会话列表
 */
func (sd *SessionDivider) FilterByEventCount(sessions []*models.Session, minEvents ...int) []*models.Session {
	minCount := sd.config.MinEvents
	if len(minEvents) > 0 {
		minCount = minEvents[0]
	}

	var filtered []*models.Session
	for _, session := range sessions {
		if session.EventCount >= minCount {
			filtered = append(filtered, session)
		}
	}

	return filtered
}

/**
 * GetSessionStats 获取会话统计信息
 *
 * Parameters:
 *   - sessions: 会话列表
 *
 * Returns: *SessionStats - 统计信息
 */
func (sd *SessionDivider) GetSessionStats(sessions []*models.Session) *SessionStats {
	stats := &SessionStats{
		TotalSessions:      len(sessions),
		ByApplication:      make(map[string]int),
		TotalEvents:        0,
		AverageEvents:      0,
		ShortestSession:    nil,
		LongestSession:     nil,
	}

	if len(sessions) == 0 {
		return stats
	}

	var totalDuration time.Duration
	var shortestDuration, longestDuration time.Duration

	for _, session := range sessions {
		// 统计应用
		stats.ByApplication[session.Application]++

		// 统计事件
		stats.TotalEvents += session.EventCount

		// 统计持续时间
		duration := session.Duration()
		totalDuration += duration

		// 找最短和最长会话
		if stats.ShortestSession == nil || duration < shortestDuration {
			shortestDuration = duration
			stats.ShortestSession = session
		}
		if stats.LongestSession == nil || duration > longestDuration {
			longestDuration = duration
			stats.LongestSession = session
		}
	}

	// 计算平均事件数
	if len(sessions) > 0 {
		stats.AverageEvents = float64(stats.TotalEvents) / float64(len(sessions))
		stats.AverageDuration = totalDuration / time.Duration(len(sessions))
	}

	return stats
}

/**
 * SessionStats 会话统计信息
 */
type SessionStats struct {
	// TotalSessions 总会话数
	TotalSessions int

	// ByApplication 按应用统计的会话数
	ByApplication map[string]int

	// TotalEvents 总事件数
	TotalEvents int

	// AverageEvents 平均每个会话的事件数
	AverageEvents float64

	// AverageDuration 平均会话持续时间
	AverageDuration time.Duration

	// ShortestSession 最短的会话
	ShortestSession *models.Session

	// LongestSession 最长的会话
	LongestSession *models.Session
}
