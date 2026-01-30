package events

import (
	"sync"
	"time"
)

// FilterRule 事件过滤规则
//
// 定义了事件过滤的条件和阈值
type FilterRule struct {
	// MinInterval 事件最小间隔（同一事件类型的两次事件之间的最小时间间隔）
	// 小于此间隔的事件将被过滤
	MinInterval time.Duration

	// MaxPerSecond 每秒最大事件数（同一事件类型）
	// 超过此速率的事件将被丢弃
	MaxPerSecond int
}

// EventFilterManager 事件过滤器管理器
//
// 用于过滤过于频繁的事件，避免事件风暴。
// 支持基于时间间隔和速率限制的过滤策略。
// 注意：不要与EventFilter函数类型混淆。
type EventFilterManager struct {
	// rules 每种事件类型的过滤规则
	rules map[EventType]*FilterRule

	// lastEventTime 每种事件类型最后一次事件的时间
	lastEventTime map[EventType]time.Time

	// eventCounters 用于速率限制的事件计数器（滑动窗口）
	eventCounters map[EventType][]time.Time

	// mu 互斥锁，保护并发访问
	mu sync.RWMutex

	// windowSize 速率限制的时间窗口（默认1秒）
	windowSize time.Duration
}

// NewEventFilterManager 创建事件过滤器管理器
//
// 创建一个新的事件过滤器管理器实例。
//
// Returns: *EventFilterManager - 新创建的事件过滤器管理器实例
func NewEventFilterManager() *EventFilterManager {
	return &EventFilterManager{
		rules:          make(map[EventType]*FilterRule),
		lastEventTime:  make(map[EventType]time.Time),
		eventCounters:  make(map[EventType][]time.Time),
		windowSize:     1 * time.Second,
	}
}

// SetRule 设置过滤规则
//
// 为指定的事件类型设置过滤规则。
//
// Parameters:
//   - eventType: 事件类型
//   - rule: 过滤规则（nil 表示移除规则）
func (f *EventFilterManager) SetRule(eventType EventType, rule *FilterRule) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if rule == nil {
		delete(f.rules, eventType)
	} else {
		f.rules[eventType] = rule
	}
}

// SetRules 批量设置过滤规则
//
// 批量设置多个事件类型的过滤规则。
//
// Parameters: rules - 事件类型到过滤规则的映射
func (f *EventFilterManager) SetRules(rules map[EventType]*FilterRule) {
	f.mu.Lock()
	defer f.mu.Unlock()

	for eventType, rule := range rules {
		if rule == nil {
			delete(f.rules, eventType)
		} else {
			f.rules[eventType] = rule
		}
	}
}

// ShouldPass 判断事件是否应该通过过滤器
//
// 检查事件是否符合过滤规则，决定是否允许事件通过。
//
// Parameters:
//   - eventType: 事件类型
//
// Returns: bool - true 表示事件应该通过，false 表示应该被过滤
func (f *EventFilterManager) ShouldPass(eventType EventType) bool {
	f.mu.Lock()
	defer f.mu.Unlock()

	rule, exists := f.rules[eventType]

	now := time.Now()

	if !exists {
		// 没有规则，允许所有事件通过，但记录时间
		f.lastEventTime[eventType] = now
		return true
	}

	// 检查最小时间间隔
	if rule.MinInterval > 0 {
		if lastTime, ok := f.lastEventTime[eventType]; ok {
			elapsed := now.Sub(lastTime)
			if elapsed < rule.MinInterval {
				// 时间间隔太小，过滤事件
				return false
			}
		}
	}

	// 检查速率限制
	if rule.MaxPerSecond > 0 {
		// 清理过期的计数器
		f.cleanupCounters(eventType, now)

		// 获取当前时间窗口内的事件计数
		count := len(f.eventCounters[eventType])
		if count >= rule.MaxPerSecond {
			// 超过速率限制，过滤事件
			return false
		}

		// 记录当前事件时间
		f.eventCounters[eventType] = append(f.eventCounters[eventType], now)
	}

	// 更新最后事件时间
	f.lastEventTime[eventType] = now

	return true
}

// cleanupCounters 清理过期的计数器
//
// 清理指定事件类型的过期计数器（超过时间窗口的记录）。
//
// Parameters:
//   - eventType: 事件类型
//   - now: 当前时间
func (f *EventFilterManager) cleanupCounters(eventType EventType, now time.Time) {
	timestamps := f.eventCounters[eventType]

	// 找到第一个未过期的索引
	cutoff := now.Add(-f.windowSize)
	var validStart int
	for i, ts := range timestamps {
		if ts.After(cutoff) {
			validStart = i
			break
		}
	}

	// 保留有效的时间戳
	if validStart > 0 {
		f.eventCounters[eventType] = timestamps[validStart:]
	} else if validStart == 0 && len(timestamps) > 0 && timestamps[0].After(cutoff) {
		// 所有时间戳都有效
		// 不需要处理
	} else {
		// 所有时间戳都过期，清空
		f.eventCounters[eventType] = f.eventCounters[eventType][:0]
	}
}

// GetLastEventTime 获取指定事件类型的最后事件时间
//
// Parameters: eventType - 事件类型
//
// Returns: time.Time - 最后事件时间，零值表示没有记录
func (f *EventFilterManager) GetLastEventTime(eventType EventType) time.Time {
	f.mu.RLock()
	defer f.mu.RUnlock()

	return f.lastEventTime[eventType]
}

// Reset 重置过滤器状态
//
// 清除所有过滤规则和状态，重置过滤器到初始状态。
func (f *EventFilterManager) Reset() {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.rules = make(map[EventType]*FilterRule)
	f.lastEventTime = make(map[EventType]time.Time)
	f.eventCounters = make(map[EventType][]time.Time)
}

// SetWindowSize 设置速率限制的时间窗口大小
//
// Parameters: duration - 时间窗口大小
func (f *EventFilterManager) SetWindowSize(duration time.Duration) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.windowSize = duration
}

// GetEventCount 获取指定事件类型在当前时间窗口内的事件计数
//
// 用于监控事件频率，帮助调试和调优过滤规则。
//
// Parameters: eventType - 事件类型
//
// Returns: int - 当前时间窗口内的事件计数
func (f *EventFilterManager) GetEventCount(eventType EventType) int {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.cleanupCounters(eventType, time.Now())
	return len(f.eventCounters[eventType])
}
