/**
 * Package storage 提供数据持久化功能
 *
 * 负责将监控事件和分析结果持久化到数据库
 */

package storage

import (
	"context"
	"sync"
	"time"

	"github.com/chenyang-zz/flowmind/pkg/events"
	"github.com/chenyang-zz/flowmind/internal/infrastructure/logger"
	"go.uber.org/zap"
)

/**
 * BatchWriterConfig 批量写入器配置
 */
type BatchWriterConfig struct {
	// BatchSize 批量大小（达到此数量时自动刷新）
	BatchSize int

	// FlushInterval 刷新间隔（定时刷新）
	FlushInterval time.Duration

	// EventBuffer 缓冲区大小（channel 容量）
	EventBuffer int
}

/**
 * DefaultBatchWriterConfig 默认配置
 */
func DefaultBatchWriterConfig() BatchWriterConfig {
	return BatchWriterConfig{
		BatchSize:     100,  // 100 个事件一批
		FlushInterval: 5 * time.Second, // 5 秒刷新一次
		EventBuffer:   1000, // 缓冲 1000 个事件
	}
}

/**
 * BatchWriterStats 批量写入器统计信息
 */
type BatchWriterStats struct {
	// TotalEvents 总事件数
	TotalEvents int64

	// PersistedEvents 成功持久化的事件数
	PersistedEvents int64

	// FailedEvents 失败的事件数
	FailedEvents int64

	// AverageLatency 平均延迟
	AverageLatency time.Duration

	mu sync.Mutex
}

/**
 * BatchWriter 批量写入器
 *
 * 缓冲事件并批量写入数据库，提升持久化性能
 */
type BatchWriter struct {
	repo   EventRepository
	config BatchWriterConfig

	// 事件通道
	eventChan chan events.Event

	// 批量缓冲区
	buffer []events.Event

	// 统计信息
	stats *BatchWriterStats

	// 并发控制
	mu     sync.Mutex
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// 状态
	started bool
}

/**
 * NewBatchWriter 创建批量写入器
 *
 * Parameters:
 *   - repo: 事件仓储
 *   - config: 配置（使用 DefaultBatchWriterConfig() 获取默认配置）
 *
 * Returns: *BatchWriter - 批量写入器实例
 */
func NewBatchWriter(repo EventRepository, config BatchWriterConfig) *BatchWriter {
	ctx, cancel := context.WithCancel(context.Background())

	return &BatchWriter{
		repo:      repo,
		config:    config,
		eventChan: make(chan events.Event, config.EventBuffer),
		buffer:    make([]events.Event, 0, config.BatchSize),
		stats:     &BatchWriterStats{},
		ctx:       ctx,
		cancel:    cancel,
		started:   false,
	}
}

/**
 * Start 启动批量写入器
 *
 * 开始处理事件通道和定时刷新
 */
func (bw *BatchWriter) Start() {
	bw.mu.Lock()
	defer bw.mu.Unlock()

	if bw.started {
		logger.Warn("批量写入器已经启动", zap.Any("config", bw.config))
		return
	}

	bw.started = true

	// 启动事件处理循环
	bw.wg.Add(1)
	go bw.processEvents()

	// 启动定时刷新循环
	bw.wg.Add(1)
	go bw.flushLoop()

	logger.Info("批量写入器已启动",
		zap.Int("batch_size", bw.config.BatchSize),
		zap.Duration("flush_interval", bw.config.FlushInterval),
		zap.Int("event_buffer", bw.config.EventBuffer),
	)
}

/**
 * Stop 停止批量写入器
 *
 * 停止接收新事件，刷新缓冲区，等待所有写入完成
 */
func (bw *BatchWriter) Stop() {
	bw.mu.Lock()
	if !bw.started {
		bw.mu.Unlock()
		return
	}
	bw.started = false
	bw.mu.Unlock()

	logger.Info("正在停止批量写入器...")

	// 关闭事件通道
	close(bw.eventChan)

	// 取消上下文
	bw.cancel()

	// 刷新剩余事件
	bw.flush()

	// 等待所有 goroutine 完成
	bw.wg.Wait()

	logger.Info("批量写入器已停止")
}

/**
 * Write 写入单个事件
 *
 * 非阻塞方法，将事件放入通道
 *
 * Parameters:
 *   - event: 事件对象
 *
 * Returns: bool - 是否成功写入（通道满时返回 false）
 */
func (bw *BatchWriter) Write(event events.Event) bool {
	select {
	case bw.eventChan <- event:
		return true
	default:
		// 通道已满
		logger.Warn("批量写入器通道已满，事件丢弃",
			zap.String("event_id", event.ID),
			zap.String("event_type", string(event.Type)),
		)
		return false
	}
}

/**
 * WriteBatch 批量写入事件
 *
 * Parameters:
 *   - eventList: 事件列表
 *
 * Returns: int - 成功写入的事件数量
 */
func (bw *BatchWriter) WriteBatch(eventList []events.Event) int {
	successCount := 0
	for _, event := range eventList {
		if bw.Write(event) {
			successCount++
		}
	}
	return successCount
}

/**
 * ForceFlush 强制刷新缓冲区
 *
 * 立即将缓冲区中的所有事件写入数据库
 */
func (bw *BatchWriter) ForceFlush() {
	bw.mu.Lock()
	defer bw.mu.Unlock()
	bw.flush()
}

/**
 * processEvents 事件处理循环
 *
 * 从通道接收事件并放入缓冲区
 */
func (bw *BatchWriter) processEvents() {
	defer bw.wg.Done()

	for {
		select {
		case <-bw.ctx.Done():
			// 上下文取消，退出循环
			return

		case event, ok := <-bw.eventChan:
			if !ok {
				// 通道关闭
				return
			}

			bw.mu.Lock()
			bw.buffer = append(bw.buffer, event)

			// 达到批量大小，立即刷新
			if len(bw.buffer) >= bw.config.BatchSize {
				bw.flush()
			}
			bw.mu.Unlock()
		}
	}
}

/**
 * flushLoop 定时刷新循环
 *
 * 定期刷新缓冲区
 */
func (bw *BatchWriter) flushLoop() {
	defer bw.wg.Done()

	ticker := time.NewTicker(bw.config.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-bw.ctx.Done():
			return
		case <-ticker.C:
			bw.mu.Lock()
			bw.flush()
			bw.mu.Unlock()
		}
	}
}

/**
 * flush 刷新缓冲区到数据库
 *
 * 必须在持有锁的情况下调用
 */
func (bw *BatchWriter) flush() {
	if len(bw.buffer) == 0 {
		return
	}

	startTime := time.Now()
	eventCount := len(bw.buffer)

	// 批量写入
	err := bw.repo.SaveBatch(bw.buffer)
	if err != nil {
		logger.Error("批量写入失败",
			zap.Int("count", eventCount),
			zap.Error(err),
		)
		return
	}

	// 清空缓冲区
	bw.buffer = bw.buffer[:0]

	duration := time.Since(startTime)

	logger.Debug("批量刷新完成",
		zap.Int("count", eventCount),
		zap.Duration("duration", duration),
	)
}

/**
 * GetBufferSize 获取当前缓冲区大小
 *
 * Returns: int - 缓冲区中的事件数量
 */
func (bw *BatchWriter) GetBufferSize() int {
	bw.mu.Lock()
	defer bw.mu.Unlock()
	return len(bw.buffer)
}

/**
 * IsStarted 检查批量写入器是否已启动
 *
 * Returns: bool - 是否已启动
 */
func (bw *BatchWriter) IsStarted() bool {
	bw.mu.Lock()
	defer bw.mu.Unlock()
	return bw.started
}

/**
 * GetStats 获取统计信息
 *
 * Returns: *BatchWriterStats - 统计信息
 */
func (bw *BatchWriter) GetStats() *BatchWriterStats {
	return bw.stats
}
