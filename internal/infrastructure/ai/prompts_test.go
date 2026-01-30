/**
 * Package ai AI 服务基础设施层
 *
 * 提示词模板单元测试
 */

package ai

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestBuildPatternAnalysisPrompt 测试构建模式分析提示词
func TestBuildPatternAnalysisPrompt(t *testing.T) {
	patternData := map[string]interface{}{
		"pattern_id":     "test-pattern-123",
		"support_count":  15,
		"confidence":     0.85,
		"frequency_hour": 2.5,
		"description":    "这是一个测试模式",
		"length":         5,
	}

	prompt := BuildPatternAnalysisPrompt(patternData)

	// 验证提示词包含关键信息
	assert.Contains(t, prompt, "模式信息")
	assert.Contains(t, prompt, "test-pattern-123")
	assert.Contains(t, prompt, "15")
	assert.Contains(t, prompt, "0.85")
	assert.Contains(t, prompt, "2.5")
	assert.Contains(t, prompt, "这是一个测试模式")

	// 验证提示词包含分析维度
	assert.Contains(t, prompt, "频率")
	assert.Contains(t, prompt, "时间节省")
	assert.Contains(t, prompt, "复杂度")
	assert.Contains(t, prompt, "可行性")

	// 验证提示词包含输出格式说明
	assert.Contains(t, prompt, "should_automate")
	assert.Contains(t, prompt, "reason")
	assert.Contains(t, prompt, "estimated_time_saving")
	assert.Contains(t, prompt, "complexity")
	assert.Contains(t, prompt, "suggested_name")
	assert.Contains(t, prompt, "suggested_steps")
}

// TestFormatPatternForAnalysis 测试格式化模式数据
func TestFormatPatternForAnalysis(t *testing.T) {
	sequence := []EventStepInfo{
		{
			Type:   "keyboard",
			Action: "keypress",
			Context: &StepContextInfo{
				Application:  "VSCode",
				PatternValue: "letter",
			},
		},
		{
			Type:   "clipboard",
			Action: "copy",
			Context: &StepContextInfo{
				Application: "VSCode",
			},
		},
	}

	result := FormatPatternForAnalysis(
		"pattern-123",
		sequence,
		10,
		0.8,
		2.5,
		"测试模式描述",
	)

	// 验证返回的数据结构
	assert.Equal(t, "pattern-123", result["pattern_id"])
	assert.Equal(t, sequence, result["sequence"])
	assert.Equal(t, 10, result["support_count"])
	assert.Equal(t, 0.8, result["confidence"])
	assert.Equal(t, 2.5, result["frequency_hour"])
	assert.Equal(t, "测试模式描述", result["description"])
	assert.Equal(t, 2, result["length"])

	// 验证 step_summary 包含正确的序列描述
	stepSummary, ok := result["step_summary"].(string)
	assert.True(t, ok)
	// step_summary 应该包含步骤信息
	assert.NotEmpty(t, stepSummary)
	t.Logf("Step summary: %s", stepSummary)
}

// TestFormatPatternForAnalysis_EmptySequence 测试空序列
func TestFormatPatternForAnalysis_EmptySequence(t *testing.T) {
	sequence := []EventStepInfo{}

	result := FormatPatternForAnalysis(
		"pattern-empty",
		sequence,
		0,
		0.0,
		0.0,
		"",
	)

	// 验证空序列也能正确处理
	assert.Equal(t, "pattern-empty", result["pattern_id"])
	assert.Equal(t, sequence, result["sequence"])
	assert.Equal(t, 0, result["length"])
}

// TestEventStepInfoSerialization 测试EventStepInfo序列化
func TestEventStepInfoSerialization(t *testing.T) {
	step := EventStepInfo{
		Type:   "keyboard",
		Action: "function_key",
		Context: &StepContextInfo{
			Application:  "Chrome",
			BundleID:     "com.google.Chrome",
			PatternValue: "F5",
		},
	}

	// 验证字段可以正确访问
	assert.Equal(t, "keyboard", step.Type)
	assert.Equal(t, "function_key", step.Action)
	assert.NotNil(t, step.Context)
	assert.Equal(t, "Chrome", step.Context.Application)
	assert.Equal(t, "com.google.Chrome", step.Context.BundleID)
	assert.Equal(t, "F5", step.Context.PatternValue)
}

// TestEventStepInfo_EmptyContext 测试空上下文
func TestEventStepInfo_EmptyContext(t *testing.T) {
	step := EventStepInfo{
		Type:   "clipboard",
		Action: "paste",
		Context: nil,
	}

	// 验证空上下文不会导致错误
	assert.Equal(t, "clipboard", step.Type)
	assert.Equal(t, "paste", step.Action)
	assert.Nil(t, step.Context)
}

// TestStepContextInfo_AllFields 测试所有字段
func TestStepContextInfo_AllFields(t *testing.T) {
	context := &StepContextInfo{
		Application:  "VSCode",
		BundleID:     "com.microsoft.VSCode",
		PatternValue: "cmd+s",
	}

	assert.Equal(t, "VSCode", context.Application)
	assert.Equal(t, "com.microsoft.VSCode", context.BundleID)
	assert.Equal(t, "cmd+s", context.PatternValue)
}
