/**
 * Package ai AI 服务基础设施层
 *
 * 提示词模板和响应解析
 */

package ai

import (
	"encoding/json"
	"fmt"
	"strings"
)

/**
 * BuildPatternAnalysisPrompt 构建模式分析提示词
 *
 * Parameters:
 *   - patternData: 模式数据
 *
 * Returns: string - 提示词
 */
func BuildPatternAnalysisPrompt(patternData map[string]interface{}) string {
	// 将模式数据转换为可读的字符串
	patternJSON, _ := json.MarshalIndent(patternData, "", "  ")

	prompt := `请分析以下用户操作模式，判断是否值得自动化。

## 模式信息

` + string(patternJSON) + `

## 分析维度

请从以下维度评估该模式：

1. **频率**：模式出现的频率（支持计数、每小时出现次数）
2. **时间节省**：如果自动化，每次可节省的时间（秒）
3. **复杂度**：实现自动化的技术难度（low/medium/high）
4. **可行性**：技术实现的可行性
5. **价值**：对用户体验的提升程度

## 判断标准

**值得自动化**的情况：
- 高频率（每天多次）
- 明显的时间节省（每次 > 10秒）
- 技术可行性高
- 实现复杂度合理（low 或 medium）

**不值得自动化**的情况：
- 低频率（每周少于 1 次）
- 时间节省不明显（< 5秒）
- 技术实现困难或复杂度高（high）
- 用户可能需要灵活调整的操作

## 输出格式

请严格按照以下 JSON 格式返回分析结果：

{
  "should_automate": true,
  "reason": "简明扼要的原因说明（1-2句话）",
  "estimated_time_saving": 30,
  "complexity": "low",
  "suggested_name": "自动化建议名称",
  "suggested_steps": [
    "步骤1描述",
    "步骤2描述",
    "步骤3描述"
  ]
}

## 字段说明

- should_automate: boolean - 是否值得自动化
- reason: string - 原因说明（中文）
- estimated_time_saving: number - 预计每次节省的时间（秒）
- complexity: string - 实现复杂度，必须是 "low"、"medium" 或 "high" 之一
- suggested_name: string - 建议的自动化名称（简洁明了）
- suggested_steps: array<string> - 建议的自动化步骤列表

请基于以上信息，返回 JSON 格式的分析结果：`

	return prompt
}

/**
 * FormatPatternForAnalysis 格式化模式数据用于分析
 *
 * Parameters:
 *   - patternID: 模式 ID
 *   - sequence: 事件步骤序列
 *   - supportCount: 支持计数
 *   - confidence: 置信度
 *   - frequency: 频率（每小时出现次数）
 *   - description: 模式描述
 *
 * Returns: map[string]interface{} - 格式化后的模式数据
 */
func FormatPatternForAnalysis(
	patternID string,
	sequence []EventStepInfo,
	supportCount int,
	confidence float64,
	frequency float64,
	description string,
) map[string]interface{} {
	// 构建步骤摘要
	stepSummary := make([]string, len(sequence))
	for i, step := range sequence {
		stepSummary[i] = fmt.Sprintf("%s: %s", step.Type, step.Action)
	}

	return map[string]interface{}{
		"pattern_id":     patternID,
		"sequence":       sequence,
		"step_summary":   strings.Join(stepSummary, " → "),
		"support_count":  supportCount,
		"confidence":     confidence,
		"frequency_hour": frequency,
		"description":    description,
		"length":         len(sequence),
	}
}

/**
 * EventStepInfo 事件步骤信息（用于序列化）
 */
type EventStepInfo struct {
	// Type 事件类型
	Type string `json:"type"`

	// Action 动作
	Action string `json:"action"`

	// Context 上下文（可选）
	Context *StepContextInfo `json:"context,omitempty"`
}

/**
 * StepContextInfo 步骤上下文信息
 */
type StepContextInfo struct {
	// Application 应用名称
	Application string `json:"application,omitempty"`

	// BundleID Bundle ID
	BundleID string `json:"bundle_id,omitempty"`

	// PatternValue 模式值
	PatternValue string `json:"pattern_value,omitempty"`
}
