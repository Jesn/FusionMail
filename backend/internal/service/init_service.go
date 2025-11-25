package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"fusionmail/internal/model"
	cryptoutil "fusionmail/pkg/crypto"
	"fusionmail/pkg/database"

	"gorm.io/gorm"
)

// InitService 系统初始化服务
type InitService struct {
	db *gorm.DB
}

// NewInitService 创建初始化服务
func NewInitService() *InitService {
	return &InitService{
		db: database.DB,
	}
}

// InitializeSystem 初始化系统
func (s *InitService) InitializeSystem() error {
	log.Println("Starting system initialization...")

	// 检查是否已存在管理员用户
	var adminUser model.User
	err := s.db.Where("role = ?", "admin").First(&adminUser).Error
	if err == nil {
		log.Println("Admin user already exists, skipping initialization")
		return nil
	}

	if err != gorm.ErrRecordNotFound {
		return fmt.Errorf("failed to check admin user: %w", err)
	}

	// 生成随机密码
	password, err := generateRandomPassword(16)
	if err != nil {
		return fmt.Errorf("failed to generate random password: %w", err)
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

	// 保存密码到文件（测试环境）
	if err := s.savePasswordToFile(password); err != nil {
		log.Printf("Warning: failed to save password to file: %v", err)
	}

	log.Println("System initialization completed successfully!")
	log.Printf("Admin user created with username: admin")
	log.Printf("Initial password: %s", password)
	log.Println("Please change the password after first login!")

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

// savePasswordToFile 保存密码到文件（仅用于测试环境）
func (s *InitService) savePasswordToFile(password string) error {
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

	log.Printf("Password saved to: %s", passwordFile)
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
	return s.db.Model(&model.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"last_login_at":   lastLoginAt,
		"last_login_ip":   lastLoginIP,
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