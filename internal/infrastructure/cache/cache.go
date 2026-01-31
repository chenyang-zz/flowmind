/**
 * Package cache 提供缓存抽象和实现
 *
 * 支持多种缓存策略，包括内存缓存和持久化缓存
 */

package cache

import (
	"sync"
	"time"
)

/**
 * Cache 缓存接口
 *
 * 定义缓存的基本操作，支持不同实现（内存、BBolt、Redis等）
 */
type Cache interface {
	// Get 获取缓存值
	// Parameters:
	//   - key: 缓存键
	// Returns: interface{} - 缓存值, bool - 是否找到
	Get(key string) (interface{}, bool)

	// Set 设置缓存值
	// Parameters:
	//   - key: 缓存键
	//   - value: 缓存值
	//   - ttl: 过期时间（0表示永不过期）
	// Returns: error - 错误信息
	Set(key string, value interface{}, ttl time.Duration) error

	// Delete 删除缓存
	// Parameters:
	//   - key: 缓存键
	// Returns: error - 错误信息
	Delete(key string) error

	// Clear 清空所有缓存
	// Returns: error - 错误信息
	Clear() error

	// Exists 检查键是否存在
	// Parameters:
	//   - key: 缓存键
	// Returns: bool - 是否存在
	Exists(key string) bool

	// Count 获取缓存项数量
	// Returns: int - 缓存项数量
	Count() int

	// Stop 停止缓存（清理资源）
	Stop()
}

/**
 * CacheStats 缓存统计信息
 */
type CacheStats struct {
	// Hits 缓存命中次数
	Hits int64

	// Misses 缓存未命中次数
	Misses int64

	// Sets 设置缓存次数
	Sets int64

	// Deletes 删除缓存次数
	Deletes int64

	// Evictions 淘汰缓存次数
	Evolutions int64

	mu sync.RWMutex
}

/**
 * RecordHit 记录缓存命中
 */
func (s *CacheStats) RecordHit() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Hits++
}

/**
 * RecordMiss 记录缓存未命中
 */
func (s *CacheStats) RecordMiss() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Misses++
}

/**
 * RecordSet 记录设置缓存
 */
func (s *CacheStats) RecordSet() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Sets++
}

/**
 * RecordDelete 记录删除缓存
 */
func (s *CacheStats) RecordDelete() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Deletes++
}

/**
 * RecordEviction 记录缓存淘汰
 */
func (s *CacheStats) RecordEviction() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Evolutions++
}

/**
 * GetStats 获取统计信息快照
 * Returns: Hits, Misses, Sets, Deletes, Evolutions
 */
func (s *CacheStats) GetStats() (int64, int64, int64, int64, int64) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Hits, s.Misses, s.Sets, s.Deletes, s.Evolutions
}

/**
 * HitRate 计算缓存命中率
 * Returns: float64 - 命中率（0-1之间）
 */
func (s *CacheStats) HitRate() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	total := s.Hits + s.Misses
	if total == 0 {
		return 0
	}
	return float64(s.Hits) / float64(total)
}

/**
 * Reset 重置统计信息
 */
func (s *CacheStats) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Hits = 0
	s.Misses = 0
	s.Sets = 0
	s.Deletes = 0
	s.Evolutions = 0
}
