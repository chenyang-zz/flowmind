package monitor

import (
	"testing"
	"time"

	"github.com/chenyang-zz/flowmind/internal/infrastructure/platform"
	"github.com/chenyang-zz/flowmind/pkg/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockAppSwitchMonitor 模拟的应用切换监控器
// 用于测试 ApplicationMonitor 的业务逻辑
type MockAppSwitchMonitor struct {
	isRunning bool
	callback  platform.AppSwitchCallback
}

// Start 启动模拟监控器
func (m *MockAppSwitchMonitor) Start(callback platform.AppSwitchCallback) error {
	if m.isRunning {
		return nil
	}
	m.callback = callback
	m.isRunning = true
	return nil
}

// Stop 停止模拟监控器
func (m *MockAppSwitchMonitor) Stop() error {
	if !m.isRunning {
		return nil
	}
	m.isRunning = false
	m.callback = nil
	return nil
}

// IsRunning 检查运行状态
func (m *MockAppSwitchMonitor) IsRunning() bool {
	return m.isRunning
}

// SimulateAppSwitch 模拟应用切换事件（用于测试）
func (m *MockAppSwitchMonitor) SimulateAppSwitch(from, to, bundleID, window string) {
	if m.callback != nil {
		m.callback(platform.AppSwitchEvent{
			From:     from,
			To:       to,
			BundleID: bundleID,
			Window:   window,
		})
	}
}

// MockContextProvider 模拟的上下文提供者
type MockContextProvider struct {
	appName    string
	bundleID   string
	windowName string
}

// GetFrontmostApp 获取最前端应用名称
func (m *MockContextProvider) GetFrontmostApp() string {
	return m.appName
}

// GetBundleID 获取应用 Bundle ID
func (m *MockContextProvider) GetBundleID() string {
	return m.bundleID
}

// GetFocusedWindowTitle 获取焦点窗口标题
func (m *MockContextProvider) GetFocusedWindowTitle() string {
	return m.windowName
}

// GetContext 获取完整的应用上下文
func (m *MockContextProvider) GetContext() *events.EventContext {
	return &events.EventContext{
		Application:  m.GetFrontmostApp(),
		BundleID:     m.GetBundleID(),
		WindowTitle:  m.GetFocusedWindowTitle(),
	}
}

// TestNewApplicationMonitor 测试ApplicationMonitor的创建
//
// 验证ApplicationMonitor能够正确创建，并且初始状态为未运行。
func TestNewApplicationMonitor(t *testing.T) {
	eventBus := events.NewEventBus()

	monitor := NewApplicationMonitor(eventBus)

	assert.NotNil(t, monitor)
	assert.False(t, monitor.IsRunning())
}

// TestApplicationMonitor_StartStop 测试启动和停止
//
// 验证ApplicationMonitor能够正确启动和停止，并且状态正确更新。
func TestApplicationMonitor_StartStop(t *testing.T) {
	eventBus := events.NewEventBus()

	// 创建监控器并注入模拟实现
	monitor := NewApplicationMonitor(eventBus)
	appMonitor := monitor.(*ApplicationMonitor)

	// 注入模拟的平台层监控器
	mockPlatform := &MockAppSwitchMonitor{}
	appMonitor.platform = mockPlatform

	// 启动监控器
	err := monitor.Start()
	assert.NoError(t, err)
	assert.True(t, monitor.IsRunning())
	assert.True(t, mockPlatform.IsRunning())

	// 停止监控器
	err = monitor.Stop()
	assert.NoError(t, err)
	assert.False(t, monitor.IsRunning())
	assert.False(t, mockPlatform.IsRunning())
}

// TestApplicationMonitor_HandleAppSwitch 测试应用切换事件处理
//
// 验证应用切换事件能够正确处理，并发布到事件总线。
func TestApplicationMonitor_HandleAppSwitch(t *testing.T) {
	eventBus := events.NewEventBus()

	monitor := NewApplicationMonitor(eventBus)
	appMonitor := monitor.(*ApplicationMonitor)

	// 注入模拟的平台层监控器
	mockPlatform := &MockAppSwitchMonitor{}
	appMonitor.platform = mockPlatform

	// 注入模拟的上下文管理器
	mockContext := &MockContextProvider{
		appName:    "Safari",
		bundleID:   "com.apple.Safari",
		windowName: "Safari Window",
	}
	appMonitor.contextMgr = mockContext

	// 启动监控器
	err := monitor.Start()
	require.NoError(t, err)

	// 订阅应用切换事件
	eventReceived := false
	var receivedEvent events.Event

	eventBus.Subscribe("app_switch", func(event events.Event) error {
		eventReceived = true
		receivedEvent = event
		return nil
	})

	// 模拟应用切换
	mockPlatform.SimulateAppSwitch("Chrome", "Safari", "com.apple.Safari", "Safari Window")

	// 等待事件处理
	time.Sleep(50 * time.Millisecond)

	// 验证事件已发布
	assert.True(t, eventReceived, "应用切换事件应该被发布")
	assert.Equal(t, "Chrome", receivedEvent.Data["from"])
	assert.Equal(t, "Safari", receivedEvent.Data["to"])
	assert.Equal(t, "com.apple.Safari", receivedEvent.Data["bundle_id"])
	assert.Equal(t, "Safari Window", receivedEvent.Data["window"])

	// 验证上下文信息
	assert.NotNil(t, receivedEvent.Context)
	assert.Equal(t, "Safari", receivedEvent.Context.Application)
	assert.Equal(t, "com.apple.Safari", receivedEvent.Context.BundleID)
	assert.Equal(t, "Safari Window", receivedEvent.Context.WindowTitle)

	// 停止监控器
	monitor.Stop()
}

// TestApplicationMonitor_AppSessionTracking 测试应用会话追踪
//
// 验证应用切换时能够正确创建和管理应用会话。
func TestApplicationMonitor_AppSessionTracking(t *testing.T) {
	eventBus := events.NewEventBus()

	monitor := NewApplicationMonitor(eventBus)
	appMonitor := monitor.(*ApplicationMonitor)

	// 注入模拟的平台层监控器
	mockPlatform := &MockAppSwitchMonitor{}
	appMonitor.platform = mockPlatform

	// 启动监控器
	err := monitor.Start()
	require.NoError(t, err)

	// 模拟首次应用切换
	mockPlatform.SimulateAppSwitch("", "Chrome", "com.google.Chrome", "Chrome Window")
	time.Sleep(50 * time.Millisecond)

	// 验证Chrome会话已创建
	session := appMonitor.appTracker.GetActiveSession("Chrome")
	assert.NotNil(t, session)
	assert.Equal(t, "Chrome", session.AppName)
	assert.Equal(t, "com.google.Chrome", session.BundleID)
	assert.True(t, session.IsAlive())

	// 模拟第二次应用切换
	mockPlatform.SimulateAppSwitch("Chrome", "Safari", "com.apple.Safari", "Safari Window")
	time.Sleep(50 * time.Millisecond)

	// 验证Chrome会话已结束，Safari会话已创建
	chromeSession := appMonitor.appTracker.GetActiveSession("Chrome")
	assert.Nil(t, chromeSession)

	safariSession := appMonitor.appTracker.GetActiveSession("Safari")
	assert.NotNil(t, safariSession)
	assert.Equal(t, "Safari", safariSession.AppName)

	// 停止监控器
	monitor.Stop()
}

// TestApplicationMonitor_StartWhenRunning 测试重复启动
//
// 验证重复启动监控器时幂等地返回成功（不会报错）。
func TestApplicationMonitor_StartWhenRunning(t *testing.T) {
	eventBus := events.NewEventBus()

	monitor := NewApplicationMonitor(eventBus)
	appMonitor := monitor.(*ApplicationMonitor)

	// 注入模拟的平台层监控器
	mockPlatform := &MockAppSwitchMonitor{}
	appMonitor.platform = mockPlatform

	// 首次启动
	err := monitor.Start()
	assert.NoError(t, err)
	assert.True(t, monitor.IsRunning())

	// 重复启动（应该幂等地返回成功）
	err = monitor.Start()
	assert.NoError(t, err)
	assert.True(t, monitor.IsRunning())

	// 清理
	monitor.Stop()
}

// TestApplicationMonitor_StopWhenNotRunning 测试重复停止
//
// 验证重复停止监控器时幂等地返回成功（不会报错）。
func TestApplicationMonitor_StopWhenNotRunning(t *testing.T) {
	eventBus := events.NewEventBus()

	monitor := NewApplicationMonitor(eventBus)

	// 未启动就停止（应该幂等地返回成功）
	err := monitor.Stop()
	assert.NoError(t, err)
	assert.False(t, monitor.IsRunning())
}

// TestApplicationMonitor_EndAllSessions 测试停止时结束所有会话
//
// 验证监控器停止时，所有活跃的应用会话都被正确结束。
func TestApplicationMonitor_EndAllSessions(t *testing.T) {
	eventBus := events.NewEventBus()

	monitor := NewApplicationMonitor(eventBus)
	appMonitor := monitor.(*ApplicationMonitor)

	// 注入模拟的平台层监控器
	mockPlatform := &MockAppSwitchMonitor{}
	appMonitor.platform = mockPlatform

	// 启动监控器
	err := monitor.Start()
	require.NoError(t, err)

	// 创建多个活跃会话
	mockPlatform.SimulateAppSwitch("", "Chrome", "com.google.Chrome", "Window 1")
	time.Sleep(50 * time.Millisecond)

	mockPlatform.SimulateAppSwitch("Chrome", "Safari", "com.apple.Safari", "Window 2")
	time.Sleep(50 * time.Millisecond)

	// 验证有活跃会话
	sessions := appMonitor.appTracker.GetAllActiveSessions()
	assert.NotEmpty(t, sessions)

	// 停止监控器
	monitor.Stop()

	// 验证所有会话已结束
	sessions = appMonitor.appTracker.GetAllActiveSessions()
	assert.Empty(t, sessions)
}
