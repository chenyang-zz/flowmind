package cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/**
 * TestMemoryCache_SetGet 测试基本的 Set 和 Get 操作
 */
func TestMemoryCache_SetGet(t *testing.T) {
	cache := NewMemoryCache(100, 10*time.Minute)
	defer cache.Stop()

	// 设置缓存
	err := cache.Set("key1", "value1", 0)
	require.NoError(t, err)

	// 获取缓存
	value, found := cache.Get("key1")
	assert.True(t, found)
	assert.Equal(t, "value1", value)

	// 获取不存在的键
	value, found = cache.Get("key2")
	assert.False(t, found)
	assert.Nil(t, value)
}

/**
 * TestMemoryCache_Expiration 测试 TTL 过期功能
 */
func TestMemoryCache_Expiration(t *testing.T) {
	cache := NewMemoryCache(100, 10*time.Minute)
	defer cache.Stop()

	// 设置带 TTL 的缓存（100ms）
	err := cache.Set("key1", "value1", 100*time.Millisecond)
	require.NoError(t, err)

	// 立即获取，应该存在
	value, found := cache.Get("key1")
	assert.True(t, found)
	assert.Equal(t, "value1", value)

	// 等待过期
	time.Sleep(150 * time.Millisecond)

	// 再次获取，应该不存在
	value, found = cache.Get("key1")
	assert.False(t, found)
	assert.Nil(t, value)
}

/**
 * TestMemoryCache_LRUEviction 测试 LRU 淘汰策略
 */
func TestMemoryCache_LRUEviction(t *testing.T) {
	// 创建最大容量为 3 的缓存
	cache := NewMemoryCache(3, 10*time.Minute)
	defer cache.Stop()

	// 添加 3 个缓存项
	err := cache.Set("key1", "value1", 0)
	require.NoError(t, err)
	err = cache.Set("key2", "value2", 0)
	require.NoError(t, err)
	err = cache.Set("key3", "value3", 0)
	require.NoError(t, err)

	// 访问 key1，使其成为最近使用
	cache.Get("key1")

	// 添加第 4 个缓存项，应该淘汰最久未使用的 key2
	err = cache.Set("key4", "value4", 0)
	require.NoError(t, err)

	// key2 应该被淘汰
	_, found := cache.Get("key2")
	assert.False(t, found)

	// key1, key3, key4 应该还存在
	_, found = cache.Get("key1")
	assert.True(t, found)
	_, found = cache.Get("key3")
	assert.True(t, found)
	_, found = cache.Get("key4")
	assert.True(t, found)

	// 验证缓存数量
	assert.Equal(t, 3, cache.Count())
}

/**
 * TestMemoryCache_Cleanup 测试定期清理过期项
 */
func TestMemoryCache_Cleanup(t *testing.T) {
	// 创建清理间隔为 50ms 的缓存
	cache := NewMemoryCache(100, 50*time.Millisecond)
	defer cache.Stop()

	// 添加多个带 TTL 的缓存项
	for i := 0; i < 5; i++ {
		key := "key" + string(rune('0'+i))
		err := cache.Set(key, i, 100*time.Millisecond)
		require.NoError(t, err)
	}

	// 添加一个永不过期的缓存项
	err := cache.Set("permanent", "value", 0)
	require.NoError(t, err)

	// 等待清理周期 + 过期时间
	time.Sleep(200 * time.Millisecond)

	// 临时缓存项应该被清理
	assert.Equal(t, 1, cache.Count())

	// 永久缓存项应该还在
	value, found := cache.Get("permanent")
	assert.True(t, found)
	assert.Equal(t, "value", value)
}

/**
 * TestMemoryCache_Delete 测试删除缓存
 */
func TestMemoryCache_Delete(t *testing.T) {
	cache := NewMemoryCache(100, 10*time.Minute)
	defer cache.Stop()

	// 添加缓存项
	err := cache.Set("key1", "value1", 0)
	require.NoError(t, err)

	// 删除缓存项
	err = cache.Delete("key1")
	require.NoError(t, err)

	// 验证已删除
	_, found := cache.Get("key1")
	assert.False(t, found)

	// 删除不存在的键不应报错
	err = cache.Delete("key2")
	assert.NoError(t, err)
}

/**
 * TestMemoryCache_Clear 测试清空缓存
 */
func TestMemoryCache_Clear(t *testing.T) {
	cache := NewMemoryCache(100, 10*time.Minute)
	defer cache.Stop()

	// 添加多个缓存项
	for i := 0; i < 10; i++ {
		key := "key" + string(rune('0'+i))
		err := cache.Set(key, i, 0)
		require.NoError(t, err)
	}

	assert.Equal(t, 10, cache.Count())

	// 清空缓存
	err := cache.Clear()
	require.NoError(t, err)

	// 验证已清空
	assert.Equal(t, 0, cache.Count())
}

/**
 * TestMemoryCache_Exists 测试检查键是否存在
 */
func TestMemoryCache_Exists(t *testing.T) {
	cache := NewMemoryCache(100, 10*time.Minute)
	defer cache.Stop()

	// 键不存在
	assert.False(t, cache.Exists("key1"))

	// 添加缓存项
	err := cache.Set("key1", "value1", 0)
	require.NoError(t, err)

	// 键应该存在
	assert.True(t, cache.Exists("key1"))

	// 等待过期
	err = cache.Set("key2", "value2", 50*time.Millisecond)
	require.NoError(t, err)
	time.Sleep(100 * time.Millisecond)

	// 过期的键应该不存在
	assert.False(t, cache.Exists("key2"))
}

/**
 * TestMemoryCache_Count 测试统计缓存项数量
 */
func TestMemoryCache_Count(t *testing.T) {
	cache := NewMemoryCache(100, 10*time.Minute)
	defer cache.Stop()

	// 初始数量为 0
	assert.Equal(t, 0, cache.Count())

	// 添加缓存项
	for i := 0; i < 5; i++ {
		key := "key" + string(rune('0'+i))
		err := cache.Set(key, i, 0)
		require.NoError(t, err)
	}

	assert.Equal(t, 5, cache.Count())

	// 删除部分缓存项
	cache.Delete("key1")
	cache.Delete("key2")

	assert.Equal(t, 3, cache.Count())
}

/**
 * TestMemoryCache_Stats 测试缓存统计
 */
func TestMemoryCache_Stats(t *testing.T) {
	cache := NewMemoryCache(100, 10*time.Minute)
	defer cache.Stop()

	// 设置缓存项
	err := cache.Set("key1", "value1", 0)
	require.NoError(t, err)

	stats := cache.GetStats()
	hits, misses, sets, _, _ := stats.GetStats()

	// 验证设置次数
	assert.Equal(t, int64(1), sets)

	// 缓存命中
	cache.Get("key1")
	hits, misses, sets, _, _ = stats.GetStats()
	assert.Equal(t, int64(1), hits)

	// 缓存未命中
	cache.Get("key2")
	hits, misses, sets, _, _ = stats.GetStats()
	assert.Equal(t, int64(1), misses)

	// 验证命中率
	hitRate := stats.HitRate()
	assert.Equal(t, 0.5, hitRate)
}

/**
 * TestMemoryCache_ConcurrentAccess 测试并发访问
 */
func TestMemoryCache_ConcurrentAccess(t *testing.T) {
	cache := NewMemoryCache(1000, 10*time.Minute)
	defer cache.Stop()

	// 并发写入
	done := make(chan bool)
	for i := 0; i < 100; i++ {
		go func(idx int) {
			key := "key" + string(rune('0'+idx%10))
			cache.Set(key, idx, 0)
			done <- true
		}(i)
	}

	// 等待所有写入完成
	for i := 0; i < 100; i++ {
		<-done
	}

	// 并发读取
	for i := 0; i < 100; i++ {
		go func(idx int) {
			key := "key" + string(rune('0'+idx%10))
			cache.Get(key)
			done <- true
		}(i)
	}

	// 等待所有读取完成
	for i := 0; i < 100; i++ {
		<-done
	}

	// 验证缓存仍然正常工作
	stats := cache.GetStats()
	hits, _, _, _, _ := stats.GetStats()
	assert.Greater(t, hits, int64(0))
}

/**
 * TestMemoryCache_Stop 测试停止缓存
 */
func TestMemoryCache_Stop(t *testing.T) {
	cache := NewMemoryCache(100, 50*time.Millisecond)

	// 添加缓存项
	err := cache.Set("key1", "value1", 0)
	require.NoError(t, err)

	// 停止缓存
	cache.Stop()

	// 停止后不能设置缓存
	err = cache.Set("key2", "value2", 0)
	assert.Error(t, err)

	// 停止后不能读取缓存
	_, found := cache.Get("key1")
	assert.False(t, found)

	// 重复停止应该安全
	cache.Stop()
}
