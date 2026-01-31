/**
 * Package analyzer 模式识别引擎的分析组件
 *
 * 负责会话划分、事件标准化、模式挖掘等核心功能
 */

package analyzer

import (
	"fmt"
	"sync"
	"time"

	"github.com/chenyang-zz/flowmind/internal/domain/models"
	"github.com/chenyang-zz/flowmind/pkg/events"
	"hash/fnv"
)

/**
 * PrefixSpanConfig PrefixSpan 算法配置
 */
type PrefixSpanConfig struct {
	// MinSupport 最小支持度（出现次数）
	MinSupport int

	// MaxPatternLength 最大模式长度
	MaxPatternLength int

	// MinPatternLength 最小模式长度
	MinPatternLength int
}

/**
 * DefaultPrefixSpanConfig 默认配置
 */
func DefaultPrefixSpanConfig() PrefixSpanConfig {
	return PrefixSpanConfig{
		MinSupport:       3,
		MaxPatternLength: 10,
		MinPatternLength: 2,
	}
}

/**
 * PrefixSpan PrefixSpan 算法实现
 *
 * 用于从事件序列中挖掘频繁模式
 */
type PrefixSpan struct {
	config PrefixSpanConfig
}

/**
 * NewPrefixSpan 创建 PrefixSpan 实例
 *
 * Parameters:
 *   - config: 算法配置
 *
 * Returns: *PrefixSpan - PrefixSpan 实例
 */
func NewPrefixSpan(config PrefixSpanConfig) *PrefixSpan {
	return &PrefixSpan{
		config: config,
	}
}

/**
 * Mine 挖掘频繁模式
 *
 * 从会话列表中挖掘频繁序列模式
 *
 * Parameters:
 *   - sessions: 会话列表
 *
 * Returns: []*models.Pattern - 发现的模式列表
 */
func (ps *PrefixSpan) Mine(sessions []*models.Session) ([]*models.Pattern, error) {
	if len(sessions) == 0 {
		return []*models.Pattern{}, nil
	}

	// 1. 标准化所有会话的事件
	normalizer := NewEventNormalizer(DefaultEventNormalizerConfig())
	var sequences [][]models.EventStep

	for _, session := range sessions {
		steps := normalizer.NormalizeEvents(session.Events)
		if len(steps) >= ps.config.MinPatternLength {
			sequences = append(sequences, steps)
		}
	}

	if len(sequences) == 0 {
		return []*models.Pattern{}, nil
	}

	// 2. 挖掘频繁模式
	patterns := ps.mineRecursive(sequences, []models.EventStep{}, 0, len(sequences))

	// 3. 转换为 Pattern 模型
	result := make([]*models.Pattern, 0, len(patterns))
	for _, p := range patterns {
		pattern := ps.buildPattern(p, sessions)
		if pattern != nil {
			result = append(result, pattern)
		}
	}

	return result, nil
}

/**
 * mineRecursive 递归挖掘频繁模式
 *
 * Parameters:
 *   - sequences: 序列数据库
 *   - prefix: 当前前缀
 *   - depth: 递归深度
 *   - totalSessions: 总会话数（用于支持度计算）
 *
 * Returns: []频繁模式列表
 */
func (ps *PrefixSpan) mineRecursive(
	sequences [][]models.EventStep,
	prefix []models.EventStep,
	depth int,
	totalSessions int,
) [][]models.EventStep {
	// 达到最大深度，停止挖掘
	if depth >= ps.config.MaxPatternLength {
		return [][]models.EventStep{}
	}

	// 1. 构建投影数据库并计算频繁项
	projectedDB := ps.buildProjectedDatabase(sequences, prefix)
	frequentItems := ps.findFrequentItems(projectedDB, totalSessions)

	var results [][]models.EventStep

	// 2. 对于每个频繁项，生成新模式并递归挖掘
	for item := range frequentItems {
		// 构建新模式
		newPrefix := append(prefix, item)

		// 如果达到最小长度，添加到结果
		if len(newPrefix) >= ps.config.MinPatternLength {
			results = append(results, newPrefix)
		}

		// 递归挖掘更长的模式
		if len(newPrefix) < ps.config.MaxPatternLength {
			subPatterns := ps.mineRecursive(sequences, newPrefix, depth+1, totalSessions)
			results = append(results, subPatterns...)
		}
	}

	return results
}

/**
 * buildProjectedDatabase 构建投影数据库
 *
 * 找出包含指定前缀的所有序列，并返回前缀之后的部分
 *
 * Parameters:
 *   - sequences: 原始序列数据库
 *   - prefix: 前缀模式
 *
 * Returns: [][]models.EventStep - 投影数据库
 */
func (ps *PrefixSpan) buildProjectedDatabase(
	sequences [][]models.EventStep,
	prefix []models.EventStep,
) [][]models.EventStep {
	if len(prefix) == 0 {
		// 空前缀，返回原始数据库
		return sequences
	}

	var projected [][]models.EventStep

	for _, sequence := range sequences {
		// 查找前缀在序列中的位置
		index := ps.findPrefixIndex(sequence, prefix)
		if index != -1 && index+len(prefix) < len(sequence) {
			// 返回前缀之后的部分
			suffix := sequence[index+len(prefix):]
			projected = append(projected, suffix)
		}
	}

	return projected
}

/**
 * findPrefixIndex 查找前缀在序列中的位置
 *
 * Parameters:
 *   - sequence: 序列
 *   - prefix: 前缀
 *
 * Returns: int - 前缀起始位置，-1表示未找到
 */
func (ps *PrefixSpan) findPrefixIndex(
	sequence []models.EventStep,
	prefix []models.EventStep,
) int {
	if len(prefix) == 0 {
		return 0
	}

	for i := 0; i <= len(sequence)-len(prefix); i++ {
		match := true
		for j := 0; j < len(prefix); j++ {
			if !ps.stepEqual(sequence[i+j], prefix[j]) {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}

	return -1
}

/**
 * stepEqual 判断两个事件步骤是否相等
 *
 * Parameters:
 *   - a: 步骤a
 *   - b: 步骤b
 *
 * Returns: bool - true表示相等
 */
func (ps *PrefixSpan) stepEqual(a, b models.EventStep) bool {
	// 比较事件类型
	if a.Type != b.Type {
		return false
	}

	// 比较动作
	if a.Action != b.Action {
		return false
	}

	// 如果两个都有上下文，比较应用
	if a.Context != nil && b.Context != nil {
		// 可以选择是否比较应用
		// 这里我们只比较类型和动作，不比较具体应用
		return true
	}

	// 如果一个有上下文一个没有，认为不相等
	if (a.Context != nil && b.Context == nil) || (a.Context == nil && b.Context != nil) {
		return false
	}

	return true
}

/**
 * findFrequentItems 查找频繁项
 *
 * Parameters:
 *   - projectedDB: 投影数据库
 *   - totalSessions: 总会话数（保留参数以备将来使用）
 *
 * Returns: map[models.EventStep]int - 频繁项及其支持度
 */
func (ps *PrefixSpan) findFrequentItems(
	projectedDB [][]models.EventStep,
	totalSessions int,
) map[models.EventStep]int {
	frequency := make(map[models.EventStep]int)

	// 统计每个项的出现频率
	for _, sequence := range projectedDB {
		if len(sequence) == 0 {
			continue
		}

		// 考虑序列的第一个元素（前缀后的下一个元素）
		firstItem := sequence[0]
		frequency[firstItem]++
	}

	// 过滤非频繁项
	frequentItems := make(map[models.EventStep]int)
	for item, count := range frequency {
		if count >= ps.config.MinSupport {
			frequentItems[item] = count
		}
	}

	return frequentItems
}

/**
 * buildPattern 构建模式模型
 *
 * Parameters:
 *   - sequence: 模式序列
 *   - sessions: 原始会话列表
 *
 * Returns: *models.Pattern - 模式对象
 */
func (ps *PrefixSpan) buildPattern(
	sequence []models.EventStep,
	sessions []*models.Session,
) *models.Pattern {
	if len(sequence) == 0 {
		return nil
	}

	// 计算支持度（出现次数）
	supportCount := ps.calculateSupport(sequence, sessions)

	// 计算置信度
	confidence := ps.calculateConfidence(sequence, sessions)

	// 找出首次和最后出现时间
	firstSeen, lastSeen := ps.findTimeRange(sequence, sessions)

	pattern := &models.Pattern{
		ID:           ps.generatePatternID(sequence),
		Sequence:     sequence,
		SupportCount: supportCount,
		Confidence:   confidence,
		FirstSeen:    firstSeen,
		LastSeen:     lastSeen,
		IsAutomated:  false,
	}

	return pattern
}

/**
 * calculateSupport 计算模式支持度
 *
 * Parameters:
 *   - sequence: 模式序列
 *   - sessions: 会话列表
 *
 * Returns: int - 支持度（出现次数）
 */
func (ps *PrefixSpan) calculateSupport(sequence []models.EventStep, sessions []*models.Session) int {
	count := 0

	for _, session := range sessions {
		if ps.containsPattern(session.Events, sequence) {
			count++
		}
	}

	return count
}

/**
 * containsPattern 检查会话是否包含指定模式
 *
 * Parameters:
 *   - events: 事件列表
 *   - pattern: 模式序列
 *
 * Returns: bool - true表示包含
 */
func (ps *PrefixSpan) containsPattern(events []events.Event, pattern []models.EventStep) bool {
	if len(pattern) == 0 || len(events) < len(pattern) {
		return false
	}

	// 标准化事件
	normalizer := NewEventNormalizer(DefaultEventNormalizerConfig())
	steps := normalizer.NormalizeEvents(events)

	// 查找模式
	for i := 0; i <= len(steps)-len(pattern); i++ {
		match := true
		for j := 0; j < len(pattern); j++ {
			if !ps.stepEqual(steps[i+j], pattern[j]) {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}

	return false
}

/**
 * calculateConfidence 计算模式置信度
 *
 * 置信度 = 支持度 / 总会话数
 *
 * Parameters:
 *   - sequence: 模式序列
 *   - sessions: 会话列表
 *
 * Returns: float64 - 置信度（0-1之间）
 */
func (ps *PrefixSpan) calculateConfidence(sequence []models.EventStep, sessions []*models.Session) float64 {
	if len(sessions) == 0 {
		return 0
	}

	support := ps.calculateSupport(sequence, sessions)
	return float64(support) / float64(len(sessions))
}

/**
 * findTimeRange 查找模式的出现时间范围
 *
 * Parameters:
 *   - sequence: 模式序列
 *   - sessions: 会话列表
 *
 * Returns: (time.Time, time.Time) - 首次和最后出现时间
 */
func (ps *PrefixSpan) findTimeRange(
	sequence []models.EventStep,
	sessions []*models.Session,
) (firstSeen, lastSeen time.Time) {
	var first, last *time.Time

	for _, session := range sessions {
		if ps.containsPattern(session.Events, sequence) {
			if first == nil || session.StartTime.Before(*first) {
				first = &session.StartTime
			}

			sessionEnd := session.StartTime
			if session.EndTime != nil {
				sessionEnd = *session.EndTime
			}
			if last == nil || sessionEnd.After(*last) {
				last = &sessionEnd
			}
		}
	}

	if first != nil {
		firstSeen = *first
	}
	if last != nil {
		lastSeen = *last
	}

	return firstSeen, lastSeen
}

/**
 * generatePatternID 生成模式ID
 *
 * Parameters:
 *   - sequence: 模式序列
 *
 * Returns: string - 模式ID
 */
func (ps *PrefixSpan) generatePatternID(sequence []models.EventStep) string {
	// 生成一个简化的模式签名
	signature := ""
	for _, step := range sequence {
		signature += fmt.Sprintf("%s-%s_", step.Type, step.Action)
	}

	// 使用 hash 生成唯一ID
	hash := fnv.New32a()
	hash.Write([]byte(signature))
	return fmt.Sprintf("pattern-%x", hash.Sum32())
}

/**
 * ParallelMine 并行挖掘模式（用于大数据集）
 *
 * Parameters:
 *   - sessions: 会话列表
 *   - workers: 并发工作数
 *
 * Returns: []*models.Pattern - 发现的模式列表
 */
func (ps *PrefixSpan) ParallelMine(sessions []*models.Session, workers int) ([]*models.Pattern, error) {
	if len(sessions) == 0 {
		return []*models.Pattern{}, nil
	}

	// 分割会话列表
	chunkSize := (len(sessions) + workers - 1) / workers
	var chunks [][]*models.Session

	for i := 0; i < len(sessions); i += chunkSize {
		end := i + chunkSize
		if end > len(sessions) {
			end = len(sessions)
		}
		chunks = append(chunks, sessions[i:end])
	}

	// 并行挖掘
	var wg sync.WaitGroup
	results := make(chan []*models.Pattern, workers)
	errors := make(chan error, workers)

	for _, chunk := range chunks {
		wg.Add(1)
		go func(sessions []*models.Session) {
			defer wg.Done()
			patterns, err := ps.Mine(sessions)
			if err != nil {
				errors <- err
				return
			}
			results <- patterns
		}(chunk)
	}

	// 等待所有worker完成
	wg.Wait()
	close(results)
	close(errors)

	// 检查错误
	select {
	case err := <-errors:
		return nil, err
	default:
	}

	// 合并结果
	var allPatterns []*models.Pattern
	for result := range results {
		allPatterns = append(allPatterns, result...)
	}

	// 去重
	return ps.deduplicatePatterns(allPatterns), nil
}

/**
 * deduplicatePatterns 去重模式
 *
 * Parameters:
 *   - patterns: 模式列表
 *
 * Returns: []*models.Pattern - 去重后的模式列表
 */
func (ps *PrefixSpan) deduplicatePatterns(patterns []*models.Pattern) []*models.Pattern {
	seen := make(map[string]bool)
	var result []*models.Pattern

	for _, pattern := range patterns {
		signature := ps.generatePatternSignature(pattern.Sequence)
		if !seen[signature] {
			seen[signature] = true
			result = append(result, pattern)
		}
	}

	return result
}

/**
 * generatePatternSignature 生成模式签名（用于去重）
 *
 * Parameters:
 *   - sequence: 模式序列
 *
 * Returns: string - 模式签名
 */
func (ps *PrefixSpan) generatePatternSignature(sequence []models.EventStep) string {
	signature := ""
	for _, step := range sequence {
		signature += fmt.Sprintf("%s:%s|", step.Type, step.Action)
	}
	return signature
}
