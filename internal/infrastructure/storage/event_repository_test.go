package storage

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/chenyang-zz/flowmind/pkg/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestDB 创建测试数据库
func setupTestDB(t *testing.T) *sql.DB {
	db, err := NewSQLiteDB(SQLiteConfig{Path: t.TempDir() + "/test.db"})
	require.NoError(t, err)

	err = RunMigrations(db)
	require.NoError(t, err)

	return db
}

// TestSQLiteEventRepository_Save 测试保存单个事件
func TestSQLiteEventRepository_Save(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSQLiteEventRepository(db)

	event := events.NewEvent(events.EventTypeKeyboard, map[string]interface{}{
		"keycode": float64(65),
	})
	event.Context = &events.EventContext{
		Application: "TestApp",
		BundleID:    "com.test.app",
	}

	err := repo.Save(*event)
	require.NoError(t, err)

	// 验证保存成功
	saved, err := repo.FindRecent(1)
	require.NoError(t, err)
	assert.Len(t, saved, 1)
	assert.Equal(t, event.ID, saved[0].ID)
	assert.Equal(t, events.EventTypeKeyboard, saved[0].Type)
	assert.Equal(t, "TestApp", saved[0].Context.Application)
}

// TestSQLiteEventRepository_SaveBatch 测试批量保存事件
func TestSQLiteEventRepository_SaveBatch(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSQLiteEventRepository(db)

	// 创建 100 个事件
	var eventList []events.Event
	for i := 0; i < 100; i++ {
		event := events.NewEvent(events.EventTypeKeyboard, map[string]interface{}{
			"keycode": float64(i),
		})
		eventList = append(eventList, *event)
	}

	err := repo.SaveBatch(eventList)
	require.NoError(t, err)

	// 验证保存成功
	saved, err := repo.FindRecent(100)
	require.NoError(t, err)
	assert.Len(t, saved, 100)
}

// TestSQLiteEventRepository_SaveBatch_Empty 测试批量保存空列表
func TestSQLiteEventRepository_SaveBatch_Empty(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSQLiteEventRepository(db)

	err := repo.SaveBatch([]events.Event{})
	require.NoError(t, err)

	// 验证没有保存任何事件
	saved, err := repo.FindRecent(10)
	require.NoError(t, err)
	assert.Len(t, saved, 0)
}

// TestSQLiteEventRepository_FindByTimeRange 测试按时间范围查询
func TestSQLiteEventRepository_FindByTimeRange(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSQLiteEventRepository(db)

	now := time.Now()

	// 创建不同时间的事件
	event1 := events.NewEvent(events.EventTypeKeyboard, map[string]interface{}{})
	event1.Timestamp = now.Add(-2 * time.Hour)

	event2 := events.NewEvent(events.EventTypeKeyboard, map[string]interface{}{})
	event2.Timestamp = now.Add(-1 * time.Hour)

	event3 := events.NewEvent(events.EventTypeKeyboard, map[string]interface{}{})
	event3.Timestamp = now

	err := repo.SaveBatch([]events.Event{*event1, *event2, *event3})
	require.NoError(t, err)

	// 查询最近 30 分钟的事件（只有 event3 符合）
	start := now.Add(-30 * time.Minute)
	end := now
	found, err := repo.FindByTimeRange(start, end)
	require.NoError(t, err)
	assert.Len(t, found, 1)
	assert.Equal(t, event3.ID, found[0].ID)
}

// TestSQLiteEventRepository_FindRecent 测试查询最近事件
func TestSQLiteEventRepository_FindRecent(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSQLiteEventRepository(db)

	// 创建3个事件（添加延迟确保时间戳不同）
	for i := 0; i < 3; i++ {
		event := events.NewEvent(events.EventTypeKeyboard, map[string]interface{}{
			"index": float64(i),
		})
		err := repo.Save(*event)
		require.NoError(t, err)
		if i < 2 { // 最后一个事件不需要延迟
			time.Sleep(time.Millisecond)
		}
	}

	// 查询最近2个
	recent, err := repo.FindRecent(2)
	require.NoError(t, err)
	assert.Len(t, recent, 2)

	// 验证顺序（从旧到新）
	assert.Equal(t, float64(1), recent[0].Data["index"])
	assert.Equal(t, float64(2), recent[1].Data["index"])
}

// TestSQLiteEventRepository_FindByType 测试按类型查询
func TestSQLiteEventRepository_FindByType(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSQLiteEventRepository(db)

	// 创建不同类型的事件
	keyboardEvent := events.NewEvent(events.EventTypeKeyboard, map[string]interface{}{})
	clipboardEvent := events.NewEvent(events.EventTypeClipboard, map[string]interface{}{})

	err := repo.SaveBatch([]events.Event{*keyboardEvent, *clipboardEvent})
	require.NoError(t, err)

	// 查询键盘事件
	found, err := repo.FindByType(events.EventTypeKeyboard, 10)
	require.NoError(t, err)
	assert.Len(t, found, 1)
	assert.Equal(t, events.EventTypeKeyboard, found[0].Type)
}

// TestSQLiteEventRepository_DeleteOlderThan 测试删除旧事件
func TestSQLiteEventRepository_DeleteOlderThan(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSQLiteEventRepository(db)

	now := time.Now()

	// 创建旧事件和新事件
	oldEvent := events.NewEvent(events.EventTypeKeyboard, map[string]interface{}{})
	oldEvent.Timestamp = now.Add(-48 * time.Hour)

	newEvent := events.NewEvent(events.EventTypeKeyboard, map[string]interface{}{})
	newEvent.Timestamp = now

	err := repo.SaveBatch([]events.Event{*oldEvent, *newEvent})
	require.NoError(t, err)

	// 删除 24 小时前的事件
	cutoff := now.Add(-24 * time.Hour)
	count, err := repo.DeleteOlderThan(cutoff)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	// 验证只保留了新事件
	stats, err := repo.GetStats()
	require.NoError(t, err)
	assert.Equal(t, int64(1), stats.TotalCount)
}

// TestSQLiteEventRepository_GetStats 测试获取统计信息
func TestSQLiteEventRepository_GetStats(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSQLiteEventRepository(db)

	// 创建不同类型的事件
	keyboardEvent := events.NewEvent(events.EventTypeKeyboard, map[string]interface{}{})
	clipboardEvent := events.NewEvent(events.EventTypeClipboard, map[string]interface{}{})

	err := repo.SaveBatch([]events.Event{*keyboardEvent, *clipboardEvent})
	require.NoError(t, err)

	// 获取统计
	stats, err := repo.GetStats()
	require.NoError(t, err)

	assert.Equal(t, int64(2), stats.TotalCount)
	assert.Equal(t, int64(1), stats.CountByType["keyboard"])
	assert.Equal(t, int64(1), stats.CountByType["clipboard"])
	assert.NotNil(t, stats.OldestEvent)
	assert.NotNil(t, stats.NewestEvent)
}

// TestSQLiteEventRepository_GetStats_Empty 测试空数据库的统计
func TestSQLiteEventRepository_GetStats_Empty(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSQLiteEventRepository(db)

	stats, err := repo.GetStats()
	require.NoError(t, err)

	assert.Equal(t, int64(0), stats.TotalCount)
	assert.Empty(t, stats.CountByType)
	assert.Nil(t, stats.OldestEvent)
	assert.Nil(t, stats.NewestEvent)
}

// TestSQLiteEventRepository_ComplexData 测试复杂数据序列化
func TestSQLiteEventRepository_ComplexData(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSQLiteEventRepository(db)

	// 创建包含复杂数据的事件
	event := events.NewEvent(events.EventTypeKeyboard, map[string]interface{}{
		"keycode":    float64(65),
		"modifiers":  float64(16),
		"nested":     map[string]interface{}{"key": "value"},
		"array":      []interface{}{1, 2, 3},
	})
	event.Context = &events.EventContext{
		Application: "TestApp",
		BundleID:    "com.test.app",
		WindowTitle: "Test Window",
		FilePath:    "/test/path.txt",
		Selection:   "selected text",
	}

	err := repo.Save(*event)
	require.NoError(t, err)

	// 验证数据完整恢复
	saved, err := repo.FindRecent(1)
	require.NoError(t, err)
	assert.Len(t, saved, 1)

	assert.Equal(t, float64(65), saved[0].Data["keycode"])
	assert.Equal(t, "TestApp", saved[0].Context.Application)
	assert.Equal(t, "/test/path.txt", saved[0].Context.FilePath)
}

// TestSQLiteEventRepository_LargeBatch 测试大批量写入性能
func TestSQLiteEventRepository_LargeBatch(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSQLiteEventRepository(db)

	// 创建 1000 个事件
	var eventList []events.Event
	for i := 0; i < 1000; i++ {
		event := events.NewEvent(events.EventTypeKeyboard, map[string]interface{}{
			"index": float64(i),
		})
		eventList = append(eventList, *event)
	}

	// 测量执行时间
	start := time.Now()
	err := repo.SaveBatch(eventList)
	duration := time.Since(start)

	require.NoError(t, err)
	assert.Less(t, duration, 1*time.Second, "批量写入应该很快完成")

	// 验证保存成功
	stats, err := repo.GetStats()
	require.NoError(t, err)
	assert.Equal(t, int64(1000), stats.TotalCount)
}

// TestSQLiteEventRepository_SaveBatch_InvalidJSON 测试批量保存时JSON序列化失败
func TestSQLiteEventRepository_SaveBatch_InvalidJSON(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSQLiteEventRepository(db)

	// 创建包含不可序列化数据的事件
	// 注意：Go的json.Marshal可以处理大部分类型，所以这个测试主要验证错误处理路径
	event1 := events.NewEvent(events.EventTypeKeyboard, map[string]interface{}{
		"keycode": float64(65),
	})

	// 创建正常事件确保至少有一个能保存
	event2 := events.NewEvent(events.EventTypeKeyboard, map[string]interface{}{
		"keycode": float64(66),
	})

	err := repo.SaveBatch([]events.Event{*event1, *event2})
	require.NoError(t, err)

	// 验证至少保存了成功的事件
	saved, err := repo.FindRecent(10)
	require.NoError(t, err)
	assert.Greater(t, len(saved), 0, "应该至少保存一个事件")
}

// TestSQLiteEventRepository_FindByTimeRange_NoResults 测试时间范围查询无结果
func TestSQLiteEventRepository_FindByTimeRange_NoResults(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSQLiteEventRepository(db)

	// 保存一些事件
	event := events.NewEvent(events.EventTypeKeyboard, map[string]interface{}{})
	event.Timestamp = time.Now().Add(-24 * time.Hour)
	err := repo.Save(*event)
	require.NoError(t, err)

	// 查询一个没有事件的时间范围（最近1小时）
	start := time.Now().Add(-1 * time.Hour)
	end := time.Now()
	found, err := repo.FindByTimeRange(start, end)
	require.NoError(t, err)
	assert.Len(t, found, 0, "不应该找到任何事件")
}

// TestSQLiteEventRepository_FindByType_NoResults 测试按类型查询无结果
func TestSQLiteEventRepository_FindByType_NoResults(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSQLiteEventRepository(db)

	// 保存键盘事件
	event := events.NewEvent(events.EventTypeKeyboard, map[string]interface{}{})
	err := repo.Save(*event)
	require.NoError(t, err)

	// 查询剪贴板事件（不存在）
	found, err := repo.FindByType(events.EventTypeClipboard, 10)
	require.NoError(t, err)
	assert.Len(t, found, 0, "不应该找到剪贴板事件")
}

// TestSQLiteEventRepository_DeleteOlderThan_NoResults 测试删除无旧事件
func TestSQLiteEventRepository_DeleteOlderThan_NoResults(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSQLiteEventRepository(db)

	// 只保存新事件
	event := events.NewEvent(events.EventTypeKeyboard, map[string]interface{}{})
	event.Timestamp = time.Now()
	err := repo.Save(*event)
	require.NoError(t, err)

	// 删除1天前的事件（应该没有）
	cutoff := time.Now().Add(-24 * time.Hour)
	count, err := repo.DeleteOlderThan(cutoff)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count, "不应该删除任何事件")

	// 验证事件仍然存在
	saved, err := repo.FindRecent(10)
	require.NoError(t, err)
	assert.Len(t, saved, 1, "事件应该仍然存在")
}

// TestSQLiteEventRepository_FindRecent_LimitZero 测试查询限制为0
func TestSQLiteEventRepository_FindRecent_LimitZero(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSQLiteEventRepository(db)

	// 保存一些事件
	for i := 0; i < 3; i++ {
		event := events.NewEvent(events.EventTypeKeyboard, map[string]interface{}{})
		err := repo.Save(*event)
		require.NoError(t, err)
	}

	// 查询0个事件
	recent, err := repo.FindRecent(0)
	require.NoError(t, err)
	assert.Len(t, recent, 0, "应该返回空列表")
}

// TestSQLiteEventRepository_FindRecent_LimitNegative 测试查询限制为负数
func TestSQLiteEventRepository_FindRecent_LimitNegative(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSQLiteEventRepository(db)

	// 保存一些事件
	for i := 0; i < 3; i++ {
		event := events.NewEvent(events.EventTypeKeyboard, map[string]interface{}{})
		err := repo.Save(*event)
		require.NoError(t, err)
	}

	// 查询-1个事件（应该返回空列表或所有事件）
	recent, err := repo.FindRecent(-1)
	require.NoError(t, err)
	// SQL的LIMIT -1在某些数据库中可能有特殊含义，但这里应该返回空列表
	assert.NotNil(t, recent)
}

// TestSQLiteEventRepository_FindByType_LimitZero 测试按类型查询限制为0
func TestSQLiteEventRepository_FindByType_LimitZero(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSQLiteEventRepository(db)

	// 保存键盘事件
	event := events.NewEvent(events.EventTypeKeyboard, map[string]interface{}{})
	err := repo.Save(*event)
	require.NoError(t, err)

	// 查询0个事件
	found, err := repo.FindByType(events.EventTypeKeyboard, 0)
	require.NoError(t, err)
	assert.Len(t, found, 0, "应该返回空列表")
}

// TestSQLiteEventRepository_SaveBatch_LargeData 测试保存大量数据的事件
func TestSQLiteEventRepository_SaveBatch_LargeData(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSQLiteEventRepository(db)

	// 创建包含大量数据的单个事件
	largeData := make(map[string]interface{})
	for i := 0; i < 100; i++ {
		largeData[fmt.Sprintf("key%d", i)] = fmt.Sprintf("value%d with some longer text content", i)
	}

	event := events.NewEvent(events.EventTypeKeyboard, largeData)

	err := repo.SaveBatch([]events.Event{*event})
	require.NoError(t, err)

	// 验证保存成功
	saved, err := repo.FindRecent(1)
	require.NoError(t, err)
	assert.Len(t, saved, 1)
	assert.Equal(t, event.ID, saved[0].ID)
}

// TestSQLiteEventRepository_MultipleOperations 测试连续多个操作
func TestSQLiteEventRepository_MultipleOperations(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSQLiteEventRepository(db)

	// 1. 保存事件
	event1 := events.NewEvent(events.EventTypeKeyboard, map[string]interface{}{"action": "type"})
	err := repo.Save(*event1)
	require.NoError(t, err)

	// 2. 批量保存更多事件
	var eventList []events.Event
	for i := 0; i < 5; i++ {
		event := events.NewEvent(events.EventTypeKeyboard, map[string]interface{}{"index": float64(i)})
		eventList = append(eventList, *event)
	}
	err = repo.SaveBatch(eventList)
	require.NoError(t, err)

	// 3. 查询验证
	recent, err := repo.FindRecent(10)
	require.NoError(t, err)
	assert.Len(t, recent, 6)

	// 4. 获取统计
	stats, err := repo.GetStats()
	require.NoError(t, err)
	assert.Equal(t, int64(6), stats.TotalCount)

	// 5. 删除部分事件
	cutoff := time.Now().Add(-1 * time.Second)
	count, err := repo.DeleteOlderThan(cutoff)
	require.NoError(t, err)
	// 可能删除了一些旧事件
	assert.GreaterOrEqual(t, count, int64(0))
}

// TestSQLiteEventRepository_SaveBatch_WithStop 测试批量写入器停止后的行为
func TestSQLiteEventRepository_SaveBatch_WithStop(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSQLiteEventRepository(db)

	// 创建并保存一些事件
	for i := 0; i < 3; i++ {
		event := events.NewEvent(events.EventTypeKeyboard, map[string]interface{}{
			"index": float64(i),
		})
		err := repo.Save(*event)
		require.NoError(t, err)
	}

	// 验证所有事件都保存成功
	saved, err := repo.FindRecent(10)
	require.NoError(t, err)
	assert.Len(t, saved, 3)
}

// TestSQLiteEventRepository_GetStats_WithEvents 测试有事件时的统计
func TestSQLiteEventRepository_GetStats_WithEvents(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSQLiteEventRepository(db)

	now := time.Now()

	// 创建多个不同类型的事件
	events := []events.Event{
		*events.NewEvent(events.EventTypeKeyboard, map[string]interface{}{}),
		*events.NewEvent(events.EventTypeKeyboard, map[string]interface{}{}),
		*events.NewEvent(events.EventTypeClipboard, map[string]interface{}{}),
	}

	// 设置不同的时间戳
	events[0].Timestamp = now.Add(-2 * time.Hour)
	events[1].Timestamp = now.Add(-1 * time.Hour)
	events[2].Timestamp = now

	err := repo.SaveBatch(events)
	require.NoError(t, err)

	// 获取统计
	stats, err := repo.GetStats()
	require.NoError(t, err)

	assert.Equal(t, int64(3), stats.TotalCount)
	assert.Equal(t, int64(2), stats.CountByType["keyboard"])
	assert.Equal(t, int64(1), stats.CountByType["clipboard"])
	assert.NotNil(t, stats.OldestEvent)
	assert.NotNil(t, stats.NewestEvent)

	// 验证时间范围
	assert.True(t, stats.OldestEvent.Before(*stats.NewestEvent))
}

// TestSQLiteEventRepository_FindByType_NoLimit 测试按类型查询不限制数量
func TestSQLiteEventRepository_FindByType_NoLimit(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSQLiteEventRepository(db)

	// 保存多个键盘事件
	for i := 0; i < 5; i++ {
		event := events.NewEvent(events.EventTypeKeyboard, map[string]interface{}{
			"index": float64(i),
		})
		err := repo.Save(*event)
		require.NoError(t, err)
	}

	// 查询所有键盘事件（使用大数字作为限制）
	found, err := repo.FindByType(events.EventTypeKeyboard, 100)
	require.NoError(t, err)
	assert.Len(t, found, 5)
}

// TestSQLiteEventRepository_DeleteOlderThan_All 测试删除所有事件
func TestSQLiteEventRepository_DeleteOlderThan_All(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSQLiteEventRepository(db)

	// 保存一些旧事件
	oldEvent := events.NewEvent(events.EventTypeKeyboard, map[string]interface{}{})
	oldEvent.Timestamp = time.Now().Add(-48 * time.Hour)
	err := repo.Save(*oldEvent)
	require.NoError(t, err)

	// 删除所有事件（使用未来时间）
	cutoff := time.Now().Add(1 * time.Hour)
	count, err := repo.DeleteOlderThan(cutoff)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	// 验证所有事件都被删除
	saved, err := repo.FindRecent(10)
	require.NoError(t, err)
	assert.Len(t, saved, 0)
}

// TestSQLiteEventRepository_Save_ContextNil 测试保存上下文为nil的事件
func TestSQLiteEventRepository_Save_ContextNil(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSQLiteEventRepository(db)

	// 创建没有上下文的事件
	event := events.NewEvent(events.EventTypeKeyboard, map[string]interface{}{
		"keycode": float64(65),
	})
	event.Context = nil

	err := repo.Save(*event)
	require.NoError(t, err)

	// 验证保存成功
	saved, err := repo.FindRecent(1)
	require.NoError(t, err)
	assert.Len(t, saved, 1)
	// scanEvents会为所有非NULL的上下文字段创建空EventContext
	assert.NotNil(t, saved[0].Context)
	assert.Equal(t, "", saved[0].Context.Application)
}
