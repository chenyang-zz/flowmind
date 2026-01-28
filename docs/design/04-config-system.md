# 配置系统设计

FlowMind 采用分层配置管理，支持默认配置、用户配置、环境变量覆盖。

---

## 配置架构

### 配置优先级

```
1. 环境变量 (最高优先级)
2. 用户配置文件 (~/.config/flowmind/config.yaml)
3. 默认配置 (内置)
```

### 配置文件位置

- **macOS**: `~/Library/Application Support/FlowMind/config.yaml`
- **Linux**: `~/.config/flowmind/config.yaml`
- **开发**: `./config/dev.yaml`

---

## 配置结构

### 主配置

```yaml
# config.yaml

# 应用配置
app:
  name: "FlowMind"
  version: "1.0.0"
  log_level: "info"  # debug, info, warn, error

# 监控配置
monitor:
  enabled: true
  keyboard: true
  clipboard: true
  app_switch: true
  file_system: true
  poll_interval: "1s"

  # 隐私保护
  ignored_apps:
    - "1Password"
    - "Keychain Access"
  sensitive_patterns:
    - "password"
    - "token"
    - "api_key"

# 分析配置
analyzer:
  # 模式挖掘
  pattern_min_support: 3  # 最小出现次数
  pattern_window: "30m"   # 时间窗口大小
  session_timeout: "10m"  # 会话超时

  # AI 过滤
  ai_filter_enabled: true
  ai_cache_ttl: "1h"

# AI 配置
ai:
  # Claude API
  claude:
    enabled: true
    api_key: "${CLAUDE_API_KEY}"  # 从环境变量读取
    model: "claude-3-5-sonnet-20241022"
    max_tokens: 4096
    temperature: 0.7

  # Ollama
  ollama:
    enabled: true
    base_url: "http://localhost:11434"
    model: "llama3.2"
    timeout: "120s"

  # 路由策略
  router:
    simple_tasks: "ollama"
    complex_tasks: "claude"
    max_prompt_length: 500

# 自动化配置
automation:
  enabled: true
  max_concurrent: 5
  default_timeout: "300s"

  # 沙箱
  sandbox:
    enabled: true
    allowed_paths:
      - "~/Downloads"
      - "~/Documents"
      - "~/Desktop"
    allowed_actions:
      - "shell.exec"
      - "file.move"
      - "git.commit"
    max_memory: "100MB"
    max_duration: "30s"
    network_access: false

  # 调度器
  scheduler:
    enabled: true
    timezone: "Asia/Shanghai"

# 知识库配置
knowledge:
  enabled: true
  auto_tag: true
  auto_summarize: true

  # 向量搜索
  vector_search:
    enabled: true
    model: "nomic-embed-text"
    top_k: 10

# 通知配置
notifications:
  desktop: true
  sound: false
  cooldown: "30s"  # 通知冷却时间

# 数据库配置
database:
  sqlite:
    path: "~/.local/share/flowmind/flowmind.db"
    wal_mode: true
    cache_size: 1000
    max_connections: 25

  bbolt:
    path: "~/.local/share/flowmind/cache.db"

# 事件配置
events:
  retention:
    events: "30d"      # 事件保留时间
    patterns: "90d"    # 模式保留时间
    results: "180d"    # 执行结果保留时间

  cleanup:
    enabled: true
    schedule: "0 2 * * *"  # 每天凌晨 2 点

# API 配置
api:
  enabled: false  # 默认关闭，可选启用
  port: 8080
  cors_origins:
    - "http://localhost:3000"

# 界面配置
ui:
  theme: "system"  # light, dark, system
  language: "zh-CN"
  global_hotkey: "Cmd+Shift+M"

  # 仪表板
  dashboard:
    refresh_interval: "30s"
    show_graphs: true

# 调试配置
debug:
  enabled: false
  profile_port: 6060
  pprof_enabled: false
```

---

## 配置管理

### 配置结构

```go
// internal/config/config.go
package config

type Config struct {
    App        AppConfig        `yaml:"app"`
    Monitor    MonitorConfig    `yaml:"monitor"`
    Analyzer   AnalyzerConfig   `yaml:"analyzer"`
    AI         AIConfig         `yaml:"ai"`
    Automation AutomationConfig `yaml:"automation"`
    Knowledge  KnowledgeConfig  `yaml:"knowledge"`
    Database   DatabaseConfig   `yaml:"database"`
    Events     EventsConfig     `yaml:"events"`
    API        APIConfig        `yaml:"api"`
    UI         UIConfig         `yaml:"ui"`
    Debug      DebugConfig      `yaml:"debug"`
}

type AppConfig struct {
    Name     string `yaml:"name"`
    Version  string `yaml:"version"`
    LogLevel string `yaml:"log_level"`
}

type MonitorConfig struct {
    Enabled         bool          `yaml:"enabled"`
    Keyboard        bool          `yaml:"keyboard"`
    Clipboard       bool          `yaml:"clipboard"`
    AppSwitch       bool          `yaml:"app_switch"`
    FileSystem      bool          `yaml:"file_system"`
    PollInterval    time.Duration `yaml:"poll_interval"`
    IgnoredApps     []string      `yaml:"ignored_apps"`
    SensitivePatterns []string    `yaml:"sensitive_patterns"`
}

type AIConfig struct {
    Claude   ClaudeConfig   `yaml:"claude"`
    Ollama   OllamaConfig   `yaml:"ollama"`
    Router   RouterConfig   `yaml:"router"`
}

type ClaudeConfig struct {
    Enabled     bool          `yaml:"enabled"`
    APIKey      string        `yaml:"api_key"`
    Model       string        `yaml:"model"`
    MaxTokens   int           `yaml:"max_tokens"`
    Temperature float32       `yaml:"temperature"`
    Timeout     time.Duration `yaml:"timeout"`
}
```

### 配置加载

```go
// internal/config/loader.go
type Loader struct {
    configPath string
    config     *Config
}

func NewLoader(configPath string) *Loader {
    if configPath == "" {
        // 使用默认路径
        configPath = getDefaultConfigPath()
    }

    return &Loader{
        configPath: configPath,
    }
}

func (l *Loader) Load() (*Config, error) {
    // 1. 加载默认配置
    config := l.getDefaultConfig()

    // 2. 加载用户配置文件
    if _, err := os.Stat(l.configPath); err == nil {
        userConfig, err := l.loadFromFile(l.configPath)
        if err != nil {
            return nil, err
        }

        // 合并配置
        config = l.merge(config, userConfig)
    }

    // 3. 环境变量覆盖
    l.applyEnvVars(config)

    // 4. 验证配置
    if err := l.validate(config); err != nil {
        return nil, err
    }

    l.config = config
    return config, nil
}

func (l *Loader) loadFromFile(path string) (*Config, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }

    var config Config
    if err := yaml.Unmarshal(data, &config); err != nil {
        return nil, err
    }

    return &config, nil
}

func (l *Loader) getDefaultConfig() *Config {
    return &Config{
        App: AppConfig{
            Name:     "FlowMind",
            Version:  "1.0.0",
            LogLevel: "info",
        },
        Monitor: MonitorConfig{
            Enabled:      true,
            Keyboard:     true,
            Clipboard:    true,
            AppSwitch:    true,
            FileSystem:   true,
            PollInterval: 1 * time.Second,
            IgnoredApps: []string{
                "1Password",
                "Keychain Access",
            },
        },
        // ... 默认值
    }
}

func (l *Loader) merge(base, override *Config) *Config {
    // 深度合并配置
    merged := *base

    // 简化：使用 mergo 库
    mergo.Merge(&merged, override, mergo.WithOverride)

    return &merged
}

func (l *Loader) applyEnvVars(config *Config) {
    // Claude API Key
    if apiKey := os.Getenv("CLAUDE_API_KEY"); apiKey != "" {
        config.AI.Claude.APIKey = apiKey
    }

    // Ollama URL
    if url := os.Getenv("OLLAMA_URL"); url != "" {
        config.AI.Ollama.BaseURL = url
    }

    // 日志级别
    if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
        config.App.LogLevel = logLevel
    }

    // 调试模式
    if os.Getenv("DEBUG") == "true" {
        config.Debug.Enabled = true
    }
}

func (l *Loader) validate(config *Config) error {
    // 验证必需字段
    if config.AI.Claude.Enabled && config.AI.Claude.APIKey == "" {
        return fmt.Errorf("Claude API key is required when enabled")
    }

    // 验证路径
    if !filepath.IsAbs(config.Database.SQLite.Path) {
        config.Database.SQLite.Path = expandHomeDir(config.Database.SQLite.Path)
    }

    // 验证时间间隔
    if config.Monitor.PollInterval < 100*time.Millisecond {
        return fmt.Errorf("poll interval too small (min 100ms)")
    }

    return nil
}
```

### 配置热加载

```go
// internal/config/watcher.go
import "github.com/fsnotify/fsnotify"

type Watcher struct {
    configPath string
    callback   func(*Config)
    watcher    *fsnotify.Watcher
}

func NewWatcher(configPath string, callback func(*Config)) (*Watcher, error) {
    watcher, err := fsnotify.NewWatcher()
    if err != nil {
        return nil, err
    }

    w := &Watcher{
        configPath: configPath,
        callback:   callback,
        watcher:    watcher,
    }

    // 监控文件
    if err := watcher.Add(configPath); err != nil {
        return nil, err
    }

    return w, nil
}

func (w *Watcher) Start() {
    go func() {
        for {
            select {
            case event, ok := <-w.watcher.Events:
                if !ok {
                    return
                }

                if event.Op&fsnotify.Write == fsnotify.Write {
                    // 重新加载配置
                    loader := NewLoader(w.configPath)
                    config, err := loader.Load()
                    if err == nil {
                        w.callback(config)
                        log.Info("Config reloaded")
                    }
                }

            case err, ok := <-w.watcher.Errors:
                if !ok {
                    return
                }
                log.Error("Config watcher error:", err)
            }
        }
    }()
}

func (w *Watcher) Stop() {
    w.watcher.Close()
}
```

---

## 配置验证

### 规则验证

```go
// internal/config/validator.go
type Validator struct {
    rules []ValidationRule
}

type ValidationRule struct {
    Path    string
    Validate func(interface{}) error
}

func (v *Validator) Validate(config *Config) error {
    for _, rule := range v.rules {
        value := getValueByPath(config, rule.Path)
        if err := rule.Validate(value); err != nil {
            return fmt.Errorf("config.%s: %w", rule.Path, err)
        }
    }
    return nil
}

func getDefaultValidator() *Validator {
    return &Validator{
        rules: []ValidationRule{
            {
                Path: "ai.claude.model",
                Validate: func(v interface{}) error {
                    model := v.(string)
                    validModels := []string{
                        "claude-3-5-sonnet-20241022",
                        "claude-3-opus-20240229",
                        "claude-3-haiku-20240307",
                    }
                    for _, m := range validModels {
                        if model == m {
                            return nil
                        }
                    }
                    return fmt.Errorf("invalid model: %s", model)
                },
            },
            {
                Path: "monitor.poll_interval",
                Validate: func(v interface{}) error {
                    interval := v.(time.Duration)
                    if interval < 100*time.Millisecond {
                        return fmt.Errorf("too small (min 100ms)")
                    }
                    return nil
                },
            },
        },
    }
}
```

---

## 配置迁移

### 版本迁移

```go
// internal/config/migrate.go
type Migrator struct {
    currentVersion string
}

func (m *Migrator) Migrate(config *Config) error {
    version := config.App.Version

    switch version {
    case "1.0.0":
        // 无需迁移
        return nil

    case "0.9.0":
        return m.migrateFrom090(config)

    default:
        return fmt.Errorf("unsupported version: %s", version)
    }
}

func (m *Migrator) migrateFrom090(config *Config) error {
    // 添加新字段
    if config.AI.Claude.Model == "" {
        config.AI.Claude.Model = "claude-3-5-sonnet-20241022"
    }

    // 重命名字段
    if config.AI.Claude.Key != "" {
        config.AI.Claude.APIKey = config.AI.Claude.Key
    }

    return nil
}
```

---

## 使用示例

### 加载配置

```go
// main.go
func main() {
    // 加载配置
    loader := config.NewLoader("")
    cfg, err := loader.Load()
    if err != nil {
        log.Fatal("Failed to load config:", err)
    }

    // 应用日志级别
    log.SetLevel(loglevelFromString(cfg.App.LogLevel))

    // 配置热加载
    watcher, _ := config.NewWatcher("", func(newConfig *config.Config) {
        cfg = newConfig
        applyConfig(cfg)
    })
    watcher.Start()

    // 使用配置
    app := &App{
        config: cfg,
        // ...
    }
}
```

### 环境变量

```bash
# 设置 Claude API Key
export CLAUDE_API_KEY="sk-ant-..."

# 启用调试模式
export DEBUG="true"

# 自定义日志级别
export LOG_LEVEL="debug"

# 自定义 Ollama URL
export OLLAMA_URL="http://localhost:11434"

# 运行应用
./flowmind
```

---

**相关文档**：
- [数据库设计](./01-database-design.md)
- [安全设计](./05-security-design.md)
- [开发环境搭建](../implementation/01-development-setup.md)
