/**
 * Package ai AI 服务基础设施层
 *
 * 提供多模型支持的 AI 客户端接口
 */

package ai

import (
	"context"
)

/**
 * ModelType 模型类型
 */
type ModelType string

const (
	// ModelTypeClaude Claude 模型
	ModelTypeClaude ModelType = "claude"

	// ModelTypeOpenAI OpenAI 模型
	ModelTypeOpenAI ModelType = "openai"

	// ModelTypeZhipu 智谱AI (ChatGLM)
	ModelTypeZhipu ModelType = "zhipu"

	// ModelTypeOllama Ollama 本地模型
	ModelTypeOllama ModelType = "ollama"
)

/**
 * AIModelConfig AI 模型配置接口
 */
type AIModelConfig interface {
	// GetType 获取模型类型
	GetType() ModelType

	// Validate 验证配置
	Validate() error
}

/**
 * AIModel AI 模型接口
 *
 * 定义 AI 模型的通用能力
 */
type AIModel interface {
	// AnalyzePattern 分析模式
	AnalyzePattern(ctx context.Context, patternData map[string]interface{}) (*PatternAnalysis, error)

	// AnalyzePatternBatch 批量分析模式
	AnalyzePatternBatch(ctx context.Context, patterns []map[string]interface{}) ([]*PatternAnalysis, error)

	// GetType 获取模型类型
	GetType() ModelType

	// Close 关闭连接
	Close() error
}
