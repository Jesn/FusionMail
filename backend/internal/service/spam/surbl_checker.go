package spam

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// SURBLChecker SURBL URL 黑名单检查器
type SURBLChecker struct {
	redisClient     *redis.Client
	timeout         time.Duration
	cacheTTL        time.Duration
	surblZones      []string // SURBL 查询域名列表
	cacheManager    *CacheManager
	fallbackManager *FallbackManager
}

// SURBLResult SURBL 检查结果
type SURBLResult struct {
	IsListed    bool     // 是否在黑名单中
	ListedURLs  []string // 被列入黑名单的 URL
	Score       int      // 评分增量
	CheckedURLs int      // 检查的 URL 数量
}

// NewSURBLChecker 创建 SURBL 检查器
func NewSURBLChecker(redisClient *redis.Client) *SURBLChecker {
	return &SURBLChecker{
		redisClient: redisClient,
		timeout:     3 * time.Second,  // 3 秒超时
		cacheTTL:    30 * time.Minute, // 缓存 30 分钟
		surblZones: []string{
			"multi.surbl.org",  // SURBL 多区域查询
			"multi.uribl.com",  // URIBL 多区域查询
			"dbl.spamhaus.org", // Spamhaus 域名黑名单
		},
	}
}

// NewSURBLCheckerWithFallback 创建带降级策略的 SURBL 检查器
func NewSURBLCheckerWithFallback(redisClient *redis.Client, cacheManager *CacheManager, fallbackManager *FallbackManager) *SURBLChecker {
	return &SURBLChecker{
		redisClient:     redisClient,
		timeout:         3 * time.Second,
		cacheTTL:        30 * time.Minute,
		cacheManager:    cacheManager,
		fallbackManager: fallbackManager,
		surblZones: []string{
			"multi.surbl.org",
			"multi.uribl.com",
			"dbl.spamhaus.org",
		},
	}
}

// Check 检查邮件中的 URL
func (s *SURBLChecker) Check(ctx context.Context, subject, body string) (*SURBLResult, error) {
	result := &SURBLResult{
		IsListed:   false,
		ListedURLs: make([]string, 0),
		Score:      0,
	}

	// 提取所有 URL
	urls := s.extractURLs(subject + " " + body)
	result.CheckedURLs = len(urls)

	if len(urls) == 0 {
		return result, nil
	}

	// 检查服务是否可用（降级策略）
	if s.fallbackManager != nil && !s.fallbackManager.IsServiceAvailable("surbl") {
		// 服务不可用，尝试从缓存获取
		return s.checkFromCacheOnly(ctx, urls)
	}

	// 检查每个 URL
	for _, urlStr := range urls {
		domain := s.extractDomain(urlStr)
		if domain == "" {
			continue
		}

		// 检查缓存（优先使用新的缓存管理器）
		if s.cacheManager != nil {
			if cached, ok := s.cacheManager.GetSURBLResult(ctx, domain); ok {
				if cached.IsListed {
					result.IsListed = true
					result.ListedURLs = append(result.ListedURLs, urlStr)
				}
				continue
			}
		} else {
			// 兼容旧的缓存方式
			cacheKey := fmt.Sprintf("surbl:check:%s", domain)
			cached, err := s.redisClient.Get(ctx, cacheKey).Result()
			if err == nil {
				if cached == "listed" {
					result.IsListed = true
					result.ListedURLs = append(result.ListedURLs, urlStr)
				}
				continue
			}
		}

		// 执行 SURBL 查询
		isListed := s.querySURBL(ctx, domain)

		// 记录服务调用结果
		if s.fallbackManager != nil {
			s.fallbackManager.RecordSuccess("surbl")
		}

		// 缓存结果
		if s.cacheManager != nil {
			s.cacheManager.SetSURBLResult(ctx, domain, isListed)
		} else {
			cacheKey := fmt.Sprintf("surbl:check:%s", domain)
			cacheValue := "clean"
			if isListed {
				cacheValue = "listed"
			}
			s.redisClient.Set(ctx, cacheKey, cacheValue, s.cacheTTL)
		}

		if isListed {
			result.IsListed = true
			result.ListedURLs = append(result.ListedURLs, urlStr)
		}
	}

	// 计算评分
	if result.IsListed {
		result.Score = len(result.ListedURLs) * 30
		if result.Score > 60 {
			result.Score = 60
		}
	}

	return result, nil
}

// checkFromCacheOnly 仅从缓存检查（降级策略）
func (s *SURBLChecker) checkFromCacheOnly(ctx context.Context, urls []string) (*SURBLResult, error) {
	result := &SURBLResult{
		IsListed:    false,
		ListedURLs:  make([]string, 0),
		Score:       0,
		CheckedURLs: len(urls),
	}

	for _, urlStr := range urls {
		domain := s.extractDomain(urlStr)
		if domain == "" {
			continue
		}

		// 仅从缓存获取
		if s.cacheManager != nil {
			if cached, ok := s.cacheManager.GetSURBLResult(ctx, domain); ok && cached.IsListed {
				result.IsListed = true
				result.ListedURLs = append(result.ListedURLs, urlStr)
			}
		}
	}

	// 计算评分
	if result.IsListed {
		result.Score = len(result.ListedURLs) * 30
		if result.Score > 60 {
			result.Score = 60
		}
	}

	return result, nil
}

// extractURLs 从文本中提取所有 URL
func (s *SURBLChecker) extractURLs(text string) []string {
	// URL 正则表达式
	urlRegex := regexp.MustCompile(`https?://[^\s<>"{}|\\^` + "`" + `\[\]]+`)
	matches := urlRegex.FindAllString(text, -1)

	// 去重
	urlMap := make(map[string]bool)
	urls := make([]string, 0)
	for _, match := range matches {
		if !urlMap[match] {
			urlMap[match] = true
			urls = append(urls, match)
		}
	}

	return urls
}

// extractDomain 从 URL 中提取域名
func (s *SURBLChecker) extractDomain(urlStr string) string {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return ""
	}

	host := parsedURL.Hostname()
	if host == "" {
		return ""
	}

	// 移除 www. 前缀
	host = strings.TrimPrefix(host, "www.")

	return host
}

// querySURBL 查询 SURBL 黑名单
func (s *SURBLChecker) querySURBL(ctx context.Context, domain string) bool {
	// 创建带超时的上下文
	queryCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	// 构造查询域名
	// SURBL 查询格式：<domain>.<surbl-zone>
	// 例如：example.com.multi.surbl.org
	queryDomain := fmt.Sprintf("%s.%s", domain, s.surblZones[0])

	// 执行 DNS 查询
	resolver := &net.Resolver{}
	addrs, err := resolver.LookupHost(queryCtx, queryDomain)

	// 如果查询超时或失败，返回 false（降级策略）
	if err != nil {
		// 检查是否是超时错误
		if queryCtx.Err() == context.DeadlineExceeded {
			// 超时，降级处理
			return false
		}
		// DNS 查询失败通常意味着域名不在黑名单中
		return false
	}

	// 如果返回了 IP 地址，说明域名在黑名单中
	// SURBL 返回的 IP 通常是 127.0.0.x 格式
	if len(addrs) > 0 {
		for _, addr := range addrs {
			if strings.HasPrefix(addr, "127.0.0.") {
				return true
			}
		}
	}

	return false
}

// CheckWithFallback 带降级策略的检查
func (s *SURBLChecker) CheckWithFallback(ctx context.Context, subject, body string) (*SURBLResult, error) {
	result, err := s.Check(ctx, subject, body)

	// 如果检查失败，返回空结果（降级策略）
	if err != nil {
		return &SURBLResult{
			IsListed:    false,
			ListedURLs:  make([]string, 0),
			Score:       0,
			CheckedURLs: 0,
		}, nil
	}

	return result, nil
}
