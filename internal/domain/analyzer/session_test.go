package analyzer

import (
	"testing"
	"time"

	"github.com/chenyang-zz/flowmind/internal/domain/models"
	"github.com/chenyang-zz/flowmind/pkg/events"
	"github.com/stretchr/testify/assert"
)

// TestSessionDivider_Divide 测试基本会话划分
func TestSessionDivider_Divide(t *testing.T) {
	config := DefaultSessionDividerConfig()
	config.Timeout = 5 * time.Minute
	config.MinEvents = 2

	divider := NewSessionDivider(config)

	now := time.Now()

	// 创建测试事件（3个会话）
	eventList := []events.Event{
		// 会话1：2个事件
		*createEventWithApp(now, "VSCode", "0"),
		*createEventWithApp(now.Add(1*time.Minute), "VSCode", "1"),
		// 超时后新会话
		*createEventWithApp(now.Add(10*time.Minute), "Chrome", "2"),
		*createEventWithApp(now.Add(11*time.Minute), "Chrome", "3"),
		// 应用切换后新会话
		*createEventWithApp(now.Add(12*time.Minute), "Firefox", "4"),
		*createEventWithApp(now.Add(13*time.Minute), "Firefox", "5"),
	}

	sessions := divider.Divide(eventList)

	// 应该划分为3个会话
	assert.Len(t, sessions, 3)

	// 验证第一个会话
	assert.Equal(t, "VSCode", sessions[0].Application)
	assert.Len(t, sessions[0].Events, 2)

	// 验证第二个会话
	assert.Equal(t, "Chrome", sessions[1].Application)
	assert.Len(t, sessions[1].Events, 2)

	// 验证第三个会话
	assert.Equal(t, "Firefox", sessions[2].Application)
	assert.Len(t, sessions[2].Events, 2)
}

// TestSessionDivider_Divide_Empty 测试空事件列表
func TestSessionDivider_Divide_Empty(t *testing.T) {
	config := DefaultSessionDividerConfig()
	divider := NewSessionDivider(config)

	sessions := divider.Divide([]events.Event{})

	assert.Len(t, sessions, 0)
}

// TestSessionDivider_Divide_SingleEvent 测试单个事件
func TestSessionDivider_Divide_SingleEvent(t *testing.T) {
	config := DefaultSessionDividerConfig()
	config.MinEvents = 1

	divider := NewSessionDivider(config)

	eventList := []events.Event{
		*createEventWithApp(time.Now(), "VSCode", "0"),
	}

	sessions := divider.Divide(eventList)

	// 单个事件满足最小事件数要求
	assert.Len(t, sessions, 1)
	assert.Equal(t, 1, sessions[0].EventCount)
}

// TestSessionDivider_Timeout 测试超时分割
func TestSessionDivider_Timeout(t *testing.T) {
	config := DefaultSessionDividerConfig()
	config.Timeout = 5 * time.Minute
	config.MinEvents = 2

	divider := NewSessionDivider(config)

	now := time.Now()

	eventList := []events.Event{
		*createEventWithApp(now, "VSCode", "0"),
		*createEventWithApp(now.Add(1*time.Minute), "VSCode", "1"),
		// 超时（间隔6分钟）
		*createEventWithApp(now.Add(7*time.Minute), "VSCode", "2"),
		*createEventWithApp(now.Add(8*time.Minute), "VSCode", "3"),
	}

	sessions := divider.Divide(eventList)

	// 超时应该分割成2个会话
	assert.Len(t, sessions, 2)
	assert.Len(t, sessions[0].Events, 2)
	assert.Len(t, sessions[1].Events, 2)
}

// TestSessionDivider_AppChange 测试应用切换分割
func TestSessionDivider_AppChange(t *testing.T) {
	config := DefaultSessionDividerConfig()
	config.SplitOnAppChange = true
	config.MinEvents = 1

	divider := NewSessionDivider(config)

	now := time.Now()

	eventList := []events.Event{
		*createEventWithApp(now, "VSCode", "0"),
		*createEventWithApp(now.Add(30*time.Second), "Chrome", "1"),
		*createEventWithApp(now.Add(1*time.Minute), "VSCode", "2"),
	}

	sessions := divider.Divide(eventList)

	// 应用切换应该分割成3个会话
	assert.Len(t, sessions, 3)
	assert.Equal(t, "VSCode", sessions[0].Application)
	assert.Equal(t, "Chrome", sessions[1].Application)
	assert.Equal(t, "VSCode", sessions[2].Application)
}

// TestSessionDivider_MinEventsFilter 测试最小事件数过滤
func TestSessionDivider_MinEventsFilter(t *testing.T) {
	config := DefaultSessionDividerConfig()
	config.MinEvents = 3

	divider := NewSessionDivider(config)

	now := time.Now()

	eventList := []events.Event{
		// 会话1：只有2个事件（不足最小值）
		*createEventWithApp(now, "VSCode", "0"),
		*createEventWithApp(now.Add(1*time.Minute), "VSCode", "1"),
		// 超时
		// 会话2：4个事件（满足最小值）
		*createEventWithApp(now.Add(10*time.Minute), "Chrome", "2"),
		*createEventWithApp(now.Add(11*time.Minute), "Chrome", "3"),
		*createEventWithApp(now.Add(12*time.Minute), "Chrome", "4"),
		*createEventWithApp(now.Add(13*time.Minute), "Chrome", "5"),
	}

	sessions := divider.Divide(eventList)

	// 应该只返回第二个会话（第一个被过滤）
	assert.Len(t, sessions, 1)
	assert.Equal(t, "Chrome", sessions[0].Application)
	assert.Len(t, sessions[0].Events, 4)
}

// TestSession_FilterByEventCount 测试按事件数过滤
func TestSession_FilterByEventCount(t *testing.T) {
	config := DefaultSessionDividerConfig()
	divider := NewSessionDivider(config)

	sessions := []*models.Session{
		{EventCount: 10},
		{EventCount: 3},
		{EventCount: 5},
		{EventCount: 1},
	}

	filtered := divider.FilterByEventCount(sessions, 3)

	// 应该只保留事件数>=3的会话
	assert.Len(t, filtered, 3)
}

// TestSessionDivider_GetSessionStats 测试会话统计
func TestSessionDivider_GetSessionStats(t *testing.T) {
	config := DefaultSessionDividerConfig()
	divider := NewSessionDivider(config)

	now := time.Now()

	// 创建测试会话
	sessions := []*models.Session{
		{
			ID:          "session-1",
			StartTime:   now.Add(-2 * time.Hour),
			EndTime:     func() *time.Time { t := now.Add(-1 * time.Hour); return &t }(),
			Application: "VSCode",
			EventCount:  10,
		},
		{
			ID:          "session-2",
			StartTime:   now.Add(-30 * time.Minute),
			EndTime:     func() *time.Time { t := now; return &t }(),
			Application: "Chrome",
			EventCount:  5,
		},
		{
			ID:          "session-3",
			StartTime:   now.Add(-1 * time.Hour),
			EndTime:     func() *time.Time { t := now.Add(-30 * time.Minute); return &t }(),
			Application: "VSCode",
			EventCount:  8,
		},
	}

	stats := divider.GetSessionStats(sessions)

	// 验证基本统计
	assert.Equal(t, 3, stats.TotalSessions)
	assert.Equal(t, 23, stats.TotalEvents)

	// 验证平均事件数
	assert.InDelta(t, 7.67, stats.AverageEvents, 0.01)

	// 验证应用统计
	assert.Equal(t, 2, stats.ByApplication["VSCode"])
	assert.Equal(t, 1, stats.ByApplication["Chrome"])

	// 验证最短和最长会话
	assert.NotNil(t, stats.ShortestSession)
	assert.NotNil(t, stats.LongestSession)
	assert.Equal(t, "Chrome", stats.ShortestSession.Application)
	assert.Equal(t, "VSCode", stats.LongestSession.Application)
}

// TestSessionDivider_GetSessionStats_Empty 测试空会话统计
func TestSessionDivider_GetSessionStats_Empty(t *testing.T) {
	config := DefaultSessionDividerConfig()
	divider := NewSessionDivider(config)

	stats := divider.GetSessionStats([]*models.Session{})

	assert.Equal(t, 0, stats.TotalSessions)
	assert.Equal(t, 0, stats.TotalEvents)
	assert.Empty(t, stats.ByApplication)
	assert.Nil(t, stats.ShortestSession)
	assert.Nil(t, stats.LongestSession)
}

// TestSession_Duration 测试会话持续时间
func TestSession_Duration(t *testing.T) {
	start := time.Now().Add(-2 * time.Hour)
	end := time.Now().Add(-1 * time.Hour)

	session := &models.Session{
		StartTime: start,
		EndTime:   &end,
	}

	duration := session.Duration()

	// 应该约为1小时
	assert.Greater(t, duration, 30*time.Minute)
	assert.Less(t, duration, 2*time.Hour)
}

// createEventWithApp 创建带有应用信息的测试事件
func createEventWithApp(timestamp time.Time, app string, index string) *events.Event {
	event := events.NewEvent(events.EventTypeKeyboard, map[string]interface{}{
		"index": index,
	})
	event.Timestamp = timestamp
	event.Context = &events.EventContext{
		Application: app,
		BundleID:    "com.test." + app,
	}

	return event
}
