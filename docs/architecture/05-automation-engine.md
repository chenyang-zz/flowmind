# 自动化引擎 (Automation Engine)

自动化引擎是 FlowMind 的执行组件，负责将自然语言需求转换为可执行的自动化脚本，并提供安全的执行环境和任务调度。

---

## 设计目标

1. **自然语言生成**：AI 理解需求并生成脚本
2. **安全执行**：沙箱隔离，权限控制
3. **任务调度**：支持定时和事件触发
4. **可视化编辑**：用户可查看和修改
5. **日志审计**：完整的执行记录

---

## 架构设计

```
用户需求（自然语言）
    ↓
┌─────────────────────────────────────┐
│     Script Generator                │  AI 生成脚本
│  - Claude API                       │
│  - DSL 解析                         │
└─────────────────────────────────────┘
    ↓
┌─────────────────────────────────────┐
│     Automation Editor               │  可视化编辑
│  - 步骤展示                         │
│  - 参数配置                         │
│  - 测试执行                         │
└─────────────────────────────────────┘
    ↓
┌─────────────────────────────────────┐
│     Sandbox Validator               │  安全验证
│  - 权限检查                         │
│  - 资源限制                         │
└─────────────────────────────────────┘
    ↓
┌─────────────────────────────────────┐
│     Scheduler                       │  任务调度
│  - Cron 表达式                      │
│  - 事件触发                         │
└─────────────────────────────────────┘
    ↓
┌─────────────────────────────────────┐
│     Step Executor                   │  步骤执行
│  - Git / Slack / Shell              │
│  - AI 处理                          │
│  - 文件操作                         │
└─────────────────────────────────────┘
    ↓
执行结果 + 日志
```

---

## 核心数据结构

```go
// internal/automation/types.go
package automation

import "time"

// AutomationScript 自动化脚本
type AutomationScript struct {
    ID          string         `json:"id"`
    Name        string         `json:"name"`
    Description string         `json:"description"`
    Steps       []Step         `json:"steps"`
    Trigger     TriggerConfig  `json:"trigger"`
    Enabled     bool           `json:"enabled"`
    CreatedAt   time.Time      `json:"created_at"`
    UpdatedAt   time.Time      `json:"updated_at"`
    RunCount    int            `json:"run_count"`
    LastRun     *time.Time     `json:"last_run,omitempty"`
    NextRun     *time.Time     `json:"next_run,omitempty"`
}

// Step 执行步骤
type Step struct {
    ID       string                 `json:"id"`
    Action   string                 `json:"action"`
    Params   map[string]interface{} `json:"params"`
    ContinueOnError bool            `json:"continue_on_error"`
}

// TriggerConfig 触发器配置
type TriggerConfig struct {
    Type     string                 `json:"type"` // cron, event, manual
    Config   map[string]interface{} `json:"config"`
}

// ExecutionResult 执行结果
type ExecutionResult struct {
    ScriptID    string        `json:"script_id"`
    RunID       string        `json:"run_id"`
    StartTime   time.Time     `json:"start_time"`
    EndTime     time.Time     `json:"end_time"`
    Status      string        `json:"status"` // success, failed, partial
    Steps       []StepResult  `json:"steps"`
    Error       string        `json:"error,omitempty"`
    Output      string        `json:"output,omitempty"`
}

// StepResult 步骤执行结果
type StepResult struct {
    StepID    string        `json:"step_id"`
    Status    string        `json:"status"`
    StartTime time.Time     `json:"start_time"`
    EndTime   time.Time     `json:"end_time"`
    Output    string        `json:"output"`
    Error     string        `json:"error,omitempty"`
}
```

---

## 脚本生成器

### AI 生成

```go
// internal/automation/generator.go
type Generator struct {
    aiService *ai.AIService
}

func NewGenerator(aiService *ai.AIService) *Generator {
    return &Generator{
        aiService: aiService,
    }
}

func (g *Generator) GenerateFromNaturalLanguage(description string) (*AutomationScript, error) {
    // 使用 AI 生成
    prompt := fmt.Sprintf(`用户需求: %s

请生成自动化脚本，输出 JSON 格式:
{
  "name": "自动化名称",
  "description": "简短描述",
  "steps": [
    {
      "action": "动作类型",
      "params": {
        "参数名": "参数值"
      }
    }
  ],
  "trigger": {
    "type": "cron/event/manual",
    "config": {}
  }
}

支持的动作类型:
- git.commit: Git 提交 (params: path, message)
- git.push: Git 推送 (params: path, remote, branch)
- slack.send: 发送 Slack 消息 (params: webhook, channel, message)
- shell.exec: 执行 Shell 命令 (params: command, working_dir)
- file.move: 移动文件 (params: source, destination)
- file.copy: 复制文件 (params: source, destination)
- ai.analyze: AI 分析文本 (params: text, prompt)
- ai.generate: AI 生成内容 (params: prompt, template)
- notification.show: 显示通知 (params: title, message)
- wait.wait: 等待 (params: duration)`, description)

    response, err := g.aiService.Complete(prompt)
    if err != nil {
        return nil, err
    }

    // 解析 JSON
    var script AutomationScript
    if err := json.Unmarshal([]byte(response), &script); err != nil {
        return nil, fmt.Errorf("failed to parse AI response: %w", err)
    }

    // 验证脚本
    if err := g.ValidateScript(&script); err != nil {
        return nil, err
    }

    // 生成 ID
    script.ID = generateID()
    for i := range script.Steps {
        script.Steps[i].ID = generateID()
    }

    return &script, nil
}

func (g *Generator) ValidateScript(script *AutomationScript) error {
    if script.Name == "" {
        return fmt.Errorf("script name is required")
    }

    if len(script.Steps) == 0 {
        return fmt.Errorf("script must have at least one step")
    }

    // 验证每个步骤
    for _, step := range script.Steps {
        if err := g.ValidateStep(&step); err != nil {
            return fmt.Errorf("step %d: %w", step.ID, err)
        }
    }

    // 验证触发器
    if err := g.ValidateTrigger(&script.Trigger); err != nil {
        return fmt.Errorf("trigger: %w", err)
    }

    return nil
}

func (g *Generator) ValidateStep(step *Step) error {
    // 检查动作类型
    validActions := map[string]bool{
        "git.commit":       true,
        "git.push":         true,
        "slack.send":       true,
        "shell.exec":       true,
        "file.move":        true,
        "file.copy":        true,
        "ai.analyze":       true,
        "ai.generate":      true,
        "notification.show": true,
        "wait.wait":        true,
    }

    if !validActions[step.Action] {
        return fmt.Errorf("invalid action: %s", step.Action)
    }

    // 检查必需参数
    requiredParams := g.getRequiredParams(step.Action)
    for _, param := range requiredParams {
        if _, exists := step.Params[param]; !exists {
            return fmt.Errorf("missing required parameter: %s", param)
        }
    }

    return nil
}

func (g *Generator) getRequiredParams(action string) []string {
    params := map[string][]string{
        "git.commit":       {"path", "message"},
        "git.push":         {"path"},
        "slack.send":       {"webhook", "channel", "message"},
        "shell.exec":       {"command"},
        "file.move":        {"source", "destination"},
        "file.copy":        {"source", "destination"},
        "ai.analyze":       {"text"},
        "ai.generate":      {"prompt"},
        "notification.show": {"title", "message"},
        "wait.wait":        {"duration"},
    }

    return params[action]
}

func (g *Generator) ValidateTrigger(trigger *TriggerConfig) error {
    validTypes := map[string]bool{
        "cron":   true,
        "event":  true,
        "manual": true,
    }

    if !validTypes[trigger.Type] {
        return fmt.Errorf("invalid trigger type: %s", trigger.Type)
    }

    // 验证 cron 表达式
    if trigger.Type == "cron" {
        if expr, ok := trigger.Config["expr"]; ok {
            if _, err := cron.ParseStandard(expr.(string)); err != nil {
                return fmt.Errorf("invalid cron expression: %w", err)
            }
        }
    }

    return nil
}
```

---

## 沙箱执行

### 安全验证

```go
// internal/automation/sandbox.go
type Sandbox struct {
    allowedPaths   []string
    allowedActions []string
    maxMemory      int64
    maxDuration    time.Duration
    networkAccess  bool
}

func NewSandbox(config *SandboxConfig) *Sandbox {
    return &Sandbox{
        allowedPaths:   config.AllowedPaths,
        allowedActions: config.AllowedActions,
        maxMemory:      config.MaxMemory,
        maxDuration:    config.MaxDuration,
        networkAccess:  config.NetworkAccess,
    }
}

func (s *Sandbox) ValidateScript(script *AutomationScript) error {
    // 检查动作权限
    for _, step := range script.Steps {
        if !s.isActionAllowed(step.Action) {
            return fmt.Errorf("action not allowed: %s", step.Action)
        }

        // 检查路径权限
        if err := s.validateStepPaths(&step); err != nil {
            return err
        }
    }

    return nil
}

func (s *Sandbox) isActionAllowed(action string) bool {
    for _, allowed := range s.allowedActions {
        if action == allowed || allowed == "*" {
            return true
        }
    }
    return false
}

func (s *Sandbox) validateStepPaths(step *Step) error {
    // 检查文件操作路径
    if step.Action == "file.move" || step.Action == "file.copy" {
        source := step.Params["source"].(string)
        dest := step.Params["destination"].(string)

        if !s.isPathAllowed(source) {
            return fmt.Errorf("source path not allowed: %s", source)
        }

        if !s.isPathAllowed(dest) {
            return fmt.Errorf("destination path not allowed: %s", dest)
        }
    }

    // 检查 Shell 命令路径
    if step.Action == "shell.exec" {
        command := step.Params["command"].(string)

        // 提取路径
        if strings.Contains(command, "cd ") {
            parts := strings.Split(command, "cd ")
            if len(parts) > 1 {
                path := strings.TrimSpace(strings.Split(parts[1], " ")[0])
                if !s.isPathAllowed(path) {
                    return fmt.Errorf("path not allowed: %s", path)
                }
            }
        }
    }

    return nil
}

func (s *Sandbox) isPathAllowed(path string) bool {
    // 检查是否在允许的路径下
    for _, allowed := range s.allowedPaths {
        if strings.HasPrefix(path, allowed) {
            return true
        }
    }

    // 检查是否是系统路径
    if strings.HasPrefix(path, "/tmp/") || strings.HasPrefix(path, os.TempDir()) {
        return true
    }

    return false
}

func (s *Sandbox) ExecuteStep(step *Step) (*StepResult, error) {
    result := &StepResult{
        StepID:    step.ID,
        Status:    "running",
        StartTime: time.Now(),
    }

    // 设置超时
    timeout := s.maxDuration
    if duration, ok := step.Params["timeout"].(string); ok {
        if d, err := time.ParseDuration(duration); err == nil {
            timeout = d
        }
    }

    ctx, cancel := context.WithTimeout(context.Background(), timeout)
    defer cancel()

    // 执行动作
    switch step.Action {
    case "shell.exec":
        err := s.executeShell(ctx, step, result)
        if err != nil {
            result.Status = "failed"
            result.Error = err.Error()
        } else {
            result.Status = "success"
        }

    case "git.commit":
        err := s.executeGitCommit(ctx, step, result)
        if err != nil {
            result.Status = "failed"
            result.Error = err.Error()
        } else {
            result.Status = "success"
        }

    case "slack.send":
        err := s.executeSlackSend(ctx, step, result)
        if err != nil {
            result.Status = "failed"
            result.Error = err.Error()
        } else {
            result.Status = "success"
        }

    // ... 其他动作

    default:
        result.Status = "failed"
        result.Error = fmt.Sprintf("unknown action: %s", step.Action)
    }

    result.EndTime = time.Now()

    return result, nil
}
```

### Shell 执行

```go
func (s *Sandbox) executeShell(ctx context.Context, step *Step, result *StepResult) error {
    command := step.Params["command"].(string)
    workingDir := ""

    if dir, ok := step.Params["working_dir"].(string); ok {
        workingDir = dir
    }

    // 创建命令
    cmd := exec.CommandContext(ctx, "sh", "-c", command)

    if workingDir != "" {
        cmd.Dir = workingDir
    }

    // 设置环境变量
    cmd.Env = os.Environ()

    // 执行
    output, err := cmd.CombinedOutput()
    result.Output = string(output)

    if err != nil {
        return err
    }

    return nil
}
```

### Git 操作

```go
func (s *Sandbox) executeGitCommit(ctx context.Context, step *Step, result *StepResult) error {
    path := step.Params["path"].(string)
    message := step.Params["message"].(string)

    // git add
    addCmd := exec.CommandContext(ctx, "git", "-C", path, "add", ".")
    if output, err := addCmd.CombinedOutput(); err != nil {
        result.Output += string(output)
        return fmt.Errorf("git add failed: %w", err)
    }

    // git commit
    commitCmd := exec.CommandContext(ctx, "git", "-C", path, "commit", "-m", message)
    if output, err := commitCmd.CombinedOutput(); err != nil {
        result.Output += string(output)
        return fmt.Errorf("git commit failed: %w", err)
    }

    result.Output += "Committed successfully"
    return nil
}

func (s *Sandbox) executeGitPush(ctx context.Context, step *Step, result *StepResult) error {
    path := step.Params["path"].(string)
    remote := "origin"
    branch := "main"

    if r, ok := step.Params["remote"].(string); ok {
        remote = r
    }

    if b, ok := step.Params["branch"].(string); ok {
        branch = b
    }

    cmd := exec.CommandContext(ctx, "git", "-C", path, "push", remote, branch)
    output, err := cmd.CombinedOutput()
    result.Output = string(output)

    if err != nil {
        return fmt.Errorf("git push failed: %w", err)
    }

    return nil
}
```

### Slack 发送

```go
func (s *Sandbox) executeSlackSend(ctx context.Context, step *Step, result *StepResult) error {
    webhook := step.Params["webhook"].(string)
    channel := step.Params["channel"].(string)
    message := step.Params["message"].(string)

    payload := map[string]interface{}{
        "channel": channel,
        "text":    message,
    }

    body, _ := json.Marshal(payload)

    req, err := http.NewRequestWithContext(ctx, "POST", webhook, bytes.NewReader(body))
    if err != nil {
        return err
    }

    req.Header.Set("Content-Type", "application/json")

    client := &http.Client{Timeout: 30 * time.Second}
    resp, err := client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("Slack API error: %s", resp.Status)
    }

    result.Output = "Message sent to Slack"
    return nil
}
```

---

## 任务调度

### Cron 调度器

```go
// internal/automation/scheduler.go
import "github.com/robfig/cron/v3"

type Scheduler struct {
    cron       *cron.Cron
    engine     *Engine
    scriptRepo ScriptRepository
}

func NewScheduler(engine *Engine, repo ScriptRepository) *Scheduler {
    return &Scheduler{
        cron:       cron.New(cron.WithSeconds()),
        engine:     engine,
        scriptRepo: repo,
    }
}

func (s *Scheduler) Start() error {
    // 加载所有启用的自动化
    scripts, err := s.scriptRepo.FindAllEnabled()
    if err != nil {
        return err
    }

    // 添加到调度器
    for _, script := range scripts {
        if script.Trigger.Type == "cron" {
            s.ScheduleScript(&script)
        }
    }

    s.cron.Start()

    return nil
}

func (s *Scheduler) Stop() {
    s.cron.Stop()
}

func (s *Scheduler) ScheduleScript(script *AutomationScript) (cron.EntryID, error) {
    expr := script.Trigger.Config["expr"].(string)

    id, err := s.cron.AddFunc(expr, func() {
        log.Info("Executing scheduled automation:", script.Name)

        result := s.engine.Execute(script)

        if result.Status == "success" {
            log.Info("Automation completed successfully")
        } else {
            log.Error("Automation failed:", result.Error)
        }

        // 保存执行结果
        s.engine.SaveResult(result)
    })

    if err != nil {
        return 0, err
    }

    // 计算下次运行时间
    entry := s.cron.Entry(id)
    if !entry.Time.IsZero() {
        next := entry.Time
        script.NextRun = &next
        s.scriptRepo.Update(script)
    }

    return id, nil
}

func (s *Scheduler) UnscheduleScript(id cron.EntryID) {
    s.cron.Remove(id)
}
```

### 事件触发

```go
// internal/automation/trigger.go
type EventTrigger struct {
    engine     *Engine
    scriptRepo ScriptRepository
    eventBus   *events.EventBus
}

func NewEventTrigger(engine *Engine, repo ScriptRepository, bus *events.EventBus) *EventTrigger {
    return &EventTrigger{
        engine:     engine,
        scriptRepo: repo,
        eventBus:   bus,
    }
}

func (et *EventTrigger) Start() error {
    // 加载所有事件触发的自动化
    scripts, err := et.scriptRepo.FindByTriggerType("event")
    if err != nil {
        return err
    }

    // 订阅事件
    for _, script := range scripts {
        eventType := script.Trigger.Config["event_type"].(string)

        et.eventBus.Subscribe(eventType, func(event events.Event) {
            // 检查条件
            if et.shouldTrigger(&script, event) {
                log.Info("Event triggered automation:", script.Name)
                result := et.engine.Execute(&script)
                et.engine.SaveResult(result)
            }
        })
    }

    return nil
}

func (et *EventTrigger) shouldTrigger(script *AutomationScript, event events.Event) bool {
    // 检查条件
    if conditions, ok := script.Trigger.Config["conditions"].([]map[string]interface{}); ok {
        for _, condition := range conditions {
            field := condition["field"].(string)
            operator := condition["operator"].(string)
            value := condition["value"]

            // 获取事件字段值
            eventValue := et.getEventField(event, field)

            // 比较
            if !et.compare(eventValue, operator, value) {
                return false
            }
        }
    }

    return true
}

func (et *EventTrigger) getEventField(event events.Event, field string) interface{} {
    switch field {
    case "type":
        return event.Type
    case "application":
        return event.Context.Application
    case "bundle_id":
        return event.Context.BundleID
    default:
        return nil
    }
}

func (et *EventTrigger) compare(actual interface{}, operator string, expected interface{}) bool {
    switch operator {
    case "eq":
        return fmt.Sprintf("%v", actual) == fmt.Sprintf("%v", expected)
    case "neq":
        return fmt.Sprintf("%v", actual) != fmt.Sprintf("%v", expected)
    case "contains":
        return strings.Contains(fmt.Sprintf("%v", actual), fmt.Sprintf("%v", expected))
    default:
        return false
    }
}
```

---

## 执行引擎

```go
// internal/automation/engine.go
type Engine struct {
    sandbox      *Sandbox
    generator    *Generator
    scriptRepo   ScriptRepository
    resultRepo   ResultRepository
    notifier     *notifier.Notifier
}

func NewEngine(config *Config) (*Engine, error) {
    sandbox := NewSandbox(config.Sandbox)

    return &Engine{
        sandbox:    sandbox,
        generator:  NewGenerator(config.AIService),
        scriptRepo: config.ScriptRepo,
        resultRepo: config.ResultRepo,
        notifier:   config.Notifier,
    }, nil
}

func (e *Engine) Execute(script *AutomationScript) *ExecutionResult {
    result := &ExecutionResult{
        ScriptID:  script.ID,
        RunID:     generateRunID(),
        StartTime: time.Now(),
        Status:    "running",
        Steps:     make([]StepResult, 0, len(script.Steps)),
    }

    // 验证脚本
    if err := e.sandbox.ValidateScript(script); err != nil {
        result.Status = "failed"
        result.Error = err.Error()
        result.EndTime = time.Now()
        return result
    }

    // 执行每个步骤
    for _, step := range script.Steps {
        stepResult, err := e.sandbox.ExecuteStep(&step)
        result.Steps = append(result.Steps, *stepResult)

        if err != nil && !step.ContinueOnError {
            result.Status = "failed"
            result.Error = err.Error()
            break
        }
    }

    // 更新状态
    if result.Status == "running" {
        result.Status = "success"
    }

    result.EndTime = time.Now()

    // 更新脚本统计
    script.RunCount++
    now := time.Now()
    script.LastRun = &now
    e.scriptRepo.Update(script)

    // 发送通知
    e.notifier.Notify(script, result)

    return result
}

func (e *Engine) SaveResult(result *ExecutionResult) error {
    return e.resultRepo.Save(result)
}

func (e *Engine) GenerateScript(description string) (*AutomationScript, error) {
    return e.generator.GenerateFromNaturalLanguage(description)
}
```

---

## 使用示例

### 创建自动化

```go
engine, _ := NewEngine(config)

// 生成脚本
script, err := engine.GenerateScript(
    "每天下午 5 点，总结今天的 Git 提交并发到 Slack",
)

if err != nil {
    log.Fatal(err)
}

// 保存脚本
err = engine.scriptRepo.Save(script)
```

### 手动执行

```go
result := engine.Execute(script)

fmt.Printf("Status: %s\n", result.Status)
fmt.Printf("Duration: %v\n", result.EndTime.Sub(result.StartTime))
```

### 调度执行

```go
scheduler := NewScheduler(engine, repo)
scheduler.Start()

// 脚本会自动按 cron 表达式执行
```

---

**相关文档**：
- [系统架构](./01-system-architecture.md)
- [AI 服务](./04-ai-service.md)
- [实施指南 Phase 5](../implementation/06-phase5-automation.md)
