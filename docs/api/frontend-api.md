# 前端 API 参考

FlowMind 前端（Wails）API 完整参考。

---

## Wails 绑定

### 导入

```typescript
import {
  GetRecentEvents,
  GetPatterns,
  CreateAutomation,
  // ... 其他函数
} from '../../wailsjs/go/main/App'

import {
  EventsOn,
  EventsOff,
  EventsEmit,
} from '../../wailsjs/runtime'
```

### 函数调用

所有后端函数都返回 Promise。

```typescript
// 获取最近事件
const events = await GetRecentEvents(100)

// 创建自动化
const automation = await CreateAutomation(
  "每天下午 5 点，总结代码并发送 Slack"
)
```

---

## 事件订阅

### 订阅事件

```typescript
import { EventsOn } from '../../wailsjs/runtime'

// 订阅新事件
EventsOn('event:new', (event: Event) => {
  console.log('New event:', event)
})

// 订阅模式发现
EventsOn('pattern:discovered', (pattern: Pattern) => {
  showNotification('New pattern discovered!')
})
```

### 取消订阅

```typescript
import { EventsOff } from '../../wailsjs/runtime'

EventsOff('event:new')
```

---

## TypeScript 类型

### Event

```typescript
interface Event {
  id: string
  type: string
  timestamp: string
  data: Record<string, any>
  context?: EventContext
}

interface EventContext {
  application: string
  bundle_id: string
  window_title: string
  file_path?: string
  selection?: string
}
```

### Pattern

```typescript
interface Pattern {
  id: string
  sequence: EventStep[]
  support_count: number
  confidence: number
  is_automated: boolean
}

interface EventStep {
  type: string
  application?: string
  data?: Record<string, any>
}
```

### AutomationScript

```typescript
interface AutomationScript {
  id: string
  name: string
  description: string
  steps: Step[]
  trigger: TriggerConfig
  enabled: boolean
}

interface Step {
  action: string
  params: Record<string, any>
}

interface TriggerConfig {
  type: 'cron' | 'event' | 'manual'
  config: Record<string, any>
}
```

---

## 使用示例

### 获取并显示事件

```typescript
import { GetRecentEvents } from '../../wailsjs/go/main/App'
import { useEffect, useState } from 'react'

function EventList() {
  const [events, setEvents] = useState<Event[]>([])

  useEffect(() => {
    GetRecentEvents(100).then(setEvents)
  }, [])

  return (
    <ul>
      {events.map(event => (
        <li key={event.id}>
          {event.type} - {event.timestamp}
        </li>
      ))}
    </ul>
  )
}
```

### 订阅实时事件

```typescript
import { EventsOn } from '../../wailsjs/runtime'

useEffect(() => {
  const handler = (event: Event) => {
    setEvents(prev => [event, ...prev])
  }

  EventsOn('event:new', handler)

  return () => {
    EventsOff('event:new')
  }
}, [])
```

---

**相关文档**：
- [Go API](./go-api.md)
- [事件 API](./event-api.md)
- [API 设计](../design/02-api-design.md)
