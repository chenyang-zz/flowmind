# Go API 参考

FlowMind 后端 Go API 完整参考。

---

## 核心 API

### 事件 API

#### GetRecentEvents

```go
func (a *App) GetRecentEvents(limit int) ([]Event, error)
```

获取最近的事件。

**参数**:
- `limit`: 返回数量限制（默认 100）

**返回**:
- `[]Event`: 事件列表
- `error`: 错误信息

**示例**:
```go
events, err := app.GetRecentEvents(100)
```

#### GetEventsByTimeRange

```go
func (a *App) GetEventsByTimeRange(start, end string) ([]Event, error)
```

获取时间范围内的事件。

**参数**:
- `start`: 开始时间（RFC3339 格式）
- `end`: 结束时间（RFC3339 格式）

**返回**:
- `[]Event`: 事件列表

**示例**:
```go
events, err := app.GetEventsByTimeRange(
    "2026-01-01T00:00:00Z",
    "2026-01-31T23:59:59Z",
)
```

---

### 模式 API

#### GetPatterns

```go
func (a *App) GetPatterns() ([]Pattern, error)
```

获取所有发现的模式。

#### ApprovePattern

```go
func (a *App) ApprovePattern(patternID string) error
```

批准模式并创建自动化。

---

### 自动化 API

#### CreateAutomation

```go
func (a *App) CreateAutomation(description string) (*AutomationScript, error)
```

从自然语言生成自动化脚本。

#### ExecuteAutomation

```go
func (a *App) ExecuteAutomation(scriptID string) (*ExecutionResult, error)
```

手动执行自动化。

---

### 知识库 API

#### AddKnowledgeItem

```go
func (a *App) AddKnowledgeItem(title, content, sourceURL string) (*KnowledgeItem, error)
```

添加知识项。

#### SearchKnowledge

```go
func (a *App) SearchKnowledge(query string, limit int) ([]KnowledgeItem, error)
```

搜索知识库（语义搜索）。

---

### AI 助手 API

#### Chat

```go
func (a *App) Chat(message string, ctx *EventContext) (string, error)
```

与 AI 对话。

#### GenerateCode

```go
func (a *App) GenerateCode(request string, ctx *EventContext) (string, error)
```

生成代码。

---

## 数据结构

### Event

```go
type Event struct {
    ID        string                 `json:"id"`
    Type      string                 `json:"type"`
    Timestamp time.Time              `json:"timestamp"`
    Data      map[string]interface{} `json:"data"`
    Context   *EventContext          `json:"context"`
}

type EventContext struct {
    Application string `json:"application"`
    BundleID    string `json:"bundle_id"`
    WindowTitle string `json:"window_title"`
    FilePath    string `json:"file_path"`
    Selection   string `json:"selection"`
}
```

### Pattern

```go
type Pattern struct {
    ID           string        `json:"id"`
    Sequence     []EventStep   `json:"sequence"`
    SupportCount int           `json:"support_count"`
    Confidence   float64       `json:"confidence"`
    IsAutomated  bool          `json:"is_automated"`
}

type EventStep struct {
    Type        string                 `json:"type"`
    Application string                 `json:"application"`
    Data        map[string]interface{} `json:"data"`
}
```

### AutomationScript

```go
type AutomationScript struct {
    ID          string         `json:"id"`
    Name        string         `json:"name"`
    Description string         `json:"description"`
    Steps       []Step         `json:"steps"`
    Trigger     TriggerConfig  `json:"trigger"`
    Enabled     bool           `json:"enabled"`
}

type Step struct {
    Action string                 `json:"action"`
    Params map[string]interface{} `json:"params"`
}

type TriggerConfig struct {
    Type   string                 `json:"type"`
    Config map[string]interface{} `json:"config"`
}
```

---

## 错误处理

所有 API 方法在失败时返回非 nil 的 error。

错误类型：
- `ErrNotFound`: 资源未找到
- `ErrInvalidInput`: 输入无效
- `ErrUnauthorized`: 未授权
- `ErrInternal`: 内部错误

---

**相关文档**：
- [前端 API](./frontend-api.md)
- [事件 API](./event-api.md)
- [API 设计](../design/02-api-design.md)
