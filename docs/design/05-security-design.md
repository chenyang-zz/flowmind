# 安全设计

FlowMind 处理敏感用户数据和系统操作，安全性至关重要。

---

## 安全架构

### 威胁模型

```
┌─────────────────────────────────────┐
│          威胁源                      │
│  - 恶意软件                         │
│  - 网络攻击                         │
│  - 物理访问                         │
│  - 内部威胁                         │
└─────────────────────────────────────┘
              │
              ▼
┌─────────────────────────────────────┐
│         防御层                       │
│  1. 访问控制                        │
│  2. 数据加密                        │
│  3. 沙箱隔离                        │
│  4. 审计日志                        │
│  5. 最小权限                        │
└─────────────────────────────────────┘
              │
              ▼
┌─────────────────────────────────────┐
│          保护资产                    │
│  - 用户数据                         │
│  - 系统资源                         │
│  - API 密钥                         │
│  - 自动化脚本                       │
└─────────────────────────────────────┘
```

---

## 1. 权限管理

### macOS 权限

#### 辅助功能权限

```xml
<!-- Info.plist -->
<key>NSAccessibilityUsageDescription</key>
<string>FlowMind 需要辅助功能权限来监听你的操作，以便发现可自动化的模式。</string>
```

#### 完整磁盘访问

```xml
<key>NSDocumentsFolderUsageDescription</key>
<string>FlowMind 需要访问文档文件夹以支持知识管理和文件自动化。</string>

<key>NSDownloadsFolderUsageDescription</key>
<string>FlowMind 需要访问下载文件夹以支持文件自动整理。</string>
```

#### 通知权限

```xml
<key>NSUserNotificationUsageDescription</key>
<string>FlowMind 需要通知权限以推送模式发现和自动化执行结果。</string>
```

### 权限检查

```go
// internal/permissions/check.go
import "github.com/go-vgo/robotgo"

func CheckAccessibilityPermission() bool {
    // macOS 检查
    trusted := robotgo.HasAccessibility()

    if !trusted {
        promptUserForPermission()
    }

    return trusted
}

func promptUserForPermission() {
    alert := &dialogs.Alert{
        Title:   "需要辅助功能权限",
        Message: "FlowMind 需要辅助功能权限来监听你的操作。\n\n" +
                  "请前往 系统设置 > 隐私与安全性 > 辅助功能，\n" +
                  "勾选 FlowMind。",
        Buttons: []string{"打开系统设置", "稍后"},
    }

    if alert.Show() == 0 {
        // 打开系统设置
        exec.Command("open", "x-apple.systempreferences:com.apple.preference.security?Privacy_Accessibility").Run()
    }
}
```

---

## 2. 沙箱隔离

### 执行沙箱

```go
// internal/automation/sandbox.go
type Sandbox struct {
    allowedPaths   []string
    allowedActions []string
    maxMemory      int64
    maxDuration    time.Duration
    networkAccess  bool
}

func (s *Sandbox) ValidateScript(script *AutomationScript) error {
    // 检查动作白名单
    for _, step := range script.Steps {
        if !s.isActionAllowed(step.Action) {
            return fmt.Errorf("action not allowed: %s", step.Action)
        }

        // 检查路径白名单
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
```

### 资源限制

```go
func (s *Sandbox) ExecuteStep(step *Step) error {
    // 超时限制
    ctx, cancel := context.WithTimeout(context.Background(), s.maxDuration)
    defer cancel()

    // 内存限制（通过 cgroups 或 ulimit）
    cmd := exec.CommandContext(ctx, "sh", "-c", step.Params["command"].(string))

    // 限制进程
    cmd.SysProcAttr = &syscall.SysProcAttr{
        Setpgid: true, // 创建新进程组
    }

    // 执行
    output, err := cmd.CombinedOutput()

    if err != nil {
        return fmt.Errorf("execution failed: %w, output: %s", err, output)
    }

    return nil
}
```

---

## 3. 数据加密

### API 密钥存储

```go
// internal/security/keychain.go
import "github.com/keybase/go-keychain"

type KeychainStorage struct {
    service string
}

func NewKeychainStorage() *KeychainStorage {
    return &KeychainStorage{
        service: "com.flowmind.app",
    }
}

func (k *KeychainStorage) SetAPIKey(provider, key string) error {
    item := keychain.NewItem()
    item.SetSecClass(keychain.SecClassGenericPassword)
    item.SetService(k.service)
    item.SetAccount(provider)
    item.SetData([]byte(key))
    item.SetSynchronizable(keychain.SynchronizableNo)
    item.SetAccessible(keychain.AccessibleWhenUnlocked)

    return keychain.AddItem(item)
}

func (k *KeychainStorage) GetAPIKey(provider string) (string, error) {
    query := keychain.NewItem()
    query.SetSecClass(keychain.SecClassGenericPassword)
    query.SetService(k.service)
    query.SetAccount(provider)

    results, err := keychain.QueryItem(query)
    if err != nil {
        return "", err
    }

    return string(results[0].Data), nil
}

// 使用
storage := NewKeychainStorage()
storage.SetAPIKey("claude", "sk-ant-...")
key, _ := storage.GetAPIKey("claude")
```

### 数据库加密

```go
// 使用 SQLCipher 加密 SQLite
func openEncryptedDB(path, key string) (*sql.DB, error) {
    // PRAGMA key = 'your-key'
    // PRAGMA cipher_memory_security = ON

    db, err := sql.Open("sqlite3", path)
    if err != nil {
        return nil, err
    }

    // 设置加密密钥
    if _, err := db.Exec("PRAGMA key = '" + key + "'"); err != nil {
        return nil, err
    }

    return db, nil
}
```

---

## 4. 敏感信息过滤

### 剪贴板过滤

```go
// internal/monitor/filter.go
type SensitiveFilter struct {
    patterns []string
}

func (f *SensitiveFilter) ShouldRecord(content string) bool {
    // 检查敏感模式
    for _, pattern := range f.patterns {
        if matched, _ := regexp.MatchString(pattern, content); matched {
            log.Info("Filtered sensitive content")
            return false
        }
    }

    // 检查内容长度
    if len(content) > 10000 {
        log.Info("Filtered oversized content")
        return false
    }

    return true
}

// 默认敏感模式
var defaultPatterns = []string{
    `(?i)password\s*[:=]\s*\S+`,
    `(?i)api[_-]?key\s*[:=]\s*\S+`,
    `(?i)token\s*[:=]\s*\S+`,
    `(?i)secret\s*[:=]\s*\S+`,
    `\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`, // Email
    `\b\d{4}[-\s]?\d{4}[-\s]?\d{4}[-\s]?\d{4}\b`,           // 信用卡号
}
```

### 应用过滤

```go
var ignoredApps = []string{
    "1Password",
    "Keychain Access",
    "Bitwarden",
    "LastPass",
}

func (f *SensitiveFilter) ShouldMonitorApp(appName string) bool {
    for _, ignored := range ignoredApps {
        if appName == ignored {
            return false
        }
    }
    return true
}
```

---

## 5. 审计日志

### 日志记录

```go
// internal/security/audit.go
type AuditLogger struct {
    db *sql.DB
}

type AuditEvent struct {
    Timestamp   time.Time              `json:"timestamp"`
    Actor       string                 `json:"actor"`        // user, system, automation
    Action      string                 `json:"action"`       // create, delete, execute
    Resource    string                 `json:"resource"`     // automation, pattern, knowledge
    ResourceID  string                 `json:"resource_id"`
    Status      string                 `json:"status"`       // success, failure
    Details     map[string]interface{} `json:"details"`
}

func (a *AuditLogger) Log(event AuditEvent) error {
    query := `
        INSERT INTO audit_log
        (timestamp, actor, action, resource, resource_id, status, details)
        VALUES (?, ?, ?, ?, ?, ?, ?)
    `

    details, _ := json.Marshal(event.Details)

    _, err := a.db.Exec(query,
        event.Timestamp,
        event.Actor,
        event.Action,
        event.Resource,
        event.ResourceID,
        event.Status,
        details,
    )

    return err
}

// 使用
auditLog.Log(AuditEvent{
    Timestamp: time.Now(),
    Actor:     "automation",
    Action:    "execute",
    Resource:  "automation",
    ResourceID: script.ID,
    Status:    "success",
    Details: map[string]interface{}{
        "duration": result.Duration,
        "steps": len(result.Steps),
    },
})
```

### 日志查询

```go
func (a *AuditLogger) Query(filter AuditFilter) ([]AuditEvent, error) {
    query := `
        SELECT timestamp, actor, action, resource, resource_id, status, details
        FROM audit_log
        WHERE 1=1
    `

    args := []interface{}{}

    if filter.Actor != "" {
        query += " AND actor = ?"
        args = append(args, filter.Actor)
    }

    if filter.Action != "" {
        query += " AND action = ?"
        args = append(args, filter.Action)
    }

    if filter.StartTime.IsZero() {
        query += " AND timestamp >= ?"
        args = append(args, filter.StartTime)
    }

    query += " ORDER BY timestamp DESC LIMIT ?"
    args = append(args, filter.Limit)

    rows, err := a.db.Query(query, args...)
    // ...
}
```

---

## 6. 用户确认

### 首次执行确认

```go
func (e *Engine) confirmFirstExecution(script *AutomationScript) bool {
    // 显示确认对话框
    message := fmt.Sprintf(`即将执行自动化脚本: %s

包含以下步骤:
`, script.Name)

    for i, step := range script.Steps {
        message += fmt.Sprintf("%d. %s\n", i+1, step.Action)
    }

    message += "\n是否继续?"

    dialog := &dialogs.MessageDialog{
        Title:   "确认执行",
        Message: message,
        Buttons: []string{"取消", "执行"},
    }

    return dialog.Show() == 1
}
```

### 危险操作确认

```go
type DangerousAction struct {
    Action       string
    Reason       string
    AlwaysPrompt bool
}

var dangerousActions = []DangerousAction{
    {"shell.exec", "执行 Shell 命令", true},
    {"file.delete", "删除文件", true},
    {"git.push", "推送到远程仓库", false},
}

func (s *Sandbox) isDangerous(step Step) bool {
    for _, action := range dangerousActions {
        if step.Action == action.Action && action.AlwaysPrompt {
            return true
        }
    }
    return false
}
```

---

## 7. 网络安全

### HTTPS 证书验证

```go
func NewHTTPClient() *http.Client {
    return &http.Client{
        Transport: &http.Transport{
            TLSClientConfig: &tls.Config{
                MinVersion: tls.VersionTLS12,
            },
        },
        Timeout: 30 * time.Second,
    }
}
```

### Webhook 验证

```go
func (s *SlackStep) validateWebhook(url string) error {
    // 检查 URL 格式
    if !strings.HasPrefix(url, "https://hooks.slack.com/") {
        return fmt.Errorf("invalid Slack webhook URL")
    }

    // 测试连接
    resp, err := http.Post(url, "application/json", strings.NewReader(`{"text":"test"}`))
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("webhook validation failed")
    }

    return nil
}
```

---

## 8. 隐私保护

### 数据最小化

```go
// 仅记录必要信息
type Event struct {
    Type      string
    Timestamp time.Time
    // 不记录具体内容，仅记录类型
    // 不记录剪贴板完整内容，仅记录长度和类型
}
```

### 用户控制

```go
type PrivacySettings struct {
    MonitorKeyboard    bool
    MonitorClipboard   bool
    MonitorAppSwitch   bool
    RecordScreenContent bool
    RetentionPolicy    string // 7d, 30d, 90d
    AllowAnalytics     bool
}

func (p *PrivacySettings) ApplyToConfig(config *Config) {
    config.Monitor.Keyboard = p.MonitorKeyboard
    config.Monitor.Clipboard = p.MonitorClipboard
    config.Events.Retention.Events = p.RetentionPolicy
}
```

### 数据删除

```go
func (s *Storage) DeleteUserData() error {
    // 删除所有用户数据
    tables := []string{
        "events",
        "patterns",
        "knowledge_items",
        "automations",
        "execution_results",
    }

    for _, table := range tables {
        if _, err := s.db.Exec(fmt.Sprintf("DELETE FROM %s", table)); err != nil {
            return err
        }
    }

    // 清理向量数据库
    s.vectorStore.Clear()

    return nil
}
```

---

## 9. 安全最佳实践

### 代码审查清单

- [ ] 所有用户输入都经过验证
- [ ] 敏感数据不记录在日志中
- [ ] API 密钥使用 Keychain 存储
- [ ] 沙箱执行所有自动化脚本
- [ ] 危险操作需要用户确认
- [ ] 使用 HTTPS 进行网络通信
- [ ] 定期更新依赖库
- [ ] 最小权限原则
- [ ] 完整的审计日志
- [ ] 用户可导出/删除数据

### 定期安全审计

```go
// internal/security/auditor.go
func (a *Auditor) Audit() *SecurityReport {
    report := &SecurityReport{}

    // 检查权限
    report.AccessibilityGranted = CheckAccessibilityPermission()

    // 检查数据保留
    report.EventRetention = a.getEventRetention()

    // 检查自动化脚本
    report.AutomationCount = a.countAutomations()
    report.DangerousAutomations = a.findDangerousAutomations()

    // 检查敏感数据暴露
    report.SensitiveDataInLogs = a.scanLogsForSensitiveData()

    return report
}
```

---

**相关文档**：
- [配置系统](./04-config-system.md)
- [数据库设计](./01-database-design.md)
- [实施指南 Phase 7](../implementation/08-phase7-optimization.md)
