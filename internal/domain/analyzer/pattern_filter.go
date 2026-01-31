/**
 * Package analyzer 模式识别引擎的分析组件
 *
 * AI 模式过滤器，使用 Claude API 评估模式价值
 */

package analyzer

import (
	"context"
	"fmt"
	"time"

	"github.com/chenyang-zz/flowmind/internal/domain/models"
	"github.com/chenyang-zz/flowmind/internal/infrastructure/ai"
	"github.com/chenyang-zz/flowmind/internal/infrastructure/cache"
	"github.com/chenyang-zz/flowmind/internal/infrastructure/logger"
	"go.uber.org/zap"
)

/**
 * AIPatternFilterConfig AI 模式过滤器配置
 */
type AIPatternFilterConfig struct {
	// AIModel AI 模型客户端（支持 Claude、智谱AI、OpenAI 等）
	AIModel ai.AIModel

	// CacheEnabled 是否启用缓存
	CacheEnabled bool

	// CacheTTL 缓存过期时间
	CacheTTL time.Duration

	// MaxConcurrent 最大并发数
	MaxConcurrent int
}

/**
 * DefaultAIPatternFilterConfig 默认配置
 */
func DefaultAIPatternFilterConfig() AIPatternFilterConfig {
	return AIPatternFilterConfig{
		CacheEnabled:  true,
		CacheTTL:      24 * time.Hour,
		MaxConcurrent: 3,
	}
}

/**
 * AIPatternFilter AI 模式过滤器
 *
 * 使用 AI（Claude/智谱AI/OpenAI 等）评估模式是否值得自动化
 */
type AIPatternFilter struct {
	config  AIPatternFilterConfig
	aiModel ai.AIModel
	cache   cache.Cache // AI 分析结果缓存
}

/**
 * NewAIPatternFilter 创建 AI 模式过滤器
 *
 * Parameters:
 *   - config: 过滤器配置
 *
 * Returns: *AIPatternFilter - AI 模式过滤器实例
 */
func NewAIPatternFilter(config AIPatternFilterConfig) (*AIPatternFilter, error) {
	if config.AIModel == nil {
		return nil, fmt.Errorf("AI 模型客户端不能为空")
	}

	// 设置默认值
	if config.MaxConcurrent == 0 {
		config.MaxConcurrent = 3
	}
	if config.CacheTTL == 0 {
		config.CacheTTL = 24 * time.Hour
	}

	// 创建缓存实例
	var cacheInstance cache.Cache
	if config.CacheEnabled {
		// 使用内存缓存，默认24小时TTL
		cacheInstance = cache.NewMemoryCache(
			1000,  // 最多1000个缓存项
			10*time.Minute,  // 每10分钟清理一次
		)
		logger.Info("AI 分析缓存已启用",
			zap.Duration("ttl", config.CacheTTL))
	}

	return &AIPatternFilter{
		config:  config,
		aiModel: config.AIModel,
		cache:   cacheInstance,
	}, nil
}

/**
 * ShouldAutomate 判断模式是否值得自动化
 *
 * Parameters:
 *   - ctx: 上下文
 *   - pattern: 模式对象
 *
 * Returns: *models.AIAnalysis - AI 分析结果
 */
func (f *AIPatternFilter) ShouldAutomate(ctx context.Context, pattern *models.Pattern) (*models.AIAnalysis, error) {
	// 检查是否已分析
	if pattern.AIAnalysis != nil {
		return pattern.AIAnalysis, nil
	}

	// 检查缓存
	if f.cache != nil {
		cacheKey := f.buildCacheKey(pattern)
		if cached, found := f.cache.Get(cacheKey); found {
			if analysis, ok := cached.(*models.AIAnalysis); ok {
				logger.Info("从缓存获取 AI 分析结果",
					zap.String("pattern_id", pattern.ID))
				return analysis, nil
			}
		}
	}

	logger.Info("开始 AI 分析模式",
		zap.String("pattern_id", pattern.ID),
		zap.Int("support_count", pattern.SupportCount),
		zap.Float64("confidence", pattern.Confidence))

	// 格式化模式数据
	sequence := make([]ai.EventStepInfo, len(pattern.Sequence))
	for i, step := range pattern.Sequence {
		stepInfo := ai.EventStepInfo{
			Type:   string(step.Type),
			Action: step.Action,
		}
		// 如果 Context 不为空，添加上下文信息
		if step.Context != nil {
			stepInfo.Context = &ai.StepContextInfo{
				Application:  step.Context.Application,
				BundleID:     step.Context.BundleID,
				PatternValue: step.Context.PatternValue,
			}
		}
		sequence[i] = stepInfo
	}

	patternData := ai.FormatPatternForAnalysis(
		pattern.ID,
		sequence,
		pattern.SupportCount,
		pattern.Confidence,
		pattern.Frequency(),
		pattern.Description,
	)

	// 调用 AI API
	analysisResult, err := f.aiModel.AnalyzePattern(ctx, patternData)
	if err != nil {
		logger.Error("AI 分析失败",
			zap.String("pattern_id", pattern.ID),
			zap.Error(err))
		return nil, fmt.Errorf("AI 分析失败: %w", err)
	}

	// 转换为 AIAnalysis
	aiAnalysis := &models.AIAnalysis{
		ShouldAutomate:      analysisResult.ShouldAutomate,
		Reason:              analysisResult.Reason,
		EstimatedTimeSaving: analysisResult.EstimatedTimeSaving,
		Complexity:          analysisResult.Complexity,
		SuggestedName:       analysisResult.SuggestedName,
		SuggestedSteps:      analysisResult.SuggestedSteps,
		AnalyzedAt:          analysisResult.AnalyzedAt,
	}

	logger.Info("AI 分析完成",
		zap.String("pattern_id", pattern.ID),
		zap.Bool("should_automate", aiAnalysis.ShouldAutomate),
		zap.String("reason", aiAnalysis.Reason),
		zap.String("complexity", aiAnalysis.Complexity))

	// 保存到缓存
	if f.cache != nil {
		cacheKey := f.buildCacheKey(pattern)
		if err := f.cache.Set(cacheKey, aiAnalysis, f.config.CacheTTL); err != nil {
			logger.Warn("缓存设置失败",
				zap.String("pattern_id", pattern.ID),
				zap.Error(err))
		} else {
			logger.Debug("AI 分析结果已缓存",
				zap.String("pattern_id", pattern.ID),
				zap.Duration("ttl", f.config.CacheTTL))
		}
	}

	return aiAnalysis, nil
}

/**
 * ShouldAutomateBatch 批量分析模式
 *
 * Parameters:
 *   - ctx: 上下文
 *   - patterns: 模式列表
 *
 * Returns: map[string]*models.AIAnalysis - 模式 ID 到分析结果的映射
 */
func (f *AIPatternFilter) ShouldAutomateBatch(ctx context.Context, patterns []*models.Pattern) (map[string]*models.AIAnalysis, error) {
	results := make(map[string]*models.AIAnalysis)

	// 过滤出未分析的模式
	unanalyzed := make([]*models.Pattern, 0)
	for _, pattern := range patterns {
		if pattern.AIAnalysis == nil {
			unanalyzed = append(unanalyzed, pattern)
		} else {
			results[pattern.ID] = pattern.AIAnalysis
		}
	}

	if len(unanalyzed) == 0 {
		return results, nil
	}

	logger.Info("开始批量 AI 分析",
		zap.Int("total", len(patterns)),
		zap.Int("unanalyzed", len(unanalyzed)))

	// 准备批量分析数据
	patternsData := make([]map[string]interface{}, len(unanalyzed))
	for i, pattern := range unanalyzed {
		sequence := make([]ai.EventStepInfo, len(pattern.Sequence))
		for j, step := range pattern.Sequence {
			stepInfo := ai.EventStepInfo{
				Type:   string(step.Type),
				Action: step.Action,
			}
			// 如果 Context 不为空，添加上下文信息
			if step.Context != nil {
				stepInfo.Context = &ai.StepContextInfo{
					Application:  step.Context.Application,
					BundleID:     step.Context.BundleID,
					PatternValue: step.Context.PatternValue,
				}
			}
			sequence[j] = stepInfo
		}

		patternsData[i] = ai.FormatPatternForAnalysis(
			pattern.ID,
			sequence,
			pattern.SupportCount,
			pattern.Confidence,
			pattern.Frequency(),
			pattern.Description,
		)
	}

	// 调用批量分析
	batchResults, err := f.aiModel.AnalyzePatternBatch(ctx, patternsData)
	if err != nil {
		logger.Error("批量 AI 分析失败", zap.Error(err))
		return nil, fmt.Errorf("批量 AI 分析失败: %w", err)
	}

	// 转换结果
	for i, result := range batchResults {
		pattern := unanalyzed[i]
		aiAnalysis := &models.AIAnalysis{
			ShouldAutomate:      result.ShouldAutomate,
			Reason:              result.Reason,
			EstimatedTimeSaving: result.EstimatedTimeSaving,
			Complexity:          result.Complexity,
			SuggestedName:       result.SuggestedName,
			SuggestedSteps:      result.SuggestedSteps,
			AnalyzedAt:          result.AnalyzedAt,
		}
		results[pattern.ID] = aiAnalysis

		logger.Info("模式分析完成",
			zap.String("pattern_id", pattern.ID),
			zap.Bool("should_automate", aiAnalysis.ShouldAutomate),
			zap.String("reason", aiAnalysis.Reason))
	}

	logger.Info("批量 AI 分析完成",
		zap.Int("analyzed", len(batchResults)))

	return results, nil
}

/**
 * FilterValuablePatterns 过滤出值得自动化的模式
 *
 * Parameters:
 *   - ctx: 上下文
 *   - patterns: 模式列表
 *
 * Returns: []*models.Pattern - 值得自动化的模式列表
 */
func (f *AIPatternFilter) FilterValuablePatterns(ctx context.Context, patterns []*models.Pattern) ([]*models.Pattern, error) {
	var valuablePatterns []*models.Pattern

	for _, pattern := range patterns {
		analysis, err := f.ShouldAutomate(ctx, pattern)
		if err != nil {
			logger.Warn("分析模式失败，跳过",
				zap.String("pattern_id", pattern.ID),
				zap.Error(err))
			continue
		}

		if analysis.ShouldAutomate {
			valuablePatterns = append(valuablePatterns, pattern)
		}
	}

	logger.Info("过滤完成",
		zap.Int("total", len(patterns)),
		zap.Int("valuable", len(valuablePatterns)))

	return valuablePatterns, nil
}

/**
 * GetAnalysisSummary 获取分析摘要
 *
 * Parameters:
 *   - analysis: AI 分析结果
 *
 * Returns: string - 分析摘要
 */
func (f *AIPatternFilter) GetAnalysisSummary(analysis *models.AIAnalysis) string {
	summary := fmt.Sprintf("建议自动化: %v\n", analysis.ShouldAutomate)
	summary += fmt.Sprintf("原因: %s\n", analysis.Reason)
	summary += fmt.Sprintf("预计节省时间: %d 秒\n", analysis.EstimatedTimeSaving)
	summary += fmt.Sprintf("复杂度: %s\n", analysis.Complexity)
	if analysis.SuggestedName != "" {
		summary += fmt.Sprintf("建议名称: %s\n", analysis.SuggestedName)
	}
	if len(analysis.SuggestedSteps) > 0 {
		summary += "建议步骤:\n"
		for i, step := range analysis.SuggestedSteps {
			summary += fmt.Sprintf("  %d. %s\n", i+1, step)
		}
	}
	summary += fmt.Sprintf("分析时间: %s\n", analysis.AnalyzedAt.Format("2006-01-02 15:04:05"))

	return summary
}

/**
 * buildCacheKey 构建缓存键
 *
 * 使用模式序列的字符串表示和支持度作为缓存键，
 * 确保相同的模式序列能够命中缓存
 *
 * Parameters:
 *   - pattern: 模式对象
 *
 * Returns: string - 缓存键
 */
func (f *AIPatternFilter) buildCacheKey(pattern *models.Pattern) string {
	// 将模式序列转换为字符串表示
	var sequenceStr string
	for _, step := range pattern.Sequence {
		sequenceStr += string(step.Type) + ":" + step.Action + ":"
		if step.Context != nil && step.Context.Application != "" {
			sequenceStr += step.Context.Application
		}
		sequenceStr += "|"
	}

	// 使用模式序列和支持度生成缓存键
	return fmt.Sprintf("pattern_analysis:%s:%d", sequenceStr, pattern.SupportCount)
}

/**
 * Close 关闭过滤器并释放资源
 *
 * 停止缓存等资源
 */
func (f *AIPatternFilter) Close() error {
	if f.cache != nil {
		f.cache.Stop()
		logger.Info("AI 模式过滤器已关闭")
	}
	return nil
}
