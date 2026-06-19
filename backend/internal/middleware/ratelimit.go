package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// localCounter 本地降级计数器（Redis 不可用时使用）
type localCounter struct {
	count       int
	windowStart time.Time
}

// RateLimitMiddleware 速率限制中间件
type RateLimitMiddleware struct {
	redisClient *redis.Client
	defaultRate int           // 默认速率（每分钟请求数）
	window      time.Duration // 时间窗口
	localLimits sync.Map      // Redis 降级时的本地计数器 map[identifier]*localCounter
}

// NewRateLimitMiddleware 创建速率限制中间件
func NewRateLimitMiddleware(redisClient *redis.Client, defaultRate int) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		redisClient: redisClient,
		defaultRate: defaultRate,
		window:      time.Minute,
	}
}

// Limit 速率限制中间件
func (m *RateLimitMiddleware) Limit() gin.HandlerFunc {
	return m.LimitWithRate(m.defaultRate)
}

// LimitWithRate 使用指定速率的限制中间件
func (m *RateLimitMiddleware) LimitWithRate(rate int) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取客户端标识（优先使用 API Key，其次使用 IP）
		identifier := m.getIdentifier(c)

		// 检查速率限制
		allowed, remaining, resetTime, err := m.checkRateLimit(c.Request.Context(), identifier, rate)
		if err != nil {
			// Redis 错误时降级到本地计数器
			allowed, remaining, resetTime = m.checkLocalRateLimit(identifier, rate)
		}

		// 设置响应头
		c.Header("X-RateLimit-Limit", strconv.Itoa(rate))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(resetTime.Unix(), 10))

		if !allowed {
			c.Header("Retry-After", strconv.FormatInt(int64(time.Until(resetTime).Seconds()), 10))
			c.JSON(http.StatusTooManyRequests, gin.H{
				"success":     false,
				"error":       "请求过于频繁，请稍后再试",
				"retry_after": time.Until(resetTime).Seconds(),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// getIdentifier 获取客户端标识
func (m *RateLimitMiddleware) getIdentifier(c *gin.Context) string {
	// 优先使用 API Key ID
	if apiKeyID, exists := c.Get("api_key_id"); exists {
		return fmt.Sprintf("apikey:%v", apiKeyID)
	}

	// 优先使用用户 ID
	if userID, exists := c.Get("userID"); exists { // 修复：使用驼峰命名 "userID"
		return fmt.Sprintf("user:%v", userID)
	}

	// 使用 IP 地址
	return fmt.Sprintf("ip:%s", c.ClientIP())
}

// checkRateLimit 检查速率限制
func (m *RateLimitMiddleware) checkRateLimit(ctx context.Context, identifier string, rate int) (bool, int, time.Time, error) {
	now := time.Now()
	windowStart := now.Truncate(m.window)
	key := fmt.Sprintf("ratelimit:%s:%d", identifier, windowStart.Unix())

	// 使用 Redis 的 INCR 命令原子性地增加计数
	pipe := m.redisClient.Pipeline()
	incrCmd := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, m.window)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return true, rate, now.Add(m.window), err
	}

	count := int(incrCmd.Val())
	remaining := rate - count
	if remaining < 0 {
		remaining = 0
	}

	resetTime := windowStart.Add(m.window)
	allowed := count <= rate

	return allowed, remaining, resetTime, nil
}

// checkLocalRateLimit 本地降级速率限制（Redis 不可用时使用）
func (m *RateLimitMiddleware) checkLocalRateLimit(identifier string, rate int) (bool, int, time.Time) {
	now := time.Now()
	windowStart := now.Truncate(m.window)
	resetTime := windowStart.Add(m.window)

	val, _ := m.localLimits.LoadOrStore(identifier, &localCounter{count: 0, windowStart: windowStart})
	lc := val.(*localCounter)

	// 窗口已重置
	if now.Sub(lc.windowStart) >= m.window {
		lc.count = 0
		lc.windowStart = windowStart
	}

	lc.count++
	remaining := rate - lc.count
	if remaining < 0 {
		remaining = 0
	}
	allowed := lc.count <= rate

	return allowed, remaining, resetTime
}

// LimitByEndpoint 根据端点设置不同的速率限制
func (m *RateLimitMiddleware) LimitByEndpoint(endpointRates map[string]int) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取当前路径
		path := c.FullPath()

		// 查找对应的速率限制
		rate, exists := endpointRates[path]
		if !exists {
			rate = m.defaultRate
		}

		// 应用速率限制
		m.LimitWithRate(rate)(c)
	}
}

// LimitByAPIKey 根据 API Key 的速率限制配置
func (m *RateLimitMiddleware) LimitByAPIKey() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 如果有 API Key，使用其配置的速率限制
		if apiKeyID, exists := c.Get("api_key_id"); exists {
			// TODO: 从数据库或缓存中获取 API Key 的速率限制配置
			// 这里暂时使用默认值
			_ = apiKeyID
		}

		m.Limit()(c)
	}
}
