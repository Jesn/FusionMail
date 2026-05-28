package main

import (
	"testing"

	"fusionmail/config"
	"fusionmail/internal/model"
	cryptoutil "fusionmail/pkg/crypto"
	"fusionmail/pkg/database"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCurrentGinMode(t *testing.T) {
	t.Run("未设置时默认为 release", func(t *testing.T) {
		t.Setenv("GIN_MODE", "")

		if currentGinMode() != gin.ReleaseMode {
			t.Fatalf("expected default gin mode %q, got %q", gin.ReleaseMode, currentGinMode())
		}
	})

	t.Run("大小写与空白会被规范化", func(t *testing.T) {
		t.Setenv("GIN_MODE", " Release ")

		if currentGinMode() != gin.ReleaseMode {
			t.Fatalf("expected normalized gin mode %q, got %q", gin.ReleaseMode, currentGinMode())
		}
	})

	t.Run("非法值按 release 处理", func(t *testing.T) {
		t.Setenv("GIN_MODE", "prod")

		if currentGinMode() != gin.ReleaseMode {
			t.Fatalf("expected invalid gin mode to fall back to %q, got %q", gin.ReleaseMode, currentGinMode())
		}
	})
}

func TestValidateProductionSecrets(t *testing.T) {
	t.Run("release 模式拒绝默认加密密钥", func(t *testing.T) {
		t.Setenv("GIN_MODE", "release")
		cfg := &config.Config{
			JWT:      config.JWTConfig{Secret: "12345678901234567890123456789012"},
			Security: config.SecurityConfig{EncryptionKey: cryptoutil.DefaultEncryptionKey},
		}

		if err := validateProductionSecrets(cfg); err == nil {
			t.Fatal("expected default encryption key to be rejected in release mode")
		}
	})

	t.Run("release 模式拒绝短加密密钥", func(t *testing.T) {
		t.Setenv("GIN_MODE", "release")
		cfg := &config.Config{
			JWT:      config.JWTConfig{Secret: "12345678901234567890123456789012"},
			Security: config.SecurityConfig{EncryptionKey: "short"},
		}

		if err := validateProductionSecrets(cfg); err == nil {
			t.Fatal("expected short encryption key to be rejected in release mode")
		}
	})

	t.Run("release 模式拒绝默认 JWT 密钥", func(t *testing.T) {
		t.Setenv("GIN_MODE", "release")
		cfg := &config.Config{
			JWT:      config.JWTConfig{Secret: config.DefaultJWTSecret},
			Security: config.SecurityConfig{EncryptionKey: "12345678901234567890123456789012"},
		}

		if err := validateProductionSecrets(cfg); err == nil {
			t.Fatal("expected default JWT secret to be rejected in release mode")
		}
	})

	t.Run("release 模式接受显式强密钥", func(t *testing.T) {
		t.Setenv("GIN_MODE", "release")
		cfg := &config.Config{
			JWT:      config.JWTConfig{Secret: "abcdefghijklmnopqrstuvwxyz123456"},
			Security: config.SecurityConfig{EncryptionKey: "12345678901234567890123456789012"},
		}

		if err := validateProductionSecrets(cfg); err != nil {
			t.Fatalf("expected explicit secrets to be accepted: %v", err)
		}
	})

	t.Run("debug 模式允许默认密钥", func(t *testing.T) {
		t.Setenv("GIN_MODE", "debug")
		cfg := &config.Config{
			JWT:      config.JWTConfig{Secret: config.DefaultJWTSecret},
			Security: config.SecurityConfig{EncryptionKey: cryptoutil.DefaultEncryptionKey},
		}

		if err := validateProductionSecrets(cfg); err != nil {
			t.Fatalf("expected default secrets to be allowed in debug mode: %v", err)
		}
	})
}

func TestEnsureUserSessionVersionColumnReady(t *testing.T) {
	originalDB := database.DB
	t.Cleanup(func() {
		database.DB = originalDB
	})

	t.Run("存在字段时通过", func(t *testing.T) {
		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		if err != nil {
			t.Fatalf("failed to open sqlite database: %v", err)
		}
		if err := db.AutoMigrate(&model.User{}); err != nil {
			t.Fatalf("failed to migrate users table: %v", err)
		}
		database.DB = db

		if err := ensureUserSessionVersionColumnReady(); err != nil {
			t.Fatalf("expected session_version column to be accepted: %v", err)
		}
	})

	t.Run("缺少字段时报错", func(t *testing.T) {
		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		if err != nil {
			t.Fatalf("failed to open sqlite database: %v", err)
		}
		if err := db.Exec("CREATE TABLE users (id INTEGER PRIMARY KEY)").Error; err != nil {
			t.Fatalf("failed to create users table: %v", err)
		}
		database.DB = db

		if err := ensureUserSessionVersionColumnReady(); err == nil {
			t.Fatal("expected missing session_version column to be rejected")
		}
	})
}

func TestShouldRunStartupMigrate(t *testing.T) {
	t.Run("release 默认关闭", func(t *testing.T) {
		t.Setenv("GIN_MODE", "release")
		t.Setenv("ENABLE_AUTO_MIGRATE", "")

		if shouldRunStartupMigrate() {
			t.Fatal("expected auto migrate to be disabled in release by default")
		}
	})

	t.Run("未设置模式时按 release 关闭", func(t *testing.T) {
		t.Setenv("GIN_MODE", "")
		t.Setenv("ENABLE_AUTO_MIGRATE", "")

		if shouldRunStartupMigrate() {
			t.Fatal("expected auto migrate to be disabled when GIN_MODE is unset")
		}
	})

	t.Run("release 显式开启", func(t *testing.T) {
		t.Setenv("GIN_MODE", "release")
		t.Setenv("ENABLE_AUTO_MIGRATE", "true")

		if !shouldRunStartupMigrate() {
			t.Fatal("expected auto migrate to be enabled when explicitly requested")
		}
	})

	t.Run("开发环境默认开启", func(t *testing.T) {
		t.Setenv("GIN_MODE", "debug")
		t.Setenv("ENABLE_AUTO_MIGRATE", "")

		if !shouldRunStartupMigrate() {
			t.Fatal("expected auto migrate to be enabled in debug mode by default")
		}
	})

	t.Run("test 模式默认关闭", func(t *testing.T) {
		t.Setenv("GIN_MODE", "test")
		t.Setenv("ENABLE_AUTO_MIGRATE", "")

		if shouldRunStartupMigrate() {
			t.Fatal("expected auto migrate to be disabled in test mode by default")
		}
	})

	t.Run("开发环境可显式关闭", func(t *testing.T) {
		t.Setenv("GIN_MODE", "debug")
		t.Setenv("ENABLE_AUTO_MIGRATE", "false")

		if shouldRunStartupMigrate() {
			t.Fatal("expected auto migrate to be disabled when explicitly requested")
		}
	})
}

func TestShouldRunStartupSeed(t *testing.T) {
	t.Run("release 默认关闭", func(t *testing.T) {
		t.Setenv("GIN_MODE", "release")
		t.Setenv("ENABLE_STARTUP_SEED", "")

		if shouldRunStartupSeed() {
			t.Fatal("expected startup seed to be disabled in release by default")
		}
	})

	t.Run("未设置模式时按 release 关闭", func(t *testing.T) {
		t.Setenv("GIN_MODE", "")
		t.Setenv("ENABLE_STARTUP_SEED", "")

		if shouldRunStartupSeed() {
			t.Fatal("expected startup seed to be disabled when GIN_MODE is unset")
		}
	})

	t.Run("release 显式开启", func(t *testing.T) {
		t.Setenv("GIN_MODE", "release")
		t.Setenv("ENABLE_STARTUP_SEED", "true")

		if !shouldRunStartupSeed() {
			t.Fatal("expected startup seed to be enabled when explicitly requested")
		}
	})

	t.Run("开发环境默认开启", func(t *testing.T) {
		t.Setenv("GIN_MODE", "debug")
		t.Setenv("ENABLE_STARTUP_SEED", "")

		if !shouldRunStartupSeed() {
			t.Fatal("expected startup seed to be enabled in debug mode by default")
		}
	})

	t.Run("test 模式默认关闭", func(t *testing.T) {
		t.Setenv("GIN_MODE", "test")
		t.Setenv("ENABLE_STARTUP_SEED", "")

		if shouldRunStartupSeed() {
			t.Fatal("expected startup seed to be disabled in test mode by default")
		}
	})

	t.Run("开发环境可显式关闭", func(t *testing.T) {
		t.Setenv("GIN_MODE", "debug")
		t.Setenv("ENABLE_STARTUP_SEED", "false")

		if shouldRunStartupSeed() {
			t.Fatal("expected startup seed to be disabled when explicitly requested")
		}
	})
}
