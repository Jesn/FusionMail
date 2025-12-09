package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"fusionmail/internal/model"
	cryptoutil "fusionmail/pkg/crypto"
	"fusionmail/pkg/database"
	"fusionmail/pkg/logger"

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
	if password == "" {
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

	// 保存密码到文件（开发/测试环境）
	if err := s.savePasswordToFile(password); err != nil {
		initLog.Warn("保存密码到文件失败: %v", err)
	}

	initLog.Info("系统初始化完成！")
	initLog.Info("管理员用户已创建，用户名: admin")

	// 只在开发环境输出密码到日志
	if os.Getenv("GIN_MODE") != "release" {
		initLog.Info("初始密码: %s", password)
	} else {
		initLog.Info("初始密码已设置（请查看 passwd 文件或 ADMIN_PASSWORD 环境变量）")
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

// savePasswordToFile 保存密码到文件（仅用于开发/测试环境）
func (s *InitService) savePasswordToFile(password string) error {
	// 检查是否为生产环境且未明确启用密码文件保存
	ginMode := os.Getenv("GIN_MODE")
	savePasswordFile := os.Getenv("SAVE_PASSWORD_FILE")

	if ginMode == "release" && savePasswordFile != "true" {
		initLog.Warn("⚠️  检测到生产模式：出于安全考虑，跳过密码文件创建")
		initLog.Info("💡 提示：设置 SAVE_PASSWORD_FILE=true 可强制创建密码文件（不推荐）")
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
	var user model.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// 验证旧密码
	if !cryptoutil.VerifyPassword(oldPassword, user.PasswordHash) {
		return fmt.Errorf("incorrect old password")
	}

	// 生成新密码哈希
	newPasswordHash, err := cryptoutil.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	// 更新密码
	user.PasswordHash = newPasswordHash
	if err := s.db.Save(&user).Error; err != nil {
		return fmt.Errorf("failed to update password: %w", err)
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
	}).Error
}

// UpdateBackupCodes 更新恢复码
func (s *InitService) UpdateBackupCodes(userID int64, backupCodes string) error {
	return s.db.Model(&model.User{}).Where("id = ?", userID).Updates(map[string]any{
		"two_factor_backup": backupCodes,
	}).Error
}
