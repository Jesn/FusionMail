package config

import (
	"fmt"
	"os"
)

// Config 应用配置
type Config struct {
	Database DatabaseConfig
	Server   ServerConfig
	Redis    RedisConfig
	JWT      JWTConfig
	Security SecurityConfig
	Storage  StorageConfig
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
}

// StorageConfig 存储配置
type StorageConfig struct {
	Type      string // local, s3, oss
	LocalPath string // 本地存储路径
	BaseURL   string // 基础 URL
}

// Load 加载配置
func Load() *Config {
	return &Config{
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
			Port: getEnv("SERVER_PORT", "8080"),
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
		},
		Storage: StorageConfig{
			Type:      getEnv("STORAGE_TYPE", "local"),
			LocalPath: getEnv("STORAGE_LOCAL_PATH", "./data/attachments"),
			BaseURL:   getEnv("STORAGE_BASE_URL", ""),
		},
	}
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
