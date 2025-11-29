package config

import (
	"fmt"
	"log"
	"os"
	"strings"
)

// Config 应用配置
type Config struct {
	Database  DatabaseConfig
	Server    ServerConfig
	Redis     RedisConfig
	JWT       JWTConfig
	Security  SecurityConfig
	Storage   StorageConfig
	OAuth2    OAuth2Config // 新增 OAuth2 配置
	RateLimit RateLimitConfig
	Swagger   SwaggerConfig // Swagger 文档配置
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Host string
	Port string
}

// RedisConfig Redis 配置
type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

// JWTConfig JWT 配置
type JWTConfig struct {
	Secret string
	Expiry int // 过期时间（小时）
}

// SecurityConfig 安全配置
type SecurityConfig struct {
	EncryptionKey  string
	MasterPassword string // 主密码（用于初始登录）
	CookieSecure   *bool  // Cookie Secure 属性（nil=自动检测，true=强制启用，false=强制禁用）
}

// StorageConfig 存储配置
type StorageConfig struct {
	Type      string // local, s3, oss
	LocalPath string // 本地存储路径
	BaseURL   string // 基础 URL
}

// OAuth2Config OAuth2 配置（现在完全基于数据库）
type OAuth2Config struct {
	Google    GoogleOAuth2Config    // Google OAuth2 配置（仅用于回退，不再使用）
	Microsoft MicrosoftOAuth2Config // Microsoft OAuth2 配置（仅用于回退，不再使用）
}

// GoogleOAuth2Config Google OAuth2 配置
type GoogleOAuth2Config struct {
	ClientID     string // Google OAuth2 客户端 ID
	ClientSecret string // Google OAuth2 客户端密钥
	RedirectURL  string // 授权回调 URL
}

// MicrosoftOAuth2Config Microsoft OAuth2 配置
type MicrosoftOAuth2Config struct {
	ClientID     string // Microsoft OAuth2 客户端 ID
	ClientSecret string // Microsoft OAuth2 客户端密钥
	RedirectURL  string // 授权回调 URL
}

// RateLimitConfig 速率限制配置
type RateLimitConfig struct {
	Enabled       bool
	SiteDefault   int
	PublicDefault int
}

// SwaggerConfig Swagger 文档配置
type SwaggerConfig struct {
	Enabled bool // 是否启用 Swagger 文档
}

// Load 加载配置
func Load() *Config {
	// 调试：打印环境变量
	log.Printf("Environment DB_HOST: %s", os.Getenv("DB_HOST"))
	log.Printf("Environment DB_USER: %s", os.Getenv("DB_USER"))
	log.Printf("Environment DB_PASSWORD exists: %v", os.Getenv("DB_PASSWORD") != "")
	log.Printf("Environment DB_NAME: %s", os.Getenv("DB_NAME"))

	cfg := &Config{
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "fusionmail"),
			Password: getEnv("DB_PASSWORD", "fusionmail_password"),
			DBName:   getEnv("DB_NAME", "fusionmail"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		Server: ServerConfig{
			Host: getEnv("SERVER_HOST", "0.0.0.0"),
			Port: getEnv("SERVER_PORT", "3333"),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", "fusionmail_redis_password"),
			DB:       0,
		},
		JWT: JWTConfig{
			Secret: getEnv("JWT_SECRET", "dev-secret-key-for-testing-only"),
			Expiry: getEnvInt("JWT_EXPIRY_HOURS", 24),
		},
		Security: SecurityConfig{
			EncryptionKey:  getEnv("ENCRYPTION_KEY", "fusionmail-default-key-32-bytes"),
			MasterPassword: getEnv("MASTER_PASSWORD", "admin123"),
			CookieSecure:   getEnvBoolPtr("COOKIE_SECURE"), // nil=自动检测，true/false=强制设置
		},
		Storage: StorageConfig{
			Type:      getEnv("STORAGE_TYPE", "local"),
			LocalPath: getEnv("STORAGE_LOCAL_PATH", "./data/attachments"),
			BaseURL:   getEnv("STORAGE_BASE_URL", ""),
		},
		OAuth2: OAuth2Config{
			Google: GoogleOAuth2Config{
				ClientID:     "", // 不再使用环境变量
				ClientSecret: "",
				RedirectURL:  "http://localhost:3333/api/v1/auth/google/callback",
			},
			Microsoft: MicrosoftOAuth2Config{
				ClientID:     "", // 不再使用环境变量
				ClientSecret: "",
				RedirectURL:  "http://localhost:3333/api/v1/auth/microsoft/callback",
			},
		},
		RateLimit: RateLimitConfig{
			Enabled:       getEnvBool("RATE_LIMIT_ENABLED", true),
			SiteDefault:   getEnvInt("RATE_LIMIT_SITE_REQUESTS_PER_MINUTE", 100),
			PublicDefault: getEnvInt("RATE_LIMIT_PUBLIC_REQUESTS_PER_MINUTE", 200),
		},
		Swagger: SwaggerConfig{
			Enabled: getEnvBool("SWAGGER_ENABLED", false), // 默认关闭
		},
	}

	return cfg
}

// GetDSN 获取数据库连接字符串
func (c *DatabaseConfig) GetDSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode,
	)
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt 获取整数类型的环境变量
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var intValue int
		if _, err := fmt.Sscanf(value, "%d", &intValue); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvBool 获取布尔类型的环境变量
func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		switch strings.ToLower(value) {
		case "1", "true", "yes", "on":
			return true
		case "0", "false", "no", "off":
			return false
		}
	}
	return defaultValue
}

// getEnvBoolPtr 获取布尔指针类型的环境变量（用于三态：nil/true/false）
func getEnvBoolPtr(key string) *bool {
	if value := os.Getenv(key); value != "" {
		switch strings.ToLower(value) {
		case "1", "true", "yes", "on":
			result := true
			return &result
		case "0", "false", "no", "off":
			result := false
			return &result
		}
	}
	return nil // 未设置，返回 nil
}
