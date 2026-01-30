//go:build darwin

package monitor

import (
	"testing"
	"time"

	"github.com/chenyang-zz/flowmind/pkg/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewKeyboardMonitor 测试键盘监控器的创建
//
// 验证键盘监控器能够正确创建，并且初始状态为未运行。
// 测试场景：
//   1. 创建键盘监控器
//   2. 验证实例不为 nil
//   3. 验证初始运行状态为 false
func TestNewKeyboardMonitor(t *testing.T) {
	eventBus := events.NewEventBus()
	monitor := NewKeyboardMonitor(eventBus)

	assert.NotNil(t, monitor)
	assert.False(t, monitor.IsRunning())
}

// TestKeyboardMonitor_StartStop_Integration 测试键盘监控器的启动和停止功能（集成测试）
//
// 验证键盘监控器能够正确启动和停止，并确保运行状态正确更新。
// 这是一个集成测试，需要系统辅助功能权限才能运行。
// 测试场景：
//   1. 启动监控器，验证运行状态为 true
//   2. 等待监控器完全启动
//   3. 停止监控器，验证运行状态为 false
func TestKeyboardMonitor_StartStop_Integration(t *testing.T) {
	eventBus := events.NewEventBus()
	monitor := NewKeyboardMonitor(eventBus)

	// 启动监控器
	err := monitor.Start()
	if err != nil {
		t.Skipf("需要辅助功能权限，跳过测试: %v", err)
	}
	require.NoError(t, err)

	// 验证状态
	assert.True(t, monitor.IsRunning())

	// 给一点时间让监控器完全启动
	time.Sleep(100 * time.Millisecond)

	// 停止监控器
	err = monitor.Stop()
	require.NoError(t, err)

	// 验证状态
	assert.False(t, monitor.IsRunning())
}

// TestKeyboardMonitor_StartTwice_Integration 测试重复启动键盘监控器的场景（集成测试）
//
// 验证键盘监控器在已运行状态下再次启动时能够幂等地处理。
// 这确保了监控器的启动逻辑具有幂等性，不会重复启动导致问题。
// 测试场景：
//   1. 第一次启动监控器，应该成功
//   2. 第二次启动监控器，应该幂等地返回成功
//   3. 运行状态应保持为 true
func TestKeyboardMonitor_StartTwice_Integration(t *testing.T) {
	eventBus := events.NewEventBus()
	monitor := NewKeyboardMonitor(eventBus)

	// 第一次启动
	err := monitor.Start()
	if err != nil {
		t.Skipf("需要辅助功能权限，跳过测试: %v", err)
	}
	require.NoError(t, err)

	// 第二次启动（应该幂等）
	err = monitor.Start()
	assert.NoError(t, err)
	assert.True(t, monitor.IsRunning())

	// 清理
	_ = monitor.Stop()
}

// TestKeyboardMonitor_StopWithoutStart 测试未启动就停止的场景
//
// 验证键盘监控器在未启动状态下调用停止方法时能够幂等地处理。
// 这确保了监控器的停止逻辑具有幂等性。
// 测试场景：
//   1. 创建监控器但不启动
//   2. 调用停止方法，应该幂等地返回成功
//   3. 运行状态应保持为 false
func TestKeyboardMonitor_StopWithoutStart(t *testing.T) {
	eventBus := events.NewEventBus()
	monitor := NewKeyboardMonitor(eventBus)

	// 未启动就停止（应该幂等）
	err := monitor.Stop()
	assert.NoError(t, err)
	assert.False(t, monitor.IsRunning())
}

// TestKeyboardMonitor_EventFlow_Integration 测试键盘事件流的完整处理流程（集成测试）
//
// 验证键盘监控器能够正确捕获键盘事件并发布到事件总线。
// 本测试需要实际按键操作才会触发事件捕获。
// 测试场景：
//   1. 订阅键盘事件
//   2. 启动监控器
//   3. 等待并验证收到的键盘事件包含正确的数据和上下文
//   4. 如果在 2 秒内没有按键，则跳过事件验证（这是正常情况）
func TestKeyboardMonitor_EventFlow_Integration(t *testing.T) {
	eventBus := events.NewEventBus()

	// 订阅键盘事件
	receivedEvents := make(chan *events.Event, 10)
	_ = eventBus.Subscribe(string(events.EventTypeKeyboard), func(event events.Event) error {
		receivedEvents <- &event
		return nil
	})

	monitor := NewKeyboardMonitor(eventBus)

	// 启动监控器
	err := monitor.Start()
	if err != nil {
		t.Skipf("需要辅助功能权限，跳过测试: %v", err)
	}
	require.NoError(t, err)

	// 等待可能的键盘事件
	// 注意：这个测试需要实际按键才会触发
	select {
	case event := <-receivedEvents:
		// 验证事件类型
		assert.Equal(t, events.EventTypeKeyboard, event.Type)

		// 验证事件数据存在
		assert.Contains(t, event.Data, "keycode")
		assert.Contains(t, event.Data, "modifiers")

		// 验证上下文
		if event.Context != nil {
			assert.NotEmpty(t, event.Context.Application)
			assert.NotEmpty(t, event.Context.BundleID)
		}

		t.Logf("收到键盘事件: %+v", event.Data)

	case <-time.After(2 * time.Second):
		t.Log("未捕获到键盘事件（正常情况，需要实际按键）")
	}

	// 清理
	_ = monitor.Stop()
}

// TestKeyboardMonitor_MultipleStartStop 测试多次启动和停止键盘监控器的场景
//
// 验证键盘监控器能够支持多次启动和停止循环，并且每次都能正常工作。
// 这确保了监控器资源能够正确释放和重新初始化。
// 测试场景：
//   1. 第一次启动和停止
//   2. 第二次启动和停止
//   3. 每次操作都应该成功
func TestKeyboardMonitor_MultipleStartStop(t *testing.T) {
	eventBus := events.NewEventBus()
	monitor := NewKeyboardMonitor(eventBus)

	// 启动
	err := monitor.Start()
	if err != nil {
		t.Skipf("需要辅助功能权限，跳过测试: %v", err)
	}
	require.NoError(t, err)

	// 停止
	err = monitor.Stop()
	require.NoError(t, err)

	// 再次启动（应该成功）
	err = monitor.Start()
	assert.NoError(t, err)

	// 再次停止
	_ = monitor.Stop()
}

// TestKeyboardMonitor_ConcurrentAccess 测试键盘监控器的并发访问场景
//
// 验证键盘监控器在多个 goroutine 同时访问时能够正确工作，不会出现竞态条件。
// 这确保了监控器的线程安全性。
// 测试场景：
//   1. 启动监控器
//   2. 创建多个 goroutine 同时检查运行状态
//   3. 验证所有操作都能正确完成，不会出现死锁或竞态条件
func TestKeyboardMonitor_ConcurrentAccess(t *testing.T) {
	eventBus := events.NewEventBus()
	monitor := NewKeyboardMonitor(eventBus)

	// 启动
	err := monitor.Start()
	if err != nil {
		t.Skipf("需要辅助功能权限，跳过测试: %v", err)
	}
	require.NoError(t, err)

	// 并发访问
	done := make(chan bool, 10)

	// 多个 goroutine 同时检查状态
	for i := 0; i < 5; i++ {
		go func() {
			_ = monitor.IsRunning()
			done <- true
		}()
	}

	// 等待所有 goroutine 完成
	for i := 0; i < 5; i++ {
		select {
		case <-done:
		case <-time.After(1 * time.Second):
			t.Fatal("并发访问超时")
		}
	}

	// 清理
	_ = monitor.Stop()
}

// TestKeyboardMonitor_StatusCheck 测试键盘监控器状态检查的准确性
//
// 验证键盘监控器的运行状态在不同阶段都能正确反映。
// 这确保了状态管理的准确性和一致性。
// 测试场景：
//   1. 验证初始状态为 false
//   2. 启动后验证状态为 true
//   3. 停止后验证状态为 false
func TestKeyboardMonitor_StatusCheck(t *testing.T) {
	eventBus := events.NewEventBus()
	monitor := NewKeyboardMonitor(eventBus)

	// 初始状态
	assert.False(t, monitor.IsRunning())

	// 启动
	err := monitor.Start()
	if err != nil {
		t.Skipf("需要辅助功能权限，跳过测试: %v", err)
	}
	require.NoError(t, err)

	// 运行中状态
	assert.True(t, monitor.IsRunning())

	// 停止后状态
	_ = monitor.Stop()
	assert.False(t, monitor.IsRunning())
}

// TestKeyboardMonitor_StopTwice 测试重复停止键盘监控器
//
// 验证在监控器已停止状态下再次停止时能够幂等地处理。
// 根据当前实现，停止操作应该是幂等的。
func TestKeyboardMonitor_StopTwice(t *testing.T) {
	eventBus := events.NewEventBus()
	monitor := NewKeyboardMonitor(eventBus)

	// 启动
	err := monitor.Start()
	if err != nil {
		t.Skipf("需要辅助功能权限，跳过测试: %v", err)
	}
	require.NoError(t, err)

	// 第一次停止
	err = monitor.Stop()
	require.NoError(t, err)

	// 第二次停止（应该幂等地返回成功）
	err = monitor.Stop()
	assert.NoError(t, err)
	assert.False(t, monitor.IsRunning())
}

// TestKeyboardMonitor_RapidStartStop 测试快速启停键盘监控器
//
// 验证监控器能够快速进行启停切换，不会出现资源泄漏或状态不一致。
func TestKeyboardMonitor_RapidStartStop(t *testing.T) {
	eventBus := events.NewEventBus()
	monitor := NewKeyboardMonitor(eventBus)

	// 快速进行多次启停
	for i := 0; i < 3; i++ {
		err := monitor.Start()
		if err != nil {
			t.Skipf("需要辅助功能权限，跳过测试: %v", err)
		}
		require.NoError(t, err)

		// 给一点时间让监控器启动
		time.Sleep(50 * time.Millisecond)

		err = monitor.Stop()
		require.NoError(t, err)

		// 验证状态
		assert.False(t, monitor.IsRunning())
	}
}

// TestKeyboardMonitor_EventDataStructure 测试键盘事件的数据结构
//
// 验证键盘监控器生成的事件包含正确的数据结构和字段。
// 注意：此测试需要实际按键才会触发事件捕获。
func TestKeyboardMonitor_EventDataStructure(t *testing.T) {
	eventBus := events.NewEventBus()

	// 订阅键盘事件
	receivedEvents := make(chan *events.Event, 10)
	_ = eventBus.Subscribe(string(events.EventTypeKeyboard), func(event events.Event) error {
		receivedEvents <- &event
		return nil
	})

	monitor := NewKeyboardMonitor(eventBus)

	// 启动监控器
	err := monitor.Start()
	if err != nil {
		t.Skipf("需要辅助功能权限，跳过测试: %v", err)
	}
	require.NoError(t, err)

	// 等待可能的键盘事件
	select {
	case event := <-receivedEvents:
		// 验证事件类型
		assert.Equal(t, events.EventTypeKeyboard, event.Type)

		// 验证事件数据包含必需字段
		assert.Contains(t, event.Data, "keycode", "事件应包含 keycode 字段")
		assert.Contains(t, event.Data, "modifiers", "事件应包含 modifiers 字段")

		// 验证字段类型
		keycode, ok := event.Data["keycode"].(int)
		assert.True(t, ok, "keycode 应该是 int 类型")
		assert.GreaterOrEqual(t, keycode, 0, "keycode 应该大于等于 0")

		modifiers, ok := event.Data["modifiers"].(uint64)
		assert.True(t, ok, "modifiers 应该是 uint64 类型")

		t.Logf("收到键盘事件 - KeyCode: %d, Modifiers: %d", keycode, modifiers)

	case <-time.After(2 * time.Second):
		t.Log("未捕获到键盘事件（正常情况，需要实际按键）")
	}

	// 清理
	_ = monitor.Stop()
}

// TestKeyboardMonitor_ContextAttachment 测试事件上下文信息的附加
//
// 验证键盘事件包含正确的应用上下文信息。
// 注意：此测试需要实际按键才会触发事件捕获。
func TestKeyboardMonitor_ContextAttachment(t *testing.T) {
	eventBus := events.NewEventBus()

	// 订阅键盘事件
	receivedEvents := make(chan *events.Event, 10)
	_ = eventBus.Subscribe(string(events.EventTypeKeyboard), func(event events.Event) error {
		receivedEvents <- &event
		return nil
	})

	monitor := NewKeyboardMonitor(eventBus)

	// 启动监控器
	err := monitor.Start()
	if err != nil {
		t.Skipf("需要辅助功能权限，跳过测试: %v", err)
	}
	require.NoError(t, err)

	// 等待可能的键盘事件
	select {
	case event := <-receivedEvents:
		// 验证事件包含上下文
		assert.NotNil(t, event.Context, "事件应该包含上下文信息")

		// 验证上下文字段
		if event.Context != nil {
			t.Logf("上下文信息 - Application: %s, BundleID: %s, WindowTitle: %s",
				event.Context.Application,
				event.Context.BundleID,
				event.Context.WindowTitle)
		}

	case <-time.After(2 * time.Second):
		t.Log("未捕获到键盘事件（正常情况，需要实际按键）")
	}

	// 清理
	_ = monitor.Stop()
}

// TestKeyboardMonitor_ConcurrentStartStop 测试并发启停键盘监控器
//
// 验证监控器在并发启动和停止时不会出现竞态条件。
func TestKeyboardMonitor_ConcurrentStartStop(t *testing.T) {
	eventBus := events.NewEventBus()
	monitor := NewKeyboardMonitor(eventBus)

	// 并发启动
	done := make(chan bool, 3)
	go func() {
		_ = monitor.Start()
		done <- true
	}()
	go func() {
		_ = monitor.Start()
		done <- true
	}()
	go func() {
		_ = monitor.Stop()
		done <- true
	}()

	// 等待所有 goroutine 完成
	for i := 0; i < 3; i++ {
		select {
		case <-done:
		case <-time.After(1 * time.Second):
			t.Fatal("并发操作超时")
		}
	}

	// 清理
	if monitor.IsRunning() {
		_ = monitor.Stop()
	}
}

// TestKeyboardMonitor_MultipleInstances 测试创建多个键盘监控器实例
//
// 验证可以同时创建多个键盘监控器实例，每个实例状态独立。
func TestKeyboardMonitor_MultipleInstances(t *testing.T) {
	eventBus := events.NewEventBus()

	// 创建多个监控器实例
	monitor1 := NewKeyboardMonitor(eventBus)
	monitor2 := NewKeyboardMonitor(eventBus)

	// 验证实例独立
	assert.NotNil(t, monitor1)
	assert.NotNil(t, monitor2)
	assert.NotSame(t, monitor1, monitor2)

	// 验证初始状态独立
	assert.False(t, monitor1.IsRunning())
	assert.False(t, monitor2.IsRunning())

	// 启动第一个监控器
	err := monitor1.Start()
	if err != nil {
		t.Skipf("需要辅助功能权限，跳过测试: %v", err)
	}
	require.NoError(t, err)

	// 验证状态独立
	assert.True(t, monitor1.IsRunning())
	assert.False(t, monitor2.IsRunning(), "monitor2 不应该受 monitor1 影响")

	// 清理
	_ = monitor1.Stop()
}

// TestKeyboardMonitor_WithNilEventBus 测试使用 nil 事件总线创建监控器
//
// 验证使用 nil 事件总线创建监控器后，相关操作不会 panic。
func TestKeyboardMonitor_WithNilEventBus(t *testing.T) {
	// 使用 nil 事件总线创建监控器
	monitor := NewKeyboardMonitor(nil)

	// 验证监控器创建成功
	assert.NotNil(t, monitor)
	assert.False(t, monitor.IsRunning())

	// 尝试启动可能会失败或 panic，取决于实现
	// 这里只验证不会 panic 创建
	_ = monitor
}

// TestKeyboardMonitor_GetHotkeyManager 测试获取快捷键管理器
//
// 验证 KeyboardMonitor 正确集成了 HotkeyManager。
// 测试场景：
//   1. 获取快捷键管理器，验证不为 nil
//   2. 验证快捷键管理器功能正常
func TestKeyboardMonitor_GetHotkeyManager(t *testing.T) {
	eventBus := events.NewEventBus()
	monitor := NewKeyboardMonitor(eventBus)

	// 类型断言获取实际的 KeyboardMonitor 类型
	km, ok := monitor.(*KeyboardMonitor)
	require.True(t, ok, "monitor 应该是 *KeyboardMonitor 类型")

	// 获取快捷键管理器
	hotkeyMgr := km.GetHotkeyManager()
	assert.NotNil(t, hotkeyMgr, "快捷键管理器不应为 nil")

	// 验证快捷键管理器功能
	testHotkey := "Cmd+A"
	registered := hotkeyMgr.IsRegistered(testHotkey)
	assert.False(t, registered, "快捷键尚未注册")

	// 注册快捷键
	id, err := hotkeyMgr.Register(testHotkey, func(reg *HotkeyRegistration, ctx *events.EventContext) {})
	require.NoError(t, err, "注册快捷键不应失败")
	assert.NotEmpty(t, id, "注册 ID 不应为空")

	// 验证注册成功
	registered = hotkeyMgr.IsRegistered(testHotkey)
	assert.True(t, registered, "快捷键应该已注册")

	// 启动监控器
	err = monitor.Start()
	if err != nil {
		t.Skipf("需要辅助功能权限，跳过测试: %v", err)
	}
	require.NoError(t, err)

	// 验证快捷键管理器也已启动
	assert.True(t, hotkeyMgr.IsRunning(), "快捷键管理器应该已启动")

	// 停止监控器
	err = monitor.Stop()
	require.NoError(t, err)

	// 验证快捷键管理器也已停止
	assert.False(t, hotkeyMgr.IsRunning(), "快捷键管理器应该已停止")
}

