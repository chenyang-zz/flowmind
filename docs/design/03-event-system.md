# 事件系统设计

FlowMind 采用事件驱动架构（EDA），实现组件间的松耦合和异步通信。

---

## 事件系统概述

### 核心概念

```
┌─────────────┐
│ Event Source│ (监控引擎、用户操作等)
└─────────────┘
      │
      │ publish
      ↓
┌─────────────┐
│  Event Bus  │ (事件总线)
└─────────────┘
      │
      │ subscribe
      ↓
┌─────────────┐
│Event Handler│ (分析器、存储、UI等)
└─────────────┘
```

### 设计目标

1. **解耦**：组件间通过事件通信，不直接依赖
2. **异步**：非阻塞式事件处理
3. **可扩展**：易于添加新的事件类型和处理者
4. **高性能**：支持高频率事件流

---

## 事件定义

### 核心事件结构

```go
// pkg/events/event.go
package events

import "time"

type Event struct {
    ID        string                 `json:"id"`
    Type      string                 `json:"type"`
    Timestamp time.Time              `json:"timestamp"`
    Data      map[string]interface{} `json:"data"`
    Metadata  map[string]string      `json:"metadata,omitempty"`
    Context   *EventContext          `json:"context,omitempty"`
}

type EventContext struct {
    Source    string `json:"source"`    // 事件源
    SessionID string `json:"session_id"` // 会话 ID
    UserID    string `json:"user_id"`   // 用户 ID
    TraceID   string `json:"trace_id"`  // 追踪 ID
}
```

### 事件类型分类

```go
const (
    // 系统事件
    EventTypeSystem      = "system"
    EventTypeError       = "error"
    EventTypeWarning     = "warning"

    // 监控事件
    EventTypeKeyboard     = "keyboard"
    EventTypeClipboard    = "clipboard"
    EventTypeAppSwitch    = "app_switch"
    EventTypeFileSystem   = "file_system"

    // 业务事件
    EventTypePatternDiscovered   = "pattern.discovered"
    EventTypePatternApproved     = "pattern.approved"
    EventTypeAutomationCreated   = "automation.created"
    EventTypeAutomationStarted   = "automation.started"
    EventTypeAutomationProgress  = "automation.progress"
    EventTypeAutomationCompleted = "automation.completed"
    EventTypeAutomationFailed    = "automation.failed"

    // 知识库事件
    EventTypeKnowledgeAdded    = "knowledge.added"
    EventTypeKnowledgeUpdated   = "knowledge.updated"
    EventTypeKnowledgeDeleted   = "knowledge.deleted"
    EventTypeKnowledgeAccessed  = "knowledge.accessed"

    // 通知事件
    EventTypeNotification = "notification"
)
```

---

## 事件总线

### 实现

```go
// pkg/events/bus.go
type EventBus struct {
    subscribers map[string][]*Subscriber
    mutex       sync.RWMutex
    wg          sync.WaitGroup
    stopChan    chan struct{}
}

type Subscriber struct {
    ID       string
    Handler  EventHandler
    Filter   EventFilter
    Once     bool
}

type EventHandler func(event Event) error

type EventFilter func(event Event) bool

func NewEventBus() *EventBus {
    return &EventBus{
        subscribers: make(map[string][]*Subscriber),
        stopChan:    make(chan struct{}),
    }
}

// Subscribe 订阅事件
func (bus *EventBus) Subscribe(eventType string, handler EventHandler) string {
    return bus.SubscribeWithFilter(eventType, handler, nil)
}

// SubscribeWithFilter 带过滤的订阅
func (bus *EventBus) SubscribeWithFilter(
    eventType string,
    handler EventHandler,
    filter EventFilter,
) string {
    bus.mutex.Lock()
    defer bus.mutex.Unlock()

    sub := &Subscriber{
        ID:      generateID(),
        Handler: handler,
        Filter:  filter,
    }

    bus.subscribers[eventType] = append(bus.subscribers[eventType], sub)

    return sub.ID
}

// Unsubscribe 取消订阅
func (bus *EventBus) Unsubscribe(eventType, subscriberID string) {
    bus.mutex.Lock()
    defer bus.mutex.Unlock()

    subs := bus.subscribers[eventType]
    for i, sub := range subs {
        if sub.ID == subscriberID {
            bus.subscribers[eventType] = append(subs[:i], subs[i+1:]...)
            break
        }
    }
}

// Publish 发布事件（同步）
func (bus *EventBus) Publish(eventType string, event Event) error {
    bus.mutex.RLock()
    defer bus.mutex.RUnlock()

    subs := bus.subscribers[eventType]

    for _, sub := range subs {
        // 过滤检查
        if sub.Filter != nil && !sub.Filter(event) {
            continue
        }

        // 执行处理
        if err := sub.Handler(event); err != nil {
            log.Error("Event handler error:", err)
        }
    }

    return nil
}

// PublishAsync 发布事件（异步）
func (bus *EventBus) PublishAsync(eventType string, event Event) {
    bus.wg.Add(1)

    go func() {
        defer bus.wg.Done()

        if err := bus.Publish(eventType, event); err != nil {
            log.Error("Async publish error:", err)
        }
    }()
}

// Stop 停止事件总线
func (bus *EventBus) Stop() {
    close(bus.stopChan)
    bus.wg.Wait()
}
```

### 通配符订阅

```go
// 订阅所有事件
func (bus *EventBus) SubscribeAll(handler EventHandler) string {
    return bus.SubscribeWithFilter("*", handler, nil)
}

// 发布到所有订阅者
func (bus *EventBus) Publish(eventType string, event Event) error {
    // 发布到具体类型
    bus.publishToType(eventType, event)

    // 发布到通配符订阅者
    bus.publishToType("*", event)

    return nil
}
```

---

## 事件中间件

### 中间件链

```go
// pkg/events/middleware.go
type Middleware func(EventHandler) EventHandler

func (bus *EventBus) Use(middleware Middleware) {
    // 应用到所有订阅者
    for eventType, subs := range bus.subscribers {
        for _, sub := range subs {
            sub.Handler = middleware(sub.Handler)
        }
    }
}

// 日志中间件
func LoggingMiddleware(logger *Logger) Middleware {
    return func(next EventHandler) EventHandler {
        return func(event Event) error {
            logger.Info("Processing event",
                "type", event.Type,
                "id", event.ID,
            )

            err := next(event)

            if err != nil {
                logger.Error("Event processing failed",
                    "type", event.Type,
                    "error", err,
                )
            }

            return err
        }
    }
}

// 恢复中间件
func RecoveryMiddleware() Middleware {
    return func(next EventHandler) EventHandler {
        return func(event Event) (err error) {
            defer func() {
                if r := recover(); r != nil {
                    log.Error("Event panic recovered", "event", event.ID, "panic", r)
                    err = fmt.Errorf("panic: %v", r)
                }
            }()

            return next(event)
        }
    }
}

// 限流中间件
func RateLimitMiddleware(maxPerSec int) Middleware {
    limiter := NewRateLimiter(maxPerSec)

    return func(next EventHandler) EventHandler {
        return func(event Event) error {
            if !limiter.Allow() {
                return fmt.Errorf("rate limit exceeded")
            }

            return next(event)
        }
    }
}

// 重试中间件
func RetryMiddleware(maxRetries int, delay time.Duration) Middleware {
    return func(next EventHandler) EventHandler {
        return func(event Event) error {
            var lastErr error

            for i := 0; i < maxRetries; i++ {
                if err := next(event); err == nil {
                    return nil
                } else {
                    lastErr = err
                    time.Sleep(delay)
                }
            }

            return lastErr
        }
    }
}
```

### 使用中间件

```go
bus := NewEventBus()

// 全局中间件
bus.Use(RecoveryMiddleware())
bus.Use(LoggingMiddleware(logger))
bus.Use(RateLimitMiddleware(1000))

// 特定事件的中间件
bus.Subscribe("automation.created", 
    RetryMiddleware(3, 1*time.Second)(handler),
)
```

---

## 事件流处理

### 事件流

```go
// pkg/events/stream.go
type EventStream struct {
    bus     *EventBus
    filters []EventFilter
    buffer  int
}

func NewEventStream(bus *EventBus, buffer int) *EventStream {
    return &EventStream{
        bus:    bus,
        buffer: buffer,
    }
}

func (s *EventStream) Filter(filter EventFilter) *EventStream {
    s.filters = append(s.filters, filter)
    return s
}

func (s *EventStream) Into(eventTypes ...string) chan Event {
    out := make(chan Event, s.buffer)

    for _, eventType := range eventTypes {
        s.bus.Subscribe(eventType, func(event Event) error {
            // 应用过滤
            for _, filter := range s.filters {
                if !filter(event) {
                    return nil
                }
            }

            // 发送到流
            select {
            case out <- event:
            default:
                log.Warn("Event stream buffer full")
            }

            return nil
        })
    }

    return out
}
```

### 使用示例

```go
// 创建事件流
stream := NewEventStream(bus, 100)

// 过滤和转换
keyboardStream := stream.
    Filter(func(e Event) bool {
        return e.Type == EventTypeKeyboard
    }).
    Into("keyboard")

// 消费事件
for event := range keyboardStream {
    fmt.Println("Keyboard event:", event)
}
```

---

## 事件持久化

### 事件存储

```go
// pkg/events/store.go
type EventStore interface {
    Save(event Event) error
    Find(query EventQuery) ([]Event, error)
    Delete(id string) error
}

type SQLiteEventStore struct {
    db *sql.DB
}

func (s *SQLiteEventStore) Save(event Event) error {
    query := `
        INSERT INTO events (uuid, type, timestamp, data, metadata)
        VALUES (?, ?, ?, ?, ?)
    `

    data, _ := json.Marshal(event.Data)
    metadata, _ := json.Marshal(event.Metadata)

    _, err := s.db.Exec(query,
        event.ID,
        event.Type,
        event.Timestamp,
        data,
        metadata,
    )

    return err
}

type EventQuery struct {
    Type      string
    StartTime time.Time
    EndTime   time.Time
    Limit     int
}

func (s *SQLiteEventStore) Find(query EventQuery) ([]Event, error) {
    // 实现查询逻辑
    return events, nil
}
```

### 事件重放

```go
// Replayer 重放事件
type Replayer struct {
    store EventStore
    bus   *EventBus
}

func (r *Replayer) Replay(query EventQuery) error {
    events, err := r.store.Find(query)
    if err != nil {
        return err
    }

    for _, event := range events {
        r.bus.Publish(event.Type, event)
    }

    return nil
}
```

---

## 调试与监控

### 事件追踪

```go
// 事件追踪
type EventTracer struct {
    traceID string
    events  []Event
}

func (t *EventTracer) Record(event Event) {
    if event.Context == nil {
        event.Context = &EventContext{}
    }

    event.Context.TraceID = t.traceID
    t.events = append(t.events, event)
}

func (t *EventTracer) GetTrace() []Event {
    return t.events
}
```

### 事件监控

```go
// 事件监控指标
type EventMetrics struct {
    TotalPublished   int64
    TotalProcessed   int64
    FailedCount      int64
    AverageLatency   time.Duration
    TypeCounts       map[string]int64
}

func (m *EventMetrics) Record(event Event, duration time.Duration, err error) {
    atomic.AddInt64(&m.TotalPublished, 1)

    if err == nil {
        atomic.AddInt64(&m.TotalProcessed, 1)
    } else {
        atomic.AddInt64(&m.FailedCount, 1)
    }

    m.TypeCounts[event.Type]++

    // 更新平均延迟
    // ...
}
```

---

## Wails 集成

### 前端事件

```go
// 将后端事件转发到前端
func (a *App) forwardEventsToFrontend() {
    go func() {
        eventChan := a.eventBus.Subscribe("*")

        for event := range eventChan {
            runtime.EventsEmit(a.ctx, "event:"+event.Type, event)
        }
    }()
}
```

### 前端订阅

```typescript
import { EventsOn } from '../../wailsjs/runtime'

EventsOn('event:keyboard', (event) => {
    console.log('Keyboard event:', event)
})
```

---

## 使用示例

### 发布事件

```go
// 发布模式发现事件
event := events.Event{
    ID:        generateID(),
    Type:      events.EventTypePatternDiscovered,
    Timestamp: time.Now(),
    Data: map[string]interface{}{
        "pattern": pattern,
    },
}

bus.Publish(events.EventTypePatternDiscovered, event)
```

### 订阅事件

```go
// 订阅模式发现事件
bus.Subscribe(events.EventTypePatternDiscovered, func(event events.Event) error {
    pattern := event.Data["pattern"].(*Pattern)

    // 处理模式
    log.Info("Pattern discovered:", pattern.ID)

    return nil
})
```

### 过滤订阅

```go
// 只订阅特定应用的事件
bus.SubscribeWithFilter(
    events.EventTypeAppSwitch,
    handler,
    func(event events.Event) bool {
        return event.Context.Application == "VS Code"
    },
)
```

---

## 最佳实践

1. **事件命名**：使用命名空间（如 `pattern.discovered`）
2. **不可变性**：事件创建后不应修改
3. **错误处理**：在事件处理器中捕获 panic
4. **性能**：异步处理耗时操作
5. **追踪**：使用 TraceID 关联相关事件
6. **测试**：模拟事件进行单元测试

---

**相关文档**：
- [API 设计](./02-api-design.md)
- [监控引擎](../architecture/02-monitor-engine.md)
- [事件 API](../api/event-api.md)
