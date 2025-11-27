package spam

import (
	"context"
	"fmt"
	"fusionmail/internal/repository"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// WhitelistChecker 白名单/黑名单检查器
type WhitelistChecker struct {
	repo  repository.EmailListRepository
	cache *redis.Client
}

// NewWhitelistChecker 创建白名单/黑名单检查器实例
func NewWhitelistChecker(repo repository.EmailListRepository, cache *redis.Client) *WhitelistChecker {
	return &WhitelistChecker{
		repo:  repo,
		cache: cache,
	}
}

// CheckWhitelist 检查发件人是否在白名单中
// 支持邮箱地址和域名匹配
func (w *WhitelistChecker) CheckWhitelist(ctx context.Context, userUID string, sender string) (bool, error) {
	// 1. 先检查缓存
	cacheKey := fmt.Sprintf("whitelist:%s:%s", userUID, sender)
	cached, err := w.cache.Get(ctx, cacheKey).Result()
	if err == nil && cached == "1" {
		return true, nil
	}

	// 2. 检查邮箱地址是否在白名单中
	isInList, err := w.repo.IsInList(ctx, userUID, sender, "whitelist")
	if err != nil {
		return false, fmt.Errorf("failed to check whitelist: %w", err)
	}

	if isInList {
		// 缓存结果（1 小时）
		w.cache.Set(ctx, cacheKey, "1", time.Hour)
		return true, nil
	}

	// 3. 提取域名并检查域名是否在白名单中
	domain := extractDomain(sender)
	if domain != "" {
		isInList, err = w.repo.IsInList(ctx, userUID, domain, "whitelist")
		if err != nil {
			return false, fmt.Errorf("failed to check whitelist domain: %w", err)
		}

		if isInList {
			// 缓存结果（1 小时）
			w.cache.Set(ctx, cacheKey, "1", time.Hour)
			return true, nil
		}
	}

	// 4. 不在白名单中，缓存负结果（10 分钟）
	w.cache.Set(ctx, cacheKey, "0", 10*time.Minute)
	return false, nil
}

// CheckBlacklist 检查发件人是否在黑名单中
// 支持邮箱地址和域名匹配
func (w *WhitelistChecker) CheckBlacklist(ctx context.Context, userUID string, sender string) (bool, error) {
	// 1. 先检查缓存
	cacheKey := fmt.Sprintf("blacklist:%s:%s", userUID, sender)
	cached, err := w.cache.Get(ctx, cacheKey).Result()
	if err == nil && cached == "1" {
		return true, nil
	}

	// 2. 检查邮箱地址是否在黑名单中
	isInList, err := w.repo.IsInList(ctx, userUID, sender, "blacklist")
	if err != nil {
		return false, fmt.Errorf("failed to check blacklist: %w", err)
	}

	if isInList {
		// 缓存结果（1 小时）
		w.cache.Set(ctx, cacheKey, "1", time.Hour)
		return true, nil
	}

	// 3. 提取域名并检查域名是否在黑名单中
	domain := extractDomain(sender)
	if domain != "" {
		isInList, err = w.repo.IsInList(ctx, userUID, domain, "blacklist")
		if err != nil {
			return false, fmt.Errorf("failed to check blacklist domain: %w", err)
		}

		if isInList {
			// 缓存结果（1 小时）
			w.cache.Set(ctx, cacheKey, "1", time.Hour)
			return true, nil
		}
	}

	// 4. 不在黑名单中，缓存负结果（10 分钟）
	w.cache.Set(ctx, cacheKey, "0", 10*time.Minute)
	return false, nil
}

// InvalidateCache 使缓存失效
// 当白名单/黑名单更新时调用
func (w *WhitelistChecker) InvalidateCache(ctx context.Context, userUID string, target string) error {
	// 删除白名单缓存
	whitelistKey := fmt.Sprintf("whitelist:%s:%s", userUID, target)
	if err := w.cache.Del(ctx, whitelistKey).Err(); err != nil {
		return fmt.Errorf("failed to invalidate whitelist cache: %w", err)
	}

	// 删除黑名单缓存
	blacklistKey := fmt.Sprintf("blacklist:%s:%s", userUID, target)
	if err := w.cache.Del(ctx, blacklistKey).Err(); err != nil {
		return fmt.Errorf("failed to invalidate blacklist cache: %w", err)
	}

	return nil
}

// InvalidateUserCache 使用户的所有缓存失效
func (w *WhitelistChecker) InvalidateUserCache(ctx context.Context, userUID string) error {
	// 使用 SCAN 命令查找所有相关的缓存键
	whitelistPattern := fmt.Sprintf("whitelist:%s:*", userUID)
	blacklistPattern := fmt.Sprintf("blacklist:%s:*", userUID)

	// 删除白名单缓存
	iter := w.cache.Scan(ctx, 0, whitelistPattern, 100).Iterator()
	for iter.Next(ctx) {
		if err := w.cache.Del(ctx, iter.Val()).Err(); err != nil {
			return fmt.Errorf("failed to delete whitelist cache: %w", err)
		}
	}
	if err := iter.Err(); err != nil {
		return fmt.Errorf("failed to scan whitelist cache: %w", err)
	}

	// 删除黑名单缓存
	iter = w.cache.Scan(ctx, 0, blacklistPattern, 100).Iterator()
	for iter.Next(ctx) {
		if err := w.cache.Del(ctx, iter.Val()).Err(); err != nil {
			return fmt.Errorf("failed to delete blacklist cache: %w", err)
		}
	}
	if err := iter.Err(); err != nil {
		return fmt.Errorf("failed to scan blacklist cache: %w", err)
	}

	return nil
}

// extractDomain 从邮箱地址中提取域名
func extractDomain(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return ""
	}
	return parts[1]
}
