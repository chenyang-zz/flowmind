# API 设计

FlowMind 提供三层 API：Go 后端 API、Wails 绑定 API 和事件 API。

---

## API 架构

```
┌─────────────────────────────────────────┐
│           前端 (React/Vue)               │
│  ┌───────────────────────────────────┐  │
│  │  Wails Binding API                │  │
│  │  - 调用 Go 函数                    │  │
│  │  - 接收返回值                      │  │
│  └───────────────────────────────────┘  │
│  ┌───────────────────────────────────┐  │
│  │  WebSocket Events                 │  │
│  │  - 实时事件推送                    │  │
│  │  - 订阅/发布                       │  │
│  └───────────────────────────────────┘  │
└─────────────────────────────────────────┘
                    ↕
┌─────────────────────────────────────────┐
│         Go 后端 (Wails)                  │
│  ┌───────────────────────────────────┐  │
│  │  Exported Functions               │  │
│  │  - 业务逻辑                        │  │
│  │  - 数据访问                        │  │
│  └───────────────────────────────────┘  │
│  ┌───────────────────────────────────┐  │
│  │  Event Bus                        │  │
│  │  - 事件路由                        │  │
│  │  - WebSocket 推送                  │  │
│  └───────────────────────────────────┘  │
└─────────────────────────────────────────┘
```

---

## Wails 绑定 API

### 导出函数规范

```go
// App 结构
type App struct {
    ctx    context.Context
    config *Config
    // 依赖注入
    monitor      *monitor.Engine
    analyzer     *analyzer.Engine
    aiService    *ai.AIService
    automation   *automation.Engine
    storage      *storage.SQLiteDB
}

// 导出方法必须：
// 1. 是 App 的方法
// 2. 首字母大写（公开）
// 3. 返回值可序列化（JSON）
```

### 核心 API

#### 1. 事件 API

```go
// GetRecentEvents 获取最近的事件
func (a *App) GetRecentEvents(limit int) ([]Event, error) {
    repo := storage.NewSQLiteEventRepository(a.storage)
    return repo.FindRecent(limit)
}

// GetEventsByTimeRange 获取时间范围内的事件
func (a *App) GetEventsByTimeRange(start, end string) ([]Event, error) {
    startTime, _ := time.Parse(time.RFC3339, start)
    endTime, _ := time.Parse(time.RFC3339, end)

    repo := storage.NewSQLiteEventRepository(a.storage)
    return repo.FindByTimeRange(startTime, endTime)
}

// GetEventStats 获取事件统计
func (a *App) GetEventStats(days int) (*EventStats, error) {
    // 实现统计逻辑
    return stats, nil
}
```

#### 2. 模式 API

```go
// GetPatterns 获取所有发现的模式
func (a *App) GetPatterns() ([]Pattern, error) {
    repo := storage.NewSQLitePatternRepository(a.storage)
    return repo.FindAll()
}

// GetAutomatedPatterns 获取已自动化的模式
func (a *App) GetAutomatedPatterns() ([]Pattern, error) {
    repo := storage.NewSQLitePatternRepository(a.storage)
    return repo.FindByAutomated(true)
}

// ApprovePattern 批准模式自动化
func (a *App) ApprovePattern(patternID string) error {
    // 生成自动化脚本
    // 保存并启用
    return nil
}
```

#### 3. 自动化 API

```go
// CreateAutomation 创建自动化
func (a *App) CreateAutomation(description string) (*AutomationScript, error) {
    return a.automation.GenerateScript(description)
}

// SaveAutomation 保存自动化
func (a *App) SaveAutomation(script *AutomationScript) error {
    repo := storage.NewSQLiteAutomationRepository(a.storage)
    return repo.Save(script)
}

// GetAutomations 获取所有自动化
func (a *App) GetAutomations() ([]AutomationScript, error) {
    repo := storage.NewSQLiteAutomationRepository(a.storage)
    return repo.FindAll()
}

// ExecuteAutomation 手动执行自动化
func (a *App) ExecuteAutomation(scriptID string) (*ExecutionResult, error) {
    repo := storage.NewSQLiteAutomationRepository(a.storage)
    script, err := repo.FindByUUID(scriptID)
    if err != nil {
        return nil, err
    }

    result := a.automation.Execute(script)
    a.automation.SaveResult(result)
    return result, nil
}

// ToggleAutomation 启用/禁用自动化
func (a *App) ToggleAutomation(scriptID string, enabled bool) error {
    repo := storage.NewSQLiteAutomationRepository(a.storage)
    script, err := repo.FindByUUID(scriptID)
    if err != nil {
        return err
    }

    script.Enabled = enabled
    return repo.Update(script)
}
```

#### 4. 知识库 API

```go
// AddKnowledgeItem 添加知识项
func (a *App) AddKnowledgeItem(title, content, sourceURL string) (*KnowledgeItem, error) {
    item := &KnowledgeItem{
        UUID:   generateUUID(),
        Title:  title,
        Content: content,
        SourceURL: sourceURL,
    }

    // AI 生成标签和摘要
    tags, summary := a.aiService.AnalyzeContent(content)
    item.Tags = tags
    item.Summary = summary

    // 保存
    repo := storage.NewSQLiteKnowledgeRepository(a.storage, a.vectorStore)
    return item, repo.Save(item)
}

// SearchKnowledge 搜索知识
func (a *App) SearchKnowledge(query string, limit int) ([]KnowledgeItem, error) {
    repo := storage.NewSQLiteKnowledgeRepository(a.storage, a.vectorStore)
    return repo.Search(query, limit)
}

// GetKnowledgeItem 获取知识项详情
func (a *App) GetKnowledgeItem(id int) (*KnowledgeItem, error) {
    repo := storage.NewSQLiteKnowledgeRepository(a.storage, a.vectorStore)
    return repo.FindByID(id)
}

// GetRelatedItems 获取相关知识项
func (a *App) GetRelatedItems(id int, limit int) ([]KnowledgeItem, error) {
    repo := storage.NewSQLiteKnowledgeRepository(a.storage, a.vectorStore)
    return repo.FindRelated(id, limit)
}
```

#### 5. AI 助手 API

```go
// Chat 流式对话
func (a *App) Chat(message string, ctx *EventContext) (string, error) {
    // 构建上下文
    prompt := a.buildChatPrompt(message, ctx)

    // 调用 AI
    response, err := a.aiService.Complete(prompt)
    if err != nil {
        return "", err
    }

    return response, nil
}

// GenerateCode 生成代码
func (a *App) GenerateCode(request string, ctx *EventContext) (string, error) {
    return a.aiService.GenerateCode(ctx, request)
}

// CompleteWithContext 上下文感知补全
func (a *App) CompleteWithContext(partial string, ctx *EventContext) (string, error) {
    prompt := fmt.Sprintf(`Context:
- Application: %s
- File: %s
- Selection: %s

Complete: %s`, ctx.Application, ctx.FilePath, ctx.Selection, partial)

    return a.aiService.Complete(prompt)
}
```

---

## WebSocket 事件 API

### 事件订阅

```typescript
// 前端订阅事件
import { EventsOn, EventsOff } from '../../wailsjs/runtime'

// 订阅新事件
EventsOn('event:new', (event: Event) => {
    console.log('New event:', event)
    updateDashboard(event)
})

// 订阅模式发现
EventsOn('pattern:discovered', (pattern: Pattern) => {
    showNotification('New pattern discovered!')
    addPatternToList(pattern)
})

// 订阅自动化执行
EventsOn('automation:started', (data: {scriptID: string}) => {
    console.log('Automation started:', data.scriptID)
})

EventsOn('automation:completed', (result: ExecutionResult) => {
    console.log('Automation completed:', result.status)
    showResult(result)
})

// 取消订阅
EventsOff('event:new')
```

### 事件发布（Go 后端）

```go
// 发布事件到前端
func (a *App) PublishEvent(eventType string, data interface{}) {
    runtime.EventsEmit(a.ctx, eventType, data)
}

// 示例：发布新模式
func (a *App) onPatternDiscovered(pattern Pattern) {
    a.PublishEvent("pattern:discovered", pattern)
}

// 示例：发布自动化结果
func (a *App) onAutomationCompleted(result ExecutionResult) {
    a.PublishEvent("automation:completed", result)
}
```

### 标准事件类型

```go
const (
    EventNew             = "event:new"              // 新事件
    EventPatternDiscovered = "pattern:discovered"   // 发现模式
    EventAutomationStarted  = "automation:started"  // 自动化开始
    EventAutomationProgress = "automation:progress" // 自动化进度
    EventAutomationCompleted = "automation:completed" // 自动化完成
    EventKnowledgeAdded = "knowledge:added"        // 知识添加
    EventNotification   = "notification"           // 通知
)
```

---

## RESTful API（可选扩展）

### HTTP 服务器

```go
// internal/api/server.go
type APIServer struct {
    port    int
    app     *App
    router  *mux.Router
}

func NewAPIServer(app *App, port int) *APIServer {
    server := &APIServer{
        app:    app,
        port:   port,
        router: mux.NewRouter(),
    }

    server.setupRoutes()
    return server
}

func (s *APIServer) setupRoutes() {
    api := s.router.PathPrefix("/api/v1").Subrouter()

    // 事件
    api.HandleFunc("/events", s.getEvents).Methods("GET")
    api.HandleFunc("/events/stats", s.getEventStats).Methods("GET")

    // 模式
    api.HandleFunc("/patterns", s.getPatterns).Methods("GET")
    api.HandleFunc("/patterns/{id}", s.approvePattern).Methods("POST")

    // 自动化
    api.HandleFunc("/automations", s.createAutomation).Methods("POST")
    api.HandleFunc("/automations", s.getAutomations).Methods("GET")
    api.HandleFunc("/automations/{id}/execute", s.executeAutomation).Methods("POST")

    // 知识库
    api.HandleFunc("/knowledge", s.addKnowledge).Methods("POST")
    api.HandleFunc("/knowledge/search", s.searchKnowledge).Methods("GET")
}

func (s *APIServer) Start() error {
    return http.ListenAndServe(fmt.Sprintf(":%d", s.port), s.router)
}
```

### API 端点示例

```
GET  /api/v1/events?limit=100&start=2026-01-01&end=2026-01-28
GET  /api/v1/events/stats?days=7
GET  /api/v1/patterns?automated=true
POST /api/v1/patterns/{id}/approve
GET  /api/v1/automations
POST /api/v1/automations
POST /api/v1/automations/{id}/execute
GET  /api/v1/knowledge/search?q=rust&limit=10
POST /api/v1/knowledge
```

---

## 错误处理

### 错误码

```go
type APIError struct {
    Code    string `json:"code"`
    Message string `json:"message"`
    Details string `json:"details,omitempty"`
}

const (
    ErrCodeNotFound      = "NOT_FOUND"
    ErrCodeInvalidInput  = "INVALID_INPUT"
    ErrCodeInternalError = "INTERNAL_ERROR"
    ErrCodeUnauthorized  = "UNAUTHORIZED"
)

func NewAPIError(code, message, details string) *APIError {
    return &APIError{
        Code:    code,
        Message: message,
        Details: details,
    }
}
```

### 错误响应格式

```json
{
  "error": {
    "code": "INVALID_INPUT",
    "message": "Invalid automation script",
    "details": "Step 2: Missing required parameter 'path'"
  }
}
```

---

## 性能优化

### 响应缓存

```go
// 使用 BBolt 缓存 API 响应
func (a *App) GetEventStats(days int) (*EventStats, error) {
    cacheKey := fmt.Sprintf("stats:%d", days)

    // 尝试从缓存获取
    if cached, found := a.cache.Get(cacheKey); found {
        return cached.(*EventStats), nil
    }

    // 计算统计
    stats, err := a.calculateStats(days)
    if err != nil {
        return nil, err
    }

    // 缓存结果（5 分钟）
    a.cache.Set(cacheKey, stats, 5*time.Minute)

    return stats, nil
}
```

### 分页

```go
type Pagination struct {
    Page     int `json:"page"`
    PageSize int `json:"page_size"`
    Total    int `json:"total"`
}

type PaginatedResult struct {
    Data       interface{} `json:"data"`
    Pagination Pagination  `json:"pagination"`
}

func (a *App) GetEvents(page, pageSize int) (*PaginatedResult, error) {
    offset := (page - 1) * pageSize

    // 查询数据
    events, err := repo.FindPaginated(offset, pageSize)
    if err != nil {
        return nil, err
    }

    // 查询总数
    total, _ := repo.Count()

    return &PaginatedResult{
        Data: events,
        Pagination: Pagination{
            Page:     page,
            PageSize: pageSize,
            Total:    total,
        },
    }, nil
}
```

---

## 使用示例

### 前端调用

```typescript
import { GetRecentEvents, CreateAutomation } from '../../wailsjs/go/main/App'

// 获取最近事件
const events = await GetRecentEvents(100)

// 创建自动化
const automation = await CreateAutomation(
  "每天下午 5 点，总结今天的 Git 提交并发到 Slack"
)

// 订阅事件
EventsOn('event:new', (event) => {
  console.log('New event:', event)
})
```

---

**相关文档**：
- [事件系统](./03-event-system.md)
- [Go API 参考](../api/go-api.md)
- [前端 API 参考](../api/frontend-api.md)
