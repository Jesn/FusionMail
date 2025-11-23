package model

// ProviderType 邮箱提供商类型枚举
// 所有使用 ProviderType 的模型都应该引用这个文件
type ProviderType int

const (
	ProviderTypeGmail    ProviderType = 1 // Gmail
	ProviderTypeOutlook  ProviderType = 2 // Outlook / Hotmail
	ProviderTypeIcloud   ProviderType = 3 // iCloud Mail
	ProviderTypeQQ       ProviderType = 4 // QQ 邮箱
	ProviderType163      ProviderType = 5 // 163 邮箱
	ProviderTypeGeneric  ProviderType = 6 // 通用邮箱
)

// String 返回 ProviderType 的字符串表示
func (p ProviderType) String() string {
	switch p {
	case ProviderTypeGmail:
		return "gmail"
	case ProviderTypeOutlook:
		return "outlook"
	case ProviderTypeIcloud:
		return "icloud"
	case ProviderTypeQQ:
		return "qq"
	case ProviderType163:
		return "163"
	case ProviderTypeGeneric:
		return "generic"
	default:
		return "unknown"
	}
}

// mapProviderNameToType 将提供商名称映射到 provider_type 枚举值
func mapProviderNameToType(providerName string) int {
	switch providerName {
	case "gmail":
		return int(ProviderTypeGmail)
	case "outlook":
		return int(ProviderTypeOutlook)
	case "icloud":
		return int(ProviderTypeIcloud)
	case "qq":
		return int(ProviderTypeQQ)
	case "163":
		return int(ProviderType163)
	case "generic":
		return int(ProviderTypeGeneric)
	default:
		return 0 // 未知类型
	}
}
