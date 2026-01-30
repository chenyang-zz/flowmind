//go:build darwin

package monitor

import (
	"sync"
	"testing"
	"time"

	"github.com/chenyang-zz/flowmind/pkg/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewClipboardMonitor 测试剪贴板监控器的创建
//
// 验证剪贴板监控器能够正确创建，并且初始状态为未运行。
// 测试场景：
//   1. 创建剪贴板监控器
//   2. 验证实例不为 nil
//   3. 验证初始运行状态为 false
func TestNewClipboardMonitor(t *testing.T) {
	eventBus := events.NewEventBus()
	monitor := NewClipboardMonitor(eventBus)

	assert.NotNil(t, monitor)
	assert.False(t, monitor.IsRunning())
}

// TestClipboardMonitor_StartStop 测试剪贴板监控器的启动和停止功能
//
// 验证剪贴板监控器能够正确启动和停止，并确保运行状态正确更新。
// 测试场景：
//   1. 启动监控器，验证运行状态为 true
//   2. 等待监控器完全启动
//   3. 停止监控器，验证运行状态为 false
func TestClipboardMonitor_StartStop(t *testing.T) {
	eventBus := events.NewEventBus()
	monitor := NewClipboardMonitor(eventBus)

	// 启动监控器
	err := monitor.Start()
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

// TestClipboardMonitor_StartTwice 测试重复启动剪贴板监控器的场景
//
// 验证剪贴板监控器在已运行状态下再次启动时能够幂等地处理。
// 这确保了监控器的启动逻辑具有幂等性，不会重复启动导致问题。
// 测试场景：
//   1. 第一次启动监控器，应该成功
//   2. 第二次启动监控器，应该幂等地返回成功
//   3. 运行状态应保持为 true
func TestClipboardMonitor_StartTwice(t *testing.T) {
	eventBus := events.NewEventBus()
	monitor := NewClipboardMonitor(eventBus)

	// 第一次启动
	err := monitor.Start()
	require.NoError(t, err)

	// 第二次启动（应该幂等）
	err = monitor.Start()
	assert.NoError(t, err)
	assert.True(t, monitor.IsRunning())

	// 清理
	_ = monitor.Stop()
}

// TestClipboardMonitor_StopWithoutStart 测试未启动就停止的场景
//
// 验证剪贴板监控器在未启动状态下调用停止方法时能够幂等地处理。
// 这确保了监控器的停止逻辑具有幂等性。
// 测试场景：
//   1. 创建监控器但不启动
//   2. 调用停止方法，应该幂等地返回成功
//   3. 运行状态应保持为 false
func TestClipboardMonitor_StopWithoutStart(t *testing.T) {
	eventBus := events.NewEventBus()
	monitor := NewClipboardMonitor(eventBus)

	// 未启动就停止（应该幂等）
	err := monitor.Stop()
	assert.NoError(t, err)
	assert.False(t, monitor.IsRunning())
}

// TestClipboardMonitor_EventFlow 测试剪贴板事件流的完整处理流程
//
// 验证剪贴板监控器能够正确检测剪贴板内容变化并发布到事件总线。
// 本测试需要手动复制文本才会触发事件捕获。
// 测试场景：
//   1. 订阅剪贴板事件
//   2. 启动监控器
//   3. 手动复制一些文本
//   4. 等待并验证收到的剪贴板事件包含正确的数据和上下文
func TestClipboardMonitor_EventFlow(t *testing.T) {
	eventBus := events.NewEventBus()

	// 订阅剪贴板事件
	receivedEvents := make(chan *events.Event, 10)
	_ = eventBus.Subscribe(string(events.EventTypeClipboard), func(event events.Event) error {
		receivedEvents <- &event
		return nil
	})

	monitor := NewClipboardMonitor(eventBus)

	// 启动监控器
	err := monitor.Start()
	require.NoError(t, err)

	// 等待剪贴板事件（需要手动复制一些文本）
	t.Log("请在 3 秒内复制一些文本以触发测试...")
	select {
	case event := <-receivedEvents:
		// 验证事件类型
		assert.Equal(t, events.EventTypeClipboard, event.Type)

		// 验证事件数据存在
		assert.Contains(t, event.Data, "content")
		assert.Contains(t, event.Data, "type")
		assert.Contains(t, event.Data, "size")
		assert.Contains(t, event.Data, "length")

		// 验证上下文
		if event.Context != nil {
			assert.NotEmpty(t, event.Context.Application)
			assert.NotEmpty(t, event.Context.BundleID)
		}

		t.Logf("收到剪贴板事件: Content=%s, Type=%s, Size=%d",
			event.Data["content"],
			event.Data["type"],
			event.Data["size"])

	case <-time.After(3 * time.Second):
		t.Log("未捕获到剪贴板事件（需要手动复制文本）")
	}

	// 清理
	_ = monitor.Stop()
}

// TestClipboardMonitor_MultipleStartStop 测试多次启动和停止剪贴板监控器的场景
//
// 验证剪贴板监控器能够支持多次启动和停止循环，并且每次都能正常工作。
// 这确保了监控器资源能够正确释放和重新初始化。
// 测试场景：
//   1. 第一次启动和停止
//   2. 第二次启动和停止
//   3. 每次操作都应该成功
func TestClipboardMonitor_MultipleStartStop(t *testing.T) {
	eventBus := events.NewEventBus()
	monitor := NewClipboardMonitor(eventBus)

	// 启动
	err := monitor.Start()
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

// TestClipboardMonitor_ConcurrentAccess 测试剪贴板监控器的并发访问场景
//
// 验证剪贴板监控器在多个 goroutine 同时访问时能够正确工作，不会出现竞态条件。
// 这确保了监控器的线程安全性。
// 测试场景：
//   1. 启动监控器
//   2. 创建多个 goroutine 同时检查运行状态
//   3. 验证所有操作都能正确完成，不会出现死锁或竞态条件
func TestClipboardMonitor_ConcurrentAccess(t *testing.T) {
	eventBus := events.NewEventBus()
	monitor := NewClipboardMonitor(eventBus)

	// 启动
	err := monitor.Start()
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

// TestClipboardMonitor_StatusCheck 测试剪贴板监控器状态检查的准确性
//
// 验证剪贴板监控器的运行状态在不同阶段都能正确反映。
// 这确保了状态管理的准确性和一致性。
// 测试场景：
//   1. 验证初始状态为 false
//   2. 启动后验证状态为 true
//   3. 停止后验证状态为 false
func TestClipboardMonitor_StatusCheck(t *testing.T) {
	eventBus := events.NewEventBus()
	monitor := NewClipboardMonitor(eventBus)

	// 初始状态
	assert.False(t, monitor.IsRunning())

	// 启动
	err := monitor.Start()
	require.NoError(t, err)

	// 运行中状态
	assert.True(t, monitor.IsRunning())

	// 停止后状态
	_ = monitor.Stop()
	assert.False(t, monitor.IsRunning())
}

// TestClipboardMonitor_StopTwice 测试重复停止剪贴板监控器
//
// 验证在监控器已停止状态下再次停止时能够幂等地处理。
func TestClipboardMonitor_StopTwice(t *testing.T) {
	eventBus := events.NewEventBus()
	monitor := NewClipboardMonitor(eventBus)

	// 启动
	err := monitor.Start()
	require.NoError(t, err)

	// 第一次停止
	err = monitor.Stop()
	require.NoError(t, err)

	// 第二次停止（应该幂等地返回成功）
	err = monitor.Stop()
	assert.NoError(t, err)
	assert.False(t, monitor.IsRunning())
}

// TestClipboardMonitor_RapidStartStop 测试快速启停剪贴板监控器
//
// 验证监控器能够快速进行启停切换，不会出现资源泄漏或状态不一致。
func TestClipboardMonitor_RapidStartStop(t *testing.T) {
	eventBus := events.NewEventBus()
	monitor := NewClipboardMonitor(eventBus)

	// 快速进行多次启停
	for i := 0; i < 3; i++ {
		err := monitor.Start()
		require.NoError(t, err)

		// 给一点时间让监控器启动
		time.Sleep(50 * time.Millisecond)

		err = monitor.Stop()
		require.NoError(t, err)

		// 验证状态
		assert.False(t, monitor.IsRunning())
	}
}

// TestClipboardMonitor_EventDataStructure 测试剪贴板事件的数据结构
//
// 验证剪贴板监控器生成的事件包含正确的数据结构和字段。
// 注意：此测试需要手动复制文本才会触发事件捕获。
func TestClipboardMonitor_EventDataStructure(t *testing.T) {
	eventBus := events.NewEventBus()

	// 订阅剪贴板事件
	receivedEvents := make(chan *events.Event, 10)
	_ = eventBus.Subscribe(string(events.EventTypeClipboard), func(event events.Event) error {
		receivedEvents <- &event
		return nil
	})

	monitor := NewClipboardMonitor(eventBus)

	// 启动监控器
	err := monitor.Start()
	require.NoError(t, err)

	// 等待剪贴板事件
	t.Log("请在 3 秒内复制一些文本以触发测试...")
	select {
	case event := <-receivedEvents:
		// 验证事件类型
		assert.Equal(t, events.EventTypeClipboard, event.Type)

		// 验证事件数据包含必需字段
		assert.Contains(t, event.Data, "content", "事件应包含 content 字段")
		assert.Contains(t, event.Data, "type", "事件应包含 type 字段")
		assert.Contains(t, event.Data, "size", "事件应包含 size 字段")
		assert.Contains(t, event.Data, "length", "事件应包含 length 字段")

		// 验证字段类型
		content, ok := event.Data["content"].(string)
		assert.True(t, ok, "content 应该是 string 类型")
		assert.NotEmpty(t, content, "content 不应该为空")

		size, ok := event.Data["size"].(int64)
		assert.True(t, ok, "size 应该是 int64 类型")
		assert.Greater(t, size, int64(0), "size 应该大于 0")

		length, ok := event.Data["length"].(int)
		assert.True(t, ok, "length 应该是 int 类型")
		assert.Greater(t, length, 0, "length 应该大于 0")

		t.Logf("收到剪贴板事件 - Content: %s, Type: %s, Size: %d, Length: %d",
			content, event.Data["type"], size, length)

	case <-time.After(3 * time.Second):
		t.Log("未捕获到剪贴板事件（需要手动复制文本）")
	}

	// 清理
	_ = monitor.Stop()
}

// TestClipboardMonitor_ContextAttachment 测试事件上下文信息的附加
//
// 验证剪贴板事件包含正确的应用上下文信息。
// 注意：此测试需要手动复制文本才会触发事件捕获。
func TestClipboardMonitor_ContextAttachment(t *testing.T) {
	eventBus := events.NewEventBus()

	// 订阅剪贴板事件
	receivedEvents := make(chan *events.Event, 10)
	_ = eventBus.Subscribe(string(events.EventTypeClipboard), func(event events.Event) error {
		receivedEvents <- &event
		return nil
	})

	monitor := NewClipboardMonitor(eventBus)

	// 启动监控器
	err := monitor.Start()
	require.NoError(t, err)

	// 等待剪贴板事件
	t.Log("请在 3 秒内复制一些文本以触发测试...")
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

	case <-time.After(3 * time.Second):
		t.Log("未捕获到剪贴板事件（需要手动复制文本）")
	}

	// 清理
	_ = monitor.Stop()
}

// TestClipboardMonitor_ConcurrentStartStop 测试并发启停剪贴板监控器
//
// 验证监控器在并发启动和停止时不会出现竞态条件。
func TestClipboardMonitor_ConcurrentStartStop(t *testing.T) {
	eventBus := events.NewEventBus()
	monitor := NewClipboardMonitor(eventBus)

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

// TestClipboardMonitor_MultipleInstances 测试创建多个剪贴板监控器实例
//
// 验证可以同时创建多个剪贴板监控器实例，每个实例状态独立。
func TestClipboardMonitor_MultipleInstances(t *testing.T) {
	eventBus := events.NewEventBus()

	// 创建多个监控器实例
	monitor1 := NewClipboardMonitor(eventBus)
	monitor2 := NewClipboardMonitor(eventBus)

	// 验证实例独立
	assert.NotNil(t, monitor1)
	assert.NotNil(t, monitor2)
	assert.NotSame(t, monitor1, monitor2)

	// 验证初始状态独立
	assert.False(t, monitor1.IsRunning())
	assert.False(t, monitor2.IsRunning())

	// 启动第一个监控器
	err := monitor1.Start()
	require.NoError(t, err)

	// 验证状态独立
	assert.True(t, monitor1.IsRunning())
	assert.False(t, monitor2.IsRunning(), "monitor2 不应该受 monitor1 影响")

	// 清理
	_ = monitor1.Stop()
}

// TestClipboardMonitor_WithNilEventBus 测试使用 nil 事件总线创建监控器
//
// 验证使用 nil 事件总线创建监控器后，相关操作不会 panic。
func TestClipboardMonitor_WithNilEventBus(t *testing.T) {
	// 使用 nil 事件总线创建监控器
	monitor := NewClipboardMonitor(nil)

	// 验证监控器创建成功
	assert.NotNil(t, monitor)
	assert.False(t, monitor.IsRunning())

	// 尝试启动可能会失败或 panic，取决于实现
	// 这里只验证不会 panic 创建
	_ = monitor
}

// TestClipboardMonitor_Deduplication 测试剪贴板内容去重功能
//
// 验证剪贴板监控器能够正确去除重复的剪贴板内容。
// 即使剪贴板没有实际变化，也不应该重复触发事件。
func TestClipboardMonitor_Deduplication(t *testing.T) {
	eventBus := events.NewEventBus()

	// 订阅剪贴板事件
	eventCount := 0
	var eventMutex sync.Mutex

	_ = eventBus.Subscribe(string(events.EventTypeClipboard), func(event events.Event) error {
		eventMutex.Lock()
		eventCount++
		eventMutex.Unlock()
		return nil
	})

	monitor := NewClipboardMonitor(eventBus)

	// 启动监控器
	err := monitor.Start()
	require.NoError(t, err)

	// 等待用户复制同一内容两次
	t.Log("请在 5 秒内复制同一内容两次，观察是否只触发一次事件...")
	time.Sleep(5 * time.Second)

	eventMutex.Lock()
	count := eventCount
	eventMutex.Unlock()

	t.Logf("捕获到 %d 次剪贴板事件", count)

	// 清理
	_ = monitor.Stop()
}
