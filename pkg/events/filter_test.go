package events

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestNewEventFilterManager 测试EventFilterManager的创建
//
// 验证EventFilterManager能够正确创建，并且初始状态正确。
func TestNewEventFilterManager(t *testing.T) {
	fm := NewEventFilterManager()

	assert.NotNil(t, fm)
	assert.NotNil(t, fm.rules)
	assert.NotNil(t, fm.lastEventTime)
	assert.NotNil(t, fm.eventCounters)
	assert.Equal(t, time.Second, fm.windowSize)
}

// TestEventFilterManager_SetRule 测试设置过滤规则
//
// 验证能够正确设置和删除过滤规则。
func TestEventFilterManager_SetRule(t *testing.T) {
	fm := NewEventFilterManager()

	// 设置规则
	rule := &FilterRule{
		MinInterval:  100 * time.Millisecond,
		MaxPerSecond: 10,
	}
	fm.SetRule(EventTypeKeyboard, rule)

	// 验证规则已设置
	assert.Equal(t, rule, fm.rules[EventTypeKeyboard])

	// 删除规则
	fm.SetRule(EventTypeKeyboard, nil)
	_, exists := fm.rules[EventTypeKeyboard]
	assert.False(t, exists)
}

// TestEventFilterManager_SetRules 测试批量设置过滤规则
//
// 验证能够批量设置多个事件类型的过滤规则。
func TestEventFilterManager_SetRules(t *testing.T) {
	fm := NewEventFilterManager()

	rules := map[EventType]*FilterRule{
		EventTypeKeyboard: {
			MinInterval:  100 * time.Millisecond,
			MaxPerSecond: 10,
		},
		EventTypeClipboard: {
			MinInterval:  200 * time.Millisecond,
			MaxPerSecond: 5,
		},
	}
	fm.SetRules(rules)

	// 验证所有规则都已设置
	assert.Equal(t, rules[EventTypeKeyboard], fm.rules[EventTypeKeyboard])
	assert.Equal(t, rules[EventTypeClipboard], fm.rules[EventTypeClipboard])
}

// TestEventFilterManager_ShouldPass_NoRule 测试无规则时的事件通过
//
// 验证没有设置规则时，所有事件都应该通过。
func TestEventFilterManager_ShouldPass_NoRule(t *testing.T) {
	fm := NewEventFilterManager()

	// 没有设置规则，应该通过
	pass := fm.ShouldPass(EventTypeKeyboard)
	assert.True(t, pass)
}

// TestEventFilterManager_MinInterval 测试最小时间间隔过滤
//
// 验证小于最小时间间隔的事件会被过滤。
func TestEventFilterManager_MinInterval(t *testing.T) {
	fm := NewEventFilterManager()

	// 设置最小间隔规则
	rule := &FilterRule{
		MinInterval: 100 * time.Millisecond,
	}
	fm.SetRule(EventTypeKeyboard, rule)

	// 第一个事件应该通过
	pass := fm.ShouldPass(EventTypeKeyboard)
	assert.True(t, pass)

	// 立即发送第二个事件，应该被过滤
	pass = fm.ShouldPass(EventTypeKeyboard)
	assert.False(t, pass)

	// 等待间隔后，应该通过
	time.Sleep(150 * time.Millisecond)
	pass = fm.ShouldPass(EventTypeKeyboard)
	assert.True(t, pass)
}

// TestEventFilterManager_MaxPerSecond 测试每秒最大事件数限制
//
// 验证超过每秒最大事件数的事件会被过滤。
func TestEventFilterManager_MaxPerSecond(t *testing.T) {
	fm := NewEventFilterManager()

	// 设置速率限制规则
	rule := &FilterRule{
		MaxPerSecond: 5,
	}
	fm.SetRule(EventTypeKeyboard, rule)

	// 前5个事件应该通过
	for i := 0; i < 5; i++ {
		pass := fm.ShouldPass(EventTypeKeyboard)
		assert.True(t, pass, "事件 %d 应该通过", i+1)
	}

	// 第6个事件应该被过滤
	pass := fm.ShouldPass(EventTypeKeyboard)
	assert.False(t, pass, "事件 6 应该被过滤")
}

// TestEventFilterManager_RateLimitSlidingWindow 测试速率限制的滑动窗口
//
// 验证滑动窗口机制能够正确恢复速率限制。
func TestEventFilterManager_RateLimitSlidingWindow(t *testing.T) {
	fm := NewEventFilterManager()

	// 设置速率限制
	rule := &FilterRule{
		MaxPerSecond: 3,
	}
	fm.SetRule(EventTypeKeyboard, rule)

	// 消耗所有配额
	for i := 0; i < 3; i++ {
		pass := fm.ShouldPass(EventTypeKeyboard)
		assert.True(t, pass)
	}

	// 应该被限制
	pass := fm.ShouldPass(EventTypeKeyboard)
	assert.False(t, pass)

	// 等待窗口过期
	time.Sleep(time.Second + 100*time.Millisecond)

	// 应该恢复
	pass = fm.ShouldPass(EventTypeKeyboard)
	assert.True(t, pass)
}

// TestEventFilterManager_GetLastEventTime 测试获取最后事件时间
//
// 验证能够正确获取最后事件的时间戳。
func TestEventFilterManager_GetLastEventTime(t *testing.T) {
	fm := NewEventFilterManager()

	// 初始时间为零值
	lastTime := fm.GetLastEventTime(EventTypeKeyboard)
	assert.True(t, lastTime.IsZero())

	// 发送事件后，应该有时间
	fm.ShouldPass(EventTypeKeyboard)
	lastTime = fm.GetLastEventTime(EventTypeKeyboard)
	assert.False(t, lastTime.IsZero())
	assert.True(t, time.Since(lastTime) < time.Second)
}

// TestEventFilterManager_Reset 测试重置过滤器
//
// 验证重置后所有规则和状态都会被清除。
func TestEventFilterManager_Reset(t *testing.T) {
	fm := NewEventFilterManager()

	// 设置规则并生成事件
	rule := &FilterRule{
		MinInterval:  100 * time.Millisecond,
		MaxPerSecond: 10,
	}
	fm.SetRule(EventTypeKeyboard, rule)
	fm.ShouldPass(EventTypeKeyboard)

	// 重置
	fm.Reset()

	// 验证规则已清除
	_, exists := fm.rules[EventTypeKeyboard]
	assert.False(t, exists)

	// 验证事件时间已清除
	lastTime := fm.GetLastEventTime(EventTypeKeyboard)
	assert.True(t, lastTime.IsZero())
}

// TestEventFilterManager_SetWindowSize 测试设置时间窗口大小
//
// 验证能够正确设置速率限制的时间窗口大小。
func TestEventFilterManager_SetWindowSize(t *testing.T) {
	fm := NewEventFilterManager()

	// 设置窗口大小
	newSize := 2 * time.Second
	fm.SetWindowSize(newSize)
	assert.Equal(t, newSize, fm.windowSize)
}

// TestEventFilterManager_GetEventCount 测试获取事件计数
//
// 验证能够正确获取当前时间窗口内的事件计数。
func TestEventFilterManager_GetEventCount(t *testing.T) {
	fm := NewEventFilterManager()

	// 设置速率限制
	rule := &FilterRule{
		MaxPerSecond: 10,
	}
	fm.SetRule(EventTypeKeyboard, rule)

	// 初始计数为0
	count := fm.GetEventCount(EventTypeKeyboard)
	assert.Equal(t, 0, count)

	// 发送5个事件
	for i := 0; i < 5; i++ {
		fm.ShouldPass(EventTypeKeyboard)
	}

	// 计数应该为5
	count = fm.GetEventCount(EventTypeKeyboard)
	assert.Equal(t, 5, count)
}

// TestEventFilterManager_Concurrent 测试并发访问
//
// 验证EventFilterManager在并发环境下是线程安全的。
func TestEventFilterManager_Concurrent(t *testing.T) {
	fm := NewEventFilterManager()

	// 设置规则
	rule := &FilterRule{
		MinInterval:  10 * time.Millisecond,
		MaxPerSecond: 100,
	}
	fm.SetRule(EventTypeKeyboard, rule)

	// 并发测试
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				fm.ShouldPass(EventTypeKeyboard)
			}
		}()
	}
	wg.Wait()

	// 验证没有panic，且状态正常
	assert.NotNil(t, fm.rules)
	assert.NotNil(t, fm.lastEventTime)
	assert.NotNil(t, fm.eventCounters)
}

// TestEventFilterManager_MultipleEventTypes 测试多种事件类型
//
// 验证能够正确处理多种不同事件类型的过滤。
func TestEventFilterManager_MultipleEventTypes(t *testing.T) {
	fm := NewEventFilterManager()

	// 为不同事件类型设置不同规则
	rules := map[EventType]*FilterRule{
		EventTypeKeyboard: {
			MinInterval: 50 * time.Millisecond,
		},
		EventTypeClipboard: {
			MaxPerSecond: 2,
		},
	}
	fm.SetRules(rules)

	// 键盘事件：第一个通过，第二个被过滤
	assert.True(t, fm.ShouldPass(EventTypeKeyboard))
	assert.False(t, fm.ShouldPass(EventTypeKeyboard))

	// 剪贴板事件：前两个通过，第三个被过滤
	assert.True(t, fm.ShouldPass(EventTypeClipboard))
	assert.True(t, fm.ShouldPass(EventTypeClipboard))
	assert.False(t, fm.ShouldPass(EventTypeClipboard))
}
