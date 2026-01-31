package storage

import (
	"testing"
	"time"

	"github.com/chenyang-zz/flowmind/pkg/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBatchWriter_StartStop 测试启动和停止
func TestBatchWriter_StartStop(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSQLiteEventRepository(db)
	config := DefaultBatchWriterConfig()
	config.BatchSize = 10
	config.FlushInterval = 100 * time.Millisecond

	bw := NewBatchWriter(repo, config)

	assert.False(t, bw.IsStarted())

	bw.Start()
	assert.True(t, bw.IsStarted())

	bw.Stop()
	assert.False(t, bw.IsStarted())
}

// TestBatchWriter_Write 测试单个事件写入
func TestBatchWriter_Write(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSQLiteEventRepository(db)
	config := DefaultBatchWriterConfig()
	config.BatchSize = 5
	config.FlushInterval = 1 * time.Second

	bw := NewBatchWriter(repo, config)
	bw.Start()
	defer bw.Stop()

	// 写入 3 个事件
	for i := 0; i < 3; i++ {
		event := events.NewEvent(events.EventTypeKeyboard, map[string]interface{}{
			"index": float64(i),
		})
		success := bw.Write(*event)
		assert.True(t, success)
	}

	// 等待批量写入
	time.Sleep(200 * time.Millisecond)

	// 强制刷新确保所有事件写入
	bw.ForceFlush()

	// 验证写入成功
	saved, err := repo.FindRecent(10)
	require.NoError(t, err)
	assert.Len(t, saved, 3)
}

// TestBatchWriter_WriteBatch 测试批量写入
func TestBatchWriter_WriteBatch(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSQLiteEventRepository(db)
	config := DefaultBatchWriterConfig()
	config.BatchSize = 20
	config.FlushInterval = 1 * time.Second

	bw := NewBatchWriter(repo, config)
	bw.Start()
	defer bw.Stop()

	// 创建 10 个事件
	var eventList []events.Event
	for i := 0; i < 10; i++ {
		event := events.NewEvent(events.EventTypeKeyboard, map[string]interface{}{
			"index": float64(i),
		})
		eventList = append(eventList, *event)
	}

	// 批量写入
	successCount := bw.WriteBatch(eventList)
	assert.Equal(t, 10, successCount)

	// 等待写入
	time.Sleep(200 * time.Millisecond)
	bw.ForceFlush()

	// 验证
	saved, err := repo.FindRecent(20)
	require.NoError(t, err)
	assert.Len(t, saved, 10)
}

// TestBatchWriter_AutoFlush 测试自动批量刷新
func TestBatchWriter_AutoFlush(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSQLiteEventRepository(db)
	config := DefaultBatchWriterConfig()
	config.BatchSize = 5 // 5 个事件触发自动刷新
	config.FlushInterval = 10 * time.Second

	bw := NewBatchWriter(repo, config)
	bw.Start()
	defer bw.Stop()

	// 写入 5 个事件（刚好达到批量大小）
	for i := 0; i < 5; i++ {
		event := events.NewEvent(events.EventTypeKeyboard, map[string]interface{}{
			"index": float64(i),
		})
		bw.Write(*event)
	}

	// 等待异步写入完成
	time.Sleep(300 * time.Millisecond)

	// 验证自动刷新成功
	saved, err := repo.FindRecent(10)
	require.NoError(t, err)
	assert.Len(t, saved, 5)

	// 缓冲区应该已清空
	assert.Equal(t, 0, bw.GetBufferSize())
}

// TestBatchWriter_TimedFlush 测试定时刷新
func TestBatchWriter_TimedFlush(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSQLiteEventRepository(db)
	config := DefaultBatchWriterConfig()
	config.BatchSize = 100 // 设置较大，避免触发自动刷新
	config.FlushInterval = 500 * time.Millisecond // 500ms 刷新

	bw := NewBatchWriter(repo, config)
	bw.Start()
	defer bw.Stop()

	// 写入 2 个事件（不够触发自动刷新）
	for i := 0; i < 2; i++ {
		event := events.NewEvent(events.EventTypeKeyboard, map[string]interface{}{
			"index": float64(i),
		})
		bw.Write(*event)
	}

	// 等待定时刷新
	time.Sleep(800 * time.Millisecond)

	// 验证定时刷新成功
	saved, err := repo.FindRecent(10)
	require.NoError(t, err)
	assert.Len(t, saved, 2)
}

// TestBatchWriter_ForceFlush 测试强制刷新
func TestBatchWriter_ForceFlush(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSQLiteEventRepository(db)
	config := DefaultBatchWriterConfig()
	config.BatchSize = 100
	config.FlushInterval = 10 * time.Second

	bw := NewBatchWriter(repo, config)
	bw.Start()
	defer bw.Stop()

	// 写入 3 个事件
	for i := 0; i < 3; i++ {
		event := events.NewEvent(events.EventTypeKeyboard, map[string]interface{}{
			"index": float64(i),
		})
		bw.Write(*event)
	}

	// 等待后台 goroutine 将事件从通道移到缓冲区
	time.Sleep(100 * time.Millisecond)

	// 强制刷新
	bw.ForceFlush()

	// 验证刷新成功
	saved, err := repo.FindRecent(10)
	require.NoError(t, err)
	assert.Len(t, saved, 3)
}

// TestBatchWriter_StopWithBuffer 测试停止时刷新缓冲区
func TestBatchWriter_StopWithBuffer(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSQLiteEventRepository(db)
	config := DefaultBatchWriterConfig()
	config.BatchSize = 100
	config.FlushInterval = 10 * time.Second

	bw := NewBatchWriter(repo, config)
	bw.Start()

	// 写入 5 个事件
	for i := 0; i < 5; i++ {
		event := events.NewEvent(events.EventTypeKeyboard, map[string]interface{}{
			"index": float64(i),
		})
		bw.Write(*event)
	}

	// 等待后台 goroutine 将事件从通道移到缓冲区
	time.Sleep(100 * time.Millisecond)

	// 停止（应该自动刷新缓冲区）
	bw.Stop()

	// 验证停止后数据已写入
	saved, err := repo.FindRecent(10)
	require.NoError(t, err)
	assert.Len(t, saved, 5)
}

// TestBatchWriter_ConcurrentWrite 测试并发写入
func TestBatchWriter_ConcurrentWrite(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSQLiteEventRepository(db)
	config := DefaultBatchWriterConfig()
	config.BatchSize = 50
	config.FlushInterval = 100 * time.Millisecond
	config.EventBuffer = 1000

	bw := NewBatchWriter(repo, config)
	bw.Start()
	defer bw.Stop()

	// 并发写入 100 个事件
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(startIdx int) {
			for j := 0; j < 10; j++ {
				event := events.NewEvent(events.EventTypeKeyboard, map[string]interface{}{
					"index": float64(startIdx*10 + j),
				})
				bw.Write(*event)
			}
			done <- true
		}(i)
	}

	// 等待所有 goroutine 完成
	for i := 0; i < 10; i++ {
		<-done
	}

	// 等待写入完成
	time.Sleep(500 * time.Millisecond)
	bw.ForceFlush()

	// 验证所有事件都写入成功
	saved, err := repo.FindRecent(200)
	require.NoError(t, err)
	assert.Len(t, saved, 100)
}

// TestBatchWriter_ChannelFull 测试通道满时的行为
func TestBatchWriter_ChannelFull(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSQLiteEventRepository(db)
	config := DefaultBatchWriterConfig()
	config.BatchSize = 10
	config.FlushInterval = 1 * time.Second
	config.EventBuffer = 5 // 小缓冲区

	bw := NewBatchWriter(repo, config)
	bw.Start()
	defer bw.Stop()

	// 写入超过缓冲区大小的事件
	successCount := 0
	for i := 0; i < 10; i++ {
		event := events.NewEvent(events.EventTypeKeyboard, map[string]interface{}{
			"index": float64(i),
		})
		if bw.Write(*event) {
			successCount++
		}
	}

	// 部分事件应该因为通道满而失败
	assert.Less(t, successCount, 10)
	assert.Greater(t, successCount, 0)

	// 等待写入
	time.Sleep(200 * time.Millisecond)
	bw.ForceFlush()

	// 验证成功的事件都写入了
	saved, err := repo.FindRecent(20)
	require.NoError(t, err)
	assert.Equal(t, successCount, len(saved))
}

// TestBatchWriter_EmptyBatch 测试空批量写入
func TestBatchWriter_EmptyBatch(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSQLiteEventRepository(db)
	config := DefaultBatchWriterConfig()
	config.BatchSize = 10
	config.FlushInterval = 100 * time.Millisecond

	bw := NewBatchWriter(repo, config)
	bw.Start()

	// 不写入任何事件
	// 直接强制刷新（应该正常处理空缓冲区）
	bw.ForceFlush()

	bw.Stop()

	// 验证没有数据写入
	saved, err := repo.FindRecent(10)
	require.NoError(t, err)
	assert.Len(t, saved, 0)
}

// TestBatchWriter_LargeVolume 测试大批量写入性能
func TestBatchWriter_LargeVolume(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSQLiteEventRepository(db)
	config := DefaultBatchWriterConfig()
	config.BatchSize = 100
	config.FlushInterval = 50 * time.Millisecond
	config.EventBuffer = 2000

	bw := NewBatchWriter(repo, config)
	bw.Start()
	defer bw.Stop()

	// 写入 1000 个事件
	startTime := time.Now()
	for i := 0; i < 1000; i++ {
		event := events.NewEvent(events.EventTypeKeyboard, map[string]interface{}{
			"index": float64(i),
		})
		bw.Write(*event)
	}
	writeDuration := time.Since(startTime)

	// 等待所有写入完成
	time.Sleep(1 * time.Second)
	bw.ForceFlush()

	// 验证写入成功
	saved, err := repo.FindRecent(2000)
	require.NoError(t, err)
	assert.Len(t, saved, 1000)

	// 性能验证：写入 1000 个事件应该很快
	assert.Less(t, writeDuration, 1*time.Second, "写入应该快速完成")

	// 验证批量写入性能（>1000 events/sec）
	eventsPerSec := float64(1000) / writeDuration.Seconds()
	assert.Greater(t, eventsPerSec, float64(1000), "批量写入性能应该 >1000 events/sec")
}

// TestBatchWriter_MultipleStart 测试重复启动
func TestBatchWriter_MultipleStart(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSQLiteEventRepository(db)
	config := DefaultBatchWriterConfig()
	bw := NewBatchWriter(repo, config)

	bw.Start()
	assert.True(t, bw.IsStarted())

	// 重复启动应该安全
	bw.Start()
	assert.True(t, bw.IsStarted())

	bw.Stop()
	assert.False(t, bw.IsStarted())
}

// TestBatchWriter_WriteBeforeStart 测试启动前写入
func TestBatchWriter_WriteBeforeStart(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSQLiteEventRepository(db)
	config := DefaultBatchWriterConfig()
	bw := NewBatchWriter(repo, config)

	// 未启动时写入
	event := events.NewEvent(events.EventTypeKeyboard, map[string]interface{}{})

	// 应该能写入通道（不会阻塞）
	success := bw.Write(*event)
	assert.True(t, success)

	bw.Start()
	defer bw.Stop()

	// 等待处理
	time.Sleep(200 * time.Millisecond)
	bw.ForceFlush()

	// 验证事件被处理
	saved, err := repo.FindRecent(10)
	require.NoError(t, err)
	assert.Len(t, saved, 1)
}

// TestBatchWriter_Write_ReturnsFalse 测试Write返回false的情况
func TestBatchWriter_Write_ReturnsFalse(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSQLiteEventRepository(db)
	config := DefaultBatchWriterConfig()
	config.BatchSize = 10
	config.FlushInterval = 10 * time.Second
	config.EventBuffer = 2 // 非常小的缓冲区

	bw := NewBatchWriter(repo, config)
	bw.Start()
	defer bw.Stop()

	// 快速写入超过缓冲区大小的事件
	successCount := 0
	for i := 0; i < 5; i++ {
		event := events.NewEvent(events.EventTypeKeyboard, map[string]interface{}{
			"index": float64(i),
		})
		if bw.Write(*event) {
			successCount++
		}
	}

	// 部分事件应该写入失败
	assert.Less(t, successCount, 5, "至少应该有一些事件写入失败")
	assert.Greater(t, successCount, 0, "至少应该有一些事件写入成功")
}

// TestBatchWriter_Stop_EmptyBuffer 测试停止时缓冲区为空
func TestBatchWriter_Stop_EmptyBuffer(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSQLiteEventRepository(db)
	config := DefaultBatchWriterConfig()
	config.BatchSize = 10
	config.FlushInterval = 1 * time.Second

	bw := NewBatchWriter(repo, config)
	bw.Start()

	// 不写入任何事件，直接停止
	bw.Stop()

	// 验证没有数据写入
	saved, err := repo.FindRecent(10)
	require.NoError(t, err)
	assert.Len(t, saved, 0)
}

// TestBatchWriter_GetBufferSize_NotStarted 测试未启动时获取缓冲区大小
func TestBatchWriter_GetBufferSize_NotStarted(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSQLiteEventRepository(db)
	config := DefaultBatchWriterConfig()
	bw := NewBatchWriter(repo, config)

	// 未启动时获取缓冲区大小
	size := bw.GetBufferSize()
	assert.Equal(t, 0, size, "未启动时缓冲区应该为空")
}

// TestBatchWriter_ForceFlush_NotStarted 测试未启动时强制刷新
func TestBatchWriter_ForceFlush_NotStarted(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewSQLiteEventRepository(db)
	config := DefaultBatchWriterConfig()
	bw := NewBatchWriter(repo, config)

	// 未启动时强制刷新（应该安全处理）
	bw.ForceFlush()

	// 验证没有错误发生
	assert.Equal(t, 0, bw.GetBufferSize())
}
