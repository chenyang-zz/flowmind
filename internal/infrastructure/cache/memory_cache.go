/**
 * Package cache 缓存实现
 *
 * 提供基于内存的缓存实现，支持 TTL 和 LRU 淘汰策略
 */

package cache

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/chenyang-zz/flowmind/internal/infrastructure/logger"
	"go.uber.org/zap"
)

/**
 * cacheItem 缓存项
 */
type cacheItem struct {
	// value 缓存值
	value interface{}

	// expiration 过期时间（零值表示永不过期）
	expiration time.Time

	// createdAt 创建时间
	createdAt time.Time

	// accessedAt 最后访问时间
	accessedAt time.Time

	// accessCount 访问次数
	accessCount int64
}

/**
 * isExpired 检查缓存项是否过期
 * Returns: bool - true表示已过期
 */
func (item *cacheItem) isExpired() bool {
	if item.expiration.IsZero() {
		return false
	}
	return time.Now().After(item.expiration)
}

/**
 * MemoryCache 内存缓存实现
 *
 * 特性：
 * - 并发安全（使用 sync.Map）
 * - TTL 支持
 * - LRU 淘汰策略
 * - 定期清理过期项
 * - 缓存统计
 */
type MemoryCache struct {
	// items 缓存项映射（使用 sync.Map 实现并发安全）
	items *sync.Map

	// maxSize 最大缓存项数（0 表示无限制）
	maxSize int

	// cleanupInterval 清理间隔
	cleanupInterval time.Duration

	// stats 缓存统计
	stats *CacheStats

	// cleanupTicker 清理定时器
	cleanupTicker *time.Ticker

	// ctx 上下文
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// stopped 是否已停止
	stopped bool
	mu      sync.RWMutex
}

/**
 * NewMemoryCache 创建内存缓存
 *
 * Parameters:
 *   - maxSize: 最大缓存项数（0 表示无限制）
 *   - cleanupInterval: 清理间隔（0 表示不定期清理）
 *
 * Returns: *MemoryCache - 内存缓存实例
 */
func NewMemoryCache(maxSize int, cleanupInterval time.Duration) *MemoryCache {
	ctx, cancel := context.WithCancel(context.Background())

	cache := &MemoryCache{
		items:           &sync.Map{},
		maxSize:         maxSize,
		cleanupInterval: cleanupInterval,
		stats:           &CacheStats{},
		ctx:             ctx,
		cancel:          cancel,
		stopped:         false,
	}

	// 启动清理循环
	if cleanupInterval > 0 {
		cache.wg.Add(1)
		go cache.cleanupLoop()
		logger.Info("内存缓存已启动",
			zap.Int("max_size", maxSize),
			zap.Duration("cleanup_interval", cleanupInterval))
	}

	return cache
}

/**
 * Set 设置缓存值
 *
 * Parameters:
 *   - key: 缓存键
 *   - value: 缓存值
 *   - ttl: 过期时间（0表示永不过期）
 *
 * Returns: error - 错误信息
 */
func (c *MemoryCache) Set(key string, value interface{}, ttl time.Duration) error {
	c.mu.Lock()
	if c.stopped {
		c.mu.Unlock()
		return fmt.Errorf("缓存已停止")
	}
	c.mu.Unlock()

	var expiration time.Time
	if ttl > 0 {
		expiration = time.Now().Add(ttl)
	}

	item := &cacheItem{
		value:      value,
		expiration: expiration,
		createdAt:  time.Now(),
		accessedAt: time.Now(),
		accessCount: 0,
	}

	// 检查缓存大小限制
	if c.maxSize > 0 {
		count := c.Count()
		if count >= c.maxSize {
			// LRU 淘汰策略
			c.evictLRU()
		}
	}

	c.items.Store(key, item)
	c.stats.RecordSet()

	logger.Debug("缓存已设置",
		zap.String("key", key),
		zap.Duration("ttl", ttl))

	return nil
}

/**
 * Get 获取缓存值
 *
 * Parameters:
 *   - key: 缓存键
 *
 * Returns: interface{} - 缓存值, bool - 是否找到
 */
func (c *MemoryCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	if c.stopped {
		c.mu.RUnlock()
		return nil, false
	}
	c.mu.RUnlock()

	value, found := c.items.Load(key)
	if !found {
		c.stats.RecordMiss()
		return nil, false
	}

	item := value.(*cacheItem)

	// 检查是否过期
	if item.isExpired() {
		c.items.Delete(key)
		c.stats.RecordMiss()
		c.stats.RecordEviction()
		return nil, false
	}

	// 更新访问信息
	item.accessedAt = time.Now()
	item.accessCount++

	c.stats.RecordHit()
	return item.value, true
}

/**
 * Delete 删除缓存
 *
 * Parameters:
 *   - key: 缓存键
 *
 * Returns: error - 错误信息
 */
func (c *MemoryCache) Delete(key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.stopped {
		return fmt.Errorf("缓存已停止")
	}

	c.items.Delete(key)
	c.stats.RecordDelete()

	logger.Debug("缓存已删除", zap.String("key", key))
	return nil
}

/**
 * Clear 清空所有缓存
 *
 * Returns: error - 错误信息
 */
func (c *MemoryCache) Clear() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.stopped {
		return fmt.Errorf("缓存已停止")
	}

	c.items.Range(func(key, value interface{}) bool {
		c.items.Delete(key)
		return true
	})

	logger.Info("缓存已清空")
	return nil
}

/**
 * Exists 检查键是否存在
 *
 * Parameters:
 *   - key: 缓存键
 *
 * Returns: bool - 是否存在
 */
func (c *MemoryCache) Exists(key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.stopped {
		return false
	}

	value, found := c.items.Load(key)
	if !found {
		return false
	}

	item := value.(*cacheItem)
	if item.isExpired() {
		c.items.Delete(key)
		return false
	}

	return true
}

/**
 * Count 获取缓存项数量
 *
 * Returns: int - 缓存项数量
 */
func (c *MemoryCache) Count() int {
	count := 0
	c.items.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}

/**
 * GetStats 获取缓存统计信息
 *
 * Returns: *CacheStats - 统计信息
 */
func (c *MemoryCache) GetStats() *CacheStats {
	return c.stats
}

/**
 * evictLRU 淘汰最久未使用的缓存项
 */
func (c *MemoryCache) evictLRU() {
	var oldestKey interface{}
	var oldestTime time.Time
	found := false

	c.items.Range(func(key, value interface{}) bool {
		item := value.(*cacheItem)
		if !found || item.accessedAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = item.accessedAt
			found = true
		}
		return true
	})

	if found && oldestKey != nil {
		c.items.Delete(oldestKey)
		c.stats.RecordEviction()
		logger.Debug("LRU 淘汰缓存项", zap.String("key", oldestKey.(string)))
	}
}

/**
 * cleanupLoop 清理过期缓存
 */
func (c *MemoryCache) cleanupLoop() {
	defer c.wg.Done()

	ticker := time.NewTicker(c.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.cleanup()
		case <-c.ctx.Done():
			logger.Info("停止缓存清理循环")
			return
		}
	}
}

/**
 * cleanup 清理过期缓存
 */
func (c *MemoryCache) cleanup() {
	deleted := 0

	c.items.Range(func(key, value interface{}) bool {
		item := value.(*cacheItem)
		if item.isExpired() {
			c.items.Delete(key)
			deleted++
			c.stats.RecordEviction()
		}
		return true
	})

	if deleted > 0 {
		logger.Debug("清理过期缓存",
			zap.Int("count", deleted),
			zap.Int("remaining", c.Count()))
	}
}

/**
 * Stop 停止缓存
 */
func (c *MemoryCache) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.stopped {
		return
	}

	logger.Info("正在停止内存缓存...")

	// 取消上下文
	c.cancel()

	// 等待清理循环结束
	c.wg.Wait()

	// 标记为已停止
	c.stopped = true

	// 清理所有缓存
	c.items.Range(func(key, value interface{}) bool {
		c.items.Delete(key)
		return true
	})

	logger.Info("内存缓存已停止",
		zap.Int("final_count", c.Count()),
		zap.Float64("hit_rate", c.stats.HitRate()))
}
