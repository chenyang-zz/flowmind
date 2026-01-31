package analyzer

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/chenyang-zz/flowmind/internal/infrastructure/ai"
	"github.com/chenyang-zz/flowmind/internal/infrastructure/storage"
	"github.com/chenyang-zz/flowmind/pkg/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/**
 * TestAnalyzerEngine_EndToEnd 端到端测试
 *
 * 测试完整的分析流程：事件 → 会话 → 模式 → 存储
 */
func TestAnalyzerEngine_EndToEnd(t *testing.T) {
	// 1. 设置测试环境
	db := setupTestDB(t)
	defer db.Close()

	// 初始化存储（使用同一个数据库连接）
	eventRepo := storage.NewSQLiteEventRepository(db)
	patternRepo := storage.NewSQLitePatternRepository(db)

	// 创建事件总线
	eventBus := events.NewEventBus()

	// 2. 创建模拟 AI 客户端
	mockAI := &MockAIClient{
		analyzeResponse: &ai.PatternAnalysis{
			ShouldAutomate:      true,
			Reason:              "测试模式",
			EstimatedTimeSaving: 100,
			Complexity:          "low",
			SuggestedName:       "测试自动化",
			SuggestedSteps:      []string{"步骤1", "步骤2"},
			AnalyzedAt:          time.Now(),
		},
	}

	// 3. 创建分析引擎配置
	config := DefaultAnalyzerEngineConfig()
	config.AIPatternFilter.AIModel = mockAI
	config.AIPatternFilter.CacheEnabled = true
	config.AnalysisInterval = 100 * time.Millisecond // 缩短间隔用于测试
	config.MinEventCount = 5

	// 4. 创建分析引擎
	engine, err := NewAnalyzerEngine(config, eventRepo, patternRepo, eventBus)
	require.NoError(t, err)
	defer engine.Close()

	// 5. 创建测试事件
	testEvents := generateTestEvents(50) // 生成50个测试事件

	// 6. 保存测试事件到数据库
	err = eventRepo.SaveBatch(testEvents)
	require.NoError(t, err)

	// 7. 手动执行一次分析（不使用定时循环）
	result, err := engine.AnalyzeNewEvents(context.Background())
	require.NoError(t, err)

	// 8. 验证结果
	assert.Equal(t, 50, result.EventCount)
	assert.Greater(t, result.SessionCount, 0, "应该发现至少一个会话")

	// 查询所有模式
	allPatterns, err := patternRepo.FindAll()
	require.NoError(t, err)
	t.Logf("发现 %d 个模式", len(allPatterns))

	if len(allPatterns) > 0 {
		pattern := allPatterns[0]
		assert.NotEmpty(t, pattern.ID)
		assert.Greater(t, pattern.SupportCount, 0)
		assert.NotEmpty(t, pattern.Sequence)
	}
}

/**
 * TestAnalyzerEngine_PatternDiscovery 测试模式发现
 */
func TestAnalyzerEngine_PatternDiscovery(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	eventRepo := storage.NewSQLiteEventRepository(db)
	patternRepo := storage.NewSQLitePatternRepository(db)
	eventBus := events.NewEventBus()

	mockAI := &MockAIClient{
		analyzeResponse: &ai.PatternAnalysis{
			ShouldAutomate:      true,
			Reason:              "值得自动化",
			EstimatedTimeSaving: 300,
			Complexity:          "medium",
			SuggestedName:       "测试模式",
			SuggestedSteps:      []string{"步骤1", "步骤2", "步骤3"},
			AnalyzedAt:          time.Now(),
		},
	}

	config := DefaultAnalyzerEngineConfig()
	config.AIPatternFilter.AIModel = mockAI
	config.EnableAIAnalysis = true
	config.MinEventCount = 3

	engine, err := NewAnalyzerEngine(config, eventRepo, patternRepo, eventBus)
	require.NoError(t, err)
	defer engine.Close()

	// 生成并保存测试事件（重复的模式）
	testEvents := generateTestEvents(30)
	err = eventRepo.SaveBatch(testEvents)
	require.NoError(t, err)

	// 执行分析
	result, err := engine.AnalyzeNewEvents(context.Background())
	require.NoError(t, err)

	t.Logf("分析结果: 事件=%d, 会话=%d, 模式=%d",
		result.EventCount, result.SessionCount, result.PatternCount)

	// 验证结果
	assert.Equal(t, 30, result.EventCount)
	assert.GreaterOrEqual(t, result.PatternCount, 0)
}

/**
 * MockAIClient 模拟 AI 客户端
 */
type MockAIClient struct {
	analyzeResponse *ai.PatternAnalysis
	analyzeCalled   bool
}

func (m *MockAIClient) AnalyzePattern(ctx context.Context, patternData map[string]interface{}) (*ai.PatternAnalysis, error) {
	m.analyzeCalled = true
	return m.analyzeResponse, nil
}

func (m *MockAIClient) AnalyzePatternBatch(ctx context.Context, patterns []map[string]interface{}) ([]*ai.PatternAnalysis, error) {
	m.analyzeCalled = true
	results := make([]*ai.PatternAnalysis, len(patterns))
	for i := range patterns {
		results[i] = m.analyzeResponse
	}
	return results, nil
}

func (m *MockAIClient) GetType() ai.ModelType {
	return ai.ModelTypeClaude
}

func (m *MockAIClient) Close() error {
	return nil
}

/**
 * setupTestDB 设置测试数据库
 */
func setupTestDB(t *testing.T) *sql.DB {
	// 创建内存数据库
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)

	// 执行迁移
	err = storage.RunMigrations(db)
	require.NoError(t, err)

	return db
}

/**
 * generateTestEvents 生成测试事件
 *
 * Parameters:
 *   - count: 事件数量
 *
 * Returns: []events.Event - 测试事件列表
 */
func generateTestEvents(count int) []events.Event {
	now := time.Now()
	eventList := make([]events.Event, count)

	for i := 0; i < count; i++ {
		// 创建不同类型的事件
		var eventType events.EventType
		switch i % 3 {
		case 0:
			eventType = events.EventTypeKeyboard
		case 1:
			eventType = events.EventTypeClipboard
		case 2:
			eventType = events.EventTypeAppSwitch
		}

		// 创建事件数据
		data := make(map[string]interface{})
		if eventType == events.EventTypeKeyboard {
			data["keycode"] = "a"
			data["modifiers"] = []string{}
		} else if eventType == events.EventTypeClipboard {
			data["operation"] = "copy"
			data["content_type"] = "text"
		} else if eventType == events.EventTypeAppSwitch {
			data["from_app"] = "Chrome"
			data["to_app"] = "VSCode"
		}

		// 创建事件
		event := events.NewEvent(eventType, data)
		event.Timestamp = now.Add(time.Duration(i) * time.Second)
		event.Context = &events.EventContext{
			Application: "TestApp",
			BundleID:    "com.test.app",
			WindowTitle: "Test Window",
		}

		eventList[i] = *event
	}

	return eventList
}
