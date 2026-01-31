/**
 * Package ai AI 服务基础设施层
 *
 * 提供多模型工厂和配置管理
 */

package ai

import (
	"fmt"
	"os"
)

/**
 * AIConfig AI 模型通用配置
 */
type AIConfig struct {
	// Provider 模型提供商（claude, openai, zhipu, ollama）
	Provider string

	// APIKey API 密钥
	APIKey string

	// Model 模型名称
	Model string

	// BaseURL API 基础 URL（可选）
	BaseURL *string

	// MaxTokens 最大生成 token 数
	MaxTokens int

	// Temperature 温度参数
	Temperature *float32

	// Timeout 请求超时时间
	Timeout int // 秒
}

/**
 * LoadFromEnv 从环境变量加载配置
 *
 * 支持的环境变量：
 * - AI_PROVIDER: 提供商（claude, openai, zhipu）
 * - AI_API_KEY: API 密钥
 * - AI_MODEL: 模型名称
 * - AI_BASE_URL: 自定义 API 端点
 * - AI_MAX_TOKENS: 最大 token 数
 * - AI_TEMPERATURE: 温度参数
 * - AI_TIMEOUT: 超时时间（秒）
 */
func (c *AIConfig) LoadFromEnv() *AIConfig {
	// 加载提供商
	if c.Provider == "" {
		c.Provider = os.Getenv("AI_PROVIDER")
		if c.Provider == "" {
			c.Provider = "claude" // 默认使用 Claude
		}
	}

	// 加载 API Key
	if c.APIKey == "" {
		c.APIKey = os.Getenv("AI_API_KEY")
		// 如果没有通用配置，尝试特定提供商的配置
		if c.APIKey == "" {
			switch c.Provider {
			case "claude":
				c.APIKey = os.Getenv("CLAUDE_API_KEY")
			case "openai":
				c.APIKey = os.Getenv("OPENAI_API_KEY")
			case "zhipu":
				c.APIKey = os.Getenv("ZHIPU_API_KEY")
			case "ollama":
				// Ollama 本地运行不需要 API Key
				c.APIKey = ""
			}
		}
	}

	// 加载模型名称
	if c.Model == "" {
		c.Model = os.Getenv("AI_MODEL")
		if c.Model == "" {
			// 使用默认模型
			switch c.Provider {
			case "claude":
				c.Model = "claude-3-5-sonnet-20241022"
			case "openai":
				c.Model = "gpt-4o"
			case "zhipu":
				c.Model = "glm-4"
			case "ollama":
				c.Model = "llama3.2"
			}
		}
	}

	// 加载 BaseURL
	if baseURL := os.Getenv("AI_BASE_URL"); baseURL != "" {
		c.BaseURL = &baseURL
	}

	// 设置默认值
	if c.MaxTokens == 0 {
		c.MaxTokens = 4096
	}

	return c
}

/**
 * Validate 验证配置
 */
func (c *AIConfig) Validate() error {
	if c.Provider == "" {
		return fmt.Errorf("提供商不能为空")
	}

	// 验证提供商是否支持
	switch c.Provider {
	case "claude", "openai", "zhipu", "ollama":
		// 支持的提供商
	default:
		return fmt.Errorf("不支持的提供商: %s", c.Provider)
	}

	// 验证 API Key（Ollama 除外）
	if c.Provider != "ollama" && c.APIKey == "" {
		return fmt.Errorf("API Key 不能为空")
	}

	return nil
}

/**
 * NewAIModel 创建 AI 模型实例（工厂方法）
 *
 * Parameters:
 *   - config: AI 配置
 *
 * Returns: AIModel - AI 模型实例
 */
func NewAIModel(config *AIConfig) (AIModel, error) {
	if config == nil {
		config = &AIConfig{}
	}
	config = config.LoadFromEnv()

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}

	// 根据提供商创建对应的客户端
	switch config.Provider {
	case "claude":
		return NewClaudeClientFromConfig(config)
	case "openai":
		// TODO: 实现 OpenAI 客户端
		return nil, fmt.Errorf("OpenAI 客户端尚未实现")
	case "zhipu":
		return NewZhipuClientFromConfig(config)
	case "ollama":
		// TODO: 实现 Ollama 客户端
		return nil, fmt.Errorf("Ollama 客户端尚未实现")
	default:
		return nil, fmt.Errorf("未知的提供商: %s", config.Provider)
	}
}

/**
 * NewClaudeClientFromConfig 从通用配置创建 Claude 客户端
 */
func NewClaudeClientFromConfig(config *AIConfig) (AIModel, error) {
	claudeConfig := &ClaudeConfig{
		APIKey:      config.APIKey,
		Model:       config.Model,
		BaseURL:     config.BaseURL,
		MaxTokens:   config.MaxTokens,
		Temperature: config.Temperature,
	}

	// 设置超时
	if config.Timeout > 0 {
		// TODO: 在 ClaudeConfig 中添加 Timeout 字段
	}

	return NewClaudeClient(claudeConfig)
}

/**
 * SwitchProvider 切换 AI 提供商
 *
 * Parameters:
 *   - provider: 新的提供商名称
 *   - apiKey: API 密钥（可选）
 *   - model: 模型名称（可选）
 *
 * Returns: AIModel - 新的 AI 模型实例
 */
func SwitchProvider(provider, apiKey, model string) (AIModel, error) {
	config := &AIConfig{
		Provider: provider,
		APIKey:   apiKey,
		Model:    model,
	}

	return NewAIModel(config)
}
