package models

import (
	"testing"
	"time"

	"github.com/chenyang-zz/flowmind/pkg/events"
	"github.com/stretchr/testify/assert"
)

// TestSession_IsCompleted 测试会话完成状态
func TestSession_IsCompleted(t *testing.T) {
	tests := []struct {
		name     string
		endTime  *time.Time
		expected bool
	}{
		{
			name:     "已完成会话",
			endTime:  func() *time.Time { t := time.Now(); return &t }(),
			expected: true,
		},
		{
			name:     "进行中会话",
			endTime:  nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := &Session{
				EndTime: tt.endTime,
			}
			assert.Equal(t, tt.expected, session.IsCompleted())
		})
	}
}

// TestSession_Duration 测试会话持续时间计算
func TestSession_Duration(t *testing.T) {
	start := time.Now().Add(-2 * time.Hour)

	tests := []struct {
		name     string
		endTime  *time.Time
		minDuration time.Duration
		maxDuration time.Duration
	}{
		{
			name:     "已完成会话",
			endTime:  func() *time.Time { t := time.Now(); return &t }(),
			minDuration: 1 * time.Hour,
			maxDuration: 3 * time.Hour,
		},
		{
			name:        "进行中会话",
			endTime:     nil,
			minDuration: 1 * time.Hour,
			maxDuration: 3 * time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := &Session{
				StartTime: start,
				EndTime:   tt.endTime,
			}
			duration := session.Duration()
			assert.GreaterOrEqual(t, duration, tt.minDuration)
			assert.LessOrEqual(t, duration, tt.maxDuration)
		})
	}
}

// TestPattern_Length 测试模式长度
func TestPattern_Length(t *testing.T) {
	pattern := &Pattern{
		Sequence: []EventStep{
			{Type: events.EventTypeKeyboard},
			{Type: events.EventTypeClipboard},
			{Type: events.EventTypeKeyboard},
		},
	}

	assert.Equal(t, 3, pattern.Length())
}

// TestPattern_Frequency 测试模式频率计算
func TestPattern_Frequency(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name         string
		supportCount int
		firstSeen    time.Time
		lastSeen     time.Time
		expectedMin  float64
		expectedMax  float64
	}{
		{
			name:         "高频率模式",
			supportCount: 10,
			firstSeen:    now.Add(-1 * time.Hour),
			lastSeen:     now,
			expectedMin:  9.0,
			expectedMax:  11.0,
		},
		{
			name:         "低频率模式",
			supportCount: 2,
			firstSeen:    now.Add(-24 * time.Hour),
			lastSeen:     now,
			expectedMin:  0.08,
			expectedMax:  0.09,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pattern := &Pattern{
				SupportCount: tt.supportCount,
				FirstSeen:    tt.firstSeen,
				LastSeen:     tt.lastSeen,
			}
			freq := pattern.Frequency()
			assert.GreaterOrEqual(t, freq, tt.expectedMin)
			assert.LessOrEqual(t, freq, tt.expectedMax)
		})
	}
}

// TestPattern_Frequency_ZeroDuration 测试零时间范围
func TestPattern_Frequency_ZeroDuration(t *testing.T) {
	now := time.Now()
	pattern := &Pattern{
		SupportCount: 5,
		FirstSeen:    now,
		LastSeen:     now,
	}

	assert.Equal(t, float64(0), pattern.Frequency())
}

// TestEventStep 测试事件步骤结构
func TestEventStep(t *testing.T) {
	step := EventStep{
		Type:   events.EventTypeKeyboard,
		Action: "keypress",
		Context: &StepContext{
			Application:  "VSCode",
			BundleID:     "com.microsoft.vscode",
			PatternValue: "letter_key",
		},
	}

	assert.Equal(t, events.EventTypeKeyboard, step.Type)
	assert.Equal(t, "keypress", step.Action)
	assert.NotNil(t, step.Context)
	assert.Equal(t, "VSCode", step.Context.Application)
}

// TestAIAnalysis 测试AI分析结果
func TestAIAnalysis(t *testing.T) {
	analysis := &AIAnalysis{
		ShouldAutomate:      true,
		Reason:              "频繁重复的操作，自动化可节省大量时间",
		EstimatedTimeSaving: 3600, // 1小时
		Complexity:          "low",
		SuggestedName:       "自动保存",
		SuggestedSteps:      []string{"1. 检测保存快捷键", "2. 执行保存操作"},
		AnalyzedAt:          time.Now(),
	}

	assert.True(t, analysis.ShouldAutomate)
	assert.Equal(t, int64(3600), analysis.EstimatedTimeSaving)
	assert.Equal(t, "low", analysis.Complexity)
	assert.Len(t, analysis.SuggestedSteps, 2)
}

// TestSession 测试会话创建
func TestSession(t *testing.T) {
	startTime := time.Now().Add(-1 * time.Hour)
	endTime := time.Now()

	session := &Session{
		ID:          "session-123",
		StartTime:   startTime,
		EndTime:    &endTime,
		Application: "Chrome",
		BundleID:    "com.google.chrome",
		EventCount:  10,
		Events:      []events.Event{},
	}

	assert.Equal(t, "session-123", session.ID)
	assert.Equal(t, "Chrome", session.Application)
	assert.True(t, session.IsCompleted())
	assert.Greater(t, session.Duration(), 30*time.Minute)
}

// TestPattern 测试模式创建
func TestPattern(t *testing.T) {
	now := time.Now()

	pattern := &Pattern{
		ID:       "pattern-456",
		Sequence: []EventStep{
			{Type: events.EventTypeKeyboard, Action: "cmd_s"},
			{Type: events.EventTypeKeyboard, Action: "wait"},
		},
		SupportCount: 5,
		Confidence:   0.85,
		FirstSeen:    now.Add(-24 * time.Hour),
		LastSeen:     now,
		IsAutomated:  false,
	}

	assert.Equal(t, "pattern-456", pattern.ID)
	assert.Equal(t, 2, pattern.Length())
	assert.Equal(t, 5, pattern.SupportCount)
	assert.Equal(t, 0.85, pattern.Confidence)
	assert.False(t, pattern.IsAutomated)
	assert.Greater(t, pattern.Frequency(), float64(0))
}
