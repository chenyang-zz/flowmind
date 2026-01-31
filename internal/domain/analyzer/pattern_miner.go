/**
 * Package analyzer 模式识别引擎的分析组件
 *
 * 负责会话划分、事件标准化、模式挖掘等核心功能
 */

package analyzer

import (
	"fmt"
	"time"

	"github.com/chenyang-zz/flowmind/internal/domain/models"
	"github.com/chenyang-zz/flowmind/pkg/events"
)

/**
 * PatternMinerConfig 模式挖掘器配置
 */
type PatternMinerConfig struct {
	// NormalizerConfig 事件标准化器配置
	NormalizerConfig EventNormalizerConfig

	// PrefixSpanConfig PrefixSpan算法配置
	PrefixSpanConfig PrefixSpanConfig
}

/**
 * DefaultPatternMinerConfig 默认配置
 */
func DefaultPatternMinerConfig() PatternMinerConfig {
	return PatternMinerConfig{
		NormalizerConfig: DefaultEventNormalizerConfig(),
		PrefixSpanConfig: DefaultPrefixSpanConfig(),
	}
}

/**
 * PatternMiner 模式挖掘器
 *
 * 整合事件标准化和模式挖掘功能，提供统一的接口
 */
type PatternMiner struct {
	config      PatternMinerConfig
	normalizer  *EventNormalizer
	prefixSpan  *PrefixSpan
}

/**
 * NewPatternMiner 创建模式挖掘器
 *
 * Parameters:
 *   - config: 挖掘器配置
 *
 * Returns: *PatternMiner - 模式挖掘器实例
 */
func NewPatternMiner(config PatternMinerConfig) *PatternMiner {
	return &PatternMiner{
		config:     config,
		normalizer: NewEventNormalizer(config.NormalizerConfig),
		prefixSpan: NewPrefixSpan(config.PrefixSpanConfig),
	}
}

/**
 * MineFromSessions 从会话列表中挖掘模式
 *
 * 这是主要的模式挖掘接口，完成以下步骤：
 * 1. 使用 SessionDivider 划分会话（如果需要）
 * 2. 标准化事件序列
 * 3. 使用 PrefixSpan 挖掘频繁模式
 * 4. 计算模式统计信息
 *
 * Parameters:
 *   - sessions: 会话列表
 *
 * Returns: []*models.Pattern - 发现的模式列表
 */
func (pm *PatternMiner) MineFromSessions(sessions []*models.Session) ([]*models.Pattern, error) {
	if len(sessions) == 0 {
		return []*models.Pattern{}, nil
	}

	// 使用 PrefixSpan 挖掘模式
	patterns, err := pm.prefixSpan.Mine(sessions)
	if err != nil {
		return nil, fmt.Errorf("模式挖掘失败: %w", err)
	}

	// 为每个模式添加额外的统计信息
	for _, pattern := range patterns {
		pm.enrichPattern(pattern, sessions)
	}

	return patterns, nil
}

/**
 * MineFromEvents 从原始事件列表中挖掘模式
 *
 * 便捷方法，自动处理会话划分和模式挖掘
 *
 * Parameters:
 *   - events: 原始事件列表
 *   - sessionDividerConfig: 会话划分配置（可选，为空则使用默认配置）
 *
 * Returns: []*models.Pattern - 发现的模式列表
 */
func (pm *PatternMiner) MineFromEvents(
	events []events.Event,
	sessionDividerConfig *SessionDividerConfig,
) ([]*models.Pattern, error) {
	if len(events) == 0 {
		return []*models.Pattern{}, nil
	}

	// 1. 划分会话
	var divider *SessionDivider
	if sessionDividerConfig != nil {
		divider = NewSessionDivider(*sessionDividerConfig)
	} else {
		divider = NewSessionDivider(DefaultSessionDividerConfig())
	}

	sessions := divider.Divide(events)

	// 2. 挖掘模式
	return pm.MineFromSessions(sessions)
}

/**
 * enrichPattern 丰富模式信息
 *
 * 为模式添加额外的统计信息和元数据
 *
 * Parameters:
 *   - pattern: 模式对象
 *   - sessions: 原始会话列表
 */
func (pm *PatternMiner) enrichPattern(pattern *models.Pattern, sessions []*models.Session) {
	// 计算模式的平均时间间隔
	avgInterval := pm.calculateAverageInterval(pattern, sessions)

	// 提取模式的应用上下文
	applications := pm.extractApplications(pattern, sessions)

	// 生成模式描述
	pattern.Description = pm.generateDescription(pattern, applications, avgInterval)
}

/**
 * calculateAverageInterval 计算模式的平均时间间隔
 *
 * Parameters:
 *   - pattern: 模式对象
 *   - sessions: 会话列表
 *
 * Returns: time.Duration - 平均时间间隔
 */
func (pm *PatternMiner) calculateAverageInterval(
	pattern *models.Pattern,
	sessions []*models.Session,
) time.Duration {
	var totalInterval time.Duration
	var count int

	for _, session := range sessions {
		// 查找模式在会话中的出现
		occurrences := pm.findPatternOccurrences(pattern, session.Events)
		count += len(occurrences)

		// 计算间隔
		for i := 1; i < len(occurrences); i++ {
			interval := occurrences[i].Sub(occurrences[i-1])
			totalInterval += interval
		}
	}

	if count == 0 {
		return 0
	}

	return totalInterval / time.Duration(count)
}

/**
 * findPatternOccurrences 查找模式在事件列表中的所有出现
 *
 * Parameters:
 *   - pattern: 模式对象
 *   - events: 事件列表
 *
 * Returns: []time.Time - 出现的时间点列表
 */
func (pm *PatternMiner) findPatternOccurrences(
	pattern *models.Pattern,
	events []events.Event,
) []time.Time {
	var occurrences []time.Time

	// 标准化事件
	steps := pm.normalizer.NormalizeEvents(events)

	// 查找所有匹配
	for i := 0; i <= len(steps)-len(pattern.Sequence); i++ {
		match := true
		for j := 0; j < len(pattern.Sequence); j++ {
			if !pm.prefixSpan.stepEqual(steps[i+j], pattern.Sequence[j]) {
				match = false
				break
			}
		}
		if match {
			// 找到匹配，记录时间
			if i < len(events) {
				occurrences = append(occurrences, events[i].Timestamp)
			}
		}
	}

	return occurrences
}

/**
 * extractApplications 提取模式相关的应用列表
 *
 * Parameters:
 *   - pattern: 模式对象
 *   - sessions: 会话列表
 *
 * Returns: map[string]bool - 应用集合
 */
func (pm *PatternMiner) extractApplications(
	pattern *models.Pattern,
	sessions []*models.Session,
) map[string]bool {
	applications := make(map[string]bool)

	for _, session := range sessions {
		// 检查会话是否包含此模式
		if pm.prefixSpan.containsPattern(session.Events, pattern.Sequence) {
			if session.Application != "" {
				applications[session.Application] = true
			}
		}
	}

	return applications
}

/**
 * generateDescription 生成模式描述
 *
 * Parameters:
 *   - pattern: 模式对象
 *   - applications: 应用集合
 *   - avgInterval: 平均时间间隔
 *
 * Returns: string - 模式描述
 */
func (pm *PatternMiner) generateDescription(
	pattern *models.Pattern,
	applications map[string]bool,
	avgInterval time.Duration,
) string {
	desc := fmt.Sprintf("长度为%d的模式序列，包含", len(pattern.Sequence))

	// 描述序列组成
	stepTypes := make(map[string]int)
	for _, step := range pattern.Sequence {
		stepTypes[string(step.Type)]++
	}

	desc += " "
	first := true
	for stepType, count := range stepTypes {
		if !first {
			desc += "、"
		}
		desc += fmt.Sprintf("%d个%s", count, stepType)
		first = false
	}

	// 描述应用
	if len(applications) > 0 {
		desc += "，主要在"
		first = true
		for app := range applications {
			if !first {
				desc += "、"
			}
			desc += app
			first = false
			if len(applications) > 3 {
				desc += "等"
				break
			}
		}
		desc += "中出现"
	}

	// 描述频率
	if avgInterval > 0 {
		desc += fmt.Sprintf("，平均每%v出现一次", avgInterval.Round(time.Second))
	}

	return desc
}

/**
 * FilterBySupport 按支持度过滤模式
 *
 * Parameters:
 *   - patterns: 模式列表
 *   - minSupport: 最小支持度
 *
 * Returns: []*models.Pattern - 过滤后的模式列表
 */
func (pm *PatternMiner) FilterBySupport(patterns []*models.Pattern, minSupport int) []*models.Pattern {
	var filtered []*models.Pattern
	for _, pattern := range patterns {
		if pattern.SupportCount >= minSupport {
			filtered = append(filtered, pattern)
		}
	}
	return filtered
}

/**
 * FilterByConfidence 按置信度过滤模式
 *
 * Parameters:
 *   - patterns: 模式列表
 *   - minConfidence: 最小置信度
 *
 * Returns: []*models.Pattern - 过滤后的模式列表
 */
func (pm *PatternMiner) FilterByConfidence(
	patterns []*models.Pattern,
	minConfidence float64,
) []*models.Pattern {
	var filtered []*models.Pattern
	for _, pattern := range patterns {
		if pattern.Confidence >= minConfidence {
			filtered = append(filtered, pattern)
		}
	}
	return filtered
}

/**
 * FilterByLength 按模式长度过滤
 *
 * Parameters:
 *   - patterns: 模式列表
 *   - minLength: 最小长度
 *   - maxLength: 最大长度
 *
 * Returns: []*models.Pattern - 过滤后的模式列表
 */
func (pm *PatternMiner) FilterByLength(
	patterns []*models.Pattern,
	minLength int,
	maxLength int,
) []*models.Pattern {
	var filtered []*models.Pattern
	for _, pattern := range patterns {
		length := len(pattern.Sequence)
		if length >= minLength && length <= maxLength {
			filtered = append(filtered, pattern)
		}
	}
	return filtered
}

/**
 * FilterUnanalyzed 过滤出未分析的模式
 *
 * Parameters:
 *   - patterns: 模式列表
 *
 * Returns: []*models.Pattern - 未分析的模式列表
 */
func (pm *PatternMiner) FilterUnanalyzed(patterns []*models.Pattern) []*models.Pattern {
	var filtered []*models.Pattern
	for _, pattern := range patterns {
		if pattern.AIAnalysis == nil {
			filtered = append(filtered, pattern)
		}
	}
	return filtered
}

/**
 * GetMiningStats 获取挖掘统计信息
 *
 * Parameters:
 *   - patterns: 模式列表
 *
 * Returns: *MiningStats - 统计信息
 */
func (pm *PatternMiner) GetMiningStats(patterns []*models.Pattern) *MiningStats {
	stats := &MiningStats{
		TotalPatterns:      len(patterns),
		ByLength:           make(map[int]int),
		BySupport:          make(map[int]int),
		UnanalyzedCount:    0,
		AutomatedCount:     0,
		TotalSupportCount:  0,
		AverageConfidence:  0.0,
		LongestPattern:     nil,
		ShortestPattern:    nil,
	}

	if len(patterns) == 0 {
		return stats
	}

	var totalConfidence float64
	var shortestLen, longestLen int

	for _, pattern := range patterns {
		// 按长度统计
		length := len(pattern.Sequence)
		stats.ByLength[length]++

		// 按支持度统计
		stats.BySupport[pattern.SupportCount]++

		// 统计未分析和已自动化
		if pattern.AIAnalysis == nil {
			stats.UnanalyzedCount++
		}
		if pattern.IsAutomated {
			stats.AutomatedCount++
		}

		// 累计支持度和置信度
		stats.TotalSupportCount += pattern.SupportCount
		totalConfidence += pattern.Confidence

		// 找最长和最短模式
		if stats.LongestPattern == nil || length > longestLen {
			longestLen = length
			stats.LongestPattern = pattern
		}
		if stats.ShortestPattern == nil || length < shortestLen {
			shortestLen = length
			stats.ShortestPattern = pattern
		}
	}

	// 计算平均置信度
	if len(patterns) > 0 {
		stats.AverageConfidence = totalConfidence / float64(len(patterns))
	}

	return stats
}

/**
 * MiningStats 挖掘统计信息
 */
type MiningStats struct {
	// TotalPatterns 总模式数
	TotalPatterns int

	// ByLength 按长度统计的模式数
	ByLength map[int]int

	// BySupport 按支持度统计的模式数
	BySupport map[int]int

	// UnanalyzedCount 未分析的模式数
	UnanalyzedCount int

	// AutomatedCount 已自动化的模式数
	AutomatedCount int

	// TotalSupportCount 总支持计数
	TotalSupportCount int

	// AverageConfidence 平均置信度
	AverageConfidence float64

	// LongestPattern 最长的模式
	LongestPattern *models.Pattern

	// ShortestPattern 最短的模式
	ShortestPattern *models.Pattern
}

/**
 * GetPatternSummary 获取模式摘要
 *
 * Parameters:
 *   - pattern: 模式对象
 *
 * Returns: string - 模式摘要
 */
func (pm *PatternMiner) GetPatternSummary(pattern *models.Pattern) string {
	summary := fmt.Sprintf("模式ID: %s\n", pattern.ID)
	summary += fmt.Sprintf("长度: %d\n", len(pattern.Sequence))
	summary += fmt.Sprintf("支持度: %d\n", pattern.SupportCount)
	summary += fmt.Sprintf("置信度: %.2f%%\n", pattern.Confidence*100)
	summary += fmt.Sprintf("频率: %.2f次/小时\n", pattern.Frequency())
	summary += fmt.Sprintf("首次发现: %s\n", pattern.FirstSeen.Format("2006-01-02 15:04"))
	summary += fmt.Sprintf("最后发现: %s\n", pattern.LastSeen.Format("2006-01-02 15:04"))

	if pattern.Description != "" {
		summary += fmt.Sprintf("描述: %s\n", pattern.Description)
	}

	if pattern.IsAutomated {
		summary += "状态: 已自动化\n"
	} else if pattern.AIAnalysis != nil {
		summary += fmt.Sprintf("状态: 已分析（建议自动化: %v）\n", pattern.AIAnalysis.ShouldAutomate)
	} else {
		summary += "状态: 未分析\n"
	}

	return summary
}
