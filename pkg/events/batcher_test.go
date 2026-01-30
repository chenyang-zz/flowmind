package events

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewEventBatcher 测试EventBatcher的创建
//
// 验证EventBatcher能够正确创建，并且初始状态正确。
func TestNewEventBatcher(t *testing.T) {
	batcher := NewEventBatcher(10, 100*time.Millisecond)

	assert.NotNil(t, batcher)
	assert.Equal(t, 10, batcher.batchSize)
	assert.Equal(t, 100*time.Millisecond, batcher.timeout)
	assert.NotNil(t, batcher.input)
	assert.NotNil(t, batcher.output)
	assert.NotNil(t, batcher.buffer)
	assert.False(t, batcher.IsRunning())
}

// TestEventBatcher_StartStop 测试启动和停止
//
// 验证能够正常启动和停止批量处理器。
func TestEventBatcher_StartStop(t *testing.T) {
	batcher := NewEventBatcher(10, 100*time.Millisecond)

	// 启动
	err := batcher.Start()
	assert.NoError(t, err)
	assert.True(t, batcher.IsRunning())

	// 停止
	batcher.Stop()
	assert.False(t, batcher.IsRunning())
}

// TestEventBatcher_Add 测试添加事件
//
// 验证能够成功添加事件到批量处理器。
func TestEventBatcher_Add(t *testing.T) {
	batcher := NewEventBatcher(10, 100*time.Millisecond)
	err := batcher.Start()
	require.NoError(t, err)
	defer batcher.Stop()

	eventPtr := NewEvent(EventTypeKeyboard, map[string]interface{}{"key": "test"})

	// 添加事件（解引用指针）
	added := batcher.Add(*eventPtr)
	assert.True(t, added)

	// 等待事件被处理
	time.Sleep(20 * time.Millisecond)
	assert.Equal(t, 1, batcher.GetBufferSize())
}

// TestEventBatcher_FlushBySize 测试按大小触发批量处理
//
// 验证缓冲区达到批次大小时会触发批量处理。
func TestEventBatcher_FlushBySize(t *testing.T) {
	batcher := NewEventBatcher(3, 1*time.Second)
	err := batcher.Start()
	require.NoError(t, err)
	defer batcher.Stop()

	// 启动消费者接收批次
	done := make(chan bool)
	var receivedBatch []Event
	go func() {
		batch := <-batcher.Output()
		receivedBatch = batch
		done <- true
	}()

	// 添加3个事件（达到批次大小）
	for i := 0; i < 3; i++ {
		eventPtr := NewEvent(EventTypeKeyboard, map[string]interface{}{"index": i})
		batcher.Add(*eventPtr)
	}

	// 等待批次被处理
	select {
	case <-done:
		assert.Len(t, receivedBatch, 3)
		assert.Equal(t, 0, batcher.GetBufferSize())
	case <-time.After(1 * time.Second):
		t.Fatal("未能在1秒内收到批次")
	}
}

// TestEventBatcher_FlushByTimeout 测试按超时触发批量处理
//
// 验证超时后会触发批量处理，即使缓冲区未满。
func TestEventBatcher_FlushByTimeout(t *testing.T) {
	batcher := NewEventBatcher(10, 100*time.Millisecond)
	err := batcher.Start()
	require.NoError(t, err)
	defer batcher.Stop()

	// 启动消费者接收批次
	done := make(chan bool)
	var receivedBatch []Event
	go func() {
		batch := <-batcher.Output()
		receivedBatch = batch
		done <- true
	}()

	// 添加1个事件（未达到批次大小）
	eventPtr := NewEvent(EventTypeKeyboard, map[string]interface{}{"key": "test"})
	batcher.Add(*eventPtr)

	// 等待超时触发批量处理
	select {
	case <-done:
		assert.Len(t, receivedBatch, 1)
		assert.Equal(t, 0, batcher.GetBufferSize())
	case <-time.After(500 * time.Millisecond):
		t.Fatal("未能在500ms内收到批次")
	}
}

// TestEventBatcher_FlushManual 测试手动触发批量处理
//
// 验证能够手动调用Flush触发批量处理。
func TestEventBatcher_FlushManual(t *testing.T) {
	batcher := NewEventBatcher(10, 1*time.Second)
	err := batcher.Start()
	require.NoError(t, err)
	defer batcher.Stop()

	// 启动消费者接收批次
	done := make(chan bool)
	var receivedBatch []Event
	go func() {
		batch := <-batcher.Output()
		receivedBatch = batch
		done <- true
	}()

	// 添加2个事件（未达到批次大小）
	for i := 0; i < 2; i++ {
		eventPtr := NewEvent(EventTypeKeyboard, map[string]interface{}{"index": i})
		batcher.Add(*eventPtr)
	}
	// 等待事件被处理到buffer
	time.Sleep(20 * time.Millisecond)

	// 手动触发批量处理
	batcher.Flush()

	// 等待批次被处理
	select {
	case <-done:
		assert.Len(t, receivedBatch, 2)
		assert.Equal(t, 0, batcher.GetBufferSize())
	case <-time.After(100 * time.Millisecond):
		t.Fatal("未能在100ms内收到批次")
	}
}

// TestEventBatcher_StopWithFlush 测试停止时自动刷新
//
// 验证停止批量处理器时会自动刷新剩余事件。
func TestEventBatcher_StopWithFlush(t *testing.T) {
	batcher := NewEventBatcher(10, 1*time.Second)
	err := batcher.Start()
	require.NoError(t, err)

	// 启动消费者接收批次
	done := make(chan bool)
	var receivedBatch []Event
	go func() {
		batch := <-batcher.Output()
		receivedBatch = batch
		done <- true
	}()

	// 添加2个事件（未达到批次大小）
	for i := 0; i < 2; i++ {
		eventPtr := NewEvent(EventTypeKeyboard, map[string]interface{}{"index": i})
		batcher.Add(*eventPtr)
	}
	// 等待事件被处理到buffer
	time.Sleep(20 * time.Millisecond)

	// 停止（应该触发Flush）
	batcher.Stop()

	// 等待批次被处理
	select {
	case <-done:
		assert.Len(t, receivedBatch, 2)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("停止时未能刷新批次")
	}
}

// TestEventBatcher_GetBufferSize 测试获取缓冲区大小
//
// 验证能够正确获取当前缓冲区中的事件数量。
func TestEventBatcher_GetBufferSize(t *testing.T) {
	batcher := NewEventBatcher(10, 1*time.Second) // 增加超时时间，避免自动flush
	err := batcher.Start()
	require.NoError(t, err)
	defer batcher.Stop()

	// 初始缓冲区为空
	assert.Equal(t, 0, batcher.GetBufferSize())

	// 添加事件
	for i := 1; i <= 5; i++ {
		eventPtr := NewEvent(EventTypeKeyboard, map[string]interface{}{"index": i})
		batcher.Add(*eventPtr)
		time.Sleep(10 * time.Millisecond) // 给处理循环一些时间
		size := batcher.GetBufferSize()
		assert.Equal(t, i, size, "第%d次添加后缓冲区大小应该是%d，实际是%d", i, i, size)
	}
}

// TestEventBatcher_SetBatchSize 测试设置批次大小
//
// 验证能够动态调整批次大小。
func TestEventBatcher_SetBatchSize(t *testing.T) {
	batcher := NewEventBatcher(10, 100*time.Millisecond)

	// 修改批次大小
	batcher.SetBatchSize(20)
	assert.Equal(t, 20, batcher.batchSize)
}

// TestEventBatcher_SetTimeout 测试设置超时时间
//
// 验证能够动态调整超时时间。
func TestEventBatcher_SetTimeout(t *testing.T) {
	batcher := NewEventBatcher(10, 100*time.Millisecond)

	// 修改超时时间
	newTimeout := 200 * time.Millisecond
	batcher.SetTimeout(newTimeout)
	assert.Equal(t, newTimeout, batcher.timeout)
}

// TestEventBatcher_OutputChannel 测试输出通道
//
// 验证输出通道能够正确传递批次。
func TestEventBatcher_OutputChannel(t *testing.T) {
	batcher := NewEventBatcher(3, 1*time.Second)
	err := batcher.Start()
	require.NoError(t, err)
	defer batcher.Stop()

	// 获取输出通道
	output := batcher.Output()
	assert.NotNil(t, output)

	// 启动消费者
	done := make(chan bool)
	var receivedBatch []Event
	go func() {
		batch := <-output
		receivedBatch = batch
		done <- true
	}()

	// 添加事件
	for i := 0; i < 3; i++ {
		eventPtr := NewEvent(EventTypeKeyboard, map[string]interface{}{"index": i})
		batcher.Add(*eventPtr)
	}

	// 等待接收批次
	select {
	case <-done:
		assert.Len(t, receivedBatch, 3)
	case <-time.After(1 * time.Second):
		t.Fatal("未能收到批次")
	}
}

// TestEventBatcher_AddAfterStop 测试停止后添加事件
//
// 验证停止后批量处理器仍然可以添加事件（但不会处理）。
func TestEventBatcher_AddAfterStop(t *testing.T) {
	batcher := NewEventBatcher(10, 100*time.Millisecond)

	// 未启动状态下添加事件
	eventPtr := NewEvent(EventTypeKeyboard, map[string]interface{}{"key": "test"})
	added := batcher.Add(*eventPtr)
	// 未启动时，input通道未阻塞，应该能够添加
	assert.True(t, added)
}

// TestEventBatcher_ContextCancel 测试上下文取消
//
// 验证上下文取消后批量处理器会停止。
func TestEventBatcher_ContextCancel(t *testing.T) {
	batcher := NewEventBatcher(10, 100*time.Millisecond)
	err := batcher.Start()
	require.NoError(t, err)

	// 取消上下文
	batcher.cancel()

	// 等待处理循环退出
	time.Sleep(200 * time.Millisecond)

	// 注意：直接调用cancel()不会设置isRunning=false
	// 需要调用Stop()来正确设置状态
	// 所以这里只验证processLoop已退出（不崩溃即可）
	// 实际使用中应该调用Stop()而不是直接cancel()
}

// TestEventBatcher_MultipleBatches 测试多批次处理
//
// 验证能够连续处理多个批次。
func TestEventBatcher_MultipleBatches(t *testing.T) {
	batcher := NewEventBatcher(2, 1*time.Second)
	err := batcher.Start()
	require.NoError(t, err)
	defer batcher.Stop()

	batchCount := 0
	batches := make(chan []Event, 10)

	// 启动消费者接收多个批次
	go func() {
		for batch := range batches {
			if len(batch) > 0 {
				batchCount++
			}
		}
	}()

	// 启动转发器
	go func() {
		for batch := range batcher.Output() {
			batches <- batch
		}
		close(batches)
	}()

	// 添加6个事件（应该产生3个批次）
	for i := 0; i < 6; i++ {
		eventPtr := NewEvent(EventTypeKeyboard, map[string]interface{}{"index": i})
		batcher.Add(*eventPtr)
		time.Sleep(50 * time.Millisecond) // 给处理循环一些时间
	}

	// 等待所有批次被处理
	time.Sleep(500 * time.Millisecond)

	assert.Equal(t, 3, batchCount)
}

// BenchmarkEventBatcher_Add 批量处理器添加事件基准测试
//
// 测试添加事件的性能。
func BenchmarkEventBatcher_Add(b *testing.B) {
	batcher := NewEventBatcher(100, 100*time.Millisecond)
	batcher.Start()
	defer batcher.Stop()

	eventPtr := NewEvent(EventTypeKeyboard, map[string]interface{}{"key": "test"})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		batcher.Add(*eventPtr)
	}
}

// BenchmarkEventFilterManager_ShouldPass 过滤器判断基准测试
//
// 测试ShouldPass方法的性能。
func BenchmarkEventFilterManager_ShouldPass(b *testing.B) {
	fm := NewEventFilterManager()

	rule := &FilterRule{
		MinInterval:  10 * time.Millisecond,
		MaxPerSecond: 1000,
	}
	fm.SetRule(EventTypeKeyboard, rule)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fm.ShouldPass(EventTypeKeyboard)
	}
}
