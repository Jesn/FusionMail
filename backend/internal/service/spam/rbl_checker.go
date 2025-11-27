package spam

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/mrichman/godnsbl"
	"github.com/redis/go-redis/v9"
)

// RBLChecker RBL 黑名单检查器
type RBLChecker struct {
	cache           *redis.Client
	timeout         time.Duration
	cacheManager    *CacheManager
	fallbackManager *FallbackManager
}

// RBLResult RBL 检查结果
type RBLResult struct {
	IsListed  bool     // 是否在黑名单中
	Lists     []string // 命中的黑名单列表
	Score     int      // 评分增量
	CheckedAt time.Time
	FromCache bool // 是否来自缓存
}

// NewRBLChecker 创建 RBL 检查器实例
func NewRBLChecker(cache *redis.Client) *RBLChecker {
	return &RBLChecker{
		cache:   cache,
		timeout: 5 * time.Second, // 5 秒超时
	}
}

// NewRBLCheckerWithFallback 创建带降级策略的 RBL 检查器
func NewRBLCheckerWithFallback(cache *redis.Client, cacheManager *CacheManager, fallbackManager *FallbackManager) *RBLChecker {
	return &RBLChecker{
		cache:           cache,
		timeout:         5 * time.Second,
		cacheManager:    cacheManager,
		fallbackManager: fallbackManager,
	}
}

// CheckIP 检查 IP 地址是否在 RBL 黑名单中
func (r *RBLChecker) CheckIP(ctx context.Context, ip string) (*RBLResult, error) {
	// 1. 验证 IP 地址格式
	if net.ParseIP(ip) == nil {
		return &RBLResult{
			IsListed:  false,
			Lists:     []string{},
			Score:     0,
			CheckedAt: time.Now(),
			FromCache: false,
		}, nil
	}

	// 2. 检查服务是否可用（降级策略）
	if r.fallbackManager != nil && !r.fallbackManager.IsServiceAvailable("rbl") {
		// 服务不可用，尝试从缓存获取
		return r.getFromCacheOrDefault(ctx, ip, "ip")
	}

	// 3. 检查缓存（优先使用新的缓存管理器）
	if r.cacheManager != nil {
		if cached, ok := r.cacheManager.GetRBLResult(ctx, ip, "ip"); ok {
			return &RBLResult{
				IsListed:  cached.IsListed,
				Lists:     cached.Lists,
				Score:     cached.Score,
				CheckedAt: cached.CachedAt,
				FromCache: true,
			}, nil
		}
	} else {
		// 兼容旧的缓存方式
		cacheKey := fmt.Sprintf("rbl:ip:%s", ip)
		cached, err := r.getFromCache(ctx, cacheKey)
		if err == nil && cached != nil {
			cached.FromCache = true
			return cached, nil
		}
	}

	// 4. 执行 RBL 查询（带超时和降级）
	result, err := r.executeRBLQuery(ctx, ip)
	if err != nil {
		// 记录错误
		if r.fallbackManager != nil {
			r.fallbackManager.RecordError("rbl", err)
		}
		// 返回默认结果（降级）
		return &RBLResult{
			IsListed:  false,
			Lists:     []string{},
			Score:     0,
			CheckedAt: time.Now(),
			FromCache: false,
		}, nil
	}

	// 记录成功
	if r.fallbackManager != nil {
		r.fallbackManager.RecordSuccess("rbl")
	}

	// 5. 缓存结果
	if r.cacheManager != nil {
		r.cacheManager.SetRBLResult(ctx, ip, "ip", result.IsListed, result.Lists, result.Score)
	} else {
		cacheKey := fmt.Sprintf("rbl:ip:%s", ip)
		r.saveToCache(ctx, cacheKey, result, 30*time.Minute)
	}

	return result, nil
}

// executeRBLQuery 执行 RBL 查询
func (r *RBLChecker) executeRBLQuery(ctx context.Context, ip string) (*RBLResult, error) {
	result := &RBLResult{
		IsListed:  false,
		Lists:     []string{},
		Score:     0,
		CheckedAt: time.Now(),
		FromCache: false,
	}

	// 创建带超时的上下文
	queryCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	// 使用常见的 RBL 列表
	rblLists := []string{
		"zen.spamhaus.org",
		"bl.spamcop.net",
		"b.barracudacentral.org",
	}

	// 在 goroutine 中执行查询，支持超时
	done := make(chan bool)
	go func() {
		for _, rblList := range rblLists {
			rblResult := godnsbl.Lookup(rblList, ip)
			for _, res := range rblResult.Results {
				if res.Listed && !res.Error {
					result.IsListed = true
					result.Lists = append(result.Lists, rblList)
					break
				}
			}
		}
		done <- true
	}()

	// 等待查询完成或超时
	select {
	case <-done:
		// 查询完成
	case <-queryCtx.Done():
		// 超时
		return result, queryCtx.Err()
	}

	// 计算评分
	if result.IsListed {
		if len(result.Lists) >= 3 {
			result.Score = 50
		} else if len(result.Lists) >= 2 {
			result.Score = 40
		} else {
			result.Score = 30
		}
	}

	return result, nil
}

// getFromCacheOrDefault 从缓存获取或返回默认值（降级策略）
func (r *RBLChecker) getFromCacheOrDefault(ctx context.Context, target, targetType string) (*RBLResult, error) {
	// 尝试从缓存获取
	if r.cacheManager != nil {
		if cached, ok := r.cacheManager.GetRBLResult(ctx, target, targetType); ok {
			return &RBLResult{
				IsListed:  cached.IsListed,
				Lists:     cached.Lists,
				Score:     cached.Score,
				CheckedAt: cached.CachedAt,
				FromCache: true,
			}, nil
		}
	}

	// 返回默认结果（降级）
	return &RBLResult{
		IsListed:  false,
		Lists:     []string{},
		Score:     0,
		CheckedAt: time.Now(),
		FromCache: false,
	}, nil
}

// CheckDomain 检查域名是否在 RBL 黑名单中
func (r *RBLChecker) CheckDomain(ctx context.Context, domain string) (*RBLResult, error) {
	// 1. 验证域名格式
	domain = strings.ToLower(strings.TrimSpace(domain))
	if domain == "" {
		return &RBLResult{
			IsListed:  false,
			Lists:     []string{},
			Score:     0,
			CheckedAt: time.Now(),
			FromCache: false,
		}, nil
	}

	// 2. 检查缓存
	cacheKey := fmt.Sprintf("rbl:domain:%s", domain)
	cached, err := r.getFromCache(ctx, cacheKey)
	if err == nil && cached != nil {
		cached.FromCache = true
		return cached, nil
	}

	// 3. 解析域名获取 IP 地址
	result := &RBLResult{
		IsListed:  false,
		Lists:     []string{},
		Score:     0,
		CheckedAt: time.Now(),
		FromCache: false,
	}

	// 创建带超时的上下文
	queryCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	// 解析域名
	ips, err := net.DefaultResolver.LookupIP(queryCtx, "ip4", domain)
	if err != nil || len(ips) == 0 {
		// 解析失败，返回默认结果
		return result, nil
	}

	// 检查第一个 IP
	ipResult, err := r.CheckIP(ctx, ips[0].String())
	if err != nil {
		return result, nil
	}

	if ipResult.IsListed {
		result.IsListed = true
		result.Lists = ipResult.Lists
		// 域名在黑名单中，评分增加 40-60 分
		if len(ipResult.Lists) >= 3 {
			result.Score = 60
		} else if len(ipResult.Lists) >= 2 {
			result.Score = 50
		} else {
			result.Score = 40
		}
	}

	// 4. 缓存结果（30 分钟）
	r.saveToCache(ctx, cacheKey, result, 30*time.Minute)

	return result, nil
}

// getFromCache 从缓存获取结果
func (r *RBLChecker) getFromCache(ctx context.Context, key string) (*RBLResult, error) {
	// 检查缓存客户端是否可用
	if r.cache == nil {
		return nil, fmt.Errorf("cache not available")
	}

	// 获取缓存的 JSON 数据
	data, err := r.cache.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	// 简单解析：格式为 "listed:score:list1,list2,list3"
	parts := strings.Split(data, ":")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid cache format")
	}

	result := &RBLResult{
		IsListed:  parts[0] == "1",
		Lists:     []string{},
		Score:     0,
		CheckedAt: time.Now(),
		FromCache: true,
	}

	// 解析评分
	fmt.Sscanf(parts[1], "%d", &result.Score)

	// 解析列表
	if len(parts) >= 3 && parts[2] != "" {
		result.Lists = strings.Split(parts[2], ",")
	}

	return result, nil
}

// saveToCache 保存结果到缓存
func (r *RBLChecker) saveToCache(ctx context.Context, key string, result *RBLResult, ttl time.Duration) {
	// 检查缓存客户端是否可用
	if r.cache == nil {
		return
	}

	// 格式：listed:score:list1,list2,list3
	listed := "0"
	if result.IsListed {
		listed = "1"
	}

	lists := strings.Join(result.Lists, ",")
	data := fmt.Sprintf("%s:%d:%s", listed, result.Score, lists)

	r.cache.Set(ctx, key, data, ttl)
}

// ExtractIPFromEmail 从邮件头中提取发件人 IP
// 这是一个辅助方法，实际使用时需要解析邮件头
func ExtractIPFromEmail(headers map[string]string) string {
	// 尝试从 Received 头中提取 IP
	received := headers["Received"]
	if received == "" {
		return ""
	}

	// 简单的 IP 提取逻辑（实际应该更复杂）
	// 查找 [xxx.xxx.xxx.xxx] 格式的 IP
	start := strings.Index(received, "[")
	end := strings.Index(received, "]")
	if start != -1 && end != -1 && end > start {
		ip := received[start+1 : end]
		if net.ParseIP(ip) != nil {
			return ip
		}
	}

	return ""
}
