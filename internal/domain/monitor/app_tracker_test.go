package monitor

import (
	"testing"
	"time"

	"github.com/chenyang-zz/flowmind/pkg/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewAppTracker 测试AppTracker的创建
//
// 验证AppTracker能够正确创建，并且初始状态为空。
func TestNewAppTracker(t *testing.T) {
	eventBus := events.NewEventBus()
	tracker := NewAppTracker(eventBus)

	assert.NotNil(t, tracker)
	assert.NotNil(t, tracker.GetAllActiveSessions())
	assert.Empty(t, tracker.GetAllActiveSessions())
}

// TestAppTracker_SwitchApp 测试应用切换逻辑
//
// 验证应用切换时能够正确创建新会话，并结束旧会话。
func TestAppTracker_SwitchApp(t *testing.T) {
	eventBus := events.NewEventBus()
	tracker := NewAppTracker(eventBus)

	// 第一次切换（从空到Chrome）
	tracker.SwitchApp("", "Chrome", "com.google.Chrome")

	// 验证Chrome会话已创建
	session := tracker.GetActiveSession("Chrome")
	require.NotNil(t, session)
	assert.Equal(t, "Chrome", session.AppName)
	assert.Equal(t, "com.google.Chrome", session.BundleID)
	assert.True(t, session.IsAlive())
	assert.True(t, session.Start.Before(time.Now()) || session.Start.Equal(time.Now()))

	// 第二次切换（从Chrome到Safari）
	tracker.SwitchApp("Chrome", "Safari", "com.apple.Safari")

	// 验证Safari会话已创建
	safariSession := tracker.GetActiveSession("Safari")
	require.NotNil(t, safariSession)
	assert.Equal(t, "Safari", safariSession.AppName)

	// 验证Chrome会话已结束
	chromeSession := tracker.GetActiveSession("Chrome")
	assert.Nil(t, chromeSession)
}

// TestAppTracker_SwitchToSameApp 测试切换到同一应用
//
// 验证切换到同一应用时，旧会话被正确结束，新会话被创建。
func TestAppTracker_SwitchToSameApp(t *testing.T) {
	eventBus := events.NewEventBus()
	tracker := NewAppTracker(eventBus)

	// 第一次切换到Chrome
	tracker.SwitchApp("", "Chrome", "com.google.Chrome")
	firstSession := tracker.GetActiveSession("Chrome")
	require.NotNil(t, firstSession)

	// 稍等片刻
	time.Sleep(10 * time.Millisecond)

	// 第二次切换到Chrome（模拟应用切换到自身）
	tracker.SwitchApp("Chrome", "Chrome", "com.google.Chrome")
	secondSession := tracker.GetActiveSession("Chrome")

	// 验证创建了新会话
	assert.NotNil(t, secondSession)
	assert.NotEqual(t, firstSession.Start, secondSession.Start)
}

// TestAppTracker_GetActiveSession 测试获取活跃会话
//
// 验证能够正确获取应用的活跃会话。
func TestAppTracker_GetActiveSession(t *testing.T) {
	eventBus := events.NewEventBus()
	tracker := NewAppTracker(eventBus)

	// 未切换时，应该返回nil
	session := tracker.GetActiveSession("Chrome")
	assert.Nil(t, session)

	// 切换后，应该返回会话
	tracker.SwitchApp("", "Chrome", "com.google.Chrome")
	session = tracker.GetActiveSession("Chrome")
	assert.NotNil(t, session)
}

// TestAppTracker_GetAllActiveSessions 测试获取所有活跃会话
//
// 验证能够获取所有活跃的应用会话。
func TestAppTracker_GetAllActiveSessions(t *testing.T) {
	eventBus := events.NewEventBus()
	tracker := NewAppTracker(eventBus)

	// 初始状态：无活跃会话
	sessions := tracker.GetAllActiveSessions()
	assert.Empty(t, sessions)

	// 切换到多个应用
	tracker.SwitchApp("", "Chrome", "com.google.Chrome")
	tracker.SwitchApp("Chrome", "Safari", "com.apple.Safari")

	// 应该有2个活跃会话（Safari活跃，Chrome已结束）
	sessions = tracker.GetAllActiveSessions()
	assert.Len(t, sessions, 1) // 只有Safari活跃
}

// TestAppTracker_EndAllSessions 测试结束所有会话
//
// 验证能够正确结束所有活跃会话，并发布会话事件。
func TestAppTracker_EndAllSessions(t *testing.T) {
	eventBus := events.NewEventBus()
	tracker := NewAppTracker(eventBus)

	// 创建多个活跃会话
	tracker.SwitchApp("", "Chrome", "com.google.Chrome")
	tracker.SwitchApp("Chrome", "Safari", "com.apple.Safari")

	// 结束所有会话
	tracker.EndAllSessions()

	// 验证所有会话已结束
	sessions := tracker.GetAllActiveSessions()
	assert.Empty(t, sessions)
}

// TestAppSession_Duration 测试会话时长计算
//
// 验证能够正确计算会话的持续时长。
func TestAppSession_Duration(t *testing.T) {
	// 活跃会话：时长应为从开始到现在
	activeSession := &AppSession{
		AppName:  "Chrome",
		BundleID: "com.google.Chrome",
		Start:    time.Now().Add(-1 * time.Second),
		End:      time.Time{}, // 零值表示活跃
	}
	duration := activeSession.Duration()
	assert.True(t, duration >= 1*time.Second)
	assert.True(t, duration < 2*time.Second)

	// 已结束会话：时长应为End - Start
	closedSession := &AppSession{
		AppName:  "Chrome",
		BundleID: "com.google.Chrome",
		Start:    time.Now().Add(-2 * time.Second),
		End:      time.Now().Add(-1 * time.Second),
	}
	duration = closedSession.Duration()
	assert.True(t, duration >= 900*time.Millisecond) // 允许100ms误差
	assert.True(t, duration < 1100*time.Millisecond)
}

// TestAppSession_IsAlive 测试会话活跃状态检查
//
// 验证能够正确判断会话是否活跃。
func TestAppSession_IsAlive(t *testing.T) {
	// 活跃会话
	activeSession := &AppSession{
		Start: time.Now(),
		End:   time.Time{}, // 零值表示活跃
	}
	assert.True(t, activeSession.IsAlive())

	// 已结束会话
	closedSession := &AppSession{
		Start: time.Now().Add(-1 * time.Second),
		End:   time.Now(),
	}
	assert.False(t, closedSession.IsAlive())
}

// TestAppTracker_SessionEvent 测试应用会话事件发布
//
// 验证应用会话结束时能够正确发布事件。
func TestAppTracker_SessionEvent(t *testing.T) {
	eventBus := events.NewEventBus()
	tracker := NewAppTracker(eventBus)

	// 订阅应用会话事件
	eventReceived := false
	var eventData map[string]interface{}

	eventBus.Subscribe("app_session", func(event events.Event) error {
		eventReceived = true
		eventData = event.Data
		return nil
	})

	// 执行应用切换
	tracker.SwitchApp("", "Chrome", "com.google.Chrome")
	time.Sleep(10 * time.Millisecond) // 稍等片刻

	// 再次切换，触发会话结束事件
	tracker.SwitchApp("Chrome", "Safari", "com.apple.Safari")

	// 等待事件处理
	time.Sleep(50 * time.Millisecond)

	// 验证事件已发布
	assert.True(t, eventReceived, "应用会话事件应该被发布")
	assert.Equal(t, "Chrome", eventData["app_name"])
	assert.Equal(t, "com.google.Chrome", eventData["bundle_id"])
}
