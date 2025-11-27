package spam

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// CacheManager 垃圾邮件检测缓存管理器
// 统一管理所有垃圾邮件检测相关的缓存
type CacheManager struct {
	redisClient *redis.Client
	localCache  *LocalCache
	config      *CacheConfig
	stats       *CacheStats
	mu          sync.RWMutex
}

// CacheConfig 缓存配置
type CacheConfig struct {
	// 发件人信誉缓存 TTL（默认 1 小时）
	ReputationTTL time.Duration
	// RBL 查询结果缓存 TTL（默认 30 分钟）
	RBLTTL time.Duration
	// 白名单/黑名单缓存 TTL（默认 1 小时）
	ListTTL time.Duration
	// 白名单/黑名单负缓存 TTL（默认 10 分钟）
	ListNegativeTTL time.Duration
	// 规则缓存 TTL（默认 10 分钟）
	RuleTTL time.Duration
	// SURBL 查询结果缓存 TTL（默认 30 分钟）
	SURBLTTL time.Duration
	// 是否启用本地缓存（二级缓存）
	EnableLocalCache bool
	// 本地缓存大小限制
	LocalCacheSize int
}

// CacheStats 缓存统计信息
type CacheStats struct {
	mu          sync.RWMutex
	Hits        int64     `json:"hits"`
	Misses      int64     `json:"misses"`
	Errors      int64     `json:"errors"`
	LocalHits   int64     `json:"local_hits"`
	RedisHits   int64     `json:"redis_hits"`
	LastResetAt time.Time `json:"last_reset_at"`
}

// LocalCache 本地内存缓存（二级缓存）
type LocalCache struct {
	mu      sync.RWMutex
	data    map[string]*localCacheEntry
	maxSize int
}

type localCacheEntry struct {
	value     interface{}
	expiresAt time.Time
}

// DefaultCacheConfig 返回默认缓存配置
func DefaultCacheConfig() *CacheConfig {
	return &CacheConfig{
		ReputationTTL:    1 * time.Hour,
		RBLTTL:           30 * time.Minute,
		ListTTL:          1 * time.Hour,
		ListNegativeTTL:  10 * time.Minute,
		RuleTTL:          10 * time.Minute,
		SURBLTTL:         30 * time.Minute,
		EnableLocalCache: true,
		LocalCacheSize:   1000,
	}
}

// NewCacheManager 创建缓存管理器
func NewCacheManager(redisClient *redis.Client, config *CacheConfig) *CacheManager {
	if config == nil {
		config = DefaultCacheConfig()
	}

	cm := &CacheManager{
		redisClient: redisClient,
		config:      config,
		stats: &CacheStats{
			LastResetAt: time.Now(),
		},
	}

	if config.EnableLocalCache {
		cm.localCache = &LocalCache{
			data:    make(map[string]*localCacheEntry),
			maxSize: config.LocalCacheSize,
		}
		// 启动本地缓存清理协程
		go cm.cleanupLocalCache()
	}

	return cm
}

// ==================== 发件人信誉缓存 ====================

// CachedReputation 缓存的信誉数据
type CachedReputation struct {
	Score      float64   `json:"score"`
	TrustLevel string    `json:"trust_level"`
	CachedAt   time.Time `json:"cached_at"`
}

// GetReputation 获取发件人信誉（带缓存）
func (cm *CacheManager) GetReputation(ctx context.Context, email string) (*CachedReputation, bool) {
	key := fmt.Sprintf("spam:reputation:%s", email)

	// 1. 检查本地缓存
	if cm.localCache != nil {
		if val, ok := cm.getFromLocalCache(key); ok {
			cm.recordHit(true)
			if rep, ok := val.(*CachedReputation); ok {
				return rep, true
			}
		}
	}

	// 2. 检查 Redis 缓存
	data, err := cm.redisClient.Get(ctx, key).Result()
	if err == nil {
		var rep CachedReputation
		if json.Unmarshal([]byte(data), &rep) == nil {
			cm.recordHit(false)
			// 写入本地缓存
			if cm.localCache != nil {
				cm.setToLocalCache(key, &rep, cm.config.ReputationTTL/2)
			}
			return &rep, true
		}
	}

	cm.recordMiss()
	return nil, false
}

// SetReputation 设置发件人信誉缓存
func (cm *CacheManager) SetReputation(ctx context.Context, email string, score float64, trustLevel string) error {
	key := fmt.Sprintf("spam:reputation:%s", email)
	rep := &CachedReputation{
		Score:      score,
		TrustLevel: trustLevel,
		CachedAt:   time.Now(),
	}

	data, err := json.Marshal(rep)
	if err != nil {
		return fmt.Errorf("failed to marshal reputation: %w", err)
	}

	// 写入 Redis
	if err := cm.redisClient.Set(ctx, key, data, cm.config.ReputationTTL).Err(); err != nil {
		cm.recordError()
		return fmt.Errorf("failed to cache reputation: %w", err)
	}

	// 写入本地缓存
	if cm.localCache != nil {
		cm.setToLocalCache(key, rep, cm.config.ReputationTTL/2)
	}

	return nil
}

// InvalidateReputation 使发件人信誉缓存失效
func (cm *CacheManager) InvalidateReputation(ctx context.Context, email string) error {
	key := fmt.Sprintf("spam:reputation:%s", email)

	// 删除本地缓存
	if cm.localCache != nil {
		cm.deleteFromLocalCache(key)
	}

	// 删除 Redis 缓存
	return cm.redisClient.Del(ctx, key).Err()
}

// ==================== RBL 查询结果缓存 ====================

// CachedRBLResult 缓存的 RBL 查询结果
type CachedRBLResult struct {
	IsListed bool      `json:"is_listed"`
	Lists    []string  `json:"lists"`
	Score    int       `json:"score"`
	CachedAt time.Time `json:"cached_at"`
}

// GetRBLResult 获取 RBL 查询结果（带缓存）
func (cm *CacheManager) GetRBLResult(ctx context.Context, target string, targetType string) (*CachedRBLResult, bool) {
	key := fmt.Sprintf("spam:rbl:%s:%s", targetType, target)

	// 1. 检查本地缓存
	if cm.localCache != nil {
		if val, ok := cm.getFromLocalCache(key); ok {
			cm.recordHit(true)
			if result, ok := val.(*CachedRBLResult); ok {
				return result, true
			}
		}
	}

	// 2. 检查 Redis 缓存
	data, err := cm.redisClient.Get(ctx, key).Result()
	if err == nil {
		var result CachedRBLResult
		if json.Unmarshal([]byte(data), &result) == nil {
			cm.recordHit(false)
			// 写入本地缓存
			if cm.localCache != nil {
				cm.setToLocalCache(key, &result, cm.config.RBLTTL/2)
			}
			return &result, true
		}
	}

	cm.recordMiss()
	return nil, false
}

// SetRBLResult 设置 RBL 查询结果缓存
func (cm *CacheManager) SetRBLResult(ctx context.Context, target string, targetType string, isListed bool, lists []string, score int) error {
	key := fmt.Sprintf("spam:rbl:%s:%s", targetType, target)
	result := &CachedRBLResult{
		IsListed: isListed,
		Lists:    lists,
		Score:    score,
		CachedAt: time.Now(),
	}

	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal RBL result: %w", err)
	}

	// 写入 Redis
	if err := cm.redisClient.Set(ctx, key, data, cm.config.RBLTTL).Err(); err != nil {
		cm.recordError()
		return fmt.Errorf("failed to cache RBL result: %w", err)
	}

	// 写入本地缓存
	if cm.localCache != nil {
		cm.setToLocalCache(key, result, cm.config.RBLTTL/2)
	}

	return nil
}

// ==================== 白名单/黑名单缓存 ====================

// GetListStatus 获取白名单/黑名单状态（带缓存）
func (cm *CacheManager) GetListStatus(ctx context.Context, userUID, target, listType string) (bool, bool) {
	key := fmt.Sprintf("spam:%s:%s:%s", listType, userUID, target)

	// 1. 检查本地缓存
	if cm.localCache != nil {
		if val, ok := cm.getFromLocalCache(key); ok {
			cm.recordHit(true)
			if status, ok := val.(bool); ok {
				return status, true
			}
		}
	}

	// 2. 检查 Redis 缓存
	data, err := cm.redisClient.Get(ctx, key).Result()
	if err == nil {
		status := data == "1"
		cm.recordHit(false)
		// 写入本地缓存
		if cm.localCache != nil {
			cm.setToLocalCache(key, status, cm.config.ListTTL/2)
		}
		return status, true
	}

	cm.recordMiss()
	return false, false
}

// SetListStatus 设置白名单/黑名单状态缓存
func (cm *CacheManager) SetListStatus(ctx context.Context, userUID, target, listType string, inList bool) error {
	key := fmt.Sprintf("spam:%s:%s:%s", listType, userUID, target)

	value := "0"
	ttl := cm.config.ListNegativeTTL
	if inList {
		value = "1"
		ttl = cm.config.ListTTL
	}

	// 写入 Redis
	if err := cm.redisClient.Set(ctx, key, value, ttl).Err(); err != nil {
		cm.recordError()
		return fmt.Errorf("failed to cache list status: %w", err)
	}

	// 写入本地缓存
	if cm.localCache != nil {
		cm.setToLocalCache(key, inList, ttl/2)
	}

	return nil
}

// InvalidateListCache 使白名单/黑名单缓存失效
func (cm *CacheManager) InvalidateListCache(ctx context.Context, userUID, target string) error {
	keys := []string{
		fmt.Sprintf("spam:whitelist:%s:%s", userUID, target),
		fmt.Sprintf("spam:blacklist:%s:%s", userUID, target),
	}

	// 删除本地缓存
	if cm.localCache != nil {
		for _, key := range keys {
			cm.deleteFromLocalCache(key)
		}
	}

	// 删除 Redis 缓存
	return cm.redisClient.Del(ctx, keys...).Err()
}

// InvalidateUserListCache 使用户的所有白名单/黑名单缓存失效
func (cm *CacheManager) InvalidateUserListCache(ctx context.Context, userUID string) error {
	patterns := []string{
		fmt.Sprintf("spam:whitelist:%s:*", userUID),
		fmt.Sprintf("spam:blacklist:%s:*", userUID),
	}

	for _, pattern := range patterns {
		iter := cm.redisClient.Scan(ctx, 0, pattern, 100).Iterator()
		for iter.Next(ctx) {
			key := iter.Val()
			// 删除本地缓存
			if cm.localCache != nil {
				cm.deleteFromLocalCache(key)
			}
			// 删除 Redis 缓存
			cm.redisClient.Del(ctx, key)
		}
		if err := iter.Err(); err != nil {
			return fmt.Errorf("failed to scan cache keys: %w", err)
		}
	}

	return nil
}

// ==================== 规则缓存 ====================

// CachedRules 缓存的规则列表
type CachedRules struct {
	Rules    []CachedRule `json:"rules"`
	CachedAt time.Time    `json:"cached_at"`
}

// CachedRule 缓存的单条规则
type CachedRule struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Category string `json:"category"`
	Pattern  string `json:"pattern"`
	Score    int    `json:"score"`
	Enabled  bool   `json:"enabled"`
}

// GetRules 获取规则列表（带缓存）
func (cm *CacheManager) GetRules(ctx context.Context) (*CachedRules, bool) {
	key := "spam:rules:enabled"

	// 1. 检查本地缓存
	if cm.localCache != nil {
		if val, ok := cm.getFromLocalCache(key); ok {
			cm.recordHit(true)
			if rules, ok := val.(*CachedRules); ok {
				return rules, true
			}
		}
	}

	// 2. 检查 Redis 缓存
	data, err := cm.redisClient.Get(ctx, key).Result()
	if err == nil {
		var rules CachedRules
		if json.Unmarshal([]byte(data), &rules) == nil {
			cm.recordHit(false)
			// 写入本地缓存
			if cm.localCache != nil {
				cm.setToLocalCache(key, &rules, cm.config.RuleTTL/2)
			}
			return &rules, true
		}
	}

	cm.recordMiss()
	return nil, false
}

// SetRules 设置规则列表缓存
func (cm *CacheManager) SetRules(ctx context.Context, rules *CachedRules) error {
	key := "spam:rules:enabled"

	data, err := json.Marshal(rules)
	if err != nil {
		return fmt.Errorf("failed to marshal rules: %w", err)
	}

	// 写入 Redis
	if err := cm.redisClient.Set(ctx, key, data, cm.config.RuleTTL).Err(); err != nil {
		cm.recordError()
		return fmt.Errorf("failed to cache rules: %w", err)
	}

	// 写入本地缓存
	if cm.localCache != nil {
		cm.setToLocalCache(key, rules, cm.config.RuleTTL/2)
	}

	return nil
}

// InvalidateRulesCache 使规则缓存失效
func (cm *CacheManager) InvalidateRulesCache(ctx context.Context) error {
	key := "spam:rules:enabled"

	// 删除本地缓存
	if cm.localCache != nil {
		cm.deleteFromLocalCache(key)
	}

	// 删除 Redis 缓存
	return cm.redisClient.Del(ctx, key).Err()
}

// ==================== SURBL 缓存 ====================

// CachedSURBLResult 缓存的 SURBL 查询结果
type CachedSURBLResult struct {
	IsListed bool      `json:"is_listed"`
	CachedAt time.Time `json:"cached_at"`
}

// GetSURBLResult 获取 SURBL 查询结果（带缓存）
func (cm *CacheManager) GetSURBLResult(ctx context.Context, domain string) (*CachedSURBLResult, bool) {
	key := fmt.Sprintf("spam:surbl:%s", domain)

	// 1. 检查本地缓存
	if cm.localCache != nil {
		if val, ok := cm.getFromLocalCache(key); ok {
			cm.recordHit(true)
			if result, ok := val.(*CachedSURBLResult); ok {
				return result, true
			}
		}
	}

	// 2. 检查 Redis 缓存
	data, err := cm.redisClient.Get(ctx, key).Result()
	if err == nil {
		var result CachedSURBLResult
		if json.Unmarshal([]byte(data), &result) == nil {
			cm.recordHit(false)
			// 写入本地缓存
			if cm.localCache != nil {
				cm.setToLocalCache(key, &result, cm.config.SURBLTTL/2)
			}
			return &result, true
		}
	}

	cm.recordMiss()
	return nil, false
}

// SetSURBLResult 设置 SURBL 查询结果缓存
func (cm *CacheManager) SetSURBLResult(ctx context.Context, domain string, isListed bool) error {
	key := fmt.Sprintf("spam:surbl:%s", domain)
	result := &CachedSURBLResult{
		IsListed: isListed,
		CachedAt: time.Now(),
	}

	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal SURBL result: %w", err)
	}

	// 写入 Redis
	if err := cm.redisClient.Set(ctx, key, data, cm.config.SURBLTTL).Err(); err != nil {
		cm.recordError()
		return fmt.Errorf("failed to cache SURBL result: %w", err)
	}

	// 写入本地缓存
	if cm.localCache != nil {
		cm.setToLocalCache(key, result, cm.config.SURBLTTL/2)
	}

	return nil
}

// ==================== 本地缓存操作 ====================

// getFromLocalCache 从本地缓存获取数据
func (cm *CacheManager) getFromLocalCache(key string) (interface{}, bool) {
	if cm.localCache == nil {
		return nil, false
	}

	cm.localCache.mu.RLock()
	defer cm.localCache.mu.RUnlock()

	entry, ok := cm.localCache.data[key]
	if !ok {
		return nil, false
	}

	// 检查是否过期
	if time.Now().After(entry.expiresAt) {
		return nil, false
	}

	return entry.value, true
}

// setToLocalCache 设置本地缓存
func (cm *CacheManager) setToLocalCache(key string, value interface{}, ttl time.Duration) {
	if cm.localCache == nil {
		return
	}

	cm.localCache.mu.Lock()
	defer cm.localCache.mu.Unlock()

	// 检查缓存大小限制
	if len(cm.localCache.data) >= cm.localCache.maxSize {
		// 简单的 LRU：删除最早的条目
		cm.evictOldestEntry()
	}

	cm.localCache.data[key] = &localCacheEntry{
		value:     value,
		expiresAt: time.Now().Add(ttl),
	}
}

// deleteFromLocalCache 从本地缓存删除数据
func (cm *CacheManager) deleteFromLocalCache(key string) {
	if cm.localCache == nil {
		return
	}

	cm.localCache.mu.Lock()
	defer cm.localCache.mu.Unlock()

	delete(cm.localCache.data, key)
}

// evictOldestEntry 删除最早的缓存条目
func (cm *CacheManager) evictOldestEntry() {
	var oldestKey string
	var oldestTime time.Time

	for key, entry := range cm.localCache.data {
		if oldestKey == "" || entry.expiresAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.expiresAt
		}
	}

	if oldestKey != "" {
		delete(cm.localCache.data, oldestKey)
	}
}

// cleanupLocalCache 定期清理过期的本地缓存
func (cm *CacheManager) cleanupLocalCache() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		if cm.localCache == nil {
			return
		}

		cm.localCache.mu.Lock()
		now := time.Now()
		for key, entry := range cm.localCache.data {
			if now.After(entry.expiresAt) {
				delete(cm.localCache.data, key)
			}
		}
		cm.localCache.mu.Unlock()
	}
}

// ==================== 统计信息 ====================

// recordHit 记录缓存命中
func (cm *CacheManager) recordHit(isLocal bool) {
	cm.stats.mu.Lock()
	defer cm.stats.mu.Unlock()

	cm.stats.Hits++
	if isLocal {
		cm.stats.LocalHits++
	} else {
		cm.stats.RedisHits++
	}
}

// recordMiss 记录缓存未命中
func (cm *CacheManager) recordMiss() {
	cm.stats.mu.Lock()
	defer cm.stats.mu.Unlock()

	cm.stats.Misses++
}

// recordError 记录缓存错误
func (cm *CacheManager) recordError() {
	cm.stats.mu.Lock()
	defer cm.stats.mu.Unlock()

	cm.stats.Errors++
}

// GetStats 获取缓存统计信息
func (cm *CacheManager) GetStats() *CacheStats {
	cm.stats.mu.RLock()
	defer cm.stats.mu.RUnlock()

	return &CacheStats{
		Hits:        cm.stats.Hits,
		Misses:      cm.stats.Misses,
		Errors:      cm.stats.Errors,
		LocalHits:   cm.stats.LocalHits,
		RedisHits:   cm.stats.RedisHits,
		LastResetAt: cm.stats.LastResetAt,
	}
}

// GetHitRate 获取缓存命中率
func (cm *CacheManager) GetHitRate() float64 {
	cm.stats.mu.RLock()
	defer cm.stats.mu.RUnlock()

	total := cm.stats.Hits + cm.stats.Misses
	if total == 0 {
		return 0
	}
	return float64(cm.stats.Hits) / float64(total) * 100
}

// ResetStats 重置统计信息
func (cm *CacheManager) ResetStats() {
	cm.stats.mu.Lock()
	defer cm.stats.mu.Unlock()

	cm.stats.Hits = 0
	cm.stats.Misses = 0
	cm.stats.Errors = 0
	cm.stats.LocalHits = 0
	cm.stats.RedisHits = 0
	cm.stats.LastResetAt = time.Now()
}

// ClearAllCache 清除所有垃圾邮件检测相关缓存
func (cm *CacheManager) ClearAllCache(ctx context.Context) error {
	// 清除本地缓存
	if cm.localCache != nil {
		cm.localCache.mu.Lock()
		cm.localCache.data = make(map[string]*localCacheEntry)
		cm.localCache.mu.Unlock()
	}

	// 清除 Redis 缓存
	patterns := []string{
		"spam:reputation:*",
		"spam:rbl:*",
		"spam:whitelist:*",
		"spam:blacklist:*",
		"spam:rules:*",
		"spam:surbl:*",
	}

	for _, pattern := range patterns {
		iter := cm.redisClient.Scan(ctx, 0, pattern, 100).Iterator()
		for iter.Next(ctx) {
			cm.redisClient.Del(ctx, iter.Val())
		}
		if err := iter.Err(); err != nil {
			return fmt.Errorf("failed to clear cache pattern %s: %w", pattern, err)
		}
	}

	return nil
}

// GetLocalCacheSize 获取本地缓存大小
func (cm *CacheManager) GetLocalCacheSize() int {
	if cm.localCache == nil {
		return 0
	}

	cm.localCache.mu.RLock()
	defer cm.localCache.mu.RUnlock()

	return len(cm.localCache.data)
}
