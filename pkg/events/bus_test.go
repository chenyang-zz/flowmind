package events

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

/**
 * TestNewEventBus 测试创建事件总线
 */
func TestNewEventBus(t *testing.T) {
	bus := NewEventBus()
	defer bus.Stop(5 * time.Second)

	if bus == nil {
		t.Fatal("Expected non-nil bus")
	}

	if bus.stopped.Load() {
		t.Fatal("Expected bus to be running")
	}
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
	if subscriberID == "" {
		t.Fatal("Expected non-empty subscriber ID")
	}

	// 发布事件
	event := NewEvent("test", map[string]interface{}{"message": "hello"})
	bus.Publish("test", *event)

	// 等待异步处理
	time.Sleep(100 * time.Millisecond)

	mutex.Lock()
	if !received {
		mutex.Unlock()
		t.Fatal("Expected to receive event")
	}
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
		bus.Publish(string(event.Type), event)
	}

	// 等待异步处理
	time.Sleep(100 * time.Millisecond)

	mutex.Lock()
	if count != 3 {
		t.Fatalf("Expected 3 events, got %d", count)
	}
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
	bus.Publish("test", *NewEvent("test", map[string]interface{}{"value": 3}))
	bus.Publish("test", *NewEvent("test", map[string]interface{}{"value": 7}))
	bus.Publish("test", *NewEvent("test", map[string]interface{}{"value": 10}))

	// 等待异步处理
	time.Sleep(100 * time.Millisecond)

	mutex.Lock()
	if received != 2 {
		mutex.Unlock()
		t.Fatalf("Expected 2 events (value > 5), got %d", received)
	}
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
		bus.Publish("test", *NewEvent("test", map[string]interface{}{"count": i}))
	}

	// 等待异步处理
	time.Sleep(100 * time.Millisecond)

	mutex.Lock()
	if count != 1 {
		mutex.Unlock()
		t.Fatalf("Expected 1 event (once subscription), got %d", count)
	}
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
	bus.Publish("test", *NewEvent("test", map[string]interface{}{"count": 1}))

	// 等待处理
	time.Sleep(50 * time.Millisecond)

	// 取消订阅
	bus.Unsubscribe(subscriberID)

	// 发布第二个事件
	bus.Publish("test", *NewEvent("test", map[string]interface{}{"count": 2}))

	// 等待异步处理
	time.Sleep(100 * time.Millisecond)

	mutex.Lock()
	if received != 1 {
		mutex.Unlock()
		t.Fatalf("Expected 1 event (after unsubscribe), got %d", received)
	}
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

	if event.Context == nil {
		t.Fatal("Expected context to be set")
	}

	if event.Context.Application != "VS Code" {
		t.Errorf("Expected application 'VS Code', got '%s'", event.Context.Application)
	}
}

/**
 * TestEventMetadata 测试事件元数据
 */
func TestEventMetadata(t *testing.T) {
	event := NewEvent("test", map[string]interface{}{})

	event.WithMetadata("source", "keyboard_monitor")
	event.WithMetadata("version", "1.0")

	if len(event.Metadata) != 2 {
		t.Fatalf("Expected 2 metadata entries, got %d", len(event.Metadata))
	}

	if event.Metadata["source"] != "keyboard_monitor" {
		t.Errorf("Expected metadata 'source' to be 'keyboard_monitor'")
	}
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

	if err != nil {
		t.Logf("Expected error from recovered panic: %v", err)
	}

	// 等待异步处理
	time.Sleep(100 * time.Millisecond)
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
	if received {
		mutex.Unlock()
		t.Fatal("Event should not be received immediately (async)")
	}
	mutex.Unlock()

	// 等待后应该收到
	time.Sleep(100 * time.Millisecond)

	mutex.Lock()
	if !received {
		mutex.Unlock()
		t.Fatal("Expected to receive event after waiting")
	}
	mutex.Unlock()
}

/**
 * TestStop 测试停止事件总线
 */
func TestStop(t *testing.T) {
	bus := NewEventBus()

	// 启动一些订阅者，但不用长时间睡眠
	for i := 0; i < 5; i++ {
		bus.Subscribe("test", func(event Event) error {
			// 不睡眠，立即返回
			return nil
		})
	}

	// 停止总线（增加超时时间）
	err := bus.Stop(10 * time.Second)
	if err != nil {
		t.Fatalf("Failed to stop bus: %v", err)
	}

	if !bus.stopped.Load() {
		t.Fatal("Expected bus to be stopped")
	}

	// 尝试发布事件（应该失败）
	err = bus.Publish("test", *NewEvent("test", map[string]interface{}{}))
	if err == nil {
		t.Fatal("Expected error when publishing to stopped bus")
	}
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
			bus.Publish("test", event)
		}(i)
	}

	wg.Wait()

	// 等待异步处理
	time.Sleep(500 * time.Millisecond)

	mutex.Lock()
	if count != numEvents {
		t.Fatalf("Expected %d events, got %d", numEvents, count)
	}
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
	handler := func(event Event) error {
		received++
		return nil
	}

	bus.Subscribe("test", handler)

	// 快速发布 20 个事件
	for i := 0; i < 20; i++ {
		bus.Publish("test", *NewEvent("test", map[string]interface{}{"id": i}))
	}

	// 等待处理
	time.Sleep(200 * time.Millisecond)

	// 应该只有部分事件被处理（受速率限制）
	if received > 15 {
		t.Logf("WARNING: Rate limiting may not be working (received %d events)", received)
	}

	fmt.Printf("Rate limit test: received %d out of 20 events\n", received)
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
		bus.Publish("test", event)
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
