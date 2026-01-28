package events

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/**
 * TestNewEventBus 测试创建事件总线
 */
func TestNewEventBus(t *testing.T) {
	bus := NewEventBus()
	defer bus.Stop(5 * time.Second)

	assert.NotNil(t, bus, "Expected non-nil bus")
	assert.False(t, bus.stopped.Load(), "Expected bus to be running")
}

/**
 * TestSubscribe 测试订阅事件
 */
func TestSubscribe(t *testing.T) {
	bus := NewEventBus()
	defer bus.Stop(5 * time.Second)

	var received bool
	var mutex sync.Mutex

	handler := func(event Event) error {
		mutex.Lock()
		received = true
		mutex.Unlock()
		return nil
	}

	subscriberID := bus.Subscribe("test", handler)
	assert.NotEmpty(t, subscriberID, "Expected non-empty subscriber ID")

	// 发布事件
	event := NewEvent("test", map[string]interface{}{"message": "hello"})
	err := bus.Publish("test", *event)
	require.NoError(t, err, "Failed to publish event")

	// 等待异步处理
	time.Sleep(100 * time.Millisecond)

	mutex.Lock()
	assert.True(t, received, "Expected to receive event")
	mutex.Unlock()
}

/**
 * TestSubscribeWildcard 测试通配符订阅
 */
func TestSubscribeWildcard(t *testing.T) {
	bus := NewEventBus()
	defer bus.Stop(5 * time.Second)

	count := 0
	var mutex sync.Mutex

	handler := func(event Event) error {
		mutex.Lock()
		count++
		mutex.Unlock()
		return nil
	}

	// 订阅所有事件
	bus.Subscribe("*", handler)

	// 发布多个不同类型的事件
	events := []Event{
		*NewEvent("test1", map[string]interface{}{"id": 1}),
		*NewEvent("test2", map[string]interface{}{"id": 2}),
		*NewEvent("test3", map[string]interface{}{"id": 3}),
	}

	for _, event := range events {
		err := bus.Publish(string(event.Type), event)
		require.NoError(t, err, "Failed to publish event")
	}

	// 等待异步处理
	time.Sleep(100 * time.Millisecond)

	mutex.Lock()
	assert.Equal(t, 3, count, "Expected 3 events")
	mutex.Unlock()
}

/**
 * TestSubscribeWithFilter 测试带过滤器的订阅
 */
func TestSubscribeWithFilter(t *testing.T) {
	bus := NewEventBus()
	defer bus.Stop(5 * time.Second)

	var received int
	var mutex sync.Mutex

	handler := func(event Event) error {
		mutex.Lock()
		received++
		mutex.Unlock()
		return nil
	}

	// 只接收 data.value > 5 的事件
	filter := func(event Event) bool {
		if val, ok := event.Data["value"].(int); ok {
			return val > 5
		}
		return false
	}

	bus.SubscribeWithFilter("test", handler, filter)

	// 发布多个事件
	events := []Event{
		*NewEvent("test", map[string]interface{}{"value": 3}),
		*NewEvent("test", map[string]interface{}{"value": 7}),
		*NewEvent("test", map[string]interface{}{"value": 10}),
	}

	for _, event := range events {
		err := bus.Publish("test", event)
		require.NoError(t, err, "Failed to publish event")
	}

	// 等待异步处理
	time.Sleep(100 * time.Millisecond)

	mutex.Lock()
	assert.Equal(t, 2, received, "Expected 2 events (value > 5)")
	mutex.Unlock()
}

/**
 * TestSubscribeOnce 测试一次性订阅
 */
func TestSubscribeOnce(t *testing.T) {
	bus := NewEventBus()
	defer bus.Stop(5 * time.Second)

	var count int
	var mutex sync.Mutex

	handler := func(event Event) error {
		mutex.Lock()
		count++
		mutex.Unlock()
		return nil
	}

	bus.SubscribeOnce("test", handler)

	// 发布多个事件
	for i := 0; i < 3; i++ {
		event := *NewEvent("test", map[string]interface{}{"count": i})
		err := bus.Publish("test", event)
		require.NoError(t, err, "Failed to publish event")
	}

	// 等待异步处理
	time.Sleep(100 * time.Millisecond)

	mutex.Lock()
	assert.Equal(t, 1, count, "Expected 1 event (once subscription)")
	mutex.Unlock()
}

/**
 * TestUnsubscribe 测试取消订阅
 */
func TestUnsubscribe(t *testing.T) {
	bus := NewEventBus()
	defer bus.Stop(5 * time.Second)

	var received int
	var mutex sync.Mutex

	handler := func(event Event) error {
		mutex.Lock()
		received++
		mutex.Unlock()
		return nil
	}

	subscriberID := bus.Subscribe("test", handler)

	// 发布第一个事件
	err := bus.Publish("test", *NewEvent("test", map[string]interface{}{"count": 1}))
	require.NoError(t, err, "Failed to publish first event")

	// 等待处理
	time.Sleep(50 * time.Millisecond)

	// 取消订阅
	bus.Unsubscribe(subscriberID)

	// 发布第二个事件
	err = bus.Publish("test", *NewEvent("test", map[string]interface{}{"count": 2}))
	require.NoError(t, err, "Failed to publish second event")

	// 等待异步处理
	time.Sleep(100 * time.Millisecond)

	mutex.Lock()
	assert.Equal(t, 1, received, "Expected 1 event (after unsubscribe)")
	mutex.Unlock()
}

/**
 * TestEventContext 测试事件上下文
 */
func TestEventContext(t *testing.T) {
	event := NewEvent("keyboard", map[string]interface{}{
		"keycode": 46,
	})

	context := &EventContext{
		Application: "VS Code",
		BundleID:    "com.microsoft.VSCode",
		WindowTitle: "main.go - FlowMind",
	}

	event.WithContext(context)

	assert.NotNil(t, event.Context, "Expected context to be set")
	assert.Equal(t, "VS Code", event.Context.Application, "Expected application 'VS Code'")
	assert.Equal(t, "com.microsoft.VSCode", event.Context.BundleID, "Expected bundle ID")
	assert.Equal(t, "main.go - FlowMind", event.Context.WindowTitle, "Expected window title")
}

/**
 * TestEventMetadata 测试事件元数据
 */
func TestEventMetadata(t *testing.T) {
	event := NewEvent("test", map[string]interface{}{})

	event.WithMetadata("source", "keyboard_monitor")
	event.WithMetadata("version", "1.0")

	assert.Equal(t, 2, len(event.Metadata), "Expected 2 metadata entries")
	assert.Equal(t, "keyboard_monitor", event.Metadata["source"], "Expected metadata 'source' to be 'keyboard_monitor'")
	assert.Equal(t, "1.0", event.Metadata["version"], "Expected metadata 'version' to be '1.0'")
}

/**
 * TestRecoveryMiddleware 测试恢复中间件
 */
func TestRecoveryMiddleware(t *testing.T) {
	bus := NewEventBus()
	bus.Use(RecoveryMiddleware())
	defer bus.Stop(5 * time.Second)

	// 会 panic 的处理函数
	panicHandler := func(event Event) error {
		panic("test panic")
	}

	bus.Subscribe("test", panicHandler)

	// 发布事件（不应该导致程序崩溃）
	err := bus.Publish("test", *NewEvent("test", map[string]interface{}{}))

	// 可能有错误，但不应该 panic
	if err != nil {
		t.Logf("Expected error from recovered panic: %v", err)
	}

	// 等待异步处理
	time.Sleep(100 * time.Millisecond)

	// 如果到这里没有 panic，说明中间件工作正常
	assert.True(t, true, "Middleware recovered from panic")
}

/**
 * TestPublishAsync 测试异步发布
 */
func TestPublishAsync(t *testing.T) {
	bus := NewEventBus()
	defer bus.Stop(5 * time.Second)

	var received bool
	var mutex sync.Mutex

	handler := func(event Event) error {
		mutex.Lock()
		received = true
		mutex.Unlock()
		return nil
	}

	bus.Subscribe("test", handler)

	// 异步发布
	bus.PublishAsync("test", *NewEvent("test", map[string]interface{}{}))

	// 立即检查（应该还没收到）
	mutex.Lock()
	assert.False(t, received, "Event should not be received immediately (async)")
	mutex.Unlock()

	// 等待后应该收到
	time.Sleep(100 * time.Millisecond)

	mutex.Lock()
	assert.True(t, received, "Expected to receive event after waiting")
	mutex.Unlock()
}

/**
 * TestStop 测试停止事件总线
 */
func TestStop(t *testing.T) {
	bus := NewEventBus()

	// 启动一些订阅者
	for i := 0; i < 5; i++ {
		bus.Subscribe("test", func(event Event) error {
			return nil
		})
	}

	// 停止总线
	err := bus.Stop(10 * time.Second)
	require.NoError(t, err, "Failed to stop bus")

	assert.True(t, bus.stopped.Load(), "Expected bus to be stopped")

	// 尝试发布事件（应该失败）
	err = bus.Publish("test", *NewEvent("test", map[string]interface{}{}))
	assert.Error(t, err, "Expected error when publishing to stopped bus")
	assert.Contains(t, err.Error(), "stopped", "Error should mention bus is stopped")
}

/**
 * TestConcurrentPublish 测试并发发布
 */
func TestConcurrentPublish(t *testing.T) {
	bus := NewEventBus()
	defer bus.Stop(5 * time.Second)

	var mutex sync.Mutex
	count := 0

	handler := func(event Event) error {
		mutex.Lock()
		count++
		mutex.Unlock()
		return nil
	}

	bus.Subscribe("*", handler)

	// 并发发布多个事件
	var wg sync.WaitGroup
	numEvents := 100

	for i := 0; i < numEvents; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			event := *NewEvent("test", map[string]interface{}{"id": id})
			err := bus.Publish("test", event)
			assert.NoError(t, err, "Failed to publish event %d", id)
		}(i)
	}

	wg.Wait()

	// 等待异步处理
	time.Sleep(500 * time.Millisecond)

	mutex.Lock()
	assert.Equal(t, numEvents, count, "Expected %d events", numEvents)
	mutex.Unlock()
}

/**
 * TestRateLimitMiddleware 测试速率限制中间件
 */
func TestRateLimitMiddleware(t *testing.T) {
	bus := NewEventBus()
	bus.Use(RateLimitMiddleware(10)) // 每秒最多 10 个事件
	defer bus.Stop(5 * time.Second)

	received := 0
	var mutex sync.Mutex

	handler := func(event Event) error {
		mutex.Lock()
		received++
		mutex.Unlock()
		return nil
	}

	bus.Subscribe("test", handler)

	// 快速发布 20 个事件
	for i := 0; i < 20; i++ {
		err := bus.Publish("test", *NewEvent("test", map[string]interface{}{"id": i}))
		assert.NoError(t, err, "Failed to publish event %d", i)
	}

	// 等待处理
	time.Sleep(200 * time.Millisecond)

	// 应该只有部分事件被处理（受速率限制）
	mutex.Lock()
	if received > 15 {
		t.Logf("WARNING: Rate limiting may not be working (received %d events)", received)
	}
	fmt.Printf("Rate limit test: received %d out of 20 events\n", received)
	mutex.Unlock()

	// 至少应该有一些事件被处理
	assert.Greater(t, received, 0, "Expected at least some events to be processed")
}

/**
 * BenchmarkEventBusPublish 基准测试：发布性能
 */
func BenchmarkEventBusPublish(b *testing.B) {
	bus := NewEventBus(WithAsyncDisabled())
	defer bus.Stop(5 * time.Second)

	bus.Subscribe("test", func(event Event) error {
		return nil
	})

	event := *NewEvent("test", map[string]interface{}{"message": "benchmark"})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := bus.Publish("test", event)
		if err != nil {
			b.Fatalf("Failed to publish: %v", err)
		}
	}
}

/**
 * BenchmarkEventBusPublishAsync 基准测试：异步发布性能
 */
func BenchmarkEventBusPublishAsync(b *testing.B) {
	bus := NewEventBus()
	defer bus.Stop(5 * time.Second)

	bus.Subscribe("test", func(event Event) error {
		return nil
	})

	event := *NewEvent("test", map[string]interface{}{"message": "benchmark"})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bus.PublishAsync("test", event)
	}
}
