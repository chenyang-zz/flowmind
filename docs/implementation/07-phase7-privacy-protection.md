# Phase 7: 隐私保护（待实施）

**目标**: 为剪贴板监控添加全面的隐私保护机制

**实施状态**: ⚠️ 待实施

**预计时间**: 9-14 天（分3个阶段）

**实施时机**: 在 Phase 1-6 完成后实施

**优先级**: 🟡 中（重要但不紧急）

---

## 📋 概述

### 背景

剪贴板监控功能在 Phase 1 实现过程中**未充分考虑隐私保护**，存在以下安全隐患：

- ⚠️ 完整的剪贴板内容被发送到事件总线
- ⚠️ 没有敏感内容检测机制
- ⚠️ 没有用户同意机制
- ⚠️ 可能记录密码、令牌等敏感信息

### 核心价值

1. **隐私优先**: 默认设置保护用户隐私
2. **用户控制**: 让用户决定数据如何使用
3. **透明公开**: 清晰的隐私政策和使用说明
4. **安全合规**: 遵守相关法律法规（GDPR、个人信息保护法）

### 飐私风险分析

**剪贴板可能包含的敏感信息**：

| 信息类型 | 示例 | 风险等级 |
|---------|------|---------|
| 认证信息 | 密码、API密钥、令牌 | 🔴 极高 |
| 个人身份 | 身份证号、银行卡号、社保号 | 🔴 极高 |
| 私密通信 | 聊天记录、邮件内容 | 🟠 高 |
| 商业机密 | 代码片段、设计文档、合同 | 🟠 高 |
| 健康信息 | 医疗记录、诊断结果 | 🟠 高 |
| 位置信息 | 地址、坐标 | 🟡 中 |
| 普通内容 | 网页链接、普通文本 | 🟢 低 |

---

## 🚨 当前存在的问题

### 问题 1：完整内容被发送到事件总线

**当前实现** (`internal/monitor/clipboard.go`):

```go
// ❌ 完整的剪贴板内容被记录
data := map[string]interface{}{
    "content": event.Content,  // 完整内容！
    "type":    event.Type,
    "size":    event.Size,
    "length":  len(event.Content),
}
```

**风险**：
- 密码、API密钥会被记录到事件总线
- 事件总线可能被多个订阅者访问
- 数据可能被持久化存储
- 数据可能被发送到远程服务器

### 问题 2：没有敏感内容检测

**当前实现**：
- 无论什么内容都会被记录
- 没有检测密码、令牌等敏感信息
- 没有应用级别的过滤

**示例风险场景**：
```
用户在 1Password 中复制密码
    ↓
剪贴板监控器捕获
    ↓
完整密码被发送到事件总线
    ↓
可能被日志记录、存储或传输
```

### 问题 3：没有用户同意机制

**当前实现**：
- 用户不知道剪贴板被监控
- 无法选择是否记录内容
- 缺少隐私政策说明

**合规风险**：
- 违反 GDPR 的"明确同意"原则
- 违反个人信息保护法的"明示同意"要求

### 问题 4：日志中记录内容预览

**当前实现**：
```go
// 日志中记录前100个字符
contentPreview := event.Content
if len(contentPreview) > 100 {
    contentPreview = contentPreview[:100] + "..."
}
logger.Info("检测到剪贴板内容变化", zap.String("preview", contentPreview))
```

**风险**：即使只有100字符，也可能包含完整的密码或令牌。

---

## 🛡️ 隐私保护方案

### 多层保护架构

```
┌─────────────────────────────────────────────────────┐
│                  用户界面层                          │
│  - 隐私设置面板                                       │
│  - 敏感应用黑名单                                     │
│  - 内容过滤规则                                       │
│  - 用户同意界面                                       │
└─────────────────┬───────────────────────────────────┘
                  │
┌─────────────────▼───────────────────────────────────┐
│                  业务逻辑层                          │
│  - 内容检测引擎                                       │
│  - 隐私策略执行                                       │
│  - 用户偏好管理                                       │
└─────────────────┬───────────────────────────────────┘
                  │
┌─────────────────▼───────────────────────────────────┐
│                  数据处理层                          │
│  - 脱敏/加密                                          │
│  - 内容哈希                                          │
│  - 元数据提取                                        │
└─────────────────┬───────────────────────────────────┘
                  │
┌─────────────────▼───────────────────────────────────┐
│                  存储传输层                          │
│  - 安全日志                                          │
│  - 加密存储                                          │
│  - 安全传输                                          │
└─────────────────────────────────────────────────────┘
```

### 默认安全策略

**原则：隐私优先，默认安全**

| 设置项 | 默认值 | 说明 |
|--------|--------|------|
| 记录完整内容 | ❌ 否 | 默认只记录元数据 |
| 记录内容哈希 | ✅ 是 | 用于去重和内容匹配 |
| 内容最大长度 | 0 字符 | 不记录内容 |
| 日志级别 | Warn | 只记录警告和错误 |
| 敏感应用过滤 | ✅ 启用 | 自动过滤密码管理器等 |
| 用户明确同意 | ✅ 必须 | 首次使用需用户同意 |

---

## 📅 实施计划

> **说明**：本方案将在 Phase 1-6 完成后实施。当前版本中的剪贴板监控功能仅供开发和测试使用，不建议用于生产环境。

### 阶段 1：紧急修复（1-2天）🟡 重要

#### 目标
修复最严重的隐私安全隐患，确保默认行为是安全的。

#### 任务清单

**1. 修改剪贴板监控器 - 默认不记录完整内容**

文件: `internal/monitor/clipboard.go`

```go
// handlePlatformEvent 处理平台层传来的剪贴板变化事件
func (cm *ClipboardMonitor) handlePlatformEvent(event platform.ClipboardEvent, config *PrivacyConfig) {
    // 1. 检查用户同意
    if config.ConsentRequired && !config.ConsentGiven {
        logger.Debug("用户未同意剪贴板监控，跳过")
        return
    }

    // 2. 检查应用黑名单
    if config.IsAppBlacklisted(event.Context.BundleID) {
        logger.Debug("跳过敏感应用的剪贴板事件",
            zap.String("app", event.Context.Application))
        return
    }

    // 3. 检测敏感内容
    if config.EnableContentFilter && cm.isSensitiveContent(event.Content) {
        logger.Warn("检测到敏感剪贴板内容，已过滤")
        // 只记录元数据
        cm.createMetadataOnlyEvent(event)
        return
    }

    // 4. 根据配置处理内容
    var processedContent interface{}

    switch {
    case config.ContentHashOnly:
        // 只记录哈希值
        hash := sha256.Sum256([]byte(event.Content))
        processedContent = fmt.Sprintf("sha256:%x", hash)

    case config.MaxContentLength > 0 && len(event.Content) > config.MaxContentLength:
        // 截断内容
        processedContent = event.Content[:config.MaxContentLength] + "... [truncated]"

    case config.RecordContent:
        // 记录完整内容（可能需要加密）
        if config.EnableEncryption {
            encrypted, err := cm.encryptContent(event.Content, config.EncryptionKey)
            if err != nil {
                logger.Error("加密内容失败", zap.Error(err))
                return
            }
            processedContent = encrypted
        } else {
            processedContent = event.Content
        }

    default:
        // 只记录元数据
        cm.createMetadataOnlyEvent(event)
        return
    }

    // 5. 创建事件
    data := map[string]interface{}{
        "content":     processedContent,
        "type":        event.Type,
        "size":        event.Size,
        "length":      len(event.Content),
        "is_filtered": !config.RecordContent,
    }

    businessEvent := events.NewEvent(events.EventTypeClipboard, data)
    businessEvent.WithContext(event.Context)
    cm.eventBus.Publish(string(events.EventTypeClipboard), *businessEvent)
}

// createMetadataOnlyEvent 创建只包含元数据的事件
func (cm *ClipboardMonitor) createMetadataOnlyEvent(event platform.ClipboardEvent) *events.Event {
    data := map[string]interface{}{
        "content":        nil, // 不记录内容
        "type":           event.Type,
        "size":           event.Size,
        "length":         len(event.Content),
        "is_filtered":    true,
        "filter_reason":  "privacy_protection",
        "content_hash":   cm.contentHash(event.Content),
    }

    businessEvent := events.NewEvent(events.EventTypeClipboard, data)
    businessEvent.WithContext(event.Context)

    return businessEvent
}
```

**2. 添加敏感应用黑名单**

文件: `internal/monitor/privacy_config.go` (新建)

```go
package monitor

import (
    "regexp"
    "sync"
)

// PrivacyConfig 隐私配置
type PrivacyConfig struct {
    mu sync.RWMutex

    // 内容记录策略
    RecordContent    bool   // 是否记录完整内容
    MaxContentLength int    // 最大内容长度
    ContentHashOnly  bool   // 仅记录哈希值

    // 敏感内容检测
    EnableContentFilter bool     // 启用内容过滤
    SensitivePatterns  []string  // 敏感内容正则表达式
    BlacklistedApps    []string  // 应用黑名单

    // 数据处理
    EnableEncryption bool   // 启用加密
    EncryptionKey    string // 加密密钥（从密钥库获取）

    // 用户同意
    ConsentRequired  bool   // 是否需要用户同意
    ConsentGiven     bool   // 用户是否已同意
    ConsentTimestamp int64  // 同意时间戳

    // 编译的敏感模式
    compiledPatterns []*regexp.Regexp
}

// DefaultPrivacyConfig 返回默认的安全配置
func DefaultPrivacyConfig() *PrivacyConfig {
    return &PrivacyConfig{
        RecordContent:       false,         // 默认不记录内容
        MaxContentLength:    0,             // 不记录内容
        ContentHashOnly:     true,          // 仅记录哈希
        EnableContentFilter: true,          // 启用过滤
        BlacklistedApps: []string{
            "com.agilebits.onepassword-osx-helper",  // 1Password
            "com.bitwarden.desktop",                 // Bitwarden
            "com.lastpass.lastpassdesktop",          // LastPass
            "com.keepassium.KeePassXC",              // KeePassXC
            "com.github.GitHub",                     // GitHub (可能复制token)
            "com.microsoft.VSCode",                   // VSCode
        },
        SensitivePatterns: []string{
            // 密码相关
            `(?i)password\s*[:=]\s*\S+`,
            `(?i)passwd\s*[:=]\s*\S+`,
            `(?i)api[_-]?key\s*[:=]\s*\S+`,
            `(?i)token\s*[:=]\s*\S+`,
            `(?i)secret\s*[:=]\s*\S+`,

            // 个人身份信息
            `\d{15,19}`,           // 银行卡号
            `\d{17}[\dXx]`,        // 身份证号
            `\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`, // 邮箱

            // 令牌和密钥
            `Bearer\s+[A-Za-z0-9\-._~+/]+=*`,
            `sk-[a-zA-Z0-9]{32,}`,   // OpenAI API Key
            `ghp_[a-zA-Z0-9]{36}`,    // GitHub Token
        },
        EnableEncryption:  false,
        ConsentRequired: true,
        ConsentGiven:    false,
    }
}

// IsAppBlacklisted 检查应用是否在黑名单中
func (p *PrivacyConfig) IsAppBlacklisted(bundleID string) bool {
    p.mu.RLock()
    defer p.mu.RUnlock()

    for _, blacklisted := range p.BlacklistedApps {
        if bundleID == blacklisted {
            return true
        }
    }
    return false
}

// isSensitiveContent 检测内容是否敏感
func (p *PrivacyConfig) isSensitiveContent(content string) bool {
    p.mu.RLock()
    defer p.mu.RUnlock()

    // 检查编译的正则表达式
    for _, pattern := range p.compiledPatterns {
        if pattern.MatchString(content) {
            return true
        }
    }
    return false
}

// CompilePatterns 编译敏感内容模式
func (p *PrivacyConfig) CompilePatterns() error {
    p.mu.Lock()
    defer p.mu.Unlock()

    p.compiledPatterns = make([]*regexp.Regexp, 0, len(p.SensitivePatterns))

    for _, pattern := range p.SensitivePatterns {
        compiled, err := regexp.Compile(pattern)
        if err != nil {
            return err
        }
        p.compiledPatterns = append(p.compiledPatterns, compiled)
    }

    return nil
}
```

**3. 添加用户同意检查**

文件: `internal/monitor/clipboard.go`

```go
// NewClipboardMonitor 创建剪贴板监控器
func NewClipboardMonitor(eventBus *events.EventBus) Monitor {
    config := DefaultPrivacyConfig()
    config.CompilePatterns() // 编译敏感内容模式

    return &ClipboardMonitor{
        platform:    platform.NewClipboardMonitor(),
        eventBus:    eventBus,
        contextMgr:  platform.NewContextProvider(),
        privacy:     config, // 添加隐私配置
        isRunning:   false,
        mu:          sync.RWMutex{},
    }
}
```

**4. 移除日志中的内容预览**

```go
// ❌ 移除或修改为不记录内容
// logger.Info("检测到剪贴板内容变化",
//     zap.String("preview", contentPreview))

// ✅ 改为只记录元数据
logger.Info("检测到剪贴板内容变化",
    zap.String("type", event.Type),
    zap.Int64("size", event.Size),
    zap.String("app", event.Context.Application),
)
```

#### 验证标准

- [ ] 默认情况下不记录完整内容到事件总线
- [ ] 敏感应用（1Password等）被自动过滤
- [ ] 密码、令牌等敏感内容被检测并过滤
- [ ] 用户首次使用时显示同意界面
- [ ] 日志中不记录剪贴板内容预览

---

### 阶段 2：核心功能（3-5天）🟠 重要

#### 目标
实现完整的隐私保护功能，包括配置系统、加密和审计。

#### 任务清单

**1. 实现配置持久化**

```go
// LoadPrivacyConfig 从文件加载隐私配置
func LoadPrivacyConfig(path string) (*PrivacyConfig, error) {
    config := DefaultPrivacyConfig()

    // 从文件读取用户配置
    data, err := os.ReadFile(path)
    if err != nil {
        if os.IsNotExist(err) {
            // 文件不存在，使用默认配置
            return config, nil
        }
        return nil, err
    }

    // 解析配置
    if err := json.Unmarshal(data, config); err != nil {
        return nil, err
    }

    // 编译正则表达式
    if err := config.CompilePatterns(); err != nil {
        return nil, err
    }

    return config, nil
}

// SavePrivacyConfig 保存隐私配置
func (p *PrivacyConfig) Save(path string) error {
    p.mu.RLock()
    defer p.mu.RUnlock()

    data, err := json.MarshalIndent(p, "", "  ")
    if err != nil {
        return err
    }

    return os.WriteFile(path, data, 0600)
}
```

**2. 实现数据加密**

```go
// encryptContent 加密内容（使用 AES-256-GCM）
func (cm *ClipboardMonitor) encryptContent(content, key string) (string, error) {
    block, err := aes.NewCipher([]byte(key))
    if err != nil {
        return "", err
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return "", err
    }

    nonce := make([]byte, gcm.NonceSize())
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return "", err
    }

    ciphertext := gcm.Seal(nonce, nonce, []byte(content), nil)
    return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decryptContent 解密内容
func (cm *ClipboardMonitor) decryptContent(ciphertext, key string) (string, error) {
    data, err := base64.StdEncoding.DecodeString(ciphertext)
    if err != nil {
        return "", err
    }

    block, err := aes.NewCipher([]byte(key))
    if err != nil {
        return "", err
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return "", err
    }

    nonceSize := gcm.NonceSize()
    if len(data) < nonceSize {
        return "", errors.New("ciphertext too short")
    }

    nonce, cipherData := data[:nonceSize], data[nonceSize:]
    plaintext, err := gcm.Open(nil, nonce, cipherData, nil)
    if err != nil {
        return "", err
    }

    return string(plaintext), nil
}
```

**3. 实现审计日志**

```go
// PrivacyAuditLog 隐私审计日志
type PrivacyAuditLog struct {
    Timestamp     time.Time `json:"timestamp"`
    EventType     string    `json:"event_type"`
    Reason        string    `json:"reason"`
    AppName       string    `json:"app_name"`
    BundleID      string    `json:"bundle_id"`
    ContentHash   string    `json:"content_hash"`
    ContentLength int       `json:"content_length"`
    UserConsent   bool      `json:"user_consent"`
}

// LogFilteredEvent 记录过滤事件
func (cm *ClipboardMonitor) LogFilteredEvent(
    event platform.ClipboardEvent,
    reason string,
) {
    auditLog := PrivacyAuditLog{
        Timestamp:     time.Now(),
        EventType:     "clipboard_filtered",
        Reason:        reason,
        AppName:       event.Context.Application,
        BundleID:      event.Context.BundleID,
        ContentHash:   cm.contentHash(event.Content),
        ContentLength: len(event.Content),
        UserConsent:   cm.privacy.ConsentGiven,
    }

    // 写入审计日志文件
    data, _ := json.Marshal(auditLog)
    logger.Info("剪贴板事件已过滤",
        zap.String("reason", reason),
        zap.String("app", event.Context.Application),
        zap.String("content_hash", auditLog.ContentHash),
    )
}
```

**4. 实现隐私指标统计**

```go
// PrivacyMetrics 隐私指标
type PrivacyMetrics struct {
    mu                sync.Mutex
    TotalEvents       int64
    FilteredEvents    int64
    BlacklistedApps    map[string]int64
    ConsentRate       float64
}

// RecordEvent 记录事件
func (m *PrivacyMetrics) RecordEvent(filtered bool, app string) {
    m.mu.Lock()
    defer m.mu.Unlock()

    m.TotalEvents++

    if filtered {
        m.FilteredEvents++
    }

    if m.BlacklistedApps == nil {
        m.BlacklistedApps = make(map[string]int64)
    }
    m.BlacklistedApps[app]++
}

// GetFilterRate 获取过滤率
func (m *PrivacyMetrics) GetFilterRate() float64 {
    m.mu.Lock()
    defer m.mu.Unlock()

    if m.TotalEvents == 0 {
        return 0
    }
    return float64(m.FilteredEvents) / float64(m.TotalEvents)
}
```

#### 验证标准

- [ ] 配置可以保存和加载
- [ ] 敏感内容可以被加密存储
- [ ] 过滤事件被记录到审计日志
- [ ] 隐私指标统计正常工作

---

### 阶段 3：增强保护（5-7天）🟡 推荐

#### 目标
提供用户界面和高级隐私保护功能。

#### 任务清单

**1. 实现用户同意界面**

```go
// ConsentDialog 用户同意对话框
type ConsentDialog struct {
    config *PrivacyConfig
}

// Show 显示同意对话框
func (d *ConsentDialog) Show() (bool, error) {
    // 在 GUI 中显示同意界面
    // 等待用户选择
    return true, nil
}

// ShowSettings 显示隐私设置界面
func (d *ConsentDialog) ShowSettings() error {
    // 显示设置面板
    return nil
}
```

**2. 实现机器学习检测**

```go
// SensitiveContentML 基于机器学习的敏感内容检测
type SensitiveContentML struct {
    model *Model // 预训练模型
}

// Predict 预测内容是否敏感
func (s *SensitiveContentML) Predict(content string) (bool, float64) {
    // 使用 ML 模型预测
    return false, 0.0
}
```

**3. 实现隐私风险评估**

```go
// PrivacyRiskReport 隐私风险评估报告
type PrivacyRiskReport struct {
    GeneratedAt    time.Time `json:"generated_at"`
    TotalEvents    int64     `json:"total_events"`
    FilteredEvents int64     `json:"filtered_events"`
    RiskScore      float64   `json:"risk_score"`
    Recommendations []string  `json:"recommendations"`
}

// GenerateReport 生成风险评估报告
func (cm *ClipboardMonitor) GenerateReport() (*PrivacyRiskReport, error) {
    // 分析历史数据
    // 生成风险评估
    // 提供改进建议
    return nil, nil
}
```

#### 验证标准

- [ ] 用户同意界面正常显示
- [ ] 隐私设置可以保存和应用
- [ ] 机器学习检测准确率 >85%
- [ ] 隐私报告可以正常生成

---

## 🎨 用户界面设计

### 首次启动同意界面

```
┌─────────────────────────────────────────────────┐
│  📋 剪贴板监控权限请求                            │
├─────────────────────────────────────────────────┤
│                                                 │
│  FlowMind 需要监控您的剪贴板内容变化，以提供     │
│  以下功能：                                      │
│                                                 │
│  ✓ 自动记录复制历史                              │
│  ✓ 智能工作流分析                                │
│  ✓ 跨设备剪贴板同步                              │
│                                                 │
│  隐私保护承诺：                                  │
│  • 默认不记录完整内容，只记录元数据               │
│  • 自动过滤敏感应用（密码管理器等）               │
│  • 数据加密存储在本地                            │
│  • 您可以随时在设置中更改隐私选项                │
│                                                 │
│  [查看完整隐私政策]                              │
│                                                 │
│  □ 我已阅读并同意剪贴板监控使用条款               │
│                                                 │
│         [拒绝]  [自定义设置]  [同意并继续]       │
│                                                 │
└─────────────────────────────────────────────────┘
```

### 隐私设置面板

```
┌─────────────────────────────────────────────────┐
│  🔒 隐私设置                                     │
├─────────────────────────────────────────────────┤
│                                                 │
│  内容记录                                        │
│  ┌───────────────────────────────────────────┐  │
│  │ ○ 不记录内容（推荐）                       │  │
│  │   只记录类型、大小、时间等元数据             │  │
│  │                                             │  │
│  │ ○ 仅记录内容哈希                           │  │
│  │   用于去重和内容匹配，不可逆                │  │
│  │                                             │  │
│  │ ● 记录完整内容                             │  │
│  │   最大长度: [100] 字符                      │  │
│  │                                             │  │
│  │   □ 启用加密存储                           │  │
│  └───────────────────────────────────────────┘  │
│                                                 │
│  敏感内容过滤                                    │
│  ┌───────────────────────────────────────────┐  │
│  │ ✅ 启用自动过滤                             │  │
│  │                                             │  │
│  │ 高风险应用（自动跳过）:                      │  │
│  │ ☑ 1Password  ☑ Bitwarden  ☑ LastPass      │  │
│  │ ☑ Signal     ☑ WeChat      ☑ WhatsApp    │  │
│  │                                             │  │
│  │ 敏感内容模式:                               │  │
│  │ ☑ 密码（password=, passwd=）                │  │
│  │ ☑ API密钥（api_key=, token=）              │  │
│  │ ☑ 信用卡号（15-19位数字）                   │  │
│  │ ☑ 身份证号（18位）                          │  │
│  │ [+ 添加自定义规则]                          │  │
│  └───────────────────────────────────────────┘  │
│                                                 │
│  日志记录                                        │
│  ┌───────────────────────────────────────────┐  │
│  │ 日志级别: [Warn ▼]                         │  │
│  │ □ 在日志中显示内容预览（前50字符）          │  │
│  └───────────────────────────────────────────┘  │
│                                                 │
│  数据管理                                        │
│  ┌───────────────────────────────────────────┐  │
│  │ [查看剪贴板历史]  [清除所有历史]           │  │
│  │ [导出数据]        [删除我的账户]           │  │
│  └───────────────────────────────────────────┘  │
│                                                 │
│         [恢复默认设置]  [保存更改]              │
│                                                 │
└─────────────────────────────────────────────────┘
```

---

## 📊 合规性检查清单

### GDPR 合规

| 要求 | 实现状态 | 验证方法 |
|------|---------|---------|
| **合法、公平、透明** | ⏳ 待实施 | 明确的用户同意机制 |
| **目的限制** | ⏳ 待实施 | 只用于声明的用途 |
| **数据最小化** | ⏳ 待实施 | 默认不记录内容 |
| **准确性** | ⏳ 待实施 | 提供数据更正和删除机制 |
| **存储限制** | ⏳ 待实施 | 设置数据保留期限，自动清理 |
| **完整性和保密性** | ⏳ 待实施 | 加密存储，访问控制 |
| **可问责性** | ⏳ 待实施 | 审计日志，数据处理记录 |

### 个人信息保护法合规

| 要求 | 实现状态 | 验证方法 |
|------|---------|---------|
| **明示同意** | ⏳ 待实施 | 首次使用需用户明确同意 |
| **最小必要** | ⏳ 待实施 | 只收集必要的数据 |
| **公开规则** | ⏳ 待实施 | 隐私政策公开可查 |
| **安全保护** | ⏳ 待实施 | 加密存储，访问控制 |
| **删除权** | ⏳ 待实施 | 提供数据删除功能 |
| **撤回同意** | ⏳ 待实施 | 可随时关闭监控 |

---

## 🧪 测试方案

### 单元测试

```go
// TestPrivacyConfig_DefaultValues 测试默认配置
func TestPrivacyConfig_DefaultValues(t *testing.T) {
    config := DefaultPrivacyConfig()

    assert.False(t, config.RecordContent, "默认不应记录内容")
    assert.True(t, config.ContentHashOnly, "默认应记录哈希")
    assert.True(t, config.EnableContentFilter, "默认应启用过滤")
    assert.True(t, config.ConsentRequired, "默认应要求用户同意")
}

// TestSensitiveContentDetection 测试敏感内容检测
func TestSensitiveContentDetection(t *testing.T) {
    config := DefaultPrivacyConfig()
    config.CompilePatterns()

    tests := []struct {
        content  string
        expected bool
    }{
        {"password=mySecret123", true},
        {"api_key=sk-1234567890", true},
        {"普通文本内容", false},
    }

    for _, tt := range tests {
        t.Run(tt.content, func(t *testing.T) {
            result := config.isSensitiveContent(tt.content)
            assert.Equal(t, tt.expected, result)
        })
    }
}

// TestAppBlacklist 测试应用黑名单
func TestAppBlacklist(t *testing.T) {
    config := DefaultPrivacyConfig()

    tests := []struct {
        bundleID string
        expected bool
    }{
        {"com.agilebits.onepassword-osx-helper", true},
        {"com.bitwarden.desktop", true},
        {"com.apple.Safari", false},
    }

    for _, tt := range tests {
        t.Run(tt.bundleID, func(t *testing.T) {
            result := config.IsAppBlacklisted(tt.bundleID)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

### 集成测试

```go
// TestClipboardMonitor_WithPrivacyProtection 测试隐私保护
func TestClipboardMonitor_WithPrivacyProtection(t *testing.T) {
    eventBus := events.NewEventBus()
    monitor := NewClipboardMonitor(eventBus)

    // 使用默认的安全配置
    config := DefaultPrivacyConfig()

    // 启动监控器
    err := monitor.Start()
    require.NoError(t, err)
    defer monitor.Stop()

    // 模拟复制密码
    event := platform.ClipboardEvent{
        Content: "password=MySecretPassword123",
        Type:    "public.utf8-plain-text",
        Size:    28,
    }

    // 处理事件
    monitor.handlePlatformEvent(event, config)

    // 验证事件被过滤
    // 断言事件总线中没有收到完整内容
}
```

### 渗透测试

- 模拟敏感内容复制
- 尝试从日志中恢复数据
- 尝试从内存中提取数据
- 验证加密强度

---

## 📈 实施时间表

| 阶段 | 任务 | 时间 | 优先级 | 建议时机 |
|------|------|------|--------|---------|
| **阶段 1** | 基础修复 | 1-2天 | 🟡 重要 | Phase 1-6 完成后 |
| | - 默认不记录内容 | | | |
| | - 应用黑名单 | | | |
| | - 用户同意检查 | | | |
| **阶段 2** | 核心功能 | 3-5天 | 🟢 推荐 | Phase 1 完成后 |
| | - 配置系统 | | | |
| | - 数据加密 | | | |
| | - 审计日志 | | | |
| | - 指标统计 | | | |
| **阶段 3** | 增强保护 | 5-7天 | 🔵 可选 | 根据需求 |
| | - 用户界面 | | | |
| | - ML检测 | | | |
| | - 风险评估 | | | |

**总计**: 9-14 天

**建议**：可以在所有核心功能实现后再进行隐私保护的实施，这样可以更全面地评估隐私需求。

---

## 🎯 成功标准

### 阶段 1 验收标准

- [ ] 默认情况下，事件总线中不包含完整剪贴板内容
- [ ] 密码管理器等敏感应用被自动过滤
- [ ] 包含密码、令牌的内容被检测并过滤
- [ ] 用户首次使用时显示同意界面
- [ ] 所有测试通过

### 阶段 2 验收标准

- [ ] 隐私配置可以保存和加载
- [ ] 敏感内容可以被加密存储（可选启用）
- [ ] 过滤事件被记录到审计日志
- [ ] 隐私指标统计正常工作
- [ ] 所有测试通过

### 阶段 3 验收标准

- [ ] 用户同意界面正常显示
- [ ] 隐私设置可以保存和应用
- [ ] 机器学习检测准确率 >85%
- [ ] 隐私报告可以正常生成
- [ ] 所有测试通过

---

## 📚 相关文档

- [剪贴板监控隐私保护方案（详细版）](../privacy/clipboard-monitor-privacy-protection.md)
- [Phase 1: 基础监控](./02-phase1-monitoring.md)
- [隐私政策](../legal/privacy-policy.md) (待创建)

---

## ⚠️ 重要说明

### 当前版本使用建议

在隐私保护功能实施前，当前版本的剪贴板监控功能：

- ✅ **适用场景**：开发和测试环境
- ✅ **适用场景**：个人使用，不记录敏感数据
- ❌ **不适用场景**：生产环境公开使用
- ❌ **不适用场景**：处理敏感数据的环境

### 临时安全措施

在使用当前版本时，建议：

1. **不要复制敏感信息**：在使用应用时避免复制密码、令牌等
2. **定期清理日志**：定期检查和清理日志文件
3. **监控应用列表**：注意哪些应用在监控剪贴板
4. **谨慎使用事件订阅**：不要将剪贴板事件发送到不可信的服务

### 实施优先级调整

根据项目实际情况，可以灵活调整实施顺序：

**如果产品面向企业用户**：
- 可以在 Phase 2 后立即实施阶段 1
- 企业用户更关注隐私保护

**如果产品面向个人用户**：
- 可以在 Phase 6 后再实施
- 个人用户更关注功能体验

**如果计划公开发布**：
- 必须在公开发布前完成阶段 1
- 避免隐私相关的法律风险

---

**最后更新**: 2026-01-30
**文档状态**: ⚠️ 待实施（Phase 1-6 完成后）
