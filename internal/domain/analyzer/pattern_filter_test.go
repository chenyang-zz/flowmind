/**
 * Package analyzer 模式识别引擎的分析组件
 *
 * AI 模式过滤器单元测试
 */

package analyzer

import (
	"context"
	"testing"
	"time"

	"github.com/chenyang-zz/flowmind/internal/domain/models"
	"github.com/chenyang-zz/flowmind/internal/infrastructure/ai"
	"github.com/chenyang-zz/flowmind/pkg/events"
	"github.com/stretchr/testify/assert"
)

// MockAIModel 模拟 AI 模型（用于测试）
type MockAIModel struct {
	analyzeFunc func(context.Context, map[string]interface{}) (*ai.PatternAnalysis, error)
}

func (m *MockAIModel) AnalyzePattern(ctx context.Context, patternData map[string]interface{}) (*ai.PatternAnalysis, error) {
	if m.analyzeFunc != nil {
		return m.analyzeFunc(ctx, patternData)
	}
	// 默认返回成功响应
	return &ai.PatternAnalysis{
		ShouldAutomate:      true,
		Reason:              "模拟分析结果",
		EstimatedTimeSaving: 60,
		Complexity:          "low",
		SuggestedName:       "模拟自动化",
		SuggestedSteps:      []string{"步骤1", "步骤2"},
		AnalyzedAt:          time.Now(),
	}, nil
}

func (m *MockAIModel) AnalyzePatternBatch(ctx context.Context, patterns []map[string]interface{}) ([]*ai.PatternAnalysis, error) {
	results := make([]*ai.PatternAnalysis, len(patterns))
	for i := range patterns {
		analysis, err := m.AnalyzePattern(ctx, patterns[i])
		if err != nil {
			return nil, err
		}
		results[i] = analysis
	}
	return results, nil
}

func (m *MockAIModel) GetType() ai.ModelType {
	return "mock"
}

func (m *MockAIModel) Close() error {
	return nil
}

// TestNewAIPatternFilter 测试创建 AI 模式过滤器
func TestNewAIPatternFilter(t *testing.T) {
	mockModel := &MockAIModel{}

	config := AIPatternFilterConfig{
		AIModel:      mockModel,
		CacheEnabled: true,
		CacheTTL:     24 * time.Hour,
		MaxConcurrent: 3,
	}

	filter, err := NewAIPatternFilter(config)

	assert.NoError(t, err)
	assert.NotNil(t, filter)
	assert.Equal(t, config.AIModel, filter.aiModel)
	assert.Equal(t, config.CacheEnabled, filter.config.CacheEnabled)
	assert.Equal(t, config.CacheTTL, filter.config.CacheTTL)
	assert.Equal(t, config.MaxConcurrent, filter.config.MaxConcurrent)
}

// TestNewAIPatternFilter_NoAIModel 测试没有 AI 模型的错误
func TestNewAIPatternFilter_NoAIModel(t *testing.T) {
	config := AIPatternFilterConfig{
		AIModel: nil,
	}

	filter, err := NewAIPatternFilter(config)

	assert.Error(t, err)
	assert.Nil(t, filter)
	assert.Contains(t, err.Error(), "AI 模型客户端不能为空")
}

// TestNewAIPatternFilter_DefaultConfig 测试默认配置
func TestNewAIPatternFilter_DefaultConfig(t *testing.T) {
	mockModel := &MockAIModel{}
	config := DefaultAIPatternFilterConfig()
	config.AIModel = mockModel

	filter, err := NewAIPatternFilter(config)

	assert.NoError(t, err)
	assert.NotNil(t, filter)
	assert.Equal(t, 24*time.Hour, filter.config.CacheTTL)
	assert.Equal(t, 3, filter.config.MaxConcurrent)
}

// TestAIPatternFilter_ShouldAutomate 测试分析模式
func TestAIPatternFilter_ShouldAutomate(t *testing.T) {
	// 创建模拟 AI 模型
	mockModel := &MockAIModel{
		analyzeFunc: func(ctx context.Context, patternData map[string]interface{}) (*ai.PatternAnalysis, error) {
			return &ai.PatternAnalysis{
				ShouldAutomate:      true,
				Reason:              "高频操作，值得自动化",
				EstimatedTimeSaving: 45,
				Complexity:          "medium",
				SuggestedName:       "快捷操作",
				SuggestedSteps:      []string{"检测触发", "执行操作"},
				AnalyzedAt:          time.Now(),
			}, nil
		},
	}

	// 创建过滤器
	config := AIPatternFilterConfig{AIModel: mockModel}
	filter, err := NewAIPatternFilter(config)
	assert.NoError(t, err)

	// 创建测试模式
	pattern := &models.Pattern{
		ID:           "test-pattern-1",
		SupportCount: 10,
		Confidence:   0.85,
		FirstSeen:    time.Now().Add(-24 * time.Hour),
		LastSeen:     time.Now(),
		Sequence: []models.EventStep{
			{Type: events.EventTypeKeyboard, Action: "keypress"},
			{Type: events.EventTypeClipboard, Action: "copy"},
		},
		Description: "测试模式",
	}

	// 分析模式
	analysis, err := filter.ShouldAutomate(context.Background(), pattern)

	assert.NoError(t, err)
	assert.NotNil(t, analysis)
	assert.True(t, analysis.ShouldAutomate)
	assert.Equal(t, "高频操作，值得自动化", analysis.Reason)
	assert.Equal(t, int64(45), analysis.EstimatedTimeSaving)
	assert.Equal(t, "medium", analysis.Complexity)
	assert.Equal(t, "快捷操作", analysis.SuggestedName)
	assert.Len(t, analysis.SuggestedSteps, 2)
}

// TestAIPatternFilter_ShouldAutomate_AlreadyAnalyzed 测试已分析的模式
func TestAIPatternFilter_ShouldAutomate_AlreadyAnalyzed(t *testing.T) {
	mockModel := &MockAIModel{}

	config := AIPatternFilterConfig{AIModel: mockModel}
	filter, err := NewAIPatternFilter(config)
	assert.NoError(t, err)

	// 创建已有分析结果的模式
	existingAnalysis := &models.AIAnalysis{
		ShouldAutomate:      true,
		Reason:              "已存在的分析",
		EstimatedTimeSaving: 30,
		Complexity:          "low",
	}

	pattern := &models.Pattern{
		ID:         "test-pattern-2",
		AIAnalysis: existingAnalysis,
	}

	// 分析模式
	analysis, err := filter.ShouldAutomate(context.Background(), pattern)

	assert.NoError(t, err)
	assert.Equal(t, existingAnalysis, analysis)
	assert.Equal(t, "已存在的分析", analysis.Reason)
}

// TestAIPatternFilter_ShouldAutomateBatch 测试批量分析
func TestAIPatternFilter_ShouldAutomateBatch(t *testing.T) {
	// 创建模拟 AI 模型
	callCount := 0
	mockModel := &MockAIModel{
		analyzeFunc: func(ctx context.Context, patternData map[string]interface{}) (*ai.PatternAnalysis, error) {
			callCount++
			return &ai.PatternAnalysis{
				ShouldAutomate: callCount%2 == 0, // 交替返回 true/false
				Reason:        "批量分析结果",
				AnalyzedAt:     time.Now(),
			}, nil
		},
	}

	config := AIPatternFilterConfig{AIModel: mockModel}
	filter, err := NewAIPatternFilter(config)
	assert.NoError(t, err)

	// 创建测试模式列表
	patterns := []*models.Pattern{
		{ID: "pattern-1", SupportCount: 5},
		{ID: "pattern-2", SupportCount: 8},
		{ID: "pattern-3", SupportCount: 12},
	}

	// 批量分析
	results, err := filter.ShouldAutomateBatch(context.Background(), patterns)

	assert.NoError(t, err)
	assert.Len(t, results, 3)
	assert.Equal(t, 3, callCount) // 应该调用3次

	// 验证每个模式都有结果
	for patternID, analysis := range results {
		assert.NotNil(t, analysis)
		assert.NotEmpty(t, patternID)
	}
}

// TestAIPatternFilter_FilterValuablePatterns 测试过滤有价值模式
func TestAIPatternFilter_FilterValuablePatterns(t *testing.T) {
	mockModel := &MockAIModel{
		analyzeFunc: func(ctx context.Context, patternData map[string]interface{}) (*ai.PatternAnalysis, error) {
			// 处理 support_count 可能是 int 或 float64 的情况
			var supportCount int
			switch v := patternData["support_count"].(type) {
			case int:
				supportCount = v
			case float64:
				supportCount = int(v)
			}
			// 支持度>=8的才值得自动化
			return &ai.PatternAnalysis{
				ShouldAutomate: supportCount >= 8,
				Reason:        "根据支持度判断",
				AnalyzedAt:     time.Now(),
			}, nil
		},
	}

	config := AIPatternFilterConfig{AIModel: mockModel}
	filter, err := NewAIPatternFilter(config)
	assert.NoError(t, err)

	// 创建测试模式
	patterns := []*models.Pattern{
		{ID: "pattern-1", SupportCount: 5},
		{ID: "pattern-2", SupportCount: 8},
		{ID: "pattern-3", SupportCount: 12},
		{ID: "pattern-4", SupportCount: 3},
	}

	// 过滤
	valuable, err := filter.FilterValuablePatterns(context.Background(), patterns)

	assert.NoError(t, err)
	assert.Len(t, valuable, 2) // pattern-2 和 pattern-3

	// 验证返回的都是有价值的模式
	for _, pattern := range valuable {
		assert.GreaterOrEqual(t, pattern.SupportCount, 8)
	}
}

// TestAIPatternFilter_GetAnalysisSummary 测试获取分析摘要
func TestAIPatternFilter_GetAnalysisSummary(t *testing.T) {
	mockModel := &MockAIModel{}
	config := AIPatternFilterConfig{AIModel: mockModel}
	filter, err := NewAIPatternFilter(config)
	assert.NoError(t, err)

	analysis := &models.AIAnalysis{
		ShouldAutomate:      true,
		Reason:              "测试原因",
		EstimatedTimeSaving: 120,
		Complexity:          "low",
		SuggestedName:       "测试名称",
		SuggestedSteps:      []string{"步骤1", "步骤2", "步骤3"},
		AnalyzedAt:          time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC),
	}

	summary := filter.GetAnalysisSummary(analysis)

	assert.Contains(t, summary, "建议自动化: true")
	assert.Contains(t, summary, "原因: 测试原因")
	assert.Contains(t, summary, "预计节省时间: 120 秒")
	assert.Contains(t, summary, "复杂度: low")
	assert.Contains(t, summary, "建议名称: 测试名称")
	assert.Contains(t, summary, "步骤1")
	assert.Contains(t, summary, "步骤2")
	assert.Contains(t, summary, "步骤3")
	assert.Contains(t, summary, "2026-01-30")
}
