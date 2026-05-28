package service

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"fusionmail/internal/model"
	cryptoutil "fusionmail/pkg/crypto"
	"fusionmail/pkg/database"
	"fusionmail/pkg/logger"
	"fusionmail/pkg/runtimeenv"

	"gorm.io/gorm"
)

// 模块日志记录器
var initLog = logger.NewWithModule("Init")

// InitService 系统初始化服务
type InitService struct {
	db *gorm.DB
}

// NewInitService 创建初始化服务
func NewInitService() *InitService {
	initLog.Debug("创建初始化服务, database.DB is nil: %v", database.DB == nil)
	if database.DB != nil {
		sqlDB, err := database.DB.DB()
		if err == nil {
			stats := sqlDB.Stats()
			initLog.Debug("数据库连接池状态: OpenConnections=%d, InUse=%d, Idle=%d",
				stats.OpenConnections, stats.InUse, stats.Idle)
		}
	}
	return &InitService{
		db: database.DB,
	}
}

// NormalizeTwoFactorStorage 将历史 2FA 明文数据升级为加密或哈希存储
func (s *InitService) NormalizeTwoFactorStorage() error {
	if s.db == nil {
		return fmt.Errorf("database is not initialized")
	}

	var users []model.User
	if err := s.db.Where("two_factor_secret <> ? OR two_factor_backup <> ?", "", "").Find(&users).Error; err != nil {
		return fmt.Errorf("failed to load users with 2FA storage: %w", err)
	}

	totpService := NewTOTPService("FusionMail")
	normalizedCount := 0
	for _, user := range users {
		updates := make(map[string]any)

		if user.TwoFactorSecret != "" && !totpService.IsEncryptedSecret(user.TwoFactorSecret) {
			encryptedSecret, err := totpService.EncryptSecret(user.TwoFactorSecret)
			if err != nil {
				return fmt.Errorf("failed to encrypt 2FA secret for user %d: %w", user.ID, err)
			}
			updates["two_factor_secret"] = encryptedSecret
		}

		if user.TwoFactorBackup != "" {
			var backupCodes []string
			if err := json.Unmarshal([]byte(user.TwoFactorBackup), &backupCodes); err != nil {
				return fmt.Errorf("failed to parse 2FA backup codes for user %d: %w", user.ID, err)
			}
			if backupCodesNeedHashing(backupCodes) {
				hashedBackupCodes, err := totpService.HashBackupCodes(backupCodes)
				if err != nil {
					return fmt.Errorf("failed to hash 2FA backup codes for user %d: %w", user.ID, err)
				}
				backupCodesJSON, err := json.Marshal(hashedBackupCodes)
				if err != nil {
					return fmt.Errorf("failed to encode 2FA backup codes for user %d: %w", user.ID, err)
				}
				updates["two_factor_backup"] = string(backupCodesJSON)
			}
		}

		if len(updates) == 0 {
			continue
		}
		if err := s.db.Model(&model.User{}).Where("id = ?", user.ID).Updates(updates).Error; err != nil {
			return fmt.Errorf("failed to normalize 2FA storage for user %d: %w", user.ID, err)
		}
		normalizedCount++
	}

	if normalizedCount > 0 {
		initLog.Info("已升级 %d 个用户的 2FA 敏感存储", normalizedCount)
	}
	return nil
}

func backupCodesNeedHashing(backupCodes []string) bool {
	for _, backupCode := range backupCodes {
		if !isHashedBackupCode(backupCode) {
			return true
		}
	}
	return false
}

func (s *InitService) ConsumeBackupCode(userID int64, code string) (bool, int, error) {
	if s.db == nil {
		return false, 0, fmt.Errorf("database is not initialized")
	}

	var user model.User
	if err := s.db.Select("id", "two_factor_backup").Where("id = ?", userID).First(&user).Error; err != nil {
		return false, 0, fmt.Errorf("failed to load 2FA backup codes for user %d: %w", userID, err)
	}
	if user.TwoFactorBackup == "" {
		return false, 0, nil
	}

	var backupCodes []string
	if err := json.Unmarshal([]byte(user.TwoFactorBackup), &backupCodes); err != nil {
		return false, 0, fmt.Errorf("failed to parse 2FA backup codes for user %d: %w", userID, err)
	}

	valid, remaining, err := NewTOTPService("FusionMail").ValidateBackupCode(backupCodes, code)
	if err != nil {
		return false, 0, err
	}
	if !valid {
		return false, len(backupCodes), nil
	}

	remainingJSON, err := json.Marshal(remaining)
	if err != nil {
		return false, 0, fmt.Errorf("failed to encode 2FA backup codes for user %d: %w", userID, err)
	}

	result := s.db.Model(&model.User{}).
		Where("id = ? AND two_factor_backup = ?", userID, user.TwoFactorBackup).
		Update("two_factor_backup", string(remainingJSON))
	if result.Error != nil {
		return false, 0, fmt.Errorf("failed to consume 2FA backup code for user %d: %w", userID, result.Error)
	}
	if result.RowsAffected != 1 {
		return false, 0, nil
	}

	return true, len(remaining), nil
}

// InitializeSystem 初始化系统
func (s *InitService) InitializeSystem() error {
	initLog.Info("开始系统初始化...")

	// 检查是否已存在管理员用户
	var adminUser model.User
	err := s.db.Where("role = ?", "admin").First(&adminUser).Error
	if err == nil {
		initLog.Info("管理员用户已存在，跳过初始化")
		return nil
	}

	if err != gorm.ErrRecordNotFound {
		return fmt.Errorf("failed to check admin user: %w", err)
	}

	// 优先使用环境变量中的密码，否则生成随机密码
	password := os.Getenv("ADMIN_PASSWORD")
	passwordFromEnv := password != ""
	if password == "" {
		if !shouldSavePasswordFile() {
			return fmt.Errorf("ADMIN_PASSWORD 未设置，且当前环境未启用安全密码交付，请显式设置 ADMIN_PASSWORD 或开启 SAVE_PASSWORD_FILE")
		}

		initLog.Info("ADMIN_PASSWORD 未设置，生成随机密码...")
		password, err = generateRandomPassword(16)
		if err != nil {
			return fmt.Errorf("failed to generate random password: %w", err)
		}
	} else {
		initLog.Info("使用环境变量 ADMIN_PASSWORD 中的密码")
		// 验证密码强度（至少8个字符）
		if len(password) < 8 {
			return fmt.Errorf("ADMIN_PASSWORD must be at least 8 characters long")
		}
	}

	passwordSaved := false
	if !passwordFromEnv && shouldSavePasswordFile() {
		if err := s.savePasswordToFile(password); err != nil {
			return fmt.Errorf("failed to persist generated admin password: %w", err)
		}
		passwordSaved = true
	}

	// 生成密码哈希
	passwordHash, err := cryptoutil.HashPassword(password)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// 创建管理员用户
	adminUser = model.User{
		Username:     "admin",
		Email:        "admin@localhost",
		PasswordHash: passwordHash,
		DisplayName:  "Administrator",
		Role:         "admin",
		IsActive:     true,
	}

	if err := s.db.Create(&adminUser).Error; err != nil {
		return fmt.Errorf("failed to create admin user: %w", err)
	}

	initLog.Info("系统初始化完成！")
	initLog.Info("管理员用户已创建，用户名: admin")

	if passwordFromEnv {
		initLog.Info("管理员初始密码已通过环境变量提供")
	} else if passwordSaved {
		initLog.Info("管理员初始密码已保存到本地 passwd 文件")
	} else {
		initLog.Info("管理员初始密码已生成，请通过安全分发渠道交付")
	}

	initLog.Warn("⚠️  重要提示：请在首次登录后修改密码！")

	return nil
}

// generateRandomPassword 生成随机密码
func generateRandomPassword(length int) (string, error) {
	bytes := make([]byte, length/2)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func shouldSavePasswordFile() bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv("SAVE_PASSWORD_FILE")))
	if value == "" {
		return runtimeenv.CurrentGinMode() == "debug"
	}

	return runtimeenv.EnvBool("SAVE_PASSWORD_FILE", false)
}

// savePasswordToFile 保存密码到文件（仅用于显式允许或本地调试环境）
func (s *InitService) savePasswordToFile(password string) error {
	if !shouldSavePasswordFile() {
		return nil
	}

	// 获取项目根目录
	pwd, err := os.Getwd()
	if err != nil {
		return err
	}

	// 构建密码文件路径
	passwordFile := filepath.Join(pwd, "passwd")

	// 写入密码到文件
	if err := os.WriteFile(passwordFile, []byte(password), 0600); err != nil {
		return fmt.Errorf("failed to write password file: %w", err)
	}

	initLog.Info("✅ 密码已保存到: %s", passwordFile)
	initLog.Warn("⚠️  警告：此文件包含敏感信息！")
	initLog.Info("💡 建议：首次登录后删除此文件或妥善保管")

	return nil
}

// ChangePassword 修改用户密码
func (s *InitService) ChangePassword(userID int64, oldPassword, newPassword string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var user model.User
		if err := tx.First(&user, userID).Error; err != nil {
			return fmt.Errorf("user not found: %w", err)
		}

		if !cryptoutil.VerifyPassword(oldPassword, user.PasswordHash) {
			return fmt.Errorf("incorrect old password")
		}

		newPasswordHash, err := cryptoutil.HashPassword(newPassword)
		if err != nil {
			return fmt.Errorf("failed to hash new password: %w", err)
		}

		result := tx.Model(&model.User{}).Where("id = ?", userID).Updates(map[string]any{
			"password_hash":   newPasswordHash,
			"session_version": gorm.Expr("session_version + ?", 1),
		})
		if result.Error != nil {
			return fmt.Errorf("failed to update password: %w", result.Error)
		}
		if result.RowsAffected != 1 {
			return fmt.Errorf("user not found")
		}
		return nil
	})
}

func (s *InitService) IncrementSessionVersion(userID int64) error {
	if s.db == nil {
		return fmt.Errorf("database is not initialized")
	}

	result := s.db.Model(&model.User{}).
		Where("id = ?", userID).
		UpdateColumn("session_version", gorm.Expr("session_version + ?", 1))
	if result.Error != nil {
		return fmt.Errorf("failed to increment session version for user %d: %w", userID, result.Error)
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("user not found")
	}
	return nil
}

// GetUserByUsername 根据用户名获取用户
func (s *InitService) GetUserByUsername(username string) (*model.User, error) {
	var user model.User
	if err := s.db.Where("username = ?", username).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByID 根据ID获取用户
func (s *InitService) GetUserByID(id int64) (*model.User, error) {
	var user model.User
	if err := s.db.First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// UpdateLastLogin 更新最后登录信息
func (s *InitService) UpdateLastLogin(userID int64, lastLoginAt *time.Time, lastLoginIP string) error {
	return s.db.Model(&model.User{}).Where("id = ?", userID).Updates(map[string]any{
		"last_login_at": lastLoginAt,
		"last_login_ip": lastLoginIP,
	}).Error
}

// ValidateUserCredentials 验证用户凭据
func (s *InitService) ValidateUserCredentials(username, password string) (*model.User, error) {
	user, err := s.GetUserByUsername(username)
	if err != nil {
		return nil, fmt.Errorf("invalid username or password")
	}

	// 检查用户是否被锁定
	if user.LockedUntil != nil && user.LockedUntil.After(time.Now()) {
		return nil, fmt.Errorf("account is locked until %s", user.LockedUntil.Format("2006-01-02 15:04:05"))
	}

	// 检查用户是否激活
	if !user.IsActive {
		return nil, fmt.Errorf("account is disabled")
	}

	// 验证密码
	if !cryptoutil.VerifyPassword(password, user.PasswordHash) {
		// 增加失败次数
		user.FailedLoginAttempts++

		// 如果失败次数超过5次，锁定账户30分钟
		if user.FailedLoginAttempts >= 5 {
			lockedUntil := time.Now().Add(30 * time.Minute)
			user.LockedUntil = &lockedUntil
		}

		s.db.Save(user)
		return nil, fmt.Errorf("invalid username or password")
	}

	// 登录成功，重置失败次数和锁定状态
	user.FailedLoginAttempts = 0
	user.LockedUntil = nil
	s.db.Save(user)

	return user, nil
}

// ==================== 2FA 双因素认证相关方法 ====================

// Update2FASetup 保存 2FA 设置（未验证状态）
func (s *InitService) Update2FASetup(userID int64, secret, backupCodes string) error {
	return s.db.Model(&model.User{}).Where("id = ?", userID).Updates(map[string]any{
		"two_factor_secret":   secret,
		"two_factor_backup":   backupCodes,
		"two_factor_verified": false,
	}).Error
}

// Enable2FA 启用 2FA
func (s *InitService) Enable2FA(userID int64, enabledAt *time.Time) error {
	return s.db.Model(&model.User{}).Where("id = ?", userID).Updates(map[string]any{
		"two_factor_enabled":    true,
		"two_factor_verified":   true,
		"two_factor_enabled_at": enabledAt,
		"session_version":       gorm.Expr("session_version + ?", 1),
	}).Error
}

// Disable2FA 禁用 2FA
func (s *InitService) Disable2FA(userID int64) error {
	return s.db.Model(&model.User{}).Where("id = ?", userID).Updates(map[string]any{
		"two_factor_enabled":    false,
		"two_factor_secret":     "",
		"two_factor_backup":     "",
		"two_factor_verified":   false,
		"two_factor_enabled_at": nil,
		"session_version":       gorm.Expr("session_version + ?", 1),
	}).Error
}

// UpdateBackupCodes 更新恢复码
func (s *InitService) UpdateBackupCodes(userID int64, backupCodes string) error {
	return s.db.Model(&model.User{}).Where("id = ?", userID).Updates(map[string]any{
		"two_factor_backup": backupCodes,
	}).Error
}
