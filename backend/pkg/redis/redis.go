package redis

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"fusionmail/config"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// Client 全局 Redis 客户端实例
var Client *redis.Client

// Initialize 初始化 Redis 连接
func Initialize(cfg *config.RedisConfig) error {
	addr := fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)
	log.Printf("Redis connecting to: %s (TLS: %v, Username: %s)", addr, cfg.TLS, cfg.Username)

	opts := &redis.Options{
		Addr:         addr,
		Username:     cfg.Username, // Aiven 等云服务需要用户名
		Password:     cfg.Password,
		DB:           cfg.DB,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
		MinIdleConns: 5,
	}

	// 启用 TLS（用于 Upstash/Aiven 等云服务）
	if cfg.TLS {
		opts.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
			ServerName: cfg.Host, // 设置服务器名称用于证书验证
		}
		log.Printf("Redis TLS enabled for host: %s", cfg.Host)
	}

	Client = redis.NewClient(opts)

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := Client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}

	log.Println("Redis connection established successfully")
	return nil
}

// Close 关闭 Redis 连接
func Close() error {
	if Client != nil {
		return Client.Close()
	}
	return nil
}

// GetClient 获取 Redis 客户端实例
func GetClient() *redis.Client {
	return Client
}

// ClientWrapper Redis 客户端包装器，提供 JSON 操作方法
type ClientWrapper struct {
	client *redis.Client
}

// NewClientWrapper 创建 Redis 客户端包装器
func NewClientWrapper(client *redis.Client) *ClientWrapper {
	return &ClientWrapper{client: client}
}

// SetJSON 设置 JSON 数据
func (c *ClientWrapper) SetJSON(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return c.client.Set(ctx, key, data, expiration).Err()
}

// GetJSON 获取 JSON 数据
func (c *ClientWrapper) GetJSON(ctx context.Context, key string, dest interface{}) error {
	data, err := c.client.Get(ctx, key).Result()
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(data), dest)
}

// Del 删除键
func (c *ClientWrapper) Del(ctx context.Context, keys ...string) error {
	return c.client.Del(ctx, keys...).Err()
}
