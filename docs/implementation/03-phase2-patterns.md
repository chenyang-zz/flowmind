# Phase 2: 模式识别引擎

**目标**: 实现 AI 驱动的模式识别引擎，自动发现用户的重复操作模式并建议自动化

**预计时间**: 20-25 天

---

## 📋 概述

本阶段是 FlowMind 的**核心创新功能**，将实现：

1. **事件序列存储** - 将监控事件持久化到 SQLite 数据库
2. **会话划分** - 将事件按时间窗口分组，识别工作会话
3. **模式挖掘** - 使用 PrefixSpan 算法识别频繁序列
4. **AI 过滤** - 使用 Claude API 判断模式是否值得自动化
5. **模式建议** - 向用户展示发现的模式并生成自动化建议

### 系统架构

```
Monitor Engine (已实现)
    ↓ 发布事件
Event Bus (pkg/events)
    ↓ 订阅事件
Analyzer Engine (新增)
  ├─ EventRepository    # 事件持久化 (SQLite)
  ├─ SessionDivider     # 会话划分 (时间窗口)
  ├─ PatternMiner       # PrefixSpan 算法
  ├─ AIPatternFilter    # Claude API 集成
  └─ PatternRecommender # 建议生成
    ↓ 输出模式建议
Frontend UI
```

---

## 🚀 实施步骤

### Step 1: 添加依赖 (1 天)

**任务清单**:
- [ ] 更新 `go.mod` 添加依赖:
  ```bash
  go get github.com/mattn/go-sqlite3
  go get go.etcd.io/bbolt
  go get github.com/philippgille/chromem-go
  go mod tidy
  ```
- [ ] 验证依赖安装: `go build ./...`

**验证标准**:
- 所有依赖安装成功
- 项目可以正常编译

---

### Step 2: 实现存储层 (3-4 天)

#### Day 1: SQLite 基础设施

**文件结构**:
```
internal/storage/
├── sqlite.go                  # SQLite 连接管理
├── migrations.go              # 迁移执行器
└── migrations/
    └── 001_init.sql           # events 表
```

**关键代码**:

`sqlite.go`:
```go
// NewSQLiteDB 创建数据库连接
//
// Parameters:
//   - dbPath: 数据库文件路径
//
// Returns: *sql.DB - 数据库连接实例, error - 错误信息
func NewSQLiteDB(dbPath string) (*sql.DB, error) {
    db, err := sql.Open("sqlite3", dbPath)
    if err != nil {
        return nil, fmt.Errorf("打开数据库失败: %w", err)
    }

    // 配置 WAL 模式 (提升并发性能)
    db.Exec("PRAGMA journal_mode=WAL")
    db.Exec("PRAGMA synchronous=NORMAL")
    db.Exec("PRAGMA cache_size=10000")
    db.SetMaxOpenConns(25) // 连接池配置

    return db, nil
}
```

`migrations/001_init.sql`:
```sql
-- 事件表
CREATE TABLE IF NOT EXISTS events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    uuid TEXT UNIQUE NOT NULL,
    type TEXT NOT NULL,
    timestamp DATETIME NOT NULL,
    data JSON,
    application TEXT,
    bundle_id TEXT,
    window_title TEXT,
    file_path TEXT,
    selection TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 索引优化
CREATE INDEX idx_events_timestamp ON events(timestamp);
CREATE INDEX idx_events_type ON events(type);
CREATE INDEX idx_events_application ON events(application);
CREATE INDEX idx_events_uuid ON events(uuid);
```

**验证标准**:
- [ ] 数据库连接成功
- [ ] 迁移执行成功
- [ ] events 表创建成功
- [ ] 单元测试通过

---

#### Day 2: EventRepository

**文件结构**:
```
internal/storage/
└── event_repository.go        # 事件仓储接口
```

**核心接口**:
```go
// EventRepository 事件存储接口
type EventRepository interface {
    // Save 保存单个事件
    Save(event *events.Event) error

    // SaveBatch 批量保存事件（性能优化）
    SaveBatch(events []events.Event) error

    // FindByTimeRange 按时间范围查询
    FindByTimeRange(start, end time.Time) ([]events.Event, error)

    // FindRecent 查询最近的事件
    FindRecent(limit int) ([]events.Event, error)

    // FindByType 按类型查询
    FindByType(eventType events.EventType, limit int) ([]events.Event, error)

    // DeleteOlderThan 删除旧数据
    DeleteOlderThan(cutoff time.Time) (int64, error)

    // GetStats 获取统计信息
    GetStats() (*EventStats, error)
}
```

**批量写入优化**:
```go
// SaveBatch 批量保存事件
//
// 使用事务和预处理语句优化批量写入性能
//
// Parameters:
//   - events: 事件数组
//
// Returns: error - 错误信息
func (r *SQLiteEventRepository) SaveBatch(events []events.Event) error {
    tx, _ := r.db.Begin()
    defer tx.Rollback()

    stmt, _ := tx.Prepare(`
        INSERT INTO events (uuid, type, timestamp, data, application, ...)
        VALUES (?, ?, ?, ?, ?, ...)
    `)
    defer stmt.Close()

    for _, event := range events {
        stmt.Exec(event.ID, event.Type, event.Timestamp, event.Data, ...)
    }

    return tx.Commit()
}
```

**验证标准**:
- [ ] 所有接口实现完成
- [ ] 单元测试覆盖率 ≥ 90%
- [ ] 批量写入性能 > 1000 events/sec

---

#### Day 3: 迁移脚本

**文件结构**:
```
internal/storage/migrations/
├── 002_add_sessions.sql       # sessions 表
├── 003_add_patterns.sql       # patterns 表
├── 004_add_automations.sql    # automations 表
├── 005_add_ai_cache.sql       # AI 缓存表
└── 006_schema_migrations.sql  # 迁移记录表
```

**关键 SQL**:

`002_add_sessions.sql`:
```sql
CREATE TABLE IF NOT EXISTS sessions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    uuid TEXT UNIQUE NOT NULL,
    application TEXT NOT NULL,
    bundle_id TEXT,
    start_time DATETIME NOT NULL,
    end_time DATETIME,
    event_count INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_sessions_time ON sessions(start_time);
CREATE INDEX idx_sessions_application ON sessions(application);
```

`003_add_patterns.sql`:
```sql
CREATE TABLE IF NOT EXISTS patterns (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    uuid TEXT UNIQUE NOT NULL,
    name TEXT,
    sequence_hash TEXT UNIQUE NOT NULL,
    sequence JSON NOT NULL,
    support_count INTEGER DEFAULT 1,
    confidence REAL DEFAULT 0.0,
    first_seen DATETIME NOT NULL,
    last_seen DATETIME NOT NULL,
    is_automated BOOLEAN DEFAULT FALSE,
    automation_id INTEGER,
    ai_analysis TEXT,
    estimated_time_saving INTEGER,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_patterns_automated ON patterns(is_automated);
CREATE INDEX idx_patterns_support ON patterns(support_count);
CREATE INDEX idx_patterns_hash ON patterns(sequence_hash);
```

---

#### Day 4: 性能优化

**文件结构**:
```
internal/storage/
└── batch_writer.go            # 批量写入器
```

**批量写入器**:
```go
// BatchWriter 批量写入器
//
// 优化数据库写入性能，通过批量写入和定时刷新减少数据库压力
type BatchWriter struct {
    repo         EventRepository
    buffer       []events.Event
    bufferSize   int           // 100条一批
    flushTimer   *time.Timer   // 或5秒刷新
    mu           sync.Mutex
}

// Add 添加事件到缓冲区
//
// 当缓冲区达到批量大小时自动刷新
func (bw *BatchWriter) Add(event events.Event) {
    bw.mu.Lock()
    defer bw.mu.Unlock()

    bw.buffer = append(bw.buffer, event)

    // 达到批量大小或定时刷新
    if len(bw.buffer) >= bw.bufferSize {
        bw.flush()
    }
}

// flush 刷新缓冲区到数据库
func (bw *BatchWriter) flush() {
    if len(bw.buffer) == 0 {
        return
    }

    // 异步批量写入
    events := make([]events.Event, len(bw.buffer))
    copy(events, bw.buffer)
    bw.buffer = bw.buffer[:0]

    go func() {
        if err := bw.repo.SaveBatch(events); err != nil {
            logger.Error("批量写入失败", zap.Error(err))
        }
    }()
}
```

**验证标准**:
- [ ] 批量写入性能达标
- [ ] 内存使用合理
- [ ] 压力测试通过 (1000 events/sec)

---

### Step 3: 集成存储层到 Monitor Engine (1 天)

**修改文件**: `internal/monitor/engine.go`

**关键代码**:
```go
// Engine 监控引擎
type Engine struct {
    // 现有字段
    keyboard   Monitor
    clipboard  Monitor
    eventBus   *events.EventBus
    isRunning  bool
    mu         sync.RWMutex

    // 新增字段
    eventRepo   storage.EventRepository
    batchWriter *storage.BatchWriter
}

// NewEngine 创建监控引擎
func NewEngine(eventBus *events.EventBus, eventRepo storage.EventRepository) Monitor {
    return &Engine{
        eventBus:     eventBus,
        eventRepo:    eventRepo,
        batchWriter:  storage.NewBatchWriter(eventRepo, 100, 5*time.Second),
    }
}

// Start 启动监控引擎
func (e *Engine) Start() error {
    e.mu.Lock()
    defer e.mu.Unlock()

    // ... 现有启动逻辑

    // 订阅所有事件并持久化
    e.eventBus.Subscribe("*", func(event events.Event) error {
        e.batchWriter.Add(event)
        return nil
    })

    e.isRunning = true
    logger.Info("监控引擎启动成功", zap.String("component", "engine"))

    return nil
}
```

**验证标准**:
- [ ] 监控事件自动保存到数据库
- [ ] 可以查询保存的事件
- [ ] 性能无明显影响

---

### Step 4: 实现 SessionDivider (2 天)

**文件结构**:
```
internal/analyzer/
├── types.go                   # 共享类型定义
└── session.go                 # 会话划分逻辑
```

**核心类型** (`types.go`):
```go
// Session 会话定义
type Session struct {
    ID          string
    StartTime   time.Time
    EndTime     *time.Time
    Application string
    BundleID    string
    EventCount  int
    Events      []events.Event
}

// SessionDividerConfig 会话划分配置
type SessionDividerConfig struct {
    Timeout         time.Duration // 会话超时 (默认10分钟)
    MinEvents       int           // 最小事件数 (默认5)
    AppSwitchBreaks bool          // 应用切换是否打断会话
}
```

**会话划分器** (`session.go`):
```go
// SessionDivider 会话划分器接口
type SessionDivider interface {
    // Divide 划分事件为会话
    Divide(events []events.Event) ([]*Session, error)

    // GetCurrentSession 获取当前活跃会话
    GetCurrentSession() (*Session, error)

    // EndSession 结束当前会话
    EndSession() error
}

// TimeBasedDivider 基于时间的会话划分
type TimeBasedDivider struct {
    config SessionDividerConfig
}

// Divide 划分事件为会话
//
// 使用超时检测和应用切换检测来划分会话
// 10分钟无操作或应用切换都会结束当前会话
func (td *TimeBasedDivider) Divide(events []events.Event) ([]*Session, error) {
    if len(events) == 0 {
        return nil, nil
    }

    var sessions []*Session
    current := &Session{
        ID:          uuid.New().String(),
        StartTime:   events[0].Timestamp,
        EndTime:     &events[0].Timestamp,
        Application: events[0].Context.Application,
        BundleID:    events[0].Context.BundleID,
    }

    for _, event := range events {
        // 检查超时 (10分钟)
        if event.Timestamp.Sub(*current.EndTime) > td.config.Timeout {
            sessions = append(sessions, current)
            current = td.newSession(event)
            continue
        }

        // 检查应用切换
        if td.config.AppSwitchBreaks &&
           event.Context.Application != current.Application {
            sessions = append(sessions, current)
            current = td.newSession(event)
            continue
        }

        // 继续当前会话
        current.Events = append(current.Events, event)
        current.EventCount++
        current.EndTime = &event.Timestamp
    }

    // 添加最后一个会话
    sessions = append(sessions, current)

    return sessions, nil
}
```

**验证标准**:
- [ ] 正确划分会话
- [ ] 超时检测准确
- [ ] 单元测试通过

---

### Step 5: 实现 PatternMiner (3-4 天)

**文件结构**:
```
internal/analyzer/
├── normalizer.go              # 事件标准化
├── prefixspan.go              # PrefixSpan 算法
└── pattern_miner.go           # 模式挖掘器
```

#### Day 1: 事件标准化

**normalizer.go**:
```go
// EventNormalizer 事件标准化器
//
// 将原始事件转换为抽象的 EventStep，便于模式挖掘
type EventNormalizer struct{}

// EventStep 事件步骤（抽象化）
type EventStep struct {
    Type        string
    Application string
    Data        map[string]interface{}
    Wildcard    bool // 是否为通配符（匹配任意）
}

// Normalize 标准化事件
//
// Parameters:
//   - event: 原始事件
//
// Returns: EventStep - 标准化后的步骤
func (en *EventNormalizer) Normalize(event events.Event) EventStep {
    step := EventStep{
        Type:        string(event.Type),
        Application: event.Context.Application,
    }

    // 根据事件类型提取关键特征
    switch event.Type {
    case events.EventTypeKeyboard:
        // 提取按键类型（字母、数字、符号、功能键）
        if keyCode, ok := event.Data["keycode"].(float64); ok {
            step.Data = map[string]interface{}{
                "key_type": en.classifyKey(int(keyCode)),
            }
        }

    case events.EventTypeClipboard:
        step.Data = map[string]interface{}{
            "has_content": true,
        }

    case events.EventTypeAppSwitch:
        // 应用切换本身就是有意义的事件
    }

    return step
}

// classifyKey 分类按键
//
// Parameters:
//   - keyCode: 按键代码
//
// Returns: string - 按键类型 (letter/number/other)
func (en *EventNormalizer) classifyKey(keyCode int) string {
    switch {
    case keyCode >= 0 && keyCode <= 26:
        return "letter"  // A-Z
    case keyCode >= 30 && keyCode <= 39:
        return "number"  // 0-9
    default:
        return "other"
    }
}
```

---

#### Day 2: PrefixSpan 算法

**prefixspan.go**:
```go
// PrefixSpan PrefixSpan算法实现
type PrefixSpan struct {
    config PatternMinerConfig
}

// Mine 从会话中挖掘模式
//
// Parameters:
//   - sessions: 会话数组
//
// Returns: []*Pattern - 发现的模式数组, error - 错误信息
func (ps *PrefixSpan) Mine(sessions []*Session) ([]*Pattern, error) {
    // 1. 构建序列数据库
    sequences := ps.buildSequences(sessions)

    // 2. 递归挖掘频繁模式
    patterns := make([]*Pattern, 0)
    ps.mineRecursive(sequences, []EventStep{}, &patterns)

    // 3. 计算置信度
    for _, pattern := range patterns {
        pattern.Confidence = ps.calculateConfidence(pattern, sequences)
    }

    return patterns, nil
}

// mineRecursive 递归挖掘频繁模式
//
// Parameters:
//   - sequences: 序列数据库
//   - prefix: 当前前缀
//   - patterns: 模式集合（输出参数）
func (ps *PrefixSpan) mineRecursive(
    sequences []EventSequence,
    prefix []EventStep,
    patterns *[]Pattern,
) {
    // 计算前缀支持度
    support := ps.calculateSupport(sequences, prefix)
    if support < ps.config.MinSupport && len(prefix) > 0 {
        return // 剪枝：支持度不足，提前终止
    }

    // 保存有效模式
    if len(prefix) >= ps.config.MinPatternLen {
        *patterns = append(*patterns, &Pattern{
            ID:           generatePatternID(prefix),
            Sequence:     prefix,
            SupportCount: support,
        })
    }

    // 生成投影数据库
    projectedDB := ps.buildProjectedDB(sequences, prefix)

    // 找到频繁项
    frequentItems := ps.findFrequentItems(projectedDB)

    // 递归挖掘
    for _, item := range frequentItems {
        newPrefix := append([]EventStep{}, prefix...)
        newPrefix = append(newPrefix, item)

        if len(newPrefix) <= ps.config.MaxPatternLen {
            ps.mineRecursive(projectedDB, newPrefix, patterns)
        }
    }
}

// buildProjectedDB 构建投影数据库
//
// Parameters:
//   - sequences: 原始序列数据库
//   - prefix: 当前前缀
//
// Returns: []EventSequence - 投影数据库
func (ps *PrefixSpan) buildProjectedDB(
    sequences []EventSequence,
    prefix []EventStep,
) []EventSequence {
    projected := make([]EventSequence, 0)

    for _, seq := range sequences {
        // 找到前缀匹配位置
        index := ps.findPrefixIndex(seq, prefix)

        if index != -1 && index < len(seq.Events)-1 {
            // 投影：从匹配位置之后的事件
            projected = append(projected, EventSequence{
                Events:    seq.Events[index+1:],
                StartTime: seq.Events[index+1].Timestamp,
                EndTime:   seq.EndTime,
                SessionID: seq.SessionID,
            })
        }
    }

    return projected
}

// calculateSupport 计算支持度
//
// Parameters:
//   - sequences: 序列数据库
//   - prefix: 前缀模式
//
// Returns: int - 支持度（包含该前缀的序列数）
func (ps *PrefixSpan) calculateSupport(
    sequences []EventSequence,
    prefix []EventStep,
) int {
    if len(prefix) == 0 {
        return len(sequences)
    }

    count := 0
    for _, seq := range sequences {
        if ps.containsPrefix(seq, prefix) {
            count++
        }
    }

    return count
}

// containsPrefix 检查序列是否包含前缀
//
// Parameters:
//   - seq: 事件序列
//   - prefix: 前缀模式
//
// Returns: bool - 是否包含
func (ps *PrefixSpan) containsPrefix(
    seq EventSequence,
    prefix []EventStep,
) bool {
    if len(prefix) == 0 {
        return true
    }

    if len(seq.Events) < len(prefix) {
        return false
    }

    // 滑动窗口匹配
    j := 0
    for _, event := range seq.Events {
        if j >= len(prefix) {
            break
        }

        if ps.matchStep(event, prefix[j]) {
            j++
        }
    }

    return j == len(prefix)
}

// matchStep 匹配事件步骤
//
// Parameters:
//   - event: 事件
//   - step: 步骤
//
// Returns: bool - 是否匹配
func (ps *PrefixSpan) matchStep(event EventStep, step EventStep) bool {
    // 通配符匹配任意
    if step.Wildcard {
        return true
    }

    // 类型匹配
    if event.Type != step.Type {
        return false
    }

    // 应用匹配（如果指定）
    if step.Application != "" && event.Application != step.Application {
        return false
    }

    return true
}
```

**验证标准**:
- [ ] 算法正确性验证
- [ ] 测试用例通过
- [ ] 性能基准测试

---

### Step 6: 实现 AI Service (2-3 天)

**文件结构**:
```
internal/ai/
├── claude_client.go           # Claude API 客户端
├── pattern_filter.go          # 模式过滤
└── prompts.go                 # 提示词模板
```

#### Day 1: Claude 客户端

**claude_client.go**:
```go
// ClaudeClient Claude API 客户端
type ClaudeClient struct {
    apiKey     string
    baseURL    string
    httpClient *http.Client
    maxRetries int
}

// NewClaudeClient 创建 Claude 客户端
//
// Parameters:
//   - apiKey: Claude API 密钥
//
// Returns: *ClaudeClient - 客户端实例
func NewClaudeClient(apiKey string) *ClaudeClient {
    return &ClaudeClient{
        apiKey:     apiKey,
        baseURL:    "https://api.anthropic.com/v1/messages",
        httpClient: &http.Client{Timeout: 60 * time.Second},
        maxRetries: 3,
    }
}

// Complete 同步调用 Claude API
//
// Parameters:
//   - ctx: 上下文
//   - prompt: 提示词
//
// Returns: string - AI 响应, error - 错误信息
func (c *ClaudeClient) Complete(ctx context.Context, prompt string) (string, error) {
    request := ClaudeRequest{
        Model:     "claude-3-5-sonnet-20241022",
        MaxTokens: 4096,
        Messages: []ClaudeMessage{
            {Role: "user", Content: prompt},
        },
    }

    body, _ := json.Marshal(request)
    req, _ := http.NewRequestWithContext(
        ctx, "POST", c.baseURL, bytes.NewReader(body),
    )

    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("x-api-key", c.apiKey)
    req.Header.Set("anthropic-version", "2023-06-01")

    // 重试逻辑
    var lastErr error
    for i := 0; i < c.maxRetries; i++ {
        resp, err := c.httpClient.Do(req)
        if err == nil && resp.StatusCode == http.StatusOK {
            defer resp.Body.Close()

            var response ClaudeResponse
            if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
                return "", err
            }

            return c.extractContent(response), nil
        }

        lastErr = err
        time.Sleep(time.Second * time.Duration(i+1)) // 指数退避
    }

    return "", fmt.Errorf("Claude API 调用失败: %w", lastErr)
}
```

#### Day 2: 提示词模板

**prompts.go**:
```go
// BuildPatternAnalysisPrompt 构建模式分析提示词
//
// Parameters:
//   - pattern: 待分析的模式
//
// Returns: string - 提示词
func BuildPatternAnalysisPrompt(pattern *Pattern) string {
    var stepsDesc []string
    for i, step := range pattern.Sequence {
        stepsDesc = append(stepsDesc, fmt.Sprintf("%d. %s", i+1, describeStep(step)))
    }

    return fmt.Sprintf(`你是一个工作流自动化专家。请分析以下重复操作模式。

## 模式信息
- **出现次数**: %d
- **首次发现**: %s
- **最近发现**: %s
- **模式长度**: %d 步

## 操作步骤
%s

## 分析任务
请判断这个模式是否值得自动化，并回答：
1. 是否值得自动化？（考虑重复频率和复杂度）
2. 如果值得，主要原因是什么？
3. 预计每次可以节省多少时间？
4. 实现复杂度如何？

## 输出格式
请严格按照以下 JSON 格式回复：
{
  "should_automate": true或false,
  "reason": "简短的原因说明（中文）",
  "estimated_time_saving": 秒数（整数）,
  "complexity": "low"或"medium"或"high",
  "suggested_name": "推荐的自动化名称",
  "suggested_steps": [
    {
      "action": "操作类型",
      "params": {"key": "value"}
    }
  ]
}

注意：
- 如果模式过于简单（如单次点击），should_automate 应为 false
- 如果模式包含用户特定内容（如具体文本），应使用通配符
- estimated_time_saving 应基于实际操作时间估算`,
        pattern.SupportCount,
        pattern.FirstSeen.Format("2006-01-02 15:04"),
        pattern.LastSeen.Format("2006-01-02 15:04"),
        len(pattern.Sequence),
        strings.Join(stepsDesc, "\n"),
    )
}

// describeStep 描述步骤
func describeStep(step EventStep) string {
    switch step.Type {
    case "keyboard":
        return "键盘输入"
    case "clipboard":
        return "复制/粘贴"
    case "app_switch":
        return fmt.Sprintf("切换到应用: %s", step.Application)
    default:
        return step.Type
    }
}
```

#### Day 3: 模式过滤

**pattern_filter.go**:
```go
// AIPatternFilter AI模式过滤器
type AIPatternFilter interface {
    // ShouldAutomate 判断模式是否值得自动化
    ShouldAutomate(pattern *Pattern) (bool, *AIAnalysis, error)

    // AnalyzePattern 深度分析模式
    AnalyzePattern(pattern *Pattern) (*AIAnalysis, error)
}

// AIAnalysis AI分析结果
type AIAnalysis struct {
    ShouldAutomate      bool   `json:"should_automate"`
    Reason              string `json:"reason"`
    Complexity          string `json:"complexity"`          // low/medium/high
    EstimatedTimeSaving int    `json:"estimated_time_saving"` // 秒
    SuggestedName       string `json:"suggested_name"`
    SuggestedSteps      []Step `json:"suggested_steps"`
}

// Step 自动化步骤
type Step struct {
    Action string                 `json:"action"`
    Params map[string]interface{} `json:"params"`
}

// PatternFilter 模式过滤器实现
type PatternFilter struct {
    client    *ClaudeClient
    cache     *bbolt.DB
    rateLimiter *RateLimiter
}

// NewPatternFilter 创建模式过滤器
func NewPatternFilter(client *ClaudeClient, cacheDB *bbolt.DB) *PatternFilter {
    return &PatternFilter{
        client:      client,
        cache:       cacheDB,
        rateLimiter: NewRateLimiter(3), // 最大3个并发
    }
}

// ShouldAutomate 判断模式是否值得自动化
//
// Parameters:
//   - pattern: 待判断的模式
//
// Returns: bool - 是否值得自动化, *AIAnalysis - AI分析结果, error - 错误信息
func (pf *PatternFilter) ShouldAutomate(pattern *Pattern) (bool, *AIAnalysis, error) {
    // 1. 检查缓存
    cacheKey := generatePatternHash(pattern)
    if cached, found := pf.getFromCache(cacheKey); found {
        var analysis AIAnalysis
        json.Unmarshal(cached, &analysis)
        return analysis.ShouldAutomate, &analysis, nil
    }

    // 2. 限流控制
    pf.rateLimiter.Acquire()
    defer pf.rateLimiter.Release()

    // 3. 调用 Claude API
    prompt := BuildPatternAnalysisPrompt(pattern)
    response, err := pf.client.Complete(context.Background(), prompt)
    if err != nil {
        return false, nil, fmt.Errorf("Claude API 调用失败: %w", err)
    }

    // 4. 解析响应
    var analysis AIAnalysis
    if err := json.Unmarshal([]byte(response), &analysis); err != nil {
        return false, nil, fmt.Errorf("解析 AI 响应失败: %w", err)
    }

    // 5. 保存到缓存
    pf.saveToCache(cacheKey, &analysis, 24*time.Hour)

    return analysis.ShouldAutomate, &analysis, nil
}
```

---

### Step 7: 实现 Analyzer Engine 主引擎 (2 天)

**文件结构**:
```
internal/analyzer/
├── engine.go                  # 主引擎
└── config.go                  # 配置管理
```

**engine.go**:
```go
// Engine 分析引擎
type Engine struct {
    // 组件
    eventRepo    EventRepository
    sessionDiv   SessionDivider
    patternMiner PatternMiner
    aiFilter     AIPatternFilter
    recommender  PatternRecommender

    // 配置
    config *Config

    // 状态
    eventBuffer   []events.Event
    knownPatterns map[string]*Pattern

    // 通道
    eventChan     <-chan events.Event
    patternChan   chan *Pattern
    recommendChan chan *Recommendation

    // 控制
    ctx    context.Context
    cancel context.CancelFunc
    wg     sync.WaitGroup
}

// Config 引擎配置
type Config struct {
    // EventRepository
    DBPath string

    // SessionDivider
    SessionTimeout time.Duration

    // PatternMiner
    MinSupport     int
    MinConfidence  float64
    MiningInterval time.Duration // 模式挖掘间隔

    // AI
    ClaudeAPIKey string

    // BatchWriter
    BatchSize     int
    FlushInterval time.Duration
}

// NewEngine 创建分析引擎
//
// Parameters:
//   - config: 引擎配置
//   - eventBus: 事件总线
//
// Returns: *Engine - 引擎实例, error - 错误信息
func NewEngine(config *Config, eventBus *events.EventBus) (*Engine, error) {
    // 初始化存储
    eventRepo, err := storage.NewSQLiteEventRepository(config.DBPath)
    if err != nil {
        return nil, err
    }

    // 初始化AI客户端
    aiClient := ai.NewClaudeClient(config.ClaudeAPIKey)

    // 创建引擎
    ctx, cancel := context.WithCancel(context.Background())

    return &Engine{
        eventRepo:     eventRepo,
        sessionDiv:    NewSessionDivider(config.SessionTimeout),
        patternMiner:  NewPatternMiner(config.MinSupport, config.MinConfidence),
        aiFilter:      ai.NewPatternFilter(aiClient),
        recommender:   NewRecommender(),
        config:        config,
        knownPatterns: make(map[string]*Pattern),
        ctx:           ctx,
        cancel:        cancel,
    }, nil
}

// Start 启动分析引擎
//
// Parameters:
//   - eventBus: 事件总线
//
// Returns: error - 错误信息
func (e *Engine) Start(eventBus *events.EventBus) error {
    logger.Info("启动分析引擎")

    // 创建事件通道
    e.eventChan = make(chan events.Event, 1000)

    // 订阅所有监控事件
    eventBus.Subscribe("*", func(event events.Event) error {
        e.eventChan <- event
        return nil
    })

    // 启动批量写入器
    batchWriter := storage.NewBatchWriter(e.eventRepo, e.config.BatchSize, e.config.FlushInterval)

    // 启动事件处理循环
    e.wg.Add(1)
    go e.processEvents(batchWriter)

    // 启动模式挖掘循环
    e.wg.Add(1)
    go e.miningLoop()

    // 启动AI过滤循环
    e.wg.Add(1)
    go e.aiFilterLoop()

    return nil
}

// processEvents 事件处理循环
func (e *Engine) processEvents(batchWriter *storage.BatchWriter) {
    defer e.wg.Done()

    for {
        select {
        case event := <-e.eventChan:
            // 添加到批量写入
            batchWriter.Add(event)

            // 添加到内存缓冲
            e.eventBuffer = append(e.eventBuffer, event)

            // 限制缓冲区大小
            if len(e.eventBuffer) > 10000 {
                e.eventBuffer = e.eventBuffer[len(e.eventBuffer)-10000:]
            }

        case <-e.ctx.Done():
            // 刷新剩余事件
            batchWriter.Flush()
            return
        }
    }
}

// miningLoop 模式挖掘循环
func (e *Engine) miningLoop() {
    defer e.wg.Done()

    ticker := time.NewTicker(e.config.MiningInterval)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            e.minePatterns()

        case <-e.ctx.Done():
            return
        }
    }
}

// minePatterns 挖掘模式
func (e *Engine) minePatterns() {
    logger.Info("开始挖掘模式")

    // 1. 从数据库查询最近的事件
    cutoff := time.Now().Add(-24 * time.Hour)
    events, err := e.eventRepo.FindByTimeRange(cutoff, time.Now())
    if err != nil {
        logger.Error("查询事件失败", zap.Error(err))
        return
    }

    // 2. 划分会话
    sessions := e.sessionDiv.Divide(events)
    logger.Info("划分会话", zap.Int("count", len(sessions)))

    // 3. 挖掘模式
    patterns, err := e.patternMiner.Mine(sessions)
    if err != nil {
        logger.Error("挖掘模式失败", zap.Error(err))
        return
    }

    logger.Info("发现模式", zap.Int("count", len(patterns)))

    // 4. 过滤已知模式
    newPatterns := e.filterKnownPatterns(patterns)

    // 5. 发送到AI过滤
    for _, pattern := range newPatterns {
        e.patternChan <- pattern
    }
}

// Stop 停止分析引擎
//
// Returns: error - 错误信息
func (e *Engine) Stop() error {
    logger.Info("停止分析引擎")

    e.cancel()
    e.wg.Wait()

    return nil
}
```

---

### Step 8-11: 前端集成、测试、部署

详见实施计划文档。

---

## ✅ 验证标准

### 功能验证
- [ ] 能记录用户所有关键操作
- [ ] 能检测到用户重复操作（如启动开发环境）
- [ ] AI 正确判断哪些模式值得自动化
- [ ] UI 显示自动化建议列表
- [ ] 用户可以接受/拒绝建议

### 性能验证
- [ ] 事件持久化: >1000 events/sec
- [ ] 模式挖掘: <5s 处理 1000 事件
- [ ] 内存占用: <100MB
- [ ] CPU 使用: <10% (空闲时)

### 质量验证
- [ ] 单元测试覆盖率 ≥80%
- [ ] 核心模块覆盖率 ≥90%
- [ ] 所有中文注释完整
- [ ] 文档完整

---

## 📊 成功指标

1. **准确性**: 能发现 90% 的重复操作模式
2. **性能**: 分析 1 天数据 <5 秒
3. **实用性**: AI 建议的自动化接受率 >70%
4. **稳定性**: 连续运行 7 天无崩溃

---

## 🔗 相关文档

- [系统架构](../architecture/01-system-architecture.md)
- [分析引擎](../architecture/03-analyzer-engine.md)
- [AI 服务](../architecture/04-ai-service.md)
- [数据库设计](../design/01-database-design.md)
- [开发环境搭建](./01-development-setup.md)
- [Phase 1: 基础监控](./02-phase1-monitoring.md)

---

**最后更新**: 2026-01-30
