//go:build darwin

package monitor

import (
	"testing"
	"time"

	"github.com/chenyang-zz/flowmind/pkg/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEngine_StartStop 测试监控引擎的启动和停止功能
//
// 验证监控引擎能够正确启动和停止，并确保运行状态正确更新。
// 测试场景：
//   1. 创建监控引擎
//   2. 启动引擎，验证运行状态为 true
//   3. 停止引擎，验证运行状态为 false
func TestEngine_StartStop(t *testing.T) {
	// 创建事件总线
	eventBus := events.NewEventBus()

	// 创建监控引擎
	engine := NewEngine(eventBus)

	// 测试启动
	err := engine.Start()
	if err != nil {
		t.Skip("需要辅助功能权限，跳过测试")
	}
	require.NoError(t, err)

	// 验证运行状态
	assert.True(t, engine.IsRunning())

	// 通过类型断言获取具体的 Engine 实例来访问 GetKeyboardMonitor
	if concreteEngine, ok := engine.(*Engine); ok {
		assert.NotNil(t, concreteEngine.GetKeyboardMonitor())
	}

	// 测试停止
	err = engine.Stop()
	require.NoError(t, err)

	// 验证停止状态
	assert.False(t, engine.IsRunning())
}

// TestEngine_StartTwice 测试重复启动监控引擎的场景
//
// 验证监控引擎在已运行状态下再次启动时会返回错误。
// 这确保了引擎不会重复启动导致资源泄漏。
// 测试场景：
//   1. 第一次启动引擎，应该成功
//   2. 第二次启动引擎，应该失败并返回 "already running" 错误
func TestEngine_StartTwice(t *testing.T) {
	// 创建事件总线
	eventBus := events.NewEventBus()

	// 创建监控引擎
	engine := NewEngine(eventBus)

	// 第一次启动
	err := engine.Start()
	if err != nil {
		t.Skip("需要辅助功能权限，跳过测试")
	}
	require.NoError(t, err)

	// 第二次启动应该失败
	err = engine.Start()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already running")

	// 清理
	_ = engine.Stop()
}

// TestEngine_StopWithoutStart 测试未启动就停止的场景
//
// 验证监控引擎在未启动状态下调用停止方法时会返回错误。
// 这确保了引擎的状态管理逻辑的正确性。
// 测试场景：
//   1. 创建监控引擎但不启动
//   2. 调用停止方法，应该失败并返回 "not running" 错误
func TestEngine_StopWithoutStart(t *testing.T) {
	// 创建事件总线
	eventBus := events.NewEventBus()

	// 创建监控引擎
	engine := NewEngine(eventBus)

	// 未启动就停止应该失败
	err := engine.Stop()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not running")
}

// TestEngine_EventPublishing 测试监控引擎的事件发布功能
//
// 验证监控引擎在启动时会正确发布状态事件到事件总线。
// 确保其他模块可以通过订阅状态事件来监控引擎的运行状态。
// 测试场景：
//   1. 订阅状态事件
//   2. 启动引擎
//   3. 验证收到启动状态事件，事件数据包含正确的状态信息
func TestEngine_EventPublishing(t *testing.T) {
	// 创建事件总线
	eventBus := events.NewEventBus()

	// 订阅键盘事件
	receivedEvents := make(chan *events.Event, 10)
	eventBus.Subscribe(string(events.EventTypeKeyboard), func(event events.Event) error {
		receivedEvents <- &event
		return nil
	})

	// 订阅状态事件
	statusEvents := make(chan *events.Event, 10)
	eventBus.Subscribe(string(events.EventTypeStatus), func(event events.Event) error {
		statusEvents <- &event
		return nil
	})

	// 创建监控引擎
	engine := NewEngine(eventBus)

	// 启动引擎
	err := engine.Start()
	if err != nil {
		t.Skip("需要辅助功能权限，跳过测试")
	}
	require.NoError(t, err)

	// 等待启动状态事件
	select {
	case statusEvent := <-statusEvents:
		assert.Equal(t, events.EventTypeStatus, statusEvent.Type)
		assert.Equal(t, "started", statusEvent.Data["status"])
	case <-time.After(100 * time.Millisecond):
		t.Fatal("未收到启动状态事件")
	}

	// 清理
	_ = engine.Stop()
}

// TestEngine_GetKeyboardMonitor_BeforeStart 测试启动前获取键盘监控器
//
// 验证在引擎启动前调用 GetKeyboardMonitor 返回 nil。
func TestEngine_GetKeyboardMonitor_BeforeStart(t *testing.T) {
	// 创建事件总线
	eventBus := events.NewEventBus()

	// 创建监控引擎
	engine := NewEngine(eventBus)

	// 通过类型断言获取具体的 Engine 实例来访问 GetKeyboardMonitor
	if concreteEngine, ok := engine.(*Engine); ok {
		// 启动前获取键盘监控器应该返回 nil
		km := concreteEngine.GetKeyboardMonitor()
		assert.Nil(t, km, "启动前键盘监控器应该为 nil")
	}
}

// TestEngine_GetKeyboardMonitor_AfterStart 测试启动后获取键盘监控器
//
// 验证引擎启动后能正确返回已初始化的键盘监控器实例。
func TestEngine_GetKeyboardMonitor_AfterStart(t *testing.T) {
	// 创建事件总线
	eventBus := events.NewEventBus()

	// 创建监控引擎
	engine := NewEngine(eventBus)

	// 启动引擎
	err := engine.Start()
	if err != nil {
		t.Skipf("需要辅助功能权限，跳过测试: %v", err)
	}
	require.NoError(t, err)

	// 通过类型断言获取具体的 Engine 实例来访问 GetKeyboardMonitor
	if concreteEngine, ok := engine.(*Engine); ok {
		// 启动后获取键盘监控器应该返回有效实例
		km := concreteEngine.GetKeyboardMonitor()
		assert.NotNil(t, km, "启动后键盘监控器应该不为 nil")
		assert.True(t, km.IsRunning(), "键盘监控器应该正在运行")
	}

	// 清理
	_ = engine.Stop()
}

// TestEngine_StopTwice 测试重复停止监控引擎
//
// 验证在引擎已停止状态下再次停止会返回错误。
func TestEngine_StopTwice(t *testing.T) {
	// 创建事件总线
	eventBus := events.NewEventBus()

	// 创建监控引擎
	engine := NewEngine(eventBus)

	// 启动并停止
	err := engine.Start()
	if err != nil {
		t.Skipf("需要辅助功能权限，跳过测试: %v", err)
	}
	require.NoError(t, err)

	err = engine.Stop()
	require.NoError(t, err)

	// 第二次停止应该失败
	err = engine.Stop()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not running")
}

// TestEngine_StatusChangeEventData 测试状态事件数据的完整性
//
// 验证引擎发布的状态事件包含正确的数据结构。
func TestEngine_StatusChangeEventData(t *testing.T) {
	// 创建事件总线
	eventBus := events.NewEventBus()

	// 订阅状态事件
	statusEvents := make(chan *events.Event, 10)
	eventBus.Subscribe(string(events.EventTypeStatus), func(event events.Event) error {
		statusEvents <- &event
		return nil
	})

	// 创建监控引擎
	engine := NewEngine(eventBus)

	// 启动引擎
	err := engine.Start()
	if err != nil {
		t.Skipf("需要辅助功能权限，跳过测试: %v", err)
	}
	require.NoError(t, err)

	// 验证启动事件
	select {
	case statusEvent := <-statusEvents:
		assert.Equal(t, events.EventTypeStatus, statusEvent.Type)
		assert.Equal(t, "started", statusEvent.Data["status"])

		// 验证监控器列表
		monitors, ok := statusEvent.Data["monitors"].([]string)
		assert.True(t, ok, "monitors 应该是字符串数组")
		assert.Contains(t, monitors, "keyboard", "应该包含 keyboard 监控器")
	case <-time.After(100 * time.Millisecond):
		t.Fatal("未收到启动状态事件")
	}

	// 停止引擎
	err = engine.Stop()
	require.NoError(t, err)

	// 验证停止事件
	select {
	case statusEvent := <-statusEvents:
		assert.Equal(t, events.EventTypeStatus, statusEvent.Type)
		assert.Equal(t, "stopped", statusEvent.Data["status"])
	case <-time.After(100 * time.Millisecond):
		t.Fatal("未收到停止状态事件")
	}
}

// TestEngine_ConcurrentStartStop 测试并发启停引擎
//
// 验证引擎在并发启动和停止时不会出现竞态条件。
func TestEngine_ConcurrentStartStop(t *testing.T) {
	// 创建事件总线
	eventBus := events.NewEventBus()

	// 创建监控引擎
	engine := NewEngine(eventBus)

	// 并发启动
	done := make(chan bool, 2)
	go func() {
		_ = engine.Start()
		done <- true
	}()
	go func() {
		_ = engine.Start()
		done <- true
	}()

	// 等待两个 goroutine 完成
	for i := 0; i < 2; i++ {
		select {
		case <-done:
		case <-time.After(1 * time.Second):
			t.Fatal("并发启动超时")
		}
	}

	// 验证状态
	if engine.IsRunning() {
		// 清理
		_ = engine.Stop()
	}
}

// TestEngine_EngineWithNilEventBus 测试使用 nil 事件总线创建引擎
//
// 验证使用 nil 事件总线创建引擎后，启动时会 panic。
// 这是一个边界测试，验证引擎的正确使用方式。
func TestEngine_EngineWithNilEventBus(t *testing.T) {
	// 使用 nil 事件总线创建引擎
	engine := NewEngine(nil)

	// 验证引擎创建成功
	assert.NotNil(t, engine)
	assert.False(t, engine.IsRunning())

	// 通过类型断言获取具体的 Engine 实例来访问 GetKeyboardMonitor
	if concreteEngine, ok := engine.(*Engine); ok {
		assert.Nil(t, concreteEngine.GetKeyboardMonitor())
	}

	// 启动时应该 panic（因为没有事件总线）
	assert.Panics(t, func() {
		_ = engine.Start()
	}, "使用 nil 事件总线启动引擎应该 panic")
}
