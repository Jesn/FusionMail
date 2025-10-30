package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RateLimitMiddleware 速率限制中间件
type RateLimitMiddleware struct {
	redisClient *redis.Client
	defaultRate int           // 默认速率（每分钟请求数）
	window      time.Duration // 时间窗口
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
			// Redis 错误时不阻止请求，但记录日志
			c.Next()
			return
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
	if userID, exists := c.Get("user_id"); exists {
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
