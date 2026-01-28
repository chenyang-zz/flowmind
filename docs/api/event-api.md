# 事件 API 参考

FlowMind 事件系统 API 完整参考。

---

## 事件类型

```go
const (
    // 系统事件
    EventTypeSystem      = "system"
    EventTypeError       = "error"

    // 监控事件
    EventTypeKeyboard     = "keyboard"
    EventTypeClipboard    = "clipboard"
    EventTypeAppSwitch    = "app_switch"
    EventTypeFileSystem   = "file_system"

    // 业务事件
    EventTypePatternDiscovered   = "pattern.discovered"
    EventTypeAutomationCreated   = "automation.created"
    EventTypeAutomationCompleted = "automation.completed"

    // 知识库事件
    EventTypeKnowledgeAdded    = "knowledge.added"
    EventTypeKnowledgeAccessed  = "knowledge.accessed"
)
```

---

## 事件订阅

### 订阅特定事件

```go
bus.Subscribe("keyboard", func(event Event) error {
    log.Info("Keyboard event:", event.ID)
    return nil
})
```

### 订阅所有事件

```go
bus.SubscribeAll(func(event Event) error {
    log.Info("Event:", event.Type, event.ID)
    return nil
})
```

### 带过滤的订阅

```go
bus.SubscribeWithFilter(
    "app_switch",
    handler,
    func(event Event) bool {
        // 只处理 VS Code 的切换
        return event.Context.Application == "VS Code"
    },
)
```

---

## 事件发布

### 同步发布

```go
err := bus.Publish("keyboard", Event{
    ID:        generateID(),
    Type:      "keyboard",
    Timestamp: time.Now(),
    Data:      map[string]interface{}{"keycode": 46},
})
```

### 异步发布

```go
bus.PublishAsync("pattern.discovered", Event{
    ID:   pattern.ID,
    Type: "pattern.discovered",
    Data: map[string]interface{}{"pattern": pattern},
})
```

---

## 前端事件订阅

```typescript
// 订阅 Go 后端事件
EventsOn('event:keyboard', (event) => {
  console.log('Keyboard event:', event)
})

// 取消订阅
EventsOff('event:keyboard')
```

---

**相关文档**：
- [Go API](./go-api.md)
- [前端 API](./frontend-api.md)
- [事件系统设计](../design/03-event-system.md)
