# AI 服务 (AI Service)

AI 服务是 FlowMind 的核心智能组件，集成 Claude API 和 Ollama，提供自然语言理解、脚本生成、内容分析等 AI 能力。

---

## 设计目标

1. **多模型支持**：Claude（复杂任务）+ Ollama（简单任务）
2. **成本优化**：智能路由，最小化 API 调用
3. **响应缓存**：避免重复请求
4. **流式响应**：提升用户体验
5. **提示词管理**：模板化、可复用

---

## 架构设计

```
AI Request
    ↓
┌─────────────────────────────────────┐
│     Request Router                  │  路由器
│  - 简单任务 → Ollama                │
│  - 复杂任务 → Claude                │
└─────────────────────────────────────┘
    ↓
┌─────────────────┬──────────────────┐
│                 │                  │
↓                 ↓                  ↓
Ollama Client   Claude Client      Cache
(本地)          (云端)            (BBolt)
    ↓                 ↓                  ↓
└─────────────────┴──────────────────┘
                    ↓
            Stream Response
```

---

## 核心接口

```go
// internal/ai/service.go
package ai

type AIService struct {
    claude    *ClaudeClient
    ollama    *OllamaClient
    cache     *ResponseCache
    prompter  *PromptEngine
    router    *RequestRouter
}

type Client interface {
    Complete(prompt string, options ...Option) (string, error)
    Stream(prompt string, handler StreamHandler) error
}

type StreamHandler func(chunk string) error

type Option func(*RequestConfig)

type RequestConfig struct {
    Model       string
    Temperature float32
    MaxTokens   int
    System      string
}
```

---

## Claude API 集成

### 客户端实现

```go
// internal/ai/claude.go
type ClaudeClient struct {
    apiKey     string
    baseURL    string
    httpClient *http.Client
    maxRetries int
}

type ClaudeMessage struct {
    Role    string `json:"role"`
    Content string `json:"content"`
}

type ClaudeRequest struct {
    Model     string          `json:"model"`
    Messages  []ClaudeMessage `json:"messages"`
    MaxTokens int             `json:"max_tokens"`
    Temperature float32       `json:"temperature,omitempty"`
    Stream    bool            `json:"stream,omitempty"`
}

type ClaudeResponse struct {
    ID      string `json:"id"`
    Type    string `json:"type"`
    Content []struct {
        Type string `json:"type"`
        Text string `json:"text"`
    } `json:"content"`
    StopReason string `json:"stop_reason"`
}

func NewClaudeClient(apiKey string) *ClaudeClient {
    return &ClaudeClient{
        apiKey:     apiKey,
        baseURL:    "https://api.anthropic.com/v1/messages",
        httpClient: &http.Client{Timeout: 60 * time.Second},
        maxRetries: 3,
    }
}

func (c *ClaudeClient) Complete(prompt string, options ...Option) (string, error) {
    config := &RequestConfig{
        Model:       "claude-3-5-sonnet-20241022",
        MaxTokens:   4096,
        Temperature: 0.7,
    }

    for _, opt := range options {
        opt(config)
    }

    request := ClaudeRequest{
        Model:       config.Model,
        MaxTokens:   config.MaxTokens,
        Temperature: config.Temperature,
        Messages: []ClaudeMessage{
            {Role: "user", Content: prompt},
        },
    }

    // 添加 system message（如果有）
    if config.System != "" {
        // Claude API 使用 system 参数
    }

    body, err := json.Marshal(request)
    if err != nil {
        return "", err
    }

    req, err := http.NewRequest("POST", c.baseURL, bytes.NewReader(body))
    if err != nil {
        return "", err
    }

    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("x-api-key", c.apiKey)
    req.Header.Set("anthropic-version", "2023-06-01")

    resp, err := c.doRequest(req)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return "", fmt.Errorf("Claude API error: %s", resp.Status)
    }

    var response ClaudeResponse
    if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
        return "", err
    }

    // 提取文本内容
    var content strings.Builder
    for _, block := range response.Content {
        if block.Type == "text" {
            content.WriteString(block.Text)
        }
    }

    return content.String(), nil
}

func (c *ClaudeClient) Stream(prompt string, handler StreamHandler) error {
    request := ClaudeRequest{
        Model:       "claude-3-5-sonnet-20241022",
        MaxTokens:   4096,
        Temperature: 0.7,
        Stream:      true,
        Messages: []ClaudeMessage{
            {Role: "user", Content: prompt},
        },
    }

    body, err := json.Marshal(request)
    if err != nil {
        return err
    }

    req, err := http.NewRequest("POST", c.baseURL, bytes.NewReader(body))
    if err != nil {
        return err
    }

    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("x-api-key", c.apiKey)
    req.Header.Set("anthropic-version", "2023-06-01")

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    // 处理 SSE 流
    scanner := bufio.NewScanner(resp.Body)
    for scanner.Scan() {
        line := scanner.Text()

        if strings.HasPrefix(line, "data: ") {
            data := strings.TrimPrefix(line, "data: ")

            if data == "[DONE]" {
                break
            }

            var event struct {
                Type    string `json:"type"`
                Delta   struct {
                    Type string `json:"type"`
                    Text string `json:"text"`
                } `json:"delta"`
            }

            if err := json.Unmarshal([]byte(data), &event); err != nil {
                continue
            }

            if event.Delta.Text != "" {
                if err := handler(event.Delta.Text); err != nil {
                    return err
                }
            }
        }
    }

    return scanner.Err()
}

func (c *ClaudeClient) doRequest(req *http.Request) (*http.Response, error) {
    var err error
    var resp *http.Response

    for i := 0; i < c.maxRetries; i++ {
        resp, err = c.httpClient.Do(req)
        if err == nil && resp.StatusCode < 500 {
            return resp, nil
        }

        if resp != nil {
            resp.Body.Close()
        }

        if i < c.maxRetries-1 {
            time.Sleep(time.Duration(i+1) * time.Second)
        }
    }

    return nil, err
}
```

---

## Ollama 集成

### 客户端实现

```go
// internal/ai/ollama.go
type OllamaClient struct {
    baseURL    string
    model      string
    httpClient *http.Client
}

func NewOllamaClient(baseURL, model string) *OllamaClient {
    if baseURL == "" {
        baseURL = "http://localhost:11434"
    }

    if model == "" {
        model = "llama3.2"
    }

    return &OllamaClient{
        baseURL:    baseURL,
        model:      model,
        httpClient: &http.Client{Timeout: 120 * time.Second},
    }
}

type OllamaRequest struct {
    Model  string `json:"model"`
    Prompt string `json:"prompt"`
    Stream bool   `json:"stream"`
}

type OllamaResponse struct {
    Model     string `json:"model"`
    Response  string `json:"response"`
    Done      bool   `json:"done"`
}

func (o *OllamaClient) Complete(prompt string, options ...Option) (string, error) {
    config := &RequestConfig{
        Model: o.model,
    }

    for _, opt := range options {
        opt(config)
    }

    request := OllamaRequest{
        Model:  config.Model,
        Prompt: prompt,
        Stream: false,
    }

    body, err := json.Marshal(request)
    if err != nil {
        return "", err
    }

    url := fmt.Sprintf("%s/api/generate", o.baseURL)
    resp, err := http.Post(url, "application/json", bytes.NewReader(body))
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return "", fmt.Errorf("Ollama error: %s", resp.Status)
    }

    var response OllamaResponse
    if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
        return "", err
    }

    return response.Response, nil
}

func (o *OllamaClient) Stream(prompt string, handler StreamHandler) error {
    request := OllamaRequest{
        Model:  o.model,
        Prompt: prompt,
        Stream: true,
    }

    body, err := json.Marshal(request)
    if err != nil {
        return err
    }

    url := fmt.Sprintf("%s/api/generate", o.baseURL)
    resp, err := http.Post(url, "application/json", bytes.NewReader(body))
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    decoder := json.NewDecoder(resp.Body)
    for {
        var response OllamaResponse
        if err := decoder.Decode(&response); err != nil {
            if err == io.EOF {
                break
            }
            return err
        }

        if response.Response != "" {
            if err := handler(response.Response); err != nil {
                return err
            }
        }

        if response.Done {
            break
        }
    }

    return nil
}

// Embeddings 用于向量搜索
func (o *OllamaClient) Embed(text string) ([]float32, error) {
    request := map[string]interface{}{
        "model":  "nomic-embed-text",
        "prompt": text,
    }

    body, _ := json.Marshal(request)
    url := fmt.Sprintf("%s/api/embeddings", o.baseURL)
    resp, err := http.Post(url, "application/json", bytes.NewReader(body))
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var result struct {
        Embedding []float32 `json:"embedding"`
    }

    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }

    return result.Embedding, nil
}

// Ping 检查 Ollama 是否可用
func (o *OllamaClient) Ping() error {
    url := fmt.Sprintf("%s/api/tags", o.baseURL)
    resp, err := http.Get(url)
    if err != nil {
        return err
    }
    resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("Ollama not available")
    }

    return nil
}
```

---

## 请求路由

### 智能路由策略

```go
// internal/ai/router.go
type RequestRouter struct {
    simplePatterns []string
    complexPatterns []string
}

func NewRequestRouter() *RequestRouter {
    return &RequestRouter{
        simplePatterns: []string{
            "summarize",
            "classify",
            "extract",
            "tag",
        },
        complexPatterns: []string{
            "generate script",
            "write code",
            "create automation",
            "analyze pattern",
        },
    }
}

func (r *RequestRouter) Route(prompt string) string {
    promptLower := strings.ToLower(prompt)

    // 检查是否需要代码生成
    if r.containsAny(promptLower, r.complexPatterns) {
        return "claude"
    }

    // 检查是否是简单任务
    if r.containsAny(promptLower, r.simplePatterns) {
        return "ollama"
    }

    // 检查提示词长度
    if len(prompt) < 500 {
        return "ollama"
    }

    // 默认使用 Claude
    return "claude"
}

func (r *RequestRouter) containsAny(text string, patterns []string) bool {
    for _, pattern := range patterns {
        if strings.Contains(text, pattern) {
            return true
        }
    }
    return false
}
```

---

## 提示词引擎

### 模板管理

```go
// internal/ai/prompt.go
type PromptEngine struct {
    templates map[string]*PromptTemplate
    cache     map[string]string
}

type PromptTemplate struct {
    Name     string            `json:"name"`
    System   string            `json:"system"`
    User     string            `json:"user"`
    Variables map[string]string `json:"variables"`
}

func NewPromptEngine() *PromptEngine {
    engine := &PromptEngine{
        templates: make(map[string]*PromptTemplate),
        cache:     make(map[string]string),
    }

    // 加载内置模板
    engine.loadBuiltinTemplates()

    return engine
}

func (pe *PromptEngine) loadBuiltinTemplates() {
    // 自动化生成模板
    pe.templates["automation"] = &PromptTemplate{
        Name:   "automation",
        System: "你是一个自动化脚本专家。根据用户需求生成可执行的自动化脚本。",
        User: `用户需求: {{.Description}}

可用能力:
- 文件操作 (监控、移动、重命名)
- Git 操作 (commit, push, PR)
- 发送通知 (Slack, Discord, Email)
- 调用 AI 处理文本
- Shell 命令执行

请生成一个 Go 脚本来实现这个需求。
输出 JSON 格式:
{
  "steps": [
    {"action": "...", "params": {...}}
  ],
  "schedule": "cron expression",
  "description": "..."
}`,
    }

    // 代码生成模板
    pe.templates["code"] = &PromptTemplate{
        Name:   "code",
        System: "你是一个编程助手。根据上下文生成高质量、可维护的代码。",
        User: `当前上下文:
- 应用: {{.Application}}
- 文件: {{.FilePath}}
- 选中文本: {{.Selection}}

用户需求: {{.Request}}

请生成代码，仅输出代码，不要解释。`,
    }

    // 摘要生成模板
    pe.templates["summary"] = &PromptTemplate{
        Name:   "summary",
        System: "你是一个文档助手。为给定内容生成简洁、准确的摘要。",
        User: `请为以下内容生成摘要（不超过 200 字）:

{{.Content}}

摘要:`,
    }
}

func (pe *PromptEngine) Render(templateName string, vars map[string]string) (string, error) {
    template, exists := pe.templates[templateName]
    if !exists {
        return "", fmt.Errorf("template not found: %s", templateName)
    }

    // 检查缓存
    cacheKey := templateName + stringify(vars)
    if cached, exists := pe.cache[cacheKey]; exists {
        return cached, nil
    }

    // 渲染模板
    userPrompt := template.User
    for key, value := range vars {
        placeholder := fmt.Sprintf("{{.%s}}", key)
        userPrompt = strings.ReplaceAll(userPrompt, placeholder, value)
    }

    // 合并 system 和 user
    fullPrompt := template.System + "\n\n" + userPrompt

    // 缓存
    pe.cache[cacheKey] = fullPrompt

    return fullPrompt, nil
}

func stringify(vars map[string]string) string {
    keys := make([]string, 0, len(vars))
    for k := range vars {
        keys = append(keys, k)
    }
    sort.Strings(keys)
    return strings.Join(keys, ",")
}
```

---

## 响应缓存

### BBolt 缓存实现

```go
// internal/ai/cache.go
import "go.etcd.io/bbolt"

type ResponseCache struct {
    db        *bbolt.DB
    bucket    []byte
    ttl       time.Duration
}

func NewResponseCache(dbPath string, ttl time.Duration) (*ResponseCache, error) {
    db, err := bbolt.Open(dbPath, 0600, &bbolt.Options{Timeout: 1 * time.Second})
    if err != nil {
        return nil, err
    }

    // 创建 bucket
    err = db.Update(func(tx *bbolt.Tx) error {
        _, err := tx.CreateBucketIfNotExists([]byte("cache"))
        return err
    })

    if err != nil {
        db.Close()
        return nil, err
    }

    return &ResponseCache{
        db:     db,
        bucket: []byte("cache"),
        ttl:    ttl,
    }, nil
}

func (rc *ResponseCache) Get(key string) (string, bool) {
    var value []byte

    err := rc.db.View(func(tx *bbolt.Tx) error {
        b := tx.Bucket(rc.bucket)
        if b == nil {
            return nil
        }

        cached := b.Get([]byte(key))
        if cached == nil {
            return nil
        }

        // 反序列化
        var entry struct {
            Value   string    `json:"value"`
            Expires time.Time `json:"expires"`
        }

        if err := json.Unmarshal(cached, &entry); err != nil {
            return err
        }

        // 检查过期
        if time.Now().After(entry.Expires) {
            return fmt.Errorf("expired")
        }

        value = []byte(entry.Value)
        return nil
    })

    if err != nil {
        return "", false
    }

    return string(value), true
}

func (rc *ResponseCache) Set(key, value string) error {
    entry := struct {
        Value   string    `json:"value"`
        Expires time.Time `json:"expires"`
    }{
        Value:   value,
        Expires: time.Now().Add(rc.ttl),
    }

    data, err := json.Marshal(entry)
    if err != nil {
        return err
    }

    return rc.db.Update(func(tx *bbolt.Tx) error {
        b := tx.Bucket(rc.bucket)
        return b.Put([]byte(key), data)
    })
}

func (rc *ResponseCache) Delete(key string) error {
    return rc.db.Update(func(tx *bbolt.Tx) error {
        b := tx.Bucket(rc.bucket)
        return b.Delete([]byte(key))
    })
}

func (rc *ResponseCache) Clear() error {
    return rc.db.Update(func(tx *bbolt.Tx) error {
        return tx.DeleteBucket(rc.bucket)
    })
}

func (rc *ResponseCache) Close() error {
    return rc.db.Close()
}

// 生成缓存 key
func cacheKey(prompt string, config RequestConfig) string {
    h := sha256.New()
    h.Write([]byte(prompt))
    h.Write([]byte(config.Model))
    h.Write([]byte(fmt.Sprintf("%f", config.Temperature)))
    return hex.EncodeToString(h.Sum(nil))
}
```

---

## AI 服务集成

```go
// internal/ai/service.go
type AIService struct {
    claude   *ClaudeClient
    ollama   *OllamaClient
    cache    *ResponseCache
    prompter *PromptEngine
    router   *RequestRouter
}

func NewAIService(config *Config) (*AIService, error) {
    // 初始化 Claude
    claude := NewClaudeClient(config.ClaudeAPIKey)

    // 初始化 Ollama
    ollama := NewOllamaClient(config.OllamaURL, config.OllamaModel)

    // 检查 Ollama 可用性
    if err := ollama.Ping(); err != nil {
        log.Warn("Ollama not available, using Claude only")
    }

    // 初始化缓存
    cache, err := NewResponseCache(config.CachePath, 1*time.Hour)
    if err != nil {
        return nil, err
    }

    return &AIService{
        claude:   claude,
        ollama:   ollama,
        cache:    cache,
        prompter: NewPromptEngine(),
        router:   NewRequestRouter(),
    }, nil
}

// Complete 完成请求（自动路由）
func (s *AIService) Complete(prompt string, options ...Option) (string, error) {
    config := &RequestConfig{}
    for _, opt := range options {
        opt(config)
    }

    // 检查缓存
    key := cacheKey(prompt, *config)
    if cached, exists := s.cache.Get(key); exists {
        return cached, nil
    }

    // 路由
    var client Client
    if s.router.Route(prompt) == "ollama" {
        client = s.ollama
    } else {
        client = s.claude
    }

    // 调用
    response, err := client.Complete(prompt, options...)
    if err != nil {
        return "", err
    }

    // 缓存
    s.cache.Set(key, response)

    return response, nil
}

// Stream 流式响应
func (s *AIService) Stream(prompt string, handler StreamHandler, options ...Option) error {
    config := &RequestConfig{}
    for _, opt := range options {
        opt(config)
    }

    // 路由
    var client Client
    if s.router.Route(prompt) == "ollama" {
        client = s.ollama
    } else {
        client = s.claude
    }

    return client.Stream(prompt, handler, options...)
}

// RenderAndApply 使用模板
func (s *AIService) RenderAndApply(templateName string, vars map[string]string, options ...Option) (string, error) {
    prompt, err := s.prompter.Render(templateName, vars)
    if err != nil {
        return "", err
    }

    return s.Complete(prompt, options...)
}

// GenerateAutomation 生成自动化脚本
func (s *AIService) GenerateAutomation(description string) (*AutomationScript, error) {
    prompt, _ := s.prompter.Render("automation", map[string]string{
        "Description": description,
    })

    response, err := s.Complete(prompt)
    if err != nil {
        return nil, err
    }

    // 解析 JSON 响应
    var script AutomationScript
    if err := json.Unmarshal([]byte(response), &script); err != nil {
        return nil, err
    }

    return &script, nil
}

// GenerateCode 生成代码
func (s *AIService) GenerateCode(ctx *EventContext, request string) (string, error) {
    prompt, _ := s.prompter.Render("code", map[string]string{
        "Application": ctx.Application,
        "FilePath":    ctx.FilePath,
        "Selection":   ctx.Selection,
        "Request":     request,
    })

    return s.Complete(prompt)
}

// GenerateSummary 生成摘要
func (s *AIService) GenerateSummary(content string) (string, error) {
    prompt, _ := s.prompter.Render("summary", map[string]string{
        "Content": content,
    })

    return s.Complete(prompt)
}

// Embed 生成向量（用于语义搜索）
func (s *AIService) Embed(text string) ([]float32, error) {
    return s.ollama.Embed(text)
}
```

---

## 使用示例

### 生成自动化

```go
service, _ := NewAIService(config)

script, err := service.GenerateAutomation(
    "每天下午 5 点，总结今天的 Git 提交并发到 Slack",
)

if err != nil {
    log.Fatal(err)
}

fmt.Printf("Generated automation: %+v\n", script)
```

### 代码助手

```go
ctx := &EventContext{
    Application: "VS Code",
    FilePath:    "/Users/user/project/src/App.tsx",
    Selection:   "useEffect(() => {}, [])",
}

code, err := service.GenerateCode(ctx, "添加清理函数")
```

### 流式对话

```go
err := service.Stream("解释 Go 的 goroutine", func(chunk string) error {
    fmt.Print(chunk)
    return nil
})
```

---

**相关文档**：
- [系统架构](./01-system-architecture.md)
- [自动化引擎](./05-automation-engine.md)
- [实施指南 Phase 3](../implementation/04-phase3-assistant.md)
