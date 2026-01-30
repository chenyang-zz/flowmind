package events

import (
	"context"
	"sync"
	"time"
)

// EventBatcher 事件批量处理器
//
// 将高频事件收集成批次，减少处理次数，提升系统性能。
// 支持按大小触发和按超时触发两种批量策略。
type EventBatcher struct {
	// batchSize 批次大小（触发批量处理的事件数量）
	batchSize int

	// timeout 超时时间（最大等待时间）
	timeout time.Duration

	// input 输入通道，接收待处理事件
	input chan Event

	// output 输出通道，发送批量事件
	output chan []Event

	// buffer 事件缓冲区
	buffer []Event

	// isRunning 运行状态标志
	isRunning bool

	// mu 互斥锁，保护并发访问
	mu sync.RWMutex

	// ctx 上下文，用于优雅停止
	ctx context.Context

	// cancel 取消函数
	cancel context.CancelFunc

	// wg 等待组，用于等待处理完成
	wg sync.WaitGroup
}

// NewEventBatcher 创建事件批量处理器
//
// 创建一个新的事件批量处理器实例。
//
// Parameters:
//   - batchSize: 批次大小（推荐 10-100）
//   - timeout: 超时时间（推荐 100ms-1s）
//
// Returns: *EventBatcher - 新创建的事件批量处理器实例
func NewEventBatcher(batchSize int, timeout time.Duration) *EventBatcher {
	ctx, cancel := context.WithCancel(context.Background())

	return &EventBatcher{
		batchSize: batchSize,
		timeout:   timeout,
		input:     make(chan Event, batchSize*2), // 缓冲区为批次大小的2倍
		output:    make(chan []Event, 10),        // 输出通道缓冲10个批次
		buffer:    make([]Event, 0, batchSize),
		ctx:       ctx,
		cancel:    cancel,
	}
}

// Start 启动批量处理器
//
// 启动后台处理协程，开始批量处理事件。
// 启动后可通过 Add() 方法添加事件，通过 Output() 方法获取批次。
//
// Returns: error - 启动失败时返回错误
func (b *EventBatcher) Start() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.isRunning {
		return nil
	}

	b.isRunning = true
	b.wg.Add(1)
	go b.processLoop()

	return nil
}

// Stop 停止批量处理器
//
// 优雅停止批量处理器，处理完所有缓冲区中的事件后关闭输出通道。
// 会先触发 Flush() 处理剩余事件。
func (b *EventBatcher) Stop() {
	b.mu.Lock()
	if !b.isRunning {
		b.mu.Unlock()
		return
	}
	b.isRunning = false // 标记为已停止
	b.mu.Unlock()

	// 取消上下文
	b.cancel()

	// 等待处理循环结束
	b.wg.Wait()

	// 最后一次刷新，处理剩余事件
	b.mu.Lock()
	b.flush()
	b.mu.Unlock()

	// 关闭输出通道
	close(b.output)
}

// Add 添加事件到批量处理器
//
// 将事件添加到缓冲区，如果缓冲区满了则触发批量处理。
// 非阻塞操作，如果通道满了会丢弃事件（防止阻塞）。
//
// Parameters:
//   - event: 待添加的事件
//
// Returns: bool - true 表示添加成功，false 表示事件被丢弃
func (b *EventBatcher) Add(event Event) bool {
	select {
	case b.input <- event:
		return true
	default:
		// 通道满了，丢弃事件
		return false
	}
}

// Output 获取输出通道
//
// 返回批量处理后的事件批次通道。
// 消费者应该从该通道接收批次并进行处理。
//
// Returns: <-chan []Event - 输出通道（只读）
func (b *EventBatcher) Output() <-chan []Event {
	return b.output
}

// Flush 手动触发批量处理
//
// 立即将缓冲区中的所有事件打包成一个批次发送到输出通道。
// 通常用于处理剩余事件或在停止前清理缓冲区。
func (b *EventBatcher) Flush() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.flush()
}

// flush 内部刷新方法（不加锁）
//
// 将缓冲区中的事件打包发送到输出通道。
func (b *EventBatcher) flush() {
	if len(b.buffer) == 0 {
		return
	}

	// 创建批次副本
	batch := make([]Event, len(b.buffer))
	copy(batch, b.buffer)

	// 清空缓冲区
	b.buffer = b.buffer[:0]

	// 发送到输出通道（非阻塞）
	select {
	case b.output <- batch:
		// 发送成功
	default:
		// 输出通道满了，丢弃批次
	}
}

// processLoop 处理循环
//
// 后台协程主循环，接收事件并批量处理。
func (b *EventBatcher) processLoop() {
	defer b.wg.Done()

	timer := time.NewTimer(b.timeout)
	defer timer.Stop()

	for {
		select {
		case <-b.ctx.Done():
			// 上下文取消，退出循环
			return

		case event, ok := <-b.input:
			if !ok {
				// 输入通道关闭，退出循环
				return
			}

			// 添加事件到缓冲区（加锁）
			b.mu.Lock()
			b.buffer = append(b.buffer, event)

			// 检查是否达到批次大小
			if len(b.buffer) >= b.batchSize {
				b.flush()
				b.mu.Unlock()
				// 重置定时器
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				timer.Reset(b.timeout)
			} else {
				b.mu.Unlock()
			}

		case <-timer.C:
			// 超时触发批量处理
			b.mu.Lock()
			b.flush()
			b.mu.Unlock()
			// 重置定时器
			timer.Reset(b.timeout)
		}
	}
}

// GetBufferSize 获取当前缓冲区大小
//
// 用于监控批量处理器的状态，调试性能问题。
//
// Returns: int - 当前缓冲区中的事件数量
func (b *EventBatcher) GetBufferSize() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.buffer)
}

// SetBatchSize 设置批次大小
//
// 动态调整批次大小，可以根据负载情况优化性能。
//
// Parameters: size - 新的批次大小
func (b *EventBatcher) SetBatchSize(size int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.batchSize = size
}

// SetTimeout 设置超时时间
//
// 动态调整超时时间，可以根据延迟要求优化性能。
//
// Parameters: duration - 新的超时时间
func (b *EventBatcher) SetTimeout(duration time.Duration) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.timeout = duration
}

// IsRunning 检查运行状态
//
// Returns: bool - true 表示正在运行，false 表示已停止
func (b *EventBatcher) IsRunning() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.isRunning
}
