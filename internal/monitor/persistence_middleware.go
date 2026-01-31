/**
 * Package monitor 监控组件
 *
 * 事件持久化中间件，自动将监控事件持久化到存储
 */

package monitor

import (
	"time"

	"github.com/chenyang-zz/flowmind/internal/infrastructure/logger"
	"github.com/chenyang-zz/flowmind/internal/infrastructure/storage"
	"github.com/chenyang-zz/flowmind/pkg/events"
	"go.uber.org/zap"
)

/**
 * PersistenceConfig 持久化配置
 */
type PersistenceConfig struct {
	// EnabledEventTypes 需要持久化的事件类型
	EnabledEventTypes map[events.EventType]bool

	// AsyncMode 是否异步持久化
	AsyncMode bool

	// RetryOnError 错误重试
	RetryOnError bool

	// MaxRetries 最大重试次数
	MaxRetries int
}

/**
 * DefaultPersistenceConfig 默认持久化配置
 */
func DefaultPersistenceConfig() PersistenceConfig {
	return PersistenceConfig{
		EnabledEventTypes: map[events.EventType]bool{
			events.EventTypeKeyboard:   true,
			events.EventTypeClipboard:  true,
			events.EventTypeAppSwitch:  true,
			events.EventTypeAppSession: true,
		},
		AsyncMode:    true,
		RetryOnError: true,
		MaxRetries:   3,
	}
}

/**
 * PersistenceMiddleware 持久化中间件
 *
 * 拦截事件总线的事件并持久化到存储
 */
type PersistenceMiddleware struct {
	batchWriter *storage.BatchWriter
	config      PersistenceConfig
}

/**
 * NewPersistenceMiddleware 创建持久化中间件
 *
 * Parameters:
 *   - batchWriter: 批量写入器
 *   - config: 持久化配置
 *
 * Returns: events.Middleware - 事件中间件
 */
func NewPersistenceMiddleware(
	batchWriter *storage.BatchWriter,
	config PersistenceConfig,
) events.Middleware {
	pm := &PersistenceMiddleware{
		batchWriter: batchWriter,
		config:      config,
	}

	logger.Info("创建持久化中间件",
		zap.Int("enabled_types", len(config.EnabledEventTypes)),
		zap.Bool("async_mode", config.AsyncMode))

	return func(next events.EventHandler) events.EventHandler {
		return func(event events.Event) error {
			// 检查事件类型是否需要持久化
			if !pm.config.EnabledEventTypes[event.Type] {
				return next(event)
			}

			// 持久化事件
			success := pm.persistEvent(event)
			if !success && pm.config.RetryOnError {
				logger.Warn("事件持久化失败，准备重试",
					zap.String("event_id", event.ID),
					zap.String("event_type", string(event.Type)))
				go pm.retryPersist(event)
			}

			// 继续处理事件链
			return next(event)
		}
	}
}

/**
 * persistEvent 持久化单个事件
 *
 * Parameters:
 *   - event: 事件对象
 *
 * Returns: bool - 是否成功
 */
func (pm *PersistenceMiddleware) persistEvent(event events.Event) bool {
	if pm.batchWriter == nil {
		logger.Error("批量写入器未初始化")
		return false
	}

	// 写入批量写入器
	success := pm.batchWriter.Write(event)
	if !success {
		logger.Warn("事件写入失败",
			zap.String("event_id", event.ID),
			zap.String("event_type", string(event.Type)))
		return false
	}

	logger.Debug("事件已持久化",
		zap.String("event_id", event.ID),
		zap.String("event_type", string(event.Type)))

	return true
}

/**
 * retryPersist 重试持久化
 *
 * Parameters:
 *   - event: 事件对象
 */
func (pm *PersistenceMiddleware) retryPersist(event events.Event) {
	maxRetries := pm.config.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}

	for i := 0; i < maxRetries; i++ {
		// 指数退避
		backoff := time.Duration(1<<uint(i)) * time.Second
		time.Sleep(backoff)

		if pm.persistEvent(event) {
			logger.Info("事件重试持久化成功",
				zap.String("event_id", event.ID),
				zap.Int("attempt", i+1))
			return
		}
	}

	logger.Error("事件持久化重试失败",
		zap.String("event_id", event.ID),
		zap.Int("max_retries", maxRetries))
}

/**
 * Stop 停止中间件
 *
 * 确保所有缓冲区数据已持久化
 */
func (pm *PersistenceMiddleware) Stop() error {
	if pm.batchWriter != nil {
		logger.Info("正在停止持久化中间件...")
		pm.batchWriter.Stop()
		logger.Info("持久化中间件已停止")
	}
	return nil
}

/**
 * GetStats 获取持久化统计信息
 *
 * Returns: map[string]interface{} - 统计信息
 */
func (pm *PersistenceMiddleware) GetStats() map[string]interface{} {
	if pm.batchWriter == nil {
		return nil
	}

	stats := pm.batchWriter.GetStats()
	return map[string]interface{}{
		"total_events":     stats.TotalEvents,
		"persisted_events": stats.PersistedEvents,
		"failed_events":    stats.FailedEvents,
		"average_latency":  stats.AverageLatency.Milliseconds(),
	}
}
