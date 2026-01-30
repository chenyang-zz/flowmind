package monitor

import (
	"sync"
	"time"

	"github.com/chenyang-zz/flowmind/pkg/events"
	"github.com/chenyang-zz/flowmind/internal/infrastructure/logger"
	"go.uber.org/zap"
)

// AppSession 应用会话
//
// 记录单个应用的使用会话信息，包括应用名称、Bundle ID、会话开始和结束时间。
type AppSession struct {
	// AppName 应用名称
	AppName string

	// BundleID 应用 Bundle ID（唯一标识符）
	BundleID string

	// Start 会话开始时间
	Start time.Time

	// End 会话结束时间（零值表示活跃会话）
	End time.Time
}

// IsAlive 检查会话是否活跃
// Returns: bool - true 表示会话活跃，false 表示已结束
func (s *AppSession) IsAlive() bool {
	return s.End.IsZero()
}

// Duration 计算会话时长
// Returns: time.Duration - 会话持续时长
func (s *AppSession) Duration() time.Duration {
	if s.IsAlive() {
		return time.Since(s.Start)
	}
	return s.End.Sub(s.Start)
}

// AppTracker 应用会话追踪器
//
// 负责追踪所有应用的使用会话，记录应用切换、会话开始和结束、会话时长等信息。
// 当应用切换或监控器停止时，会发布应用会话事件到事件总线。
type AppTracker struct {
	// sessions 活跃的会话映射（应用名称 → 会话）
	sessions map[string]*AppSession

	// eventBus 事件总线，用于发布应用会话事件
	eventBus *events.EventBus

	// mu 读写锁，保护并发访问
	mu sync.RWMutex
}

// NewAppTracker 创建应用会话追踪器
//
// 创建一个新的应用会话追踪器实例。
//
// Parameters:
//   - eventBus: 事件总线实例，用于发布应用会话事件
//
// Returns: *AppTracker - 新创建的应用会话追踪器实例
func NewAppTracker(eventBus *events.EventBus) *AppTracker {
	return &AppTracker{
		sessions: make(map[string]*AppSession),
		eventBus: eventBus,
	}
}

// SwitchApp 处理应用切换
//
// 当应用切换时调用此方法，负责：
//   1. 结束旧应用的会话
//   2. 开始新应用的会话
//   3. 发布应用会话事件
//
// Parameters:
//   - from: 切换前的应用名称
//   - to: 切换后的应用名称
//   - bundleID: 新应用的 Bundle ID
func (at *AppTracker) SwitchApp(from, to, bundleID string) {
	at.mu.Lock()
	defer at.mu.Unlock()

	now := time.Now()

	// 结束旧应用的会话
	if from != "" {
		if session, exists := at.sessions[from]; exists {
			session.End = now
			duration := session.Duration()

			// 发布应用会话事件
			at.publishSessionEvent(session)

			logger.Debug("应用会话结束",
				zap.String("app", from),
				zap.Duration("duration", duration),
			)

			// 从活跃会话中移除
			delete(at.sessions, from)
		}
	}

	// 开始新应用的会话
	if to != "" {
		// 如果新应用已有活跃会话，先结束它
		if existingSession, exists := at.sessions[to]; exists {
			existingSession.End = now
			at.publishSessionEvent(existingSession)
		}

		// 创建新会话
		at.sessions[to] = &AppSession{
			AppName:  to,
			BundleID: bundleID,
			Start:    now,
			End:     time.Time{}, // 零值表示活跃
		}

		logger.Debug("应用会话开始",
			zap.String("app", to),
			zap.String("bundle_id", bundleID),
		)
	}
}

// GetActiveSession 获取指定应用的活跃会话
//
// Parameters:
//   - appName: 应用名称
//
// Returns: *AppSession - 活跃会话，如果不存在则返回nil
func (at *AppTracker) GetActiveSession(appName string) *AppSession {
	at.mu.RLock()
	defer at.mu.RUnlock()

	return at.sessions[appName]
}

// GetAllActiveSessions 获取所有活跃会话
// Returns: []AppSession - 活跃会话列表
func (at *AppTracker) GetAllActiveSessions() []*AppSession {
	at.mu.RLock()
	defer at.mu.RUnlock()

	sessions := make([]*AppSession, 0, len(at.sessions))
	for _, session := range at.sessions {
		sessions = append(sessions, session)
	}
	return sessions
}

// EndAllSessions 结束所有活跃会话
//
// 当监控器停止时调用此方法，结束所有应用的活跃会话并发布会话事件。
func (at *AppTracker) EndAllSessions() {
	at.mu.Lock()
	defer at.mu.Unlock()

	now := time.Now()

	for appName, session := range at.sessions {
		session.End = now
		duration := session.Duration()

		// 发布应用会话事件
		at.publishSessionEvent(session)

		logger.Debug("应用会话结束",
			zap.String("app", appName),
			zap.Duration("duration", duration),
		)
	}

	// 清空所有会话
	at.sessions = make(map[string]*AppSession)
}

// publishSessionEvent 发布应用会话事件
//
// 将应用会话信息发布到事件总线，供其他模块消费。
//
// Parameters: session - 应用会话
func (at *AppTracker) publishSessionEvent(session *AppSession) {
	duration := session.Duration()

	// 构造应用会话事件
	sessionEvent := events.NewEvent(events.EventTypeAppSession, map[string]interface{}{
		"app_name":  session.AppName,
		"bundle_id": session.BundleID,
		"start":     session.Start,
		"end":       session.End,
		"duration":  duration.Seconds(), // 时长（秒）
	})

	// 发布事件到事件总线
	at.eventBus.Publish(string(events.EventTypeAppSession), *sessionEvent)
}
