package seed

import (
	"fusionmail/internal/model"

	"gorm.io/gorm"
)

// seedSettings 初始化系统设置种子数据
func seedSettings(db *gorm.DB) error {
	log.Debug("初始化系统设置数据...")

	// 定义所有默认设置种子数据（系统级，userID 为 nil）
	defaultSettings := []model.Setting{
		// UI 设置
		{Category: "ui", Key: "theme", Value: "system", ValueType: "string", IsPublic: true, Description: "界面主题：light/dark/system"},
		{Category: "ui", Key: "language", Value: "zh-CN", ValueType: "string", IsPublic: true, Description: "界面语言"},
		{Category: "ui", Key: "email_page_size", Value: "50", ValueType: "number", IsPublic: true, Description: "每页显示邮件数量"},
		{Category: "ui", Key: "default_view", Value: "list", ValueType: "string", IsPublic: true, Description: "默认视图模式：list/card"},

		// 同步设置
		{Category: "sync", Key: "enable_auto_sync", Value: "true", ValueType: "boolean", IsPublic: true, Description: "启用自动同步"},
		{Category: "sync", Key: "sync_interval", Value: "300", ValueType: "number", IsPublic: true, Description: "同步间隔（秒）"},
		{Category: "sync", Key: "max_concurrent_syncs", Value: "3", ValueType: "number", IsPublic: false, Description: "最大并发同步数"},

		// 通知设置
		{Category: "notification", Key: "enable_desktop_notification", Value: "true", ValueType: "boolean", IsPublic: true, Description: "启用桌面通知"},
		{Category: "notification", Key: "enable_email_notification", Value: "false", ValueType: "boolean", IsPublic: true, Description: "启用邮件通知"},
		{Category: "notification", Key: "notification_sound", Value: "true", ValueType: "boolean", IsPublic: true, Description: "启用通知声音"},
		{Category: "notification", Key: "unread_only", Value: "true", ValueType: "boolean", IsPublic: true, Description: "仅通知未读邮件"},

		// 安全设置
		{Category: "security", Key: "session_timeout", Value: "1440", ValueType: "number", IsPublic: false, Description: "会话超时时间（分钟）"},
		{Category: "security", Key: "login_max_attempts", Value: "5", ValueType: "number", IsPublic: false, Description: "最大登录尝试次数"},
		{Category: "security", Key: "password_complexity", Value: "true", ValueType: "boolean", IsPublic: false, Description: "启用密码复杂度检查"},
		{Category: "security", Key: "jwt_expiry", Value: "24", ValueType: "number", IsPublic: false, Description: "JWT 过期时间（小时）"},

		// API 设置
		{Category: "api", Key: "rate_limit_enabled", Value: "true", ValueType: "boolean", IsPublic: false, Description: "启用 API 速率限制"},
		{Category: "api", Key: "rate_limit_site", Value: "100", ValueType: "number", IsPublic: false, Description: "站点 API 速率限制（次/分钟）"},
		{Category: "api", Key: "rate_limit_public", Value: "200", ValueType: "number", IsPublic: false, Description: "公开 API 速率限制（次/分钟）"},

		// 系统设置
		{Category: "system", Key: "trash_auto_cleanup_days", Value: "7", ValueType: "number", IsPublic: false, Description: "回收站自动清理天数，-1 表示永不清理"},
		{Category: "system", Key: "sync_logs_retention_days", Value: "7", ValueType: "number", IsPublic: false, Description: "同步日志保留天数，-1 表示永不清理"},
		{Category: "system", Key: "webhook_logs_retention_days", Value: "14", ValueType: "number", IsPublic: false, Description: "Webhook 日志保留天数，-1 表示永不清理"},
		{Category: "system", Key: "spam_detection_logs_retention_days", Value: "7", ValueType: "number", IsPublic: false, Description: "垃圾邮件检测日志保留天数，-1 表示永不清理"},

		// 垃圾邮件设置
		{Category: "spam", Key: "spam_detection_enabled", Value: "true", ValueType: "boolean", IsPublic: true, Description: "启用垃圾邮件检测"},
		{Category: "spam", Key: "user_spam_detection_enabled", Value: "true", ValueType: "boolean", IsPublic: true, Description: "用户级垃圾邮件检测"},
		{Category: "spam", Key: "spam_threshold", Value: "60", ValueType: "number", IsPublic: true, Description: "垃圾邮件评分阈值（0-100）"},
		{Category: "spam", Key: "spam_auto_cleanup_days", Value: "30", ValueType: "number", IsPublic: true, Description: "垃圾邮件自动清理天数，-1 表示永不清理"},
		{Category: "spam", Key: "bayesian_enabled", Value: "true", ValueType: "boolean", IsPublic: true, Description: "启用贝叶斯过滤"},
		{Category: "spam", Key: "rbl_enabled", Value: "false", ValueType: "boolean", IsPublic: true, Description: "启用 RBL 检查"},
		{Category: "spam", Key: "surbl_enabled", Value: "false", ValueType: "boolean", IsPublic: true, Description: "启用 SURBL 检查"},
	}

	// 使用 FirstOrCreate 确保不会重复插入
	for _, setting := range defaultSettings {
		var existing model.Setting
		// 系统级设置 user_id 为 NULL
		result := db.Where("user_id IS NULL AND category = ? AND key = ?", setting.Category, setting.Key).First(&existing)
		if result.Error != nil {
			// 记录不存在，创建新记录
			if err := db.Create(&setting).Error; err != nil {
				log.Warn("创建设置失败: %s/%s, %v", setting.Category, setting.Key, err)
			} else {
				log.Debug("创建设置: %s/%s = %s", setting.Category, setting.Key, setting.Value)
			}
		}
		// 如果记录已存在，不更新（保留用户自定义值）
	}

	log.Debug("系统设置数据初始化完成")
	return nil
}
