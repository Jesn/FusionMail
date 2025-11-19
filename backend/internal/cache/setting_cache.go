package cache

import (
	"context"
	"fmt"
	"sync"
	"time"

	redisWrapper "fusionmail/pkg/redis"

	"github.com/redis/go-redis/v9"
)

// CachedSetting 缓存的配置数据
type CachedSetting struct {
	Settings map[string]string `json:"settings"` // 配置项，key -> value（可能已加密）
	Updated  int64            `json:"updated"`   // 缓存更新时间戳
}

// SettingCache 配置缓存管理器
// 实现二级缓存：本地缓存 + Redis缓存
type SettingCache struct {
	client        *redis.Client      // Redis客户端
	wrapper       *redisWrapper.ClientWrapper // Redis包装器（提供JSON操作）
	localCache    *sync.Map         // 本地缓存（热点数据）
	redisTTL      time.Duration     // Redis缓存TTL
	localTTL      time.Duration     // 本地缓存TTL
	cleanupTicker *time.Ticker      // 清理定时器
}

// NewSettingCache 创建缓存实例
func NewSettingCache(client *redis.Client, redisTTL, localTTL time.Duration) *SettingCache {
	wrapper := redisWrapper.NewClientWrapper(client)

	cache := &SettingCache{
		client:     client,
		wrapper:    wrapper,
		localCache: &sync.Map{},
		redisTTL:   redisTTL,
		localTTL:   localTTL,
	}

	// 启动本地缓存清理任务
	cache.startCleanupTask()

	return cache
}

// GetCacheKey 生成缓存键
// 格式：setting:system:{category} 或 setting:user:{user_id}:{category}
func (c *SettingCache) GetCacheKey(userID *int64, category string) string {
	if userID == nil {
		return fmt.Sprintf("setting:system:%s", category)
	}
	return fmt.Sprintf("setting:user:%d:%s", *userID, category)
}

// Get 获取缓存的配置
func (c *SettingCache) Get(ctx context.Context, userID *int64, category string) (*CachedSetting, error) {
	key := c.GetCacheKey(userID, category)

	// 1. 尝试从本地缓存获取（热点数据）
	if value, found := c.localCache.Load(key); found {
		if cached, ok := value.(*CachedSetting); ok {
			// 检查是否过期
			if time.Now().Unix()-cached.Updated < int64(c.localTTL.Seconds()) {
				return cached, nil
			}
			// 已过期，删除
			c.localCache.Delete(key)
		}
	}

	// 2. 尝试从Redis缓存获取
	var cached CachedSetting
	err := c.wrapper.GetJSON(ctx, key, &cached)
	if err == nil {
		// Redis缓存命中，检查是否过期
		if time.Now().Unix()-cached.Updated < int64(c.redisTTL.Seconds()) {
			// 存入本地缓存
			c.localCache.Store(key, &cached)
			return &cached, nil
		}
		// 已过期，删除Redis缓存
		c.wrapper.Del(ctx, key)
	}

	// 3. 缓存未命中，返回错误
	return nil, fmt.Errorf("cache miss")
}

// Set 设置缓存
func (c *SettingCache) Set(ctx context.Context, userID *int64, category string, settings map[string]string) error {
	key := c.GetCacheKey(userID, category)

	cached := &CachedSetting{
		Settings: settings,
		Updated:  time.Now().Unix(),
	}

	// 1. 存储到Redis
	if err := c.wrapper.SetJSON(ctx, key, cached, c.redisTTL); err != nil {
		return fmt.Errorf("failed to set redis cache: %w", err)
	}

	// 2. 存储到本地缓存
	c.localCache.Store(key, cached)

	return nil
}

// Delete 删除缓存
func (c *SettingCache) Delete(ctx context.Context, userID *int64, category string) error {
	key := c.GetCacheKey(userID, category)

	// 1. 删除Redis缓存
	if err := c.wrapper.Del(ctx, key); err != nil && err != redis.Nil {
		return fmt.Errorf("failed to delete redis cache: %w", err)
	}

	// 2. 删除本地缓存
	c.localCache.Delete(key)

	return nil
}

// DeleteUser 删除用户所有缓存
func (c *SettingCache) DeleteUser(ctx context.Context, userID int64) error {
	// 删除Redis缓存（模式匹配）
	pattern := fmt.Sprintf("setting:user:%d:*", userID)
	iter := c.client.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		if err := c.wrapper.Del(ctx, iter.Val()); err != nil && err != redis.Nil {
			return fmt.Errorf("failed to delete redis cache %s: %w", iter.Val(), err)
		}
	}
	if err := iter.Err(); err != nil {
		return fmt.Errorf("failed to scan redis keys: %w", err)
	}

	// 删除本地缓存（模式匹配）
	c.localCache.Range(func(key, value interface{}) bool {
		keyStr := key.(string)
		if len(keyStr) > len(fmt.Sprintf("setting:user:%d:", userID)) {
			prefix := keyStr[:len(fmt.Sprintf("setting:user:%d:", userID))]
			expectedPrefix := fmt.Sprintf("setting:user:%d:", userID)
			if prefix == expectedPrefix {
				c.localCache.Delete(key)
			}
		}
		return true
	})

	return nil
}

// DeleteSystem 删除系统级缓存
func (c *SettingCache) DeleteSystem(ctx context.Context, category string) error {
	return c.Delete(ctx, nil, category)
}

// WarmUp 预热缓存
// 加载常用的配置分类到缓存中
func (c *SettingCache) WarmUp(ctx context.Context, categories []string) error {
	for _, category := range categories {
		// 预热系统级缓存
		if err := c.warmUpCategory(ctx, nil, category); err != nil {
			return fmt.Errorf("failed to warm up system category %s: %w", category, err)
		}

		// 这里可以添加用户级缓存预热
		// 注意：用户级缓存预热需要用户ID列表
	}

	return nil
}

// warmUpCategory 预热单个分类的缓存
func (c *SettingCache) warmUpCategory(ctx context.Context, userID *int64, category string) error {
	// 检查是否已有缓存
	_, err := c.Get(ctx, userID, category)
	if err == nil {
		// 缓存已存在，无需预热
		return nil
	}

	// 缓存不存在，标记为需要加载
	// 注意：这里只是预留，实际加载逻辑在Service层实现
	return nil
}

// Clear 清空所有缓存
func (c *SettingCache) Clear(ctx context.Context) error {
	// 清空本地缓存
	c.localCache.Range(func(key, value interface{}) bool {
		c.localCache.Delete(key)
		return true
	})

	// 清空Redis缓存（模式匹配）
	pattern := "setting:*"
	iter := c.client.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		if err := c.wrapper.Del(ctx, iter.Val()); err != nil && err != redis.Nil {
			return fmt.Errorf("failed to delete redis cache %s: %w", iter.Val(), err)
		}
	}
	if err := iter.Err(); err != nil {
		return fmt.Errorf("failed to scan redis keys: %w", err)
	}

	return nil
}

// startCleanupTask 启动本地缓存清理任务
// 定期清理过期的本地缓存数据
func (c *SettingCache) startCleanupTask() {
	c.cleanupTicker = time.NewTicker(5 * time.Minute) // 每5分钟清理一次

	go func() {
		defer c.cleanupTicker.Stop()

		for range c.cleanupTicker.C {
			now := time.Now().Unix()
			expired := int64(c.localTTL.Seconds())

			c.localCache.Range(func(key, value interface{}) bool {
				if cached, ok := value.(*CachedSetting); ok {
					if now-cached.Updated > expired {
						c.localCache.Delete(key)
					}
				}
				return true
			})
		}
	}()
}

// Stop 停止缓存管理器
func (c *SettingCache) Stop() {
	if c.cleanupTicker != nil {
		c.cleanupTicker.Stop()
	}
}

// GetStats 获取缓存统计信息
func (c *SettingCache) GetStats() map[string]interface{} {
	// 统计本地缓存数量
	localCount := 0
	c.localCache.Range(func(key, value interface{}) bool {
		localCount++
		return true
	})

	// 统计Redis缓存数量（模式匹配）
	pattern := "setting:*"
	iter := c.client.Scan(context.Background(), 0, pattern, 0).Iterator()
	redisCount := 0
	for iter.Next(context.Background()) {
		redisCount++
	}

	return map[string]interface{}{
		"local_cache_count": localCount,
		"redis_cache_count": redisCount,
		"local_ttl_seconds": c.localTTL.Seconds(),
		"redis_ttl_seconds": c.redisTTL.Seconds(),
	}
}

// BatchGet 批量获取多个分类的缓存
func (c *SettingCache) BatchGet(ctx context.Context, userID *int64, categories []string) (map[string]*CachedSetting, error) {
	results := make(map[string]*CachedSetting)

	for _, category := range categories {
		cached, err := c.Get(ctx, userID, category)
		if err != nil {
			// 缓存未命中，记录错误但不中断
			continue
		}
		results[category] = cached
	}

	return results, nil
}

// BatchSet 批量设置多个分类的缓存
func (c *SettingCache) BatchSet(ctx context.Context, userID *int64, settingsMap map[string]map[string]string) error {
	for category, settings := range settingsMap {
		if err := c.Set(ctx, userID, category, settings); err != nil {
			return fmt.Errorf("failed to set cache for category %s: %w", category, err)
		}
	}

	return nil
}
