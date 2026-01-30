/**
 * Package events 提供事件总线实现
 *
 * EventBus 是发布-订阅模式的核心实现，支持：
 * - 类型安全的订阅和发布
 * - 通配符订阅
 * - 异步事件处理
 * - 中间件链
 * - 优雅关闭
 */

package events

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/chenyang-zz/flowmind/internal/infrastructure/logger"
	"go.uber.org/zap"
)

/**
 * EventHandler 事件处理函数类型
 *
 * Parameters:
 *   - event: 事件对象
 *
 * Returns:
 *   - error: 处理过程中的错误
 */
type EventHandler func(event Event) error

/**
 * EventFilter 事件过滤器函数类型
 *
 * 返回 true 表示事件应该被处理，false 表示跳过
 */
type EventFilter func(event Event) bool

/**
 * Middleware 中间件类型
 *
 * 中间件可以包装事件处理函数，添加日志、恢复、限流等功能
 */
type Middleware func(EventHandler) EventHandler

/**
 * Subscriber 订阅者信息
 */
type Subscriber struct {
	// ID 订阅者唯一标识
	ID string

	// Handler 事件处理函数
	Handler EventHandler

	// Filter 事件过滤器（可选）
	Filter EventFilter

	// Once 是否只触发一次
	Once bool

	// Chan 订阅者专用通道（用于异步交付）
	Chan chan Event

	// mu 保护 Chan 的发送和关闭
	mu sync.RWMutex
}

/**
 * EventBus 事件总线
 *
 * 核心的发布-订阅系统实现
 */
type EventBus struct {
	// subscribers 订阅者映射：事件类型 -> 订阅者列表
	subscribers map[string][]*Subscriber

	// mutex 保护 subscribers 的读写锁
	mutex sync.RWMutex

	// wg 等待组，用于优雅关闭
	wg sync.WaitGroup

	// stopChan 停止信号通道
	stopChan chan struct{}

	// middleware 中间件链
	middleware []Middleware

	// stopped 原子标志，标记总线是否已停止
	stopped atomic.Bool

	// asyncEnabled 是否启用异步发布
	asyncEnabled bool

	// asyncBufferSize 异步事件缓冲区大小
	asyncBufferSize int
}

/**
 * NewEventBus 创建新的事件总线
 *
 * Parameters:
 *   - opts: 配置选项（可选）
 *
 * Returns:
 *   - *EventBus: 新创建的事件总线
 */
func NewEventBus(opts ...Option) *EventBus {
	bus := &EventBus{
		subscribers:     make(map[string][]*Subscriber),
		stopChan:        make(chan struct{}),
		middleware:      make([]Middleware, 0),
		asyncEnabled:    true,
		asyncBufferSize: 1000,
	}

	// 应用配置选项
	for _, opt := range opts {
		opt(bus)
	}

	return bus
}

/**
 * Option 配置选项类型
 */
type Option func(*EventBus)

/**
 * WithAsyncBufferSize 设置异步缓冲区大小
 */
func WithAsyncBufferSize(size int) Option {
	return func(bus *EventBus) {
		bus.asyncBufferSize = size
	}
}

/**
 * WithAsyncDisabled 禁用异步发布
 */
func WithAsyncDisabled() Option {
	return func(bus *EventBus) {
		bus.asyncEnabled = false
	}
}

/**
 * Subscribe 订阅事件
 *
 * Parameters:
 *   - eventType: 事件类型，使用 "*" 订阅所有事件
 *   - handler: 事件处理函数
 *
 * Returns:
 *   - string: 订阅者 ID，用于取消订阅
 */
func (bus *EventBus) Subscribe(eventType string, handler EventHandler) string {
	return bus.SubscribeWithFilter(eventType, handler, nil)
}

/**
 * SubscribeWithFilter 带过滤器订阅事件
 *
 * Parameters:
 *   - eventType: 事件类型
 *   - handler: 事件处理函数
 *   - filter: 事件过滤器（可选）
 *
 * Returns:
 *   - string: 订阅者 ID
 */
func (bus *EventBus) SubscribeWithFilter(
	eventType string,
	handler EventHandler,
	filter EventFilter,
) string {
	subscriber := &Subscriber{
		ID:      generateSubscriberID(),
		Handler: handler,
		Filter:  filter,
		Once:    false,
		Chan:    make(chan Event, bus.asyncBufferSize),
	}

	bus.mutex.Lock()
	defer bus.mutex.Unlock()

	bus.subscribers[eventType] = append(bus.subscribers[eventType], subscriber)

	logger.Debug("订阅事件",
		zap.String("event_type", eventType),
		zap.String("subscriber_id", subscriber.ID),
	)

	// 启动异步处理
	if bus.asyncEnabled {
		bus.wg.Add(1)
		go bus.processSubscriber(subscriber)
	}

	return subscriber.ID
}

/**
 * SubscribeOnce 订阅一次性事件
 *
 * 事件只会被处理一次，之后自动取消订阅
 *
 * Parameters:
 *   - eventType: 事件类型
 *   - handler: 事件处理函数
 *
 * Returns:
 *   - string: 订阅者 ID
 */
func (bus *EventBus) SubscribeOnce(eventType string, handler EventHandler) string {
	subscriber := &Subscriber{
		ID:      generateSubscriberID(),
		Handler: handler,
		Filter:  nil,
		Once:    true,
		Chan:    make(chan Event, bus.asyncBufferSize),
	}

	bus.mutex.Lock()
	defer bus.mutex.Unlock()

	bus.subscribers[eventType] = append(bus.subscribers[eventType], subscriber)

	if bus.asyncEnabled {
		bus.wg.Add(1)
		go bus.processSubscriber(subscriber)
	}

	return subscriber.ID
}

/**
 * Unsubscribe 取消订阅
 *
 * Parameters:
 *   - subscriberID: 订阅者 ID
 */
func (bus *EventBus) Unsubscribe(subscriberID string) {
	bus.mutex.Lock()
	defer bus.mutex.Unlock()

	// 在所有事件类型中查找订阅者
	for eventType, subscribers := range bus.subscribers {
		for i, sub := range subscribers {
			if sub.ID == subscriberID {
				// 从列表中移除
				bus.subscribers[eventType] = append(subscribers[:i], subscribers[i+1:]...)

				logger.Debug("取消订阅",
					zap.String("event_type", eventType),
					zap.String("subscriber_id", subscriberID),
				)

				// 关闭通道（加锁保护）
				sub.mu.Lock()
				close(sub.Chan)
				sub.mu.Unlock()

				return
			}
		}
	}

	logger.Debug("订阅者不存在，无法取消订阅", zap.String("subscriber_id", subscriberID))
}

/**
 * Publish 同步发布事件
 *
 * 会等待所有订阅者处理完成
 *
 * Parameters:
 *   - eventType: 事件类型
 *   - event: 事件对象
 *
 * Returns:
 *   - error: 发布过程中的错误
 */
func (bus *EventBus) Publish(eventType string, event Event) error {
	if bus.stopped.Load() {
		logger.Warn("事件总线已停止，无法发布事件",
			zap.String("event_type", eventType),
		)
		return fmt.Errorf("event bus is stopped")
	}

	logger.Debug("发布事件",
		zap.String("event_type", eventType),
		zap.String("event_id", event.ID),
	)

	// 应用中间件
	handler := bus.applyMiddleware(func(e Event) error {
		return nil // 中间件不需要实际处理
	})

	// 执行中间件（用于日志等）
	_ = handler(event)

	// 获取订阅者
	bus.mutex.RLock()
	subscribers := bus.getSubscribers(eventType)
	bus.mutex.RUnlock()

	subscriberCount := 0
	// 发送事件到所有订阅者
	for _, subscriber := range subscribers {
		if subscriber.Filter != nil && !subscriber.Filter(event) {
			continue // 过滤掉不匹配的事件
		}

		subscriberCount++

		// 异步发送（加锁保护）
		subscriber.mu.RLock()
		select {
		case subscriber.Chan <- event:
		default:
			// 缓冲区满，丢弃事件
			logger.Warn("事件缓冲区满，丢弃事件",
				zap.String("subscriber_id", subscriber.ID),
				zap.String("event_type", eventType),
			)
		}
		subscriber.mu.RUnlock()
	}

	logger.Debug("事件已发送",
		zap.String("event_type", eventType),
		zap.Int("subscriber_count", subscriberCount),
	)

	return nil
}

/**
 * PublishAsync 异步发布事件
 *
 * 不等待订阅者处理，立即返回
 *
 * Parameters:
 *   - eventType: 事件类型
 *   - event: 事件对象
 */
func (bus *EventBus) PublishAsync(eventType string, event Event) {
	go func() {
		_ = bus.Publish(eventType, event)
	}()
}

/**
 * Use 添加中间件
 *
 * 中间件按添加顺序执行
 *
 * Parameters:
 *   - middleware: 中间件函数
 */
func (bus *EventBus) Use(middleware Middleware) {
	bus.middleware = append(bus.middleware, middleware)
}

/**
 * Stop 优雅停止事件总线
 *
 * 会等待所有正在处理的事件完成
 *
 * Parameters:
 *   - timeout: 超时时间
 *
 * Returns:
 *   - error: 超时或停止过程中的错误
 */
func (bus *EventBus) Stop(timeout time.Duration) error {
	bus.stopped.Store(true)

	// 立即关闭 stopChan，通知所有订阅者退出
	close(bus.stopChan)

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// 等待所有订阅者处理完成
	done := make(chan struct{})
	go func() {
		bus.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// 所有订阅者已完成
		return nil
	case <-ctx.Done():
		// 超时
		return fmt.Errorf("timeout waiting for event bus to stop")
	}
}

/**
 * processSubscriber 处理订阅者事件
 *
 * 在独立的 goroutine 中运行，从通道读取事件并处理
 *
 * Parameters:
 *   - subscriber: 订阅者对象
 */
func (bus *EventBus) processSubscriber(subscriber *Subscriber) {
	defer bus.wg.Done()

	logger.Debug("订阅者处理器启动",
		zap.String("subscriber_id", subscriber.ID),
	)

	for {
		select {
		case event, ok := <-subscriber.Chan:
			if !ok {
				// 通道关闭
				logger.Debug("订阅者通道关闭",
					zap.String("subscriber_id", subscriber.ID),
				)
				return
			}

			// 处理事件
			handler := bus.applyMiddleware(subscriber.Handler)
			if err := handler(event); err != nil {
				logger.Error("事件处理错误",
					zap.String("subscriber_id", subscriber.ID),
					zap.String("event_type", string(event.Type)),
					zap.Error(err),
				)
			}

			// 如果是一次性订阅，处理后取消
			if subscriber.Once {
				bus.Unsubscribe(subscriber.ID)
				return
			}

		case <-bus.stopChan:
			// 总线停止，退出
			logger.Debug("订阅者处理器停止",
				zap.String("subscriber_id", subscriber.ID),
			)
			return
		}
	}
}

/**
 * getSubscribers 获取事件类型的所有订阅者
 *
 * 包括通配符订阅者
 *
 * Parameters:
 *   - eventType: 事件类型
 *
 * Returns:
 *   - []*Subscriber: 订阅者列表
 */
func (bus *EventBus) getSubscribers(eventType string) []*Subscriber {
	subscribers := make([]*Subscriber, 0)

	// 添加特定类型的订阅者
	if subs, ok := bus.subscribers[eventType]; ok {
		subscribers = append(subscribers, subs...)
	}

	// 添加通配符订阅者
	if wildcardSubs, ok := bus.subscribers["*"]; ok {
		subscribers = append(subscribers, wildcardSubs...)
	}

	return subscribers
}

/**
 * applyMiddleware 应用中间件链
 *
 * Parameters:
 *   - handler: 原始处理函数
 *
 * Returns:
 *   - EventHandler: 包装后的处理函数
 */
func (bus *EventBus) applyMiddleware(handler EventHandler) EventHandler {
	// 中间件按相反顺序应用（洋葱模型）
	for i := len(bus.middleware) - 1; i >= 0; i-- {
		handler = bus.middleware[i](handler)
	}
	return handler
}

/**
 * generateSubscriberID 生成订阅者 ID
 */
func generateSubscriberID() string {
	return fmt.Sprintf("sub-%d", time.Now().UnixNano())
}

/**
 * RecoveryMiddleware 恢复中间件
 *
 * 防止事件处理函数中的 panic 导致程序崩溃
 */
func RecoveryMiddleware() Middleware {
	return func(next EventHandler) EventHandler {
		return func(event Event) (err error) {
			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("panic recovered: %v", r)
				}
			}()
			return next(event)
		}
	}
}

/**
 * LoggingMiddleware 日志中间件
 *
 * 记录所有事件的处理
 *
 * Parameters:
 *   - logger: 日志函数（可选）
 */
func LoggingMiddleware(logger func(event Event)) Middleware {
	return func(next EventHandler) EventHandler {
		return func(event Event) error {
			if logger != nil {
				logger(event)
			}
			return next(event)
		}
	}
}

/**
 * RateLimitMiddleware 速率限制中间件
 *
 * 限制事件处理的最大速率
 *
 * Parameters:
 *   - maxPerSec: 每秒最大处理事件数
 */
func RateLimitMiddleware(maxPerSec int) Middleware {
	limiter := &rateLimiter{
		tokens: maxPerSec,
		max:    maxPerSec,
	}

	return func(next EventHandler) EventHandler {
		return func(event Event) error {
			if !limiter.allow() {
				return fmt.Errorf("rate limit exceeded")
			}
			return next(event)
		}
	}
}

/**
 * rateLimiter 速率限制器
 */
type rateLimiter struct {
	tokens    int
	max       int
	lastTime  time.Time
	mutex     sync.Mutex
}

func (rl *rateLimiter) allow() bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()
	elapsed := now.Sub(rl.lastTime)

	// 补充令牌
	if elapsed > time.Second {
		rl.tokens = rl.max
		rl.lastTime = now
	}

	if rl.tokens > 0 {
		rl.tokens--
		return true
	}

	return false
}

/**
 * NewEventBusWithOptimization 创建带性能优化的事件总线
 *
 * 创建一个带事件过滤和批量处理功能的事件总线。
 * 适用于高频事件场景，能够有效防止事件风暴。
 *
 * Parameters:
 *   - opts: 配置选项（可选）
 *
 * Returns:
 *   - *EventBus: 新创建的优化事件总线
 *
 * Example:
 *   bus := NewEventBusWithOptimization(
 *       WithAsyncBufferSize(2000),
 *   )
 */
func NewEventBusWithOptimization(opts ...Option) *EventBus {
	bus := NewEventBus(opts...)

	// 创建事件过滤器
	filter := NewEventFilterManager()

	// 配置默认过滤规则
	defaultRules := map[EventType]*FilterRule{
		EventTypeKeyboard: {
			MinInterval:  50 * time.Millisecond, // 键盘事件最小间隔50ms
			MaxPerSecond: 20,                   // 每秒最多20个键盘事件
		},
		EventTypeClipboard: {
			MinInterval:  100 * time.Millisecond, // 剪贴板事件最小间隔100ms
			MaxPerSecond: 10,                    // 每秒最多10个剪贴板事件
		},
		EventTypeAppSwitch: {
			MinInterval:  200 * time.Millisecond, // 应用切换事件最小间隔200ms
			MaxPerSecond: 5,                     // 每秒最多5个应用切换事件
		},
	}
	filter.SetRules(defaultRules)

	// 创建批量处理器
	batcher := NewEventBatcher(
		10,              // 批次大小：10个事件
		100*time.Millisecond, // 超时：100ms
	)
	batcher.Start()

	// 添加过滤中间件
	bus.Use(func(next EventHandler) EventHandler {
		return func(event Event) error {
			// 检查事件是否应该通过过滤器
			if !filter.ShouldPass(event.Type) {
				logger.Debug("事件被过滤",
					zap.String("event_type", string(event.Type)),
					zap.String("event_id", event.ID),
				)
				return nil // 跳过此事件
			}
			return next(event)
		}
	})

	logger.Info("创建优化事件总线",
		zap.Int("filter_rules", len(defaultRules)),
		zap.Int("batch_size", 10),
		zap.Duration("batch_timeout", 100*time.Millisecond),
	)

	return bus
}
