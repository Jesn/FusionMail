package service

import (
	"encoding/json"
	"testing"
	"time"

	"fusionmail/internal/model"
	cryptoutil "fusionmail/pkg/crypto"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestNormalizeTwoFactorStorageEncryptsLegacySecretAndHashesBackupCodes(t *testing.T) {
	t.Setenv("ENCRYPTION_KEY", "test-two-factor-secret-key-32-bytes")

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite database: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatalf("failed to migrate users table: %v", err)
	}

	originalSecret := "JBSWY3DPEHPK3PXP"
	originalBackupCodes := []string{"11111111", "22222222"}
	backupCodesJSON, err := json.Marshal(originalBackupCodes)
	if err != nil {
		t.Fatalf("failed to encode backup codes: %v", err)
	}

	user := model.User{
		Username:        "alice",
		Email:           "alice@example.com",
		PasswordHash:    "hash",
		TwoFactorSecret: originalSecret,
		TwoFactorBackup: string(backupCodesJSON),
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	initService := &InitService{db: db}
	if err := initService.NormalizeTwoFactorStorage(); err != nil {
		t.Fatalf("NormalizeTwoFactorStorage returned error: %v", err)
	}
	if err := initService.NormalizeTwoFactorStorage(); err != nil {
		t.Fatalf("NormalizeTwoFactorStorage should be idempotent: %v", err)
	}

	var updatedUser model.User
	if err := db.First(&updatedUser, user.ID).Error; err != nil {
		t.Fatalf("failed to load updated user: %v", err)
	}

	totpService := NewTOTPService("FusionMail")
	if updatedUser.TwoFactorSecret == originalSecret {
		t.Fatal("expected legacy TOTP secret to be encrypted")
	}
	if !totpService.IsEncryptedSecret(updatedUser.TwoFactorSecret) {
		t.Fatal("expected encrypted TOTP secret prefix")
	}
	secret, err := totpService.DecryptSecret(updatedUser.TwoFactorSecret)
	if err != nil {
		t.Fatalf("DecryptSecret returned error: %v", err)
	}
	if secret != originalSecret {
		t.Fatalf("expected decrypted secret %q, got %q", originalSecret, secret)
	}

	var hashedBackupCodes []string
	if err := json.Unmarshal([]byte(updatedUser.TwoFactorBackup), &hashedBackupCodes); err != nil {
		t.Fatalf("failed to decode hashed backup codes: %v", err)
	}
	if len(hashedBackupCodes) != len(originalBackupCodes) {
		t.Fatalf("expected %d backup codes, got %d", len(originalBackupCodes), len(hashedBackupCodes))
	}
	for i, hashedBackupCode := range hashedBackupCodes {
		if hashedBackupCode == originalBackupCodes[i] {
			t.Fatalf("expected backup code %d to be hashed", i)
		}
	}

	valid, _, err := totpService.ValidateBackupCode(hashedBackupCodes, originalBackupCodes[0])
	if err != nil {
		t.Fatalf("ValidateBackupCode returned error: %v", err)
	}
	if !valid {
		t.Fatal("expected hashed migrated backup code to validate")
	}
}

func TestConsumeBackupCodeConsumesHashedCodeOnce(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite database: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatalf("failed to migrate users table: %v", err)
	}

	totpService := NewTOTPService("FusionMail")
	hashedCodes, err := totpService.HashBackupCodes([]string{"11111111", "22222222"})
	if err != nil {
		t.Fatalf("HashBackupCodes returned error: %v", err)
	}
	backupCodesJSON, err := json.Marshal(hashedCodes)
	if err != nil {
		t.Fatalf("failed to encode backup codes: %v", err)
	}

	user := model.User{
		Username:        "bob",
		Email:           "bob@example.com",
		PasswordHash:    "hash",
		TwoFactorBackup: string(backupCodesJSON),
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	initService := &InitService{db: db}
	valid, remainingCount, err := initService.ConsumeBackupCode(user.ID, "11111111")
	if err != nil {
		t.Fatalf("ConsumeBackupCode returned error: %v", err)
	}
	if !valid {
		t.Fatal("expected backup code to validate")
	}
	if remainingCount != 1 {
		t.Fatalf("expected one remaining backup code, got %d", remainingCount)
	}

	valid, _, err = initService.ConsumeBackupCode(user.ID, "11111111")
	if err != nil {
		t.Fatalf("ConsumeBackupCode returned error: %v", err)
	}
	if valid {
		t.Fatal("expected consumed backup code to be rejected")
	}
}

func TestChangePasswordIncrementsSessionVersion(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite database: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatalf("failed to migrate users table: %v", err)
	}

	passwordHash, err := cryptoutil.HashPassword("old-password")
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	user := model.User{
		Username:       "carol",
		Email:          "carol@example.com",
		PasswordHash:   passwordHash,
		SessionVersion: 7,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	initService := &InitService{db: db}
	if err := initService.ChangePassword(user.ID, "old-password", "new-password"); err != nil {
		t.Fatalf("ChangePassword returned error: %v", err)
	}

	var updatedUser model.User
	if err := db.First(&updatedUser, user.ID).Error; err != nil {
		t.Fatalf("failed to load updated user: %v", err)
	}
	if updatedUser.SessionVersion != 8 {
		t.Fatalf("expected session version 8, got %d", updatedUser.SessionVersion)
	}
	if !cryptoutil.VerifyPassword("new-password", updatedUser.PasswordHash) {
		t.Fatal("expected new password hash to validate")
	}
}

func TestTwoFactorStateChangesIncrementSessionVersion(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite database: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatalf("failed to migrate users table: %v", err)
	}

	user := model.User{
		Username:       "dave",
		Email:          "dave@example.com",
		PasswordHash:   "hash",
		SessionVersion: 4,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	initService := &InitService{db: db}
	now := time.Now()
	if err := initService.Enable2FA(user.ID, &now); err != nil {
		t.Fatalf("Enable2FA returned error: %v", err)
	}
	var enabledUser model.User
	if err := db.First(&enabledUser, user.ID).Error; err != nil {
		t.Fatalf("failed to load enabled user: %v", err)
	}
	if enabledUser.SessionVersion != 5 {
		t.Fatalf("expected session version 5 after enabling 2FA, got %d", enabledUser.SessionVersion)
	}

	if err := initService.Disable2FA(user.ID); err != nil {
		t.Fatalf("Disable2FA returned error: %v", err)
	}
	var disabledUser model.User
	if err := db.First(&disabledUser, user.ID).Error; err != nil {
		t.Fatalf("failed to load disabled user: %v", err)
	}
	if disabledUser.SessionVersion != 6 {
		t.Fatalf("expected session version 6 after disabling 2FA, got %d", disabledUser.SessionVersion)
	}
}

func TestShouldSavePasswordFile(t *testing.T) {
	originalMode := gin.Mode()
	defer gin.SetMode(originalMode)

	t.Run("debug 模式默认保存", func(t *testing.T) {
		gin.SetMode(gin.DebugMode)
		t.Setenv("GIN_MODE", "debug")
		t.Setenv("SAVE_PASSWORD_FILE", "")

		if !shouldSavePasswordFile() {
			t.Fatal("expected password file saving to default to enabled in debug mode")
		}
	})

	t.Run("release 模式默认关闭", func(t *testing.T) {
		gin.SetMode(gin.ReleaseMode)
		t.Setenv("GIN_MODE", "release")
		t.Setenv("SAVE_PASSWORD_FILE", "")

		if shouldSavePasswordFile() {
			t.Fatal("expected password file saving to default to disabled in release mode")
		}
	})

	t.Run("test 模式默认关闭", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		t.Setenv("GIN_MODE", "test")
		t.Setenv("SAVE_PASSWORD_FILE", "")

		if shouldSavePasswordFile() {
			t.Fatal("expected password file saving to default to disabled in test mode")
		}
	})

	t.Run("可显式开启", func(t *testing.T) {
		gin.SetMode(gin.ReleaseMode)
		t.Setenv("GIN_MODE", "release")
		t.Setenv("SAVE_PASSWORD_FILE", "true")

		if !shouldSavePasswordFile() {
			t.Fatal("expected password file saving to be enabled when explicitly requested")
		}
	})

	t.Run("非法值按关闭处理", func(t *testing.T) {
		gin.SetMode(gin.DebugMode)
		t.Setenv("GIN_MODE", "debug")
		t.Setenv("SAVE_PASSWORD_FILE", "maybe")

		if shouldSavePasswordFile() {
			t.Fatal("expected invalid SAVE_PASSWORD_FILE to disable password file saving")
		}
	})
}
