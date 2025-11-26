package service

// SettingConstants 配置常量定义
// 用于定义配置分类、敏感配置列表、公开配置列表等

// SensitiveSettings 需要加密的敏感配置项
// key: category, value: map of setting keys that are sensitive
var SensitiveSettings = map[string]map[string]bool{
	"oauth": {
		"gmail_client_secret":     true,
		"microsoft_client_secret": true,
	},
	"security": {
		"jwt_secret":      true,
		"encryption_key":  true,
		"master_password": true,
		"webhook_secret":  true,
		"admin_password":  true,
	},
	"smtp": {
		"smtp_password":  true,
		"smtp_from_name": false, // 非敏感
	},
	"api": {
		"secret_keys": true,
	},
}

// PublicSettings 公开配置列表（前端可获取）
// key: category, value: list of public setting keys
var PublicSettings = map[string][]string{
	"ui": {
		"theme", "language", "email_page_size", "default_view",
	},
	"sync": {
		"enable_auto_sync", "sync_interval",
	},
	"notification": {
		"enable_desktop_notification", "enable_email_notification", "notification_sound",
	},
}

// DefaultValues 默认配置值
var DefaultValues = map[string]map[string]string{
	"ui": {
		"theme":           "light",
		"language":        "zh-CN",
		"email_page_size": "50",
		"default_view":    "list",
	},
	"sync": {
		"enable_auto_sync":     "true",
		"sync_interval":        "300",
		"max_concurrent_syncs": "3",
	},
	"notification": {
		"enable_desktop_notification": "true",
		"enable_email_notification":   "true",
		"notification_sound":          "true",
		"unread_only":                 "true",
	},
	"security": {
		"session_timeout":     "1440", // 24小时
		"login_max_attempts":  "5",
		"password_complexity": "true",
		"jwt_expiry":          "24", // 24小时
	},
	"api": {
		"rate_limit_enabled": "true",
		"rate_limit_site":    "100",
		"rate_limit_public":  "200",
	},
	"system": {
		"trash_auto_cleanup_days": "7", // 回收站自动清理天数，-1 表示永不清理
	},
}

// ValueTypeMap 配置值类型映射
var ValueTypeMap = map[string]map[string]string{
	"ui": {
		"theme":           "string",
		"language":        "string",
		"email_page_size": "number",
		"default_view":    "string",
	},
	"sync": {
		"enable_auto_sync":     "boolean",
		"sync_interval":        "number",
		"max_concurrent_syncs": "number",
	},
	"notification": {
		"enable_desktop_notification": "boolean",
		"enable_email_notification":   "boolean",
		"notification_sound":          "boolean",
		"unread_only":                 "boolean",
	},
	"security": {
		"session_timeout":     "number",
		"login_max_attempts":  "number",
		"password_complexity": "boolean",
		"jwt_expiry":          "number",
	},
	"api": {
		"rate_limit_enabled": "boolean",
		"rate_limit_site":    "number",
		"rate_limit_public":  "number",
	},
	"oauth": {
		"gmail_client_id":         "string",
		"gmail_client_secret":     "string",
		"microsoft_client_id":     "string",
		"microsoft_client_secret": "string",
	},
	"smtp": {
		"smtp_host":      "string",
		"smtp_port":      "number",
		"smtp_username":  "string",
		"smtp_password":  "string",
		"smtp_from":      "string",
		"smtp_from_name": "string",
	},
	"system": {
		"trash_auto_cleanup_days": "number",
	},
}

// CategoryDescriptions 配置分类描述
var CategoryDescriptions = map[string]string{
	"ui":           "界面相关配置",
	"sync":         "同步相关配置",
	"notification": "通知相关配置",
	"security":     "安全相关配置",
	"api":          "API相关配置",
	"oauth":        "OAuth2配置",
	"smtp":         "SMTP邮件配置",
	"system":       "系统相关配置",
}

// CategoryDisplayNames 配置分类显示名称
var CategoryDisplayNames = map[string]string{
	"ui":           "界面设置",
	"sync":         "同步设置",
	"notification": "通知设置",
	"security":     "安全设置",
	"api":          "API设置",
	"oauth":        "OAuth设置",
	"smtp":         "SMTP设置",
	"system":       "系统设置",
}

// CommonCategories 常用配置分类
var CommonCategories = []string{"ui", "sync", "notification"}
