/**
 * Package ai AI 服务基础设施层
 *
 * 负责与 Claude API 的集成，使用 Eino 框架实现模式分析
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
	"github.com/cloudwego/eino-ext/components/model/claude"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"go.uber.org/zap"
)

// 确保 ClaudeClient 实现了 AIModel 接口
var _ AIModel = (*ClaudeClient)(nil)

/**
 * ClaudeClient Claude AI 客户端
 *
 * 基于 Eino 框架的 Claude API 客户端封装
 */
type ClaudeClient struct {
	// chatModel Eino ChatModel 实例
	chatModel model.ChatModel

	// config Claude 配置
	config *ClaudeConfig

	// apiKey Claude API 密钥
	apiKey string
}

/**
 * ClaudeConfig Claude 配置
 */
type ClaudeConfig struct {
	// APIKey Claude API 密钥（从环境变量读取）
	APIKey string

	// Model 模型名称（如 "claude-3-5-sonnet-20240620"）
	Model string

	// BaseURL API 基础 URL（可选，用于自定义端点）
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
 *
 * Returns: ModelType - 返回 ModelTypeClaude
 */
func (c *ClaudeConfig) GetType() ModelType {
	return ModelTypeClaude
}

/**
 * Validate 验证配置
 *
 * Returns: error - 验证错误
 */
func (c *ClaudeConfig) Validate() error {
	// 验证 API Key
	if c.APIKey == "" {
		c.APIKey = os.Getenv("CLAUDE_API_KEY")
		if c.APIKey == "" {
			return fmt.Errorf("未找到 CLAUDE_API_KEY 环境变量")
		}
	}

	// 验证模型名称
	if c.Model == "" {
		c.Model = os.Getenv("CLAUDE_MODEL")
		if c.Model == "" {
			c.Model = "claude-3-5-sonnet-20241022"
		}
	}

	// 验证 MaxTokens
	if c.MaxTokens <= 0 {
		c.MaxTokens = 4096
	}

	// 验证 Temperature
	if c.Temperature != nil && (*c.Temperature < 0 || *c.Temperature > 1) {
		return fmt.Errorf("temperature 必须在 0.0-1.0 之间")
	}

	// 验证超时
	if c.Timeout <= 0 {
		c.Timeout = 30 * time.Second
	}

	return nil
}

/**
 * NewClaudeClient 创建 Claude 客户端
 *
 * Parameters:
 *   - config: Claude 配置
 *
 * Returns: *ClaudeClient - Claude 客户端实例
 */
func NewClaudeClient(config *ClaudeConfig) (*ClaudeClient, error) {
	if config == nil {
		return nil, fmt.Errorf("配置不能为空")
	}

	// 验证配置
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}

	// 创建 Eino Claude ChatModel
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	chatModel, err := claude.NewChatModel(ctx, &claude.Config{
		APIKey:      config.APIKey,
		Model:       config.Model,
		BaseURL:     config.BaseURL,
		MaxTokens:   config.MaxTokens,
		Temperature: config.Temperature,
	})
	if err != nil {
		return nil, fmt.Errorf("创建 Claude ChatModel 失败: %w", err)
	}

	log.Printf("创建 Claude 客户端成功: model=%s, maxTokens=%d", config.Model, config.MaxTokens)

	return &ClaudeClient{
		chatModel: chatModel,
		config:    config,
		apiKey:    config.APIKey,
	}, nil
}

/**
 * AnalyzePattern 分析模式是否值得自动化
 *
 * 使用 Claude AI 分析模式的特征，判断是否值得自动化
 *
 * Parameters:
 *   - ctx: 上下文
 *   - patternData: 模式数据（JSON 格式）
 *
 * Returns: *PatternAnalysis - 分析结果
 */
func (c *ClaudeClient) AnalyzePattern(ctx context.Context, patternData map[string]interface{}) (*PatternAnalysis, error) {
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

	// 调用 Claude API
	startTime := time.Now()
	response, err := c.chatModel.Generate(ctx, messages)
	duration := time.Since(startTime)

	if err != nil {
		logger.Error("调用 Claude API 失败",
			zap.Error(err),
			zap.Duration("duration", duration))
		return nil, fmt.Errorf("调用 Claude API 失败: %w", err)
	}

	logger.Info("调用 Claude API 成功",
		zap.Duration("duration", duration),
		zap.Int("promptTokens", response.ResponseMeta.Usage.PromptTokens),
		zap.Int("completionTokens", response.ResponseMeta.Usage.CompletionTokens),
		zap.Int("totalTokens", response.ResponseMeta.Usage.TotalTokens))

	// 解析响应
	analysis, err := c.parseAnalysisResponse(response.Content)
	if err != nil {
		logger.Error("解析 Claude 响应失败",
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
func (c *ClaudeClient) AnalyzePatternBatch(ctx context.Context, patterns []map[string]interface{}) ([]*PatternAnalysis, error) {
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
 * parseAnalysisResponse 解析 Claude 响应
 *
 * Parameters:
 *   - content: Claude 返回的内容
 *
 * Returns: *PatternAnalysis - 解析后的分析结果
 */
func (c *ClaudeClient) parseAnalysisResponse(content string) (*PatternAnalysis, error) {
	// 尝试提取 JSON（如果响应包含 markdown 代码块）
	jsonStr := content
	if len(content) > 0 {
		// 移除可能的 markdown 代码块标记
		if content[0] == '`' {
			// 查找第一个 ``` 后的位置
			start := 0
			for i := 0; i < len(content)-3; i++ {
				if content[i] == '`' && content[i+1] == '`' && content[i+2] == '`' {
					start = i + 3
					// 跳过换行符
					for start < len(content) && content[start] == '\n' {
						start++
					}
					break
				}
			}
			// 查找最后一个 ``` 的位置
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
	}

	// 解析 JSON
	var analysis PatternAnalysis
	err := json.Unmarshal([]byte(jsonStr), &analysis)
	if err != nil {
		return nil, fmt.Errorf("JSON 解析失败: %w", err)
	}

	// 设置分析时间
	analysis.AnalyzedAt = time.Now()

	return &analysis, nil
}

/**
 * PatternAnalysis 模式分析结果
 */
type PatternAnalysis struct {
	// ShouldAutomate 是否值得自动化
	ShouldAutomate bool `json:"should_automate"`

	// Reason 原因说明
	Reason string `json:"reason"`

	// EstimatedTimeSaving 预计节省时间（秒）
	EstimatedTimeSaving int64 `json:"estimated_time_saving"`

	// Complexity 实现复杂度（low/medium/high）
	Complexity string `json:"complexity"`

	// SuggestedName 建议的自动化名称
	SuggestedName string `json:"suggested_name"`

	// SuggestedSteps 建议的自动化步骤
	SuggestedSteps []string `json:"suggested_steps"`

	// AnalyzedAt 分析时间
	AnalyzedAt time.Time `json:"analyzed_at"`
}

/**
 * GetType 获取模型类型
 *
 * Returns: ModelType - 返回 ModelTypeClaude
 */
func (c *ClaudeClient) GetType() ModelType {
	return ModelTypeClaude
}

/**
 * Close 关闭连接
 *
 * Returns: error - 关闭错误
 */
func (c *ClaudeClient) Close() error {
	// Eino ChatModel 不需要显式关闭
	// 这里保留接口以备将来需要
	logger.Info("Claude 客户端已关闭")
	return nil
}
