/**
 * Package config 提供配置管理功能
 *
 * 负责加载和管理应用的配置信息
 */

package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

/**
 * Config 应用配置结构体
 *
 * 包含应用的所有可配置参数
 */
type Config struct {
	// Application 应用基本配置
	Application ApplicationConfig `yaml:"application"`

	// Monitor 监控配置
	Monitor MonitorConfig `yaml:"monitor"`

	// AI AI 配置
	AI AIConfig `yaml:"ai"`

	// Automation 自动化配置
	Automation AutomationConfig `yaml:"automation"`

	// Knowledge 知识管理配置
	Knowledge KnowledgeConfig `yaml:"knowledge"`

	// Storage 存储配置
	Storage StorageConfig `yaml:"storage"`

	// Notifications 通知配置
	Notifications NotificationsConfig `yaml:"notifications"`

	// Logging 日志配置
	Logging LoggingConfig `yaml:"logging"`
}

/**
 * ApplicationConfig 应用基本配置
 */
type ApplicationConfig struct {
	/** 应用名称 */
	Name string `yaml:"name"`

	/** 应用版本 */
	Version string `yaml:"version"`

	/** 日志级别 */
	LogLevel string `yaml:"log_level"`

	/** 是否启用调试模式 */
	Debug bool `yaml:"debug"`
}

/**
 * MonitorConfig 监控配置
 */
type MonitorConfig struct {
	/** 启用的监控器列表 */
	EnabledMonitors []string `yaml:"enabled_monitors"`

	/** 采样率 */
	SampleRate string `yaml:"sample_rate"`

	/** 事件缓冲区大小 */
	EventBufferSize int `yaml:"event_buffer_size"`

	/** 过滤器配置 */
	Filters FilterConfig `yaml:"filters"`
}

/**
 * FilterConfig 过滤器配置
 */
type FilterConfig struct {
	/** 忽略的应用列表 */
	IgnoreApps []string `yaml:"ignore_apps"`

	/** 忽略的窗口标题列表 */
	IgnoreWindowTitles []string `yaml:"ignore_window_titles"`
}

/**
 * AIConfig AI 配置
 */
type AIConfig struct {
	/** AI 提供商 */
	Provider string `yaml:"provider"`

	/** Claude 配置 */
	Claude ClaudeConfig `yaml:"claude"`

	/** Ollama 配置 */
	Ollama OllamaConfig `yaml:"ollama"`

	/** 缓存配置 */
	Cache CacheConfig `yaml:"cache"`
}

/**
 * ClaudeConfig Claude API 配置
 */
type ClaudeConfig struct {
	/** API 密钥 */
	APIKey string `yaml:"api_key"`

	/** 使用的模型 */
	Model string `yaml:"model"`

	/** 最大 token 数 */
	MaxTokens int `yaml:"max_tokens"`

	/** 温度参数 */
	Temperature float64 `yaml:"temperature"`
}

/**
 * OllamaConfig Ollama 配置
 */
type OllamaConfig struct {
	/** 基础 URL */
	BaseURL string `yaml:"base_url"`

	/** 使用的模型 */
	Model string `yaml:"model"`
}

/**
 * CacheConfig 缓存配置
 */
type CacheConfig struct {
	/** 是否启用缓存 */
	Enabled bool `yaml:"enabled"`

	/** 缓存过期时间 */
	TTL string `yaml:"ttl"`

	/** 最大缓存数量 */
	MaxSize int `yaml:"max_size"`
}

/**
 * AutomationConfig 自动化配置
 */
type AutomationConfig struct {
	/** 最大执行时间 */
	MaxExecutionTime string `yaml:"max_execution_time"`

	/** 允许的路径列表 */
	AllowedPaths []string `yaml:"allowed_paths"`

	/** 沙箱配置 */
	Sandbox SandboxConfig `yaml:"sandbox"`

	/** 调度器配置 */
	Scheduler SchedulerConfig `yaml:"scheduler"`
}

/**
 * SandboxConfig 沙箱配置
 */
type SandboxConfig struct {
	/** 是否启用沙箱 */
	Enabled bool `yaml:"enabled"`

	/** 最大内存 */
	MaxMemory string `yaml:"max_memory"`

	/** 最大 CPU 时间 */
	MaxCPUTime string `yaml:"max_cpu_time"`

	/** 是否允许网络访问 */
	AllowNetwork bool `yaml:"allow_network"`
}

/**
 * SchedulerConfig 调度器配置
 */
type SchedulerConfig struct {
	/** 是否启用调度器 */
	Enabled bool `yaml:"enabled"`

	/** 最大并发任务数 */
	MaxConcurrent int `yaml:"max_concurrent"`
}

/**
 * KnowledgeConfig 知识管理配置
 */
type KnowledgeConfig struct {
	/** 剪藏配置 */
	Clipper ClipperConfig `yaml:"clipper"`

	/** 搜索配置 */
	Search SearchConfig `yaml:"search"`

	/** 图谱配置 */
	Graph GraphConfig `yaml:"graph"`
}

/**
 * ClipperConfig 剪藏配置
 */
type ClipperConfig struct {
	/** 是否自动打标签 */
	AutoTag bool `yaml:"auto_tag"`

	/** 是否自动生成摘要 */
	AutoSummarize bool `yaml:"auto_summarize"`

	/** 最大剪藏大小 */
	MaxClipSize string `yaml:"max_clip_size"`
}

/**
 * SearchConfig 搜索配置
 */
type SearchConfig struct {
	/** 返回的前 K 个结果 */
	TopK int `yaml:"top_k"`

	/** 相似度阈值 */
	SimilarityThreshold float64 `yaml:"similarity_threshold"`
}

/**
 * GraphConfig 图谱配置
 */
type GraphConfig struct {
	/** 是否启用图谱 */
	Enabled bool `yaml:"enabled"`

	/** 最大节点数 */
	MaxNodes int `yaml:"max_nodes"`

	/** 是否自动关联 */
	AutoLink bool `yaml:"auto_link"`
}

/**
 * StorageConfig 存储配置
 */
type StorageConfig struct {
	/** SQLite 配置 */
	SQLite SQLiteConfig `yaml:"sqlite"`

	/** Bolt 配置 */
	Bolt BoltConfig `yaml:"bolt"`

	/** 向量数据库配置 */
	Vector VectorConfig `yaml:"vector"`

	/** 数据保留策略 */
	Retention RetentionConfig `yaml:"retention"`
}

/**
 * SQLiteConfig SQLite 配置
 */
type SQLiteConfig struct {
	/** 数据库文件路径 */
	Path string `yaml:"path"`

	/** 最大打开连接数 */
	MaxOpenConns int `yaml:"max_open_conns"`

	/** 最大空闲连接数 */
	MaxIdleConns int `yaml:"max_idle_conns"`

	/** 连接最大生命周期 */
	ConnMaxLifetime string `yaml:"conn_max_lifetime"`
}

/**
 * BoltConfig Bolt 配置
 */
type BoltConfig struct {
	/** 数据库文件路径 */
	Path string `yaml:"path"`

	/** Bucket 名称 */
	Bucket string `yaml:"bucket"`
}

/**
 * VectorConfig 向量数据库配置
 */
type VectorConfig struct {
	/** 向量数据库路径 */
	Path string `yaml:"path"`

	/** 向量维度 */
	EmbeddingDim int `yaml:"embedding_dim"`

	/** 索引类型 */
	IndexType string `yaml:"index_type"`
}

/**
 * RetentionConfig 数据保留配置
 */
type RetentionConfig struct {
	/** 事件保留天数 */
	EventsDays int `yaml:"events_days"`

	/** 模式保留天数 */
	PatternsDays int `yaml:"patterns_days"`

	/** 知识是否永久保留 */
	KnowledgeForever bool `yaml:"knowledge_forever"`
}

/**
 * NotificationsConfig 通知配置
 */
type NotificationsConfig struct {
	/** 桌面通知配置 */
	Desktop DesktopConfig `yaml:"desktop"`

	/** Webhook 配置 */
	Webhooks []WebhookConfig `yaml:"webhooks"`

	/** 去重配置 */
	Dedup DedupConfig `yaml:"dedup"`
}

/**
 * DesktopConfig 桌面通知配置
 */
type DesktopConfig struct {
	/** 是否启用桌面通知 */
	Enabled bool `yaml:"enabled"`

	/** 是否播放声音 */
	Sound bool `yaml:"sound"`
}

/**
 * WebhookConfig Webhook 配置
 */
type WebhookConfig struct {
	/** Webhook URL */
	URL string `yaml:"url"`

	/** 是否启用 */
	Enabled bool `yaml:"enabled"`
}

/**
 * DedupConfig 去重配置
 */
type DedupConfig struct {
	/** 是否启用去重 */
	Enabled bool `yaml:"enabled"`

	/** 去重时间窗口 */
	TTL string `yaml:"ttl"`
}

/**
 * LoggingConfig 日志配置
 */
type LoggingConfig struct {
	/** 日志级别 */
	Level string `yaml:"level"`

	/** 日志格式 */
	Format string `yaml:"format"`

	/** 输出目标 */
	Output string `yaml:"output"`

	/** 文件配置 */
	File FileConfig `yaml:"file"`
}

/**
 * FileConfig 文件配置
 */
type FileConfig struct {
	/** 日志文件路径 */
	Path string `yaml:"path"`

	/** 最大文件大小 */
	MaxSize string `yaml:"max_size"`

	/** 最大备份文件数 */
	MaxBackups int `yaml:"max_backups"`

	/** 最大保留天数 */
	MaxAge string `yaml:"max_age"`

	/** 是否压缩 */
	Compress bool `yaml:"compress"`
}

/**
 * Load 加载配置文件
 *
 * 从默认路径加载配置文件，并进行环境变量替换
 *
 * Returns:
 *   - *Config: 加载的配置
 *   - error: 错误信息
 */
func Load() (*Config, error) {
	// 获取用户主目录
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	// 构造配置文件路径
	configPath := filepath.Join(homeDir, ".flowmind", "config.yaml")

	// 如果用户配置不存在，使用默认配置
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return LoadDefault()
	}

	// 读取配置文件
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	// 解析 YAML
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	// 替换环境变量
	expandEnvVars(&config)

	return &config, nil
}

/**
 * LoadDefault 加载默认配置
 *
 * Returns:
 *   - *Config: 默认配置
 *   - error: 错误信息
 */
func LoadDefault() (*Config, error) {
	// TODO: 从嵌入的默认配置文件加载
	// 目前返回一个最小配置
	return &Config{
		Application: ApplicationConfig{
			Name:     "FlowMind",
			Version:  "1.0.0",
			LogLevel: "info",
		},
	}, nil
}

/**
 * expandEnvVars 展开环境变量
 *
 * 替换配置中的环境变量占位符，如 ${HOME}
 *
 * Parameters:
 *   - config: 配置对象
 */
func expandEnvVars(config *Config) {
	// 替换存储路径中的环境变量
	homeDir := os.Getenv("HOME")
	if homeDir == "" {
		homeDir = os.Getenv("USERPROFILE")
	}

	// TODO: 实现完整的环境变量替换
	// 这里需要递归替换所有配置字段中的环境变量
	_ = homeDir
}
