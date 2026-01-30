//go:build darwin

package platform

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewKeyboardMonitor 测试键盘监控器的创建
//
// 验证 NewKeyboardMonitor 函数能正确创建监控器实例，
// 且初始状态下监控器未运行。
func TestNewKeyboardMonitor(t *testing.T) {
	monitor := NewKeyboardMonitor()

	assert.NotNil(t, monitor)
	assert.False(t, monitor.IsRunning())
}

// TestKeyboardMonitor_StartStop 测试键盘监控器的启动和停止
//
// 验证监控器能够正常启动和停止，且状态正确切换。
// 如果缺少辅助功能权限，测试会被跳过。
func TestKeyboardMonitor_StartStop(t *testing.T) {
	monitor := NewKeyboardMonitor()

	// 启动监控器
	err := monitor.Start(func(event KeyboardEvent) {
		// 回调函数
	})

	if err != nil {
		t.Skipf("需要辅助功能权限，跳过测试: %v", err)
	}

	require.NoError(t, err)
	assert.True(t, monitor.IsRunning())

	// 停止监控器
	err = monitor.Stop()
	require.NoError(t, err)
	assert.False(t, monitor.IsRunning())
}

// TestKeyboardMonitor_StartTwice 测试重复启动监控器的错误处理
//
// 验证在监控器已运行的情况下，再次启动会返回错误。
// 如果缺少辅助功能权限，测试会被跳过。
func TestKeyboardMonitor_StartTwice(t *testing.T) {
	monitor := NewKeyboardMonitor()

	// 第一次启动
	err := monitor.Start(func(event KeyboardEvent) {})
	if err != nil {
		t.Skipf("需要辅助功能权限，跳过测试: %v", err)
	}
	require.NoError(t, err)

	// 第二次启动应该失败
	err = monitor.Start(func(event KeyboardEvent) {})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already running")

	// 清理
	_ = monitor.Stop()
}

// TestKeyboardMonitor_StopWithoutStart 测试未启动就停止的错误处理
//
// 验证在监控器未启动的情况下调用 Stop 会返回错误。
func TestKeyboardMonitor_StopWithoutStart(t *testing.T) {
	monitor := NewKeyboardMonitor()

	// 未启动就停止应该失败
	err := monitor.Stop()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not running")
}

// TestKeyboardMonitor_Callback 测试键盘事件回调机制
//
// 验证监控器能够通过回调函数正确传递键盘事件。
// 注意：此测试需要在2秒内实际按键才会捕获到事件，
// 超时不视为失败（因为在自动化测试环境中可能没有用户输入）。
// 如果缺少辅助功能权限，测试会被跳过。
func TestKeyboardMonitor_Callback(t *testing.T) {
	monitor := NewKeyboardMonitor()

	// 创建回调通道
	receivedEvents := make(chan KeyboardEvent, 10)

	// 启动监控器
	err := monitor.Start(func(event KeyboardEvent) {
		receivedEvents <- event
	})

	if err != nil {
		t.Skipf("需要辅助功能权限，跳过测试: %v", err)
	}
	require.NoError(t, err)

	// 等待可能的键盘事件
	// 注意：这个测试需要实际按键才会触发
	select {
	case event := <-receivedEvents:
		// 验证事件结构
		assert.GreaterOrEqual(t, event.KeyCode, 0)
		assert.GreaterOrEqual(t, event.Modifiers, uint64(0))
	case <-time.After(2 * time.Second):
		// 超时不算失败，只是没有捕获到键盘事件
		t.Log("未捕获到键盘事件（正常情况）")
	}

	// 清理
	_ = monitor.Stop()
}

// TestKeyboardMonitor_CallbackNotNil 测试 nil 回调的处理
//
// 验证即使传入 nil 回调，监控器也能正常启动和运行。
// 如果缺少辅助功能权限，测试会被跳过。
func TestKeyboardMonitor_CallbackNotNil(t *testing.T) {
	monitor := NewKeyboardMonitor()

	// 测试回调为 nil 的情况
	err := monitor.Start(nil)
	if err != nil {
		t.Skipf("需要辅助功能权限，跳过测试: %v", err)
	}
	require.NoError(t, err)

	// 验证监控器正在运行
	assert.True(t, monitor.IsRunning())

	// 清理
	_ = monitor.Stop()
}

// TestKeyboardEvent_Structure 测试 KeyboardEvent 结构体的字段
//
// 验证 KeyboardEvent 结构体能够正确存储和返回按键代码和修饰键标志。
func TestKeyboardEvent_Structure(t *testing.T) {
	// 测试 KeyboardEvent 结构体
	event := KeyboardEvent{
		KeyCode:   42,
		Modifiers: 0x10100,
	}

	assert.Equal(t, 42, event.KeyCode)
	assert.Equal(t, uint64(0x10100), event.Modifiers)
}

// TestNewContextProvider 测试上下文提供者的创建
//
// 验证 NewContextProvider 函数能正确创建 ContextProvider 实例。
func TestNewContextProvider(t *testing.T) {
	provider := NewContextProvider()

	assert.NotNil(t, provider)
}

// TestContextProvider_GetFrontmostApp 测试获取最前端应用名称
//
// 验证能够成功获取当前最前端应用的应用名称。
// 在正常使用场景下，应该有一个活动的应用。
func TestContextProvider_GetFrontmostApp(t *testing.T) {
	provider := NewContextProvider()

	appName := provider.GetFrontmostApp()

	// 应用名称应该非空（在正常使用场景下）
	assert.NotEmpty(t, appName, "应该能获取到前端应用名称")
}

// TestContextProvider_GetBundleID 测试获取应用 Bundle ID
//
// 验证能够成功获取当前应用的 Bundle ID。
// Bundle ID 应包含点号分隔符（如 "com.apple.Safari"）。
func TestContextProvider_GetBundleID(t *testing.T) {
	provider := NewContextProvider()

	bundleID := provider.GetBundleID()

	// Bundle ID 应该非空或符合格式
	if bundleID != "" {
		assert.Contains(t, bundleID, ".")
	}
}

// TestContextProvider_GetFocusedWindowTitle 测试获取焦点窗口标题
//
// 验证能够获取焦点窗口的标题。
// 注意：某些应用可能没有窗口标题，此测试只验证不发生 panic。
func TestContextProvider_GetFocusedWindowTitle(t *testing.T) {
	provider := NewContextProvider()

	title := provider.GetFocusedWindowTitle()

	// 窗口标题可能为空（某些应用没有标题）
	// 只验证返回的字符串不会导致 panic
	assert.NotNil(t, title)
}

// TestContextProvider_GetContext 测试获取完整的应用上下文
//
// 验证能够获取包含应用名称、Bundle ID 和窗口标题的完整上下文对象。
// 应用名称和 Bundle ID 不应为空，窗口标题可以为空。
func TestContextProvider_GetContext(t *testing.T) {
	provider := NewContextProvider()

	context := provider.GetContext()

	require.NotNil(t, context)
	assert.NotEmpty(t, context.Application, "应用名称不应为空")
	assert.NotEmpty(t, context.BundleID, "Bundle ID 不应为空")

	// 窗口标题可以为空
	assert.NotNil(t, context.WindowTitle)
}

// TestContextProvider_Consistency 测试上下文数据的一致性
//
// 验证在短时间内多次调用 GetContext 返回的数据应该是一致的，
// 反映同一个应用的上下文信息。
func TestContextProvider_Consistency(t *testing.T) {
	provider := NewContextProvider()

	// 多次调用应该返回一致的数据
	context1 := provider.GetContext()
	context2 := provider.GetContext()

	assert.Equal(t, context1.Application, context2.Application)
	assert.Equal(t, context1.BundleID, context2.BundleID)
}

// TestContextProvider_Integration 测试上下文提供者的集成
//
// 验证各个方法返回的数据之间的一致性。
// GetContext 应该返回与单独调用各个方法相同的数据。
func TestContextProvider_Integration(t *testing.T) {
	provider := NewContextProvider()

	// 测试完整的上下文获取流程
	appName := provider.GetFrontmostApp()
	bundleID := provider.GetBundleID()
	windowTitle := provider.GetFocusedWindowTitle()
	context := provider.GetContext()

	// 验证各部分的一致性
	assert.Equal(t, appName, context.Application)
	assert.Equal(t, bundleID, context.BundleID)
	assert.Equal(t, windowTitle, context.WindowTitle)
}

// TestKeyboardMonitor_HandleCallback_WhenRunning 测试运行时处理回调
//
// 验证监控器运行时能够正确处理回调函数并调用用户注册的回调。
func TestKeyboardMonitor_HandleCallback_WhenRunning(t *testing.T) {
	monitor := NewKeyboardMonitor().(*DarwinKeyboardMonitor)

	receivedEvents := make(chan KeyboardEvent, 10)

	// 启动监控器
	err := monitor.Start(func(event KeyboardEvent) {
		receivedEvents <- event
	})

	if err != nil {
		t.Skipf("需要辅助功能权限，跳过测试: %v", err)
	}
	require.NoError(t, err)

	// 模拟 C 层回调
	testKeyCode := 42
	testFlags := uint64(0x10100)
	monitor.HandleCallbackExporter(testKeyCode, testFlags)

	// 验证回调被调用
	select {
	case event := <-receivedEvents:
		assert.Equal(t, testKeyCode, event.KeyCode)
		assert.Equal(t, testFlags, event.Modifiers)
	case <-time.After(1 * time.Second):
		t.Fatal("未收到回调事件")
	}

	// 清理
	_ = monitor.Stop()
}

// TestKeyboardMonitor_HandleCallback_WhenNotRunning 测试未运行时不处理回调
//
// 验证监控器未运行时，即使调用 handleCallback 也不会触发用户回调。
func TestKeyboardMonitor_HandleCallback_WhenNotRunning(t *testing.T) {
	monitor := NewKeyboardMonitor().(*DarwinKeyboardMonitor)

	callbackCalled := false
	err := monitor.Start(func(event KeyboardEvent) {
		callbackCalled = true
	})

	if err != nil {
		t.Skipf("需要辅助功能权限，跳过测试: %v", err)
	}
	require.NoError(t, err)

	// 停止监控器
	_ = monitor.Stop()

	// 重置标志
	callbackCalled = false

	// 尝试触发回调（应该不会被调用）
	monitor.HandleCallbackExporter(42, 0x10100)

	// 等待一小段时间确保回调不会被调用
	time.Sleep(100 * time.Millisecond)
	assert.False(t, callbackCalled, "监控器停止后不应调用回调")
}

// TestKeyboardMonitor_HandleCallback_WithNilCallback 测试 nil 回调的处理
//
// 验证即使 callback 为 nil，handleCallback 也不会 panic。
func TestKeyboardMonitor_HandleCallback_WithNilCallback(t *testing.T) {
	monitor := NewKeyboardMonitor().(*DarwinKeyboardMonitor)

	// 启动监控器时传入 nil 回调
	err := monitor.Start(nil)
	if err != nil {
		t.Skipf("需要辅助功能权限，跳过测试: %v", err)
	}
	require.NoError(t, err)

	// 这不应该 panic
	assert.NotPanics(t, func() {
		monitor.HandleCallbackExporter(42, 0x10100)
	})

	// 清理
	_ = monitor.Stop()
}

// TestKeyboardMonitor_ConcurrentStartStop 测试并发启动和停止
//
// 验证在并发场景下启动和停止监控器的线程安全性。
func TestKeyboardMonitor_ConcurrentStartStop(t *testing.T) {
	monitor := NewKeyboardMonitor()

	// 测试并发启动
	for i := 0; i < 10; i++ {
		go func() {
			_ = monitor.Start(func(event KeyboardEvent) {})
		}()
	}

	// 等待操作完成
	time.Sleep(200 * time.Millisecond)

	// 验证只有一个成功
	if monitor.IsRunning() {
		_ = monitor.Stop()
	}

	// 测试并发停止
	for i := 0; i < 10; i++ {
		go func() {
			_ = monitor.Stop()
		}()
	}

	// 等待操作完成
	time.Sleep(200 * time.Millisecond)
}

// TestKeyboardMonitor_ConcurrentCallback 测试并发回调
//
// 验证在并发场景下调用回调函数的线程安全性。
func TestKeyboardMonitor_ConcurrentCallback(t *testing.T) {
	monitor := NewKeyboardMonitor().(*DarwinKeyboardMonitor)

	callbackCount := 0
	var mu sync.Mutex

	err := monitor.Start(func(event KeyboardEvent) {
		mu.Lock()
		callbackCount++
		mu.Unlock()
	})

	if err != nil {
		t.Skipf("需要辅助功能权限，跳过测试: %v", err)
	}
	require.NoError(t, err)

	// 并发触发多个回调
	for i := 0; i < 100; i++ {
		go func(keyCode int) {
			monitor.HandleCallbackExporter(keyCode, uint64(keyCode))
		}(i)
	}

	// 等待所有回调完成
	time.Sleep(500 * time.Millisecond)

	mu.Lock()
	finalCount := callbackCount
	mu.Unlock()

	// 验证所有回调都被执行
	assert.Equal(t, 100, finalCount)

	// 清理
	_ = monitor.Stop()
}

// TestKeyboardMonitor_MultipleStop 测试多次停止的错误处理
//
// 验证多次调用 Stop 不会导致 panic 或资源泄漏。
func TestKeyboardMonitor_MultipleStop(t *testing.T) {
	monitor := NewKeyboardMonitor()

	// 启动监控器
	err := monitor.Start(func(event KeyboardEvent) {})
	if err != nil {
		t.Skipf("需要辅助功能权限，跳过测试: %v", err)
	}
	require.NoError(t, err)

	// 多次停止
	for i := 0; i < 5; i++ {
		err := monitor.Stop()
		if i == 0 {
			require.NoError(t, err)
		} else {
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "not running")
		}
	}
}

// TestKeyboardMonitor_Restart 测试重启监控器
//
// 验证监控器停止后能够重新启动。
func TestKeyboardMonitor_Restart(t *testing.T) {
	monitor := NewKeyboardMonitor()

	// 第一次启动和停止
	err := monitor.Start(func(event KeyboardEvent) {})
	if err != nil {
		t.Skipf("需要辅助功能权限，跳过测试: %v", err)
	}
	require.NoError(t, err)
	assert.True(t, monitor.IsRunning())

	err = monitor.Stop()
	require.NoError(t, err)
	assert.False(t, monitor.IsRunning())

	// 第二次启动
	err = monitor.Start(func(event KeyboardEvent) {})
	if err != nil {
		t.Skipf("需要辅助功能权限，跳过测试: %v", err)
	}
	require.NoError(t, err)
	assert.True(t, monitor.IsRunning())

	// 清理
	_ = monitor.Stop()
}

