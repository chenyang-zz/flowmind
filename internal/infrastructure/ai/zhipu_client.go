/**
 * Package ai AI 服务基础设施层
 *
 * 智谱AI (ChatGLM) 客户端实现
 */

package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/chenyang-zz/flowmind/internal/infrastructure/logger"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"go.uber.org/zap"
)

// 确保 ZhipuClient 实现了 AIModel 接口
var _ AIModel = (*ZhipuClient)(nil)

/**
 * ZhipuClient 智谱AI 客户端
 *
 * 使用智谱AI的 ChatGLM 系列模型进行模式分析
 * 基于 Eino 框架的 OpenAI 兼容客户端实现
 */
type ZhipuClient struct {
	// chatModel Eino ChatModel 实例
	chatModel model.ChatModel

	// config 智谱AI 配置
	config *ZhipuConfig

	// apiKey 智谱AI API 密钥
	apiKey string
}

/**
 * ZhipuConfig 智谱AI 配置
 */
type ZhipuConfig struct {
	// APIKey 智谱AI API 密钥
	APIKey string

	// Model 模型名称（glm-4, glm-4-flash, glm-4-air 等）
	Model string

	// BaseURL API 基础 URL（可选）
	BaseURL *string

	// MaxTokens 最大生成 token 数
	MaxTokens int

	// Temperature 温度参数（0.0-1.0）
	Temperature *float32

	// Timeout 请求超时时间
	Timeout time.Duration
}

/**
 * GetType 获取模型类型
 */
func (c *ZhipuConfig) GetType() ModelType {
	return ModelTypeZhipu
}

/**
 * Validate 验证配置
 */
func (c *ZhipuConfig) Validate() error {
	if c.APIKey == "" {
		c.APIKey = GetEnvOrDefault("ZHIPU_API_KEY", "")
		if c.APIKey == "" {
			return fmt.Errorf("未找到 ZHIPU_API_KEY 环境变量")
		}
	}

	if c.Model == "" {
		c.Model = GetEnvOrDefault("ZHIPU_MODEL", "glm-4")
	}

	if c.MaxTokens <= 0 {
		c.MaxTokens = 4096
	}

	if c.Temperature != nil && (*c.Temperature < 0 || *c.Temperature > 1) {
		return fmt.Errorf("temperature 必须在 0.0-1.0 之间")
	}

	if c.Timeout <= 0 {
		c.Timeout = 30 * time.Second
	}

	return nil
}

/**
 * NewZhipuClient 创建智谱AI客户端
 *
 * Parameters:
 *   - config: 智谱AI配置
 *
 * Returns: *ZhipuClient - 智谱AI客户端实例
 */
func NewZhipuClient(config *ZhipuConfig) (*ZhipuClient, error) {
	if config == nil {
		return nil, fmt.Errorf("配置不能为空")
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}

	// 设置基础 URL
	baseURL := "https://open.bigmodel.cn/api/paas/v4"
	if config.BaseURL != nil && *config.BaseURL != "" {
		baseURL = *config.BaseURL
	}

	// 准备配置参数
	maxTokens := config.MaxTokens

	// 创建 Eino OpenAI ChatModel（智谱AI兼容OpenAI API）
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL:     baseURL,
		APIKey:      config.APIKey,
		Model:       config.Model,
		MaxTokens:   &maxTokens,
		Temperature: config.Temperature,
	})
	if err != nil {
		return nil, fmt.Errorf("创建智谱AI ChatModel失败: %w", err)
	}

	log.Printf("创建智谱AI客户端成功: model=%s, maxTokens=%d", config.Model, config.MaxTokens)

	return &ZhipuClient{
		chatModel: chatModel,
		config:    config,
		apiKey:    config.APIKey,
	}, nil
}

/**
 * NewZhipuClientFromConfig 从通用配置创建智谱AI客户端
 */
func NewZhipuClientFromConfig(config *AIConfig) (AIModel, error) {
	zhipuConfig := &ZhipuConfig{
		APIKey:      config.APIKey,
		Model:       config.Model,
		BaseURL:     config.BaseURL,
		MaxTokens:   config.MaxTokens,
		Temperature: config.Temperature,
	}

	// 设置超时
	if config.Timeout > 0 {
		zhipuConfig.Timeout = time.Duration(config.Timeout) * time.Second
	}

	return NewZhipuClient(zhipuConfig)
}

/**
 * AnalyzePattern 分析模式（智谱AI实现）
 *
 * 使用智谱AI分析模式的特征，判断是否值得自动化
 *
 * Parameters:
 *   - ctx: 上下文
 *   - patternData: 模式数据（JSON格式）
 *
 * Returns: *PatternAnalysis - 分析结果
 */
func (c *ZhipuClient) AnalyzePattern(ctx context.Context, patternData map[string]interface{}) (*PatternAnalysis, error) {
	// 检查 chatModel 是否已初始化
	if c.chatModel == nil {
		return nil, fmt.Errorf("智谱AI客户端未正确初始化：chatModel为空")
	}

	// 构建提示词
	prompt := BuildPatternAnalysisPrompt(patternData)

	// 准备消息
	messages := []*schema.Message{
		{
			Role:    schema.System,
			Content: "你是一个专业的自动化分析助手，擅长评估用户操作模式是否值得自动化。请以 JSON 格式返回分析结果。",
		},
		{
			Role:    schema.User,
			Content: prompt,
		},
	}

	// 设置超时
	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	// 调用智谱AI API
	startTime := time.Now()
	response, err := c.chatModel.Generate(ctx, messages)
	duration := time.Since(startTime)

	if err != nil {
		logger.Error("调用智谱AI API失败",
			zap.Error(err),
			zap.Duration("duration", duration))
		return nil, fmt.Errorf("调用智谱AI API失败: %w", err)
	}

	logger.Info("调用智谱AI API成功",
		zap.Duration("duration", duration),
		zap.Int("promptTokens", response.ResponseMeta.Usage.PromptTokens),
		zap.Int("completionTokens", response.ResponseMeta.Usage.CompletionTokens),
		zap.Int("totalTokens", response.ResponseMeta.Usage.TotalTokens))

	// 解析响应
	analysis, err := c.parseAnalysisResponse(response.Content)
	if err != nil {
		logger.Error("解析智谱AI响应失败",
			zap.String("response", response.Content),
			zap.Error(err))
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return analysis, nil
}

/**
 * AnalyzePatternBatch 批量分析模式
 *
 * Parameters:
 *   - ctx: 上下文
 *   - patterns: 模式列表
 *
 * Returns: []*PatternAnalysis - 分析结果列表
 */
func (c *ZhipuClient) AnalyzePatternBatch(ctx context.Context, patterns []map[string]interface{}) ([]*PatternAnalysis, error) {
	results := make([]*PatternAnalysis, len(patterns))

	for i, pattern := range patterns {
		analysis, err := c.AnalyzePattern(ctx, pattern)
		if err != nil {
			logger.Error("分析模式失败",
				zap.Int("index", i),
				zap.Error(err))
			// 继续处理其他模式，不中断整个批次
			results[i] = &PatternAnalysis{
				ShouldAutomate: false,
				Reason:         fmt.Sprintf("分析失败: %v", err),
				Complexity:     "unknown",
			}
		} else {
			results[i] = analysis
		}

		// 添加延迟以避免 API 限流
		if i < len(patterns)-1 {
			time.Sleep(500 * time.Millisecond)
		}
	}

	return results, nil
}

/**
 * GetType 获取模型类型
 */
func (c *ZhipuClient) GetType() ModelType {
	return ModelTypeZhipu
}

/**
 * Close 关闭连接
 *
 * Returns: error - 关闭错误
 */
func (c *ZhipuClient) Close() error {
	logger.Info("智谱AI客户端已关闭")
	return nil
}

/**
 * parseAnalysisResponse 解析智谱AI响应
 */
func (c *ZhipuClient) parseAnalysisResponse(content string) (*PatternAnalysis, error) {
	jsonStr := content

	// 移除可能的 markdown 代码块
	if len(content) > 0 && content[0] == '`' {
		start := 0
		for i := 0; i < len(content)-3; i++ {
			if content[i] == '`' && content[i+1] == '`' && content[i+2] == '`' {
				start = i + 3
				// 跳过语言标识符（如 "json", "python" 等）
				for start < len(content) && (content[start] >= 'a' && content[start] <= 'z') ||
					(content[start] >= 'A' && content[start] <= 'Z') ||
					(content[start] >= '0' && content[start] <= '9') {
					start++
				}
				// 跳过空白字符和换行符
				for start < len(content) && (content[start] == '\n' || content[start] == '\r' || content[start] == ' ' || content[start] == '\t') {
					start++
				}
				break
			}
		}

		end := len(content)
		for i := len(content) - 1; i >= 3; i-- {
			if content[i] == '`' && content[i-1] == '`' && content[i-2] == '`' {
				end = i - 2
				break
			}
		}

		if start > 0 && end > start {
			jsonStr = content[start:end]
		}
	}

	var analysis PatternAnalysis
	err := json.Unmarshal([]byte(jsonStr), &analysis)
	if err != nil {
		return nil, fmt.Errorf("JSON 解析失败: %w", err)
	}

	analysis.AnalyzedAt = time.Now()
	return &analysis, nil
}

/**
 * GetEnvOrDefault 获取环境变量或返回默认值
 */
func GetEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
