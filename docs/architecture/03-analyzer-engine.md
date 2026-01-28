# 分析引擎 (Analyzer Engine)

分析引擎是 FlowMind 的大脑，负责从原始事件中提取模式、识别重复操作、分析上下文，为 AI 自动化提供决策依据。

---

## 设计目标

1. **模式挖掘**：自动发现重复性操作序列
2. **实时分析**：低延迟的事件流处理
3. **AI 增强**：结合 Claude API 理解语义
4. **可配置**：支持用户自定义模式识别规则

---

## 架构设计

### 组件关系

```
Event Stream (事件流)
    ↓
┌─────────────────────────────────────┐
│     Event Preprocessor              │  事件预处理
│  - 清洗、去重、标准化                │
└─────────────────────────────────────┘
    ↓
┌─────────────────────────────────────┐
│     Sliding Window                  │  滑动窗口
│  - 时间窗口聚合                      │
│  - 会话划分                          │
└─────────────────────────────────────┘
    ↓
┌─────────────────────────────────────┐
│     Pattern Miner                   │  模式挖掘
│  - PrefixSpan 算法                   │
│  - 频繁序列识别                      │
└─────────────────────────────────────┘
    ↓
┌─────────────────────────────────────┐
│     AI Filter                       │  AI 过滤
│  - Claude API 判断价值               │
│  - 排除无意义模式                    │
└─────────────────────────────────────┘
    ↓
Pattern Suggestions (模式建议)
```

---

## 核心数据结构

```go
// internal/analyzer/types.go
package analyzer

import "time"

// Event 事件（复用 Monitor 的结构）
type Event struct {
    ID        string                 `json:"id"`
    Type      string                 `json:"type"`
    Timestamp time.Time              `json:"timestamp"`
    Data      map[string]interface{} `json:"data"`
    Context   *EventContext          `json:"context"`
}

// EventSequence 事件序列
type EventSequence struct {
    Events     []Event   `json:"events"`
    StartTime  time.Time `json:"start_time"`
    EndTime    time.Time `json:"end_time"`
    SessionID  string    `json:"session_id"`
}

// Pattern 模式
type Pattern struct {
    ID           string        `json:"id"`
    Sequence     []EventStep   `json:"sequence"`
    SupportCount int           `json:"support_count"`
    FirstSeen    time.Time     `json:"first_seen"`
    LastSeen     time.Time     `json:"last_seen"`
    Confidence   float64       `json:"confidence"`
    IsAutomated  bool          `json:"is_automated"`
    AutomationID string        `json:"automation_id,omitempty"`
}

// EventStep 事件步骤（抽象）
type EventStep struct {
    Type        string                 `json:"type"`
    Application string                 `json:"application,omitempty"`
    Data        map[string]interface{} `json:"data,omitempty"`
    Tolerance   int                    `json:"tolerance,omitempty"` // 容忍度
}

// Session 会话（一组连续操作）
type Session struct {
    ID          string    `json:"id"`
    StartTime   time.Time `json:"start_time"`
    EndTime     time.Time `json:"end_time"`
    Application string    `json:"application"`
    EventCount  int       `json:"event_count"`
}
```

---

## 事件预处理

### 清洗与标准化

```go
// internal/analyzer/preprocessor.go
type Preprocessor struct {
    filters []Filter
}

type Filter interface {
    Apply(event Event) (Event, bool) // 返回过滤后的事件和是否保留
}

// 去重过滤器
type DedupFilter struct {
    seenEvents map[string]time.Time
    ttl        time.Duration
}

func (df *DedupFilter) Apply(event Event) (Event, bool) {
    key := event.Type + event.ID

    if lastSeen, exists := df.seenEvents[key]; exists {
        if time.Since(lastSeen) < df.ttl {
            return event, false // 丢弃重复事件
        }
    }

    df.seenEvents[key] = time.Now()
    return event, true
}

// 噪声过滤器（过滤无关事件）
type NoiseFilter struct {
    ignoredTypes map[string]bool
    ignoredApps  map[string]bool
}

func (nf *NoiseFilter) Apply(event Event) (Event, bool) {
    // 忽略鼠标移动等高频事件
    if nf.ignoredTypes[event.Type] {
        return event, false
    }

    // 忽略系统应用
    if nf.ignoredApps[event.Context.Application] {
        return event, false
    }

    return event, true
}

// 预处理管道
func (p *Preprocessor) Process(event Event) (Event, bool) {
    for _, filter := range p.filters {
        processed, keep := filter.Apply(event)
        if !keep {
            return processed, false
        }
        event = processed
    }
    return event, true
}
```

---

## 滑动窗口与会话划分

### 时间窗口

```go
// internal/analyzer/window.go
type SlidingWindow struct {
    events    []Event
    size      time.Duration // 窗口大小（如 30 分钟）
    slide     time.Duration // 滑动步长（如 5 分钟）
    lastSlide time.Time
}

func (sw *SlidingWindow) Add(event Event) []Event {
    sw.events = append(sw.events, event)

    // 清理过期事件
    cutoff := event.Timestamp.Add(-sw.size)
    var valid []Event
    for _, e := range sw.events {
        if e.Timestamp.After(cutoff) {
            valid = append(valid, e)
        }
    }
    sw.events = valid

    // 检查是否需要滑动
    if time.Since(sw.lastSlide) >= sw.slide {
        sw.lastSlide = time.Now()
        return sw.GetEvents()
    }

    return nil
}

func (sw *SlidingWindow) GetEvents() []Event {
    return sw.events
}
```

### 会话划分

```go
// internal/analyzer/session.go
type SessionDetector struct {
    timeout      time.Duration // 会话超时（如 10 分钟无操作）
    current      *Session
    eventChan    <-chan Event
    sessionChan  chan<- *Session
}

func (sd *SessionDetector) Start() {
    for event := range sd.eventChan {
        if sd.current == nil {
            // 开始新会话
            sd.current = &Session{
                ID:        generateSessionID(),
                StartTime: event.Timestamp,
                EndTime:   event.Timestamp,
                EventCount: 1,
            }
            continue
        }

        // 检查会话超时
        if event.Timestamp.Sub(sd.current.EndTime) > sd.timeout {
            // 结束当前会话
            sd.sessionChan <- sd.current

            // 开始新会话
            sd.current = &Session{
                ID:        generateSessionID(),
                StartTime: event.Timestamp,
                EndTime:   event.Timestamp,
                EventCount: 1,
            }
        } else {
            // 继续当前会话
            sd.current.EndTime = event.Timestamp
            sd.current.EventCount++
        }
    }

    // 发送最后一个会话
    if sd.current != nil {
        sd.sessionChan <- sd.current
    }
}
```

---

## 模式挖掘算法

### PrefixSpan 算法

```go
// internal/analyzer/prefixspan.go
type PrefixSpan struct {
    minSupport int // 最小支持度
}

func (ps *PrefixSpan) Mine(sequences []EventSequence) []Pattern {
    patterns := make([]Pattern, 0)

    // 递归挖掘频繁模式
    ps.mineRecursive(sequences, []EventStep{}, &patterns)

    return patterns
}

func (ps *PrefixSpan) mineRecursive(
    sequences []EventSequence,
    prefix []EventStep,
    patterns *[]Pattern,
) {
    // 计算前缀支持度
    support := ps.calculateSupport(sequences, prefix)
    if support < ps.minSupport && len(prefix) > 0 {
        return
    }

    // 保存模式
    if len(prefix) > 1 {
        *patterns = append(*patterns, Pattern{
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

        ps.mineRecursive(projectedDB, newPrefix, patterns)
    }
}

func (ps *PrefixSpan) calculateSupport(
    sequences []EventSequence,
    prefix []EventStep,
) int {
    count := 0

    for _, seq := range sequences {
        if ps.containsPrefix(seq, prefix) {
            count++
        }
    }

    return count
}

func (ps *PrefixSpan) containsPrefix(seq EventSequence, prefix []EventStep) bool {
    if len(prefix) == 0 {
        return true
    }

    if len(seq.Events) < len(prefix) {
        return false
    }

    j := 0
    for _, event := range seq.Events {
        if j >= len(prefix) {
            break
        }

        step := prefix[j]
        if ps.matchStep(event, step) {
            j++
        }
    }

    return j == len(prefix)
}

func (ps *PrefixSpan) matchStep(event Event, step EventStep) bool {
    if event.Type != step.Type {
        return false
    }

    if step.Application != "" && event.Context.Application != step.Application {
        return false
    }

    return true
}

func (ps *PrefixSpan) buildProjectedDB(
    sequences []EventSequence,
    prefix []EventStep,
) []EventSequence {
    projected := make([]EventSequence, 0)

    for _, seq := range sequences {
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

func (ps *PrefixSpan) findPrefixIndex(seq EventSequence, prefix []EventStep) int {
    if len(prefix) == 0 {
        return 0
    }

    for i := 0; i <= len(seq.Events)-len(prefix); i++ {
        matched := true

        for j, step := range prefix {
            if !ps.matchStep(seq.Events[i+j], step) {
                matched = false
                break
            }
        }

        if matched {
            return i
        }
    }

    return -1
}

func (ps *PrefixSpan) findFrequentItems(sequences []EventSequence) []EventStep {
    freq := make(map[string]int)

    for _, seq := range sequences {
        for _, event := range seq.Events {
            // 简化：仅用类型作为特征
            key := event.Type
            freq[key]++
        }
    }

    var items []EventStep
    for key, count := range freq {
        if count >= ps.minSupport {
            items = append(items, EventStep{
                Type: key,
            })
        }
    }

    return items
}
```

### 模式检测

```go
// internal/analyzer/detector.go
type PatternDetector struct {
    knownPatterns []Pattern
    window        *SlidingWindow
}

func (pd *PatternDetector) DetectNewPatterns() []Pattern {
    events := pd.window.GetEvents()

    // 构造序列
    sequences := pd.buildSequences(events)

    // 挖掘模式
    miner := &PrefixSpan{minSupport: 3}
    patterns := miner.Mine(sequences)

    // 过滤已知模式
    newPatterns := make([]Pattern, 0)
    for _, pattern := range patterns {
        if !pd.isKnownPattern(pattern) {
            newPatterns = append(newPatterns, pattern)
            pd.knownPatterns = append(pd.knownPatterns, pattern)
        }
    }

    return newPatterns
}

func (pd *PatternDetector) DetectMatchingPattern(recentEvents []Event) *Pattern {
    for _, pattern := range pd.knownPatterns {
        if pd.matchSequence(recentEvents, pattern.Sequence) {
            return &pattern
        }
    }
    return nil
}

func (pd *PatternDetector) matchSequence(events []Event, sequence []EventStep) bool {
    if len(events) < len(sequence) {
        return false
    }

    for i := 0; i <= len(events)-len(sequence); i++ {
        matched := true

        for j, step := range sequence {
            if !pd.matchStep(events[i+j], step) {
                matched = false
                break
            }
        }

        if matched {
            return true
        }
    }

    return false
}

func (pd *PatternDetector) matchStep(event Event, step EventStep) bool {
    // 类型匹配
    if event.Type != step.Type {
        return false
    }

    // 应用匹配（可选）
    if step.Application != "" && event.Context.Application != step.Application {
        return false
    }

    // 数据匹配（可选，如快捷键）
    if keyCode, ok := step.Data["keycode"]; ok {
        if eventKeyCode, ok := event.Data["keycode"]; !ok || eventKeyCode != keyCode {
            return false
        }
    }

    return true
}

func (pd *PatternDetector) isKnownPattern(pattern Pattern) bool {
    for _, known := range pd.knownPatterns {
        if pd.samePattern(pattern, known) {
            return true
        }
    }
    return false
}

func (pd *PatternDetector) samePattern(p1, p2 Pattern) bool {
    if len(p1.Sequence) != len(p2.Sequence) {
        return false
    }

    for i := range p1.Sequence {
        if p1.Sequence[i].Type != p2.Sequence[i].Type {
            return false
        }
    }

    return true
}

func (pd *PatternDetector) buildSequences(events []Event) []EventSequence {
    // 按时间窗口划分序列
    // 这里简化为每个时间窗口一个序列
    return []EventSequence{
        {
            Events:    events,
            StartTime: events[0].Timestamp,
            EndTime:   events[len(events)-1].Timestamp,
            SessionID: generateSessionID(),
        },
    }
}
```

---

## AI 增强

### Claude API 过滤

```go
// internal/analyzer/ai_filter.go
type AIFilter struct {
    aiClient *ai.AIClient
    cache    map[string]bool
}

func (af *AIFilter) ShouldAutomate(pattern Pattern) (bool, string) {
    // 检查缓存
    if cached, exists := af.cache[pattern.ID]; exists {
        return cached, ""
    }

    // 构建 prompt
    prompt := af.buildPrompt(pattern)

    // 调用 Claude
    response, err := af.aiClient.Complete(prompt)
    if err != nil {
        log.Error("AI filter error:", err)
        return false, ""
    }

    // 解析响应
    result := af.parseResponse(response)

    // 缓存结果
    af.cache[pattern.ID] = result.ShouldAutomate

    return result.ShouldAutomate, result.Reason
}

func (af *AIFilter) buildPrompt(pattern Pattern) string {
    var stepsDesc []string
    for i, step := range pattern.Sequence {
        desc := fmt.Sprintf("%d. %s", i+1, af.describeStep(step))
        stepsDesc = append(stepsDesc, desc)
    }

    return fmt.Sprintf(`你是一个工作流优化专家。我发现了以下重复操作模式：

模式 ID: %s
出现次数: %d
操作步骤：
%s

请分析：
1. 这个模式是否值得自动化？（是/否）
2. 如果值得，为什么？如果不值得，为什么？

请用 JSON 格式回复：
{
  "should_automate": true/false,
  "reason": "原因说明",
  "estimated_time_saving": "预计每次节省时间（分钟）",
  "complexity": "low/medium/high"
}`,
        pattern.ID,
        pattern.SupportCount,
        strings.Join(stepsDesc, "\n"),
    )
}

func (af *AIFilter) describeStep(step EventStep) string {
    switch step.Type {
    case "keyboard":
        if keyCode, ok := step.Data["keycode"]; ok {
            return fmt.Sprintf("按键 %d", keyCode)
        }
        return "按键操作"
    case "app_switch":
        return fmt.Sprintf("切换到应用 %s", step.Application)
    case "clipboard":
        return "剪贴板操作"
    default:
        return step.Type
    }
}

type AIFilterResult struct {
    ShouldAutomate       bool   `json:"should_automate"`
    Reason               string `json:"reason"`
    EstimatedTimeSaving  string `json:"estimated_time_saving"`
    Complexity           string `json:"complexity"`
}

func (af *AIFilter) parseResponse(response string) AIFilterResult {
    var result AIFilterResult

    // 提取 JSON（处理可能的额外文本）
    jsonStart := strings.Index(response, "{")
    jsonEnd := strings.LastIndex(response, "}")

    if jsonStart != -1 && jsonEnd != -1 {
        jsonStr := response[jsonStart : jsonEnd+1]
        if err := json.Unmarshal([]byte(jsonStr), &result); err == nil {
            return result
        }
    }

    // 默认：不值得自动化
    return AIFilterResult{ShouldAutomate: false}
}
```

---

## 上下文分析

### 应用上下文理解

```go
// internal/analyzer/context.go
type ContextAnalyzer struct {
    aiClient *ai.AIClient
}

func (ca *ContextAnalyzer) UnderstandContext(events []Event) (*ContextInsight, error) {
    // 提取关键信息
    apps := ca.extractApplications(events)
    files := ca.extractFiles(events)
    actions := ca.extractActions(events)

    // 构建 prompt
    prompt := fmt.Sprintf(`基于以下操作序列，分析用户的工作上下文：

应用：%s
文件：%s
操作：%s

请分析：
1. 用户在做什么任务？
2. 这是什么类型的工作？（开发/设计/文档等）
3. 可能的目标是什么？

请用 JSON 格式回复：
{
  "task_type": "开发/设计/文档/会议等",
  "task_description": "简短描述",
  "domain": "技术/设计/产品等",
  "tools": ["工具1", "工具2"]
}`,
        strings.Join(apps, ", "),
        strings.Join(files, ", "),
        strings.Join(actions, ", "),
    )

    response, err := ca.aiClient.Complete(prompt)
    if err != nil {
        return nil, err
    }

    var insight ContextInsight
    if err := json.Unmarshal([]byte(response), &insight); err != nil {
        return nil, err
    }

    return &insight, nil
}

type ContextInsight struct {
    TaskType        string   `json:"task_type"`
    TaskDescription string   `json:"task_description"`
    Domain          string   `json:"domain"`
    Tools           []string `json:"tools"`
}

func (ca *ContextAnalyzer) extractApplications(events []Event) []string {
    set := make(map[string]bool)
    for _, event := range events {
        if event.Context.Application != "" {
            set[event.Context.Application] = true
        }
    }

    var apps []string
    for app := range set {
        apps = append(apps, app)
    }

    return apps
}

func (ca *ContextAnalyzer) extractFiles(events []Event) []string {
    set := make(map[string]bool)
    for _, event := range events {
        if event.Context.FilePath != "" {
            set[event.Context.FilePath] = true
        }
    }

    var files []string
    for file := range set {
        files = append(files, filepath.Base(file))
    }

    return files
}

func (ca *ContextAnalyzer) extractActions(events []Event) []string {
    var actions []string
    for _, event := range events {
        action := ca.describeAction(event)
        actions = append(actions, action)
    }
    return actions
}

func (ca *ContextAnalyzer) describeAction(event Event) string {
    switch event.Type {
    case "keyboard":
        return "键盘输入"
    case "app_switch":
        return "切换应用"
    case "clipboard":
        return "复制粘贴"
    default:
        return event.Type
    }
}
```

---

## 分析引擎集成

```go
// internal/analyzer/engine.go
type Engine struct {
    preprocessor    *Preprocessor
    sessionDetector *SessionDetector
    patternDetector *PatternDetector
    aiFilter        *AIFilter
    contextAnalyzer *ContextAnalyzer

    eventChan    <-chan Event
    patternChan  chan<- Pattern
    contextChan  chan<- *ContextInsight
}

func NewEngine(eventChan <-chan Event) *Engine {
    return &Engine{
        preprocessor:    NewPreprocessor(),
        sessionDetector: NewSessionDetector(eventChan),
        patternDetector: NewPatternDetector(),
        aiFilter:        NewAIFilter(),
        contextAnalyzer: NewContextAnalyzer(),
        eventChan:       eventChan,
        patternChan:     make(chan Pattern, 10),
        contextChan:     make(chan *ContextInsight, 10),
    }
}

func (e *Engine) Start() error {
    // 启动会话检测
    sessionChan := make(chan *Session, 100)
    go e.sessionDetector.Start(sessionChan)

    // 启动模式检测
    go func() {
        ticker := time.NewTicker(5 * time.Minute)
        defer ticker.Stop()

        for range ticker.C {
            patterns := e.patternDetector.DetectNewPatterns()

            // AI 过滤
            for _, pattern := range patterns {
                shouldAutomate, reason := e.aiFilter.ShouldAutomate(pattern)
                if shouldAutomate {
                    log.Info("发现可自动化模式:", pattern.ID, "原因:", reason)
                    e.patternChan <- pattern
                }
            }
        }
    }()

    // 启动上下文分析
    go func() {
        window := NewSlidingWindow(30 * time.Minute)

        for event := range e.eventChan {
            // 预处理
            processed, keep := e.preprocessor.Process(event)
            if !keep {
                continue
            }

            // 添加到窗口
            window.Add(processed)

            // 定期分析上下文
            if len(window.GetEvents()) > 10 {
                insight, err := e.contextAnalyzer.UnderstandContext(window.GetEvents())
                if err == nil {
                    e.contextChan <- insight
                }
            }
        }
    }()

    return nil
}
```

---

## 性能优化

### 增量模式挖掘

```go
// internal/analyzer/incremental.go
type IncrementalMiner struct {
    basePatterns  []Pattern
    baseSupport   int
    newEvents     []Event
    threshold     int
}

func (im *IncrementalMiner) Update(newEvents []Event) []Pattern {
    im.newEvents = append(im.newEvents, newEvents...)

    // 当新事件达到阈值时，重新挖掘
    if len(im.newEvents) >= im.threshold {
        return im.remine()
    }

    return nil
}

func (im *IncrementalMiner) remine() []Pattern {
    // 合并新旧数据
    allEvents := im.mergeEvents()

    // 重新挖掘
    miner := &PrefixSpan{minSupport: im.baseSupport}
    patterns := miner.Mine(allEvents)

    // 找出新模式
    newPatterns := im.diffPatterns(patterns)

    // 更新基础模式
    im.basePatterns = patterns
    im.newEvents = nil

    return newPatterns
}
```

---

**相关文档**：
- [系统架构](./01-system-architecture.md)
- [监控引擎](./02-monitor-engine.md)
- [实施指南 Phase 2](../implementation/03-phase2-patterns.md)
