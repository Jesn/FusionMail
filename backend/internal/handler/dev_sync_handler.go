package handler

import (
	"fmt"
	"log"
	"os"

	"fusionmail/internal/dto"
	"fusionmail/internal/model"
	"fusionmail/pkg/database"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// DevSyncHandler 开发环境数据同步处理器
type DevSyncHandler struct{}

// NewDevSyncHandler 创建设备同步处理器
func NewDevSyncHandler() *DevSyncHandler {
	return &DevSyncHandler{}
}

// SyncFromEnv 从环境变量同步数据到数据库
// 注意：此端点仅用于开发环境，生产环境应删除
// @Summary 从环境变量同步数据到数据库
// @Description 同步 OAuth2 客户端配置从环境变量
// @Tags 开发工具
// @Accept json
// @Produce json
// @Success 200 {object} dto.Response
// @Router /api/v1/dev/sync-from-env [post]
func (h *DevSyncHandler) SyncFromEnv(c *gin.Context) {
	log.Println("开始从环境变量同步数据...")

	var results []struct {
		Provider   string `json:"provider"`
		Name       string `json:"name"`
		ClientID   string `json:"client_id"`
		Status     string `json:"status"`
		Message    string `json:"message"`
		ClientIDDB int64  `json:"client_id_db"`
	}

	db := database.GetDB()

	// 同步 Gmail 配置
	if gmailClientID := os.Getenv("GMAIL_CLIENT_ID"); gmailClientID != "" {
		if gmailClientSecret := os.Getenv("GMAIL_CLIENT_SECRET"); gmailClientSecret != "" {
			// Gmail provider ID = 1 (根据提供商表中的ID)
			result := syncOAuth2ClientWithProviderID(db, 1, "默认配置", gmailClientID, gmailClientSecret)
			results = append(results, result)
		} else {
			results = append(results, struct {
				Provider   string `json:"provider"`
				Name       string `json:"name"`
				ClientID   string `json:"client_id"`
				Status     string `json:"status"`
				Message    string `json:"message"`
				ClientIDDB int64  `json:"client_id_db"`
			}{
				Provider:   "gmail",
				Name:       "默认配置",
				ClientID:   gmailClientID,
				Status:     "failed",
				Message:    "未找到 GMAIL_CLIENT_SECRET",
			})
		}
	} else {
		results = append(results, struct {
			Provider   string `json:"provider"`
			Name       string `json:"name"`
			ClientID   string `json:"client_id"`
			Status     string `json:"status"`
			Message    string `json:"message"`
			ClientIDDB int64  `json:"client_id_db"`
		}{
			Provider: "gmail",
			Name:     "默认配置",
			Status:   "skipped",
			Message:  "未找到 GMAIL_CLIENT_ID",
		})
	}

	// 同步 Microsoft 配置
	if microsoftClientID := os.Getenv("MICROSOFT_CLIENT_ID"); microsoftClientID != "" {
		if microsoftClientSecret := os.Getenv("MICROSOFT_CLIENT_SECRET"); microsoftClientSecret != "" {
			// Outlook provider ID = 2
			result := syncOAuth2ClientWithProviderID(db, 2, "默认配置", microsoftClientID, microsoftClientSecret)
			results = append(results, result)
		} else {
			results = append(results, struct {
				Provider   string `json:"provider"`
				Name       string `json:"name"`
				ClientID   string `json:"client_id"`
				Status     string `json:"status"`
				Message    string `json:"message"`
				ClientIDDB int64  `json:"client_id_db"`
			}{
				Provider:   "outlook",
				Name:       "默认配置",
				ClientID:   microsoftClientID,
				Status:     "failed",
				Message:    "未找到 MICROSOFT_CLIENT_SECRET",
			})
		}
	} else {
		results = append(results, struct {
			Provider   string `json:"provider"`
			Name       string `json:"name"`
			ClientID   string `json:"client_id"`
			Status     string `json:"status"`
			Message    string `json:"message"`
			ClientIDDB int64  `json:"client_id_db"`
		}{
			Provider: "outlook",
			Name:     "默认配置",
			Status:   "skipped",
			Message:  "未找到 MICROSOFT_CLIENT_ID",
		})
	}

	log.Printf("同步完成，共处理 %d 项配置", len(results))
	dto.SuccessResponse(c, gin.H{
		"message": "同步完成",
		"results": results,
	})
}

// syncOAuth2Client 同步单个 OAuth2 客户端配置
func syncOAuth2Client(db *gorm.DB, providerName, name, clientID, clientSecret string) struct {
	Provider   string `json:"provider"`
	Name       string `json:"name"`
	ClientID   string `json:"client_id"`
	Status     string `json:"status"`
	Message    string `json:"message"`
	ClientIDDB int64  `json:"client_id_db"`
} {
	result := struct {
		Provider   string `json:"provider"`
		Name       string `json:"name"`
		ClientID   string `json:"client_id"`
		Status     string `json:"status"`
		Message    string `json:"message"`
		ClientIDDB int64  `json:"client_id_db"`
	}{
		Provider: providerName,
		Name:     name,
		ClientID: clientID,
		Status:   "failed",
	}

	fmt.Printf("  处理 %s 配置: %s\n", providerName, name)
	fmt.Printf("    ClientID: %s\n", clientID)

	// 检查是否已存在（使用 provider_id 进行关联）
	var existingClient model.OAuth2Client
	result_db := db.Where("provider_id = ? AND name = ?", MapProviderNameToType(providerName), name).First(&existingClient)

	if result_db.Error == nil {
		// 已存在，更新
		fmt.Println("  🔄 更新现有配置...")
		existingClient.ProviderID = int64(MapProviderNameToType(providerName)) // 使用 provider_id 字段
		existingClient.ClientID = clientID
		existingClient.RedirectURI = fmt.Sprintf("http://localhost:3333/api/v1/auth/%s/callback", mapProviderToAuth(providerName))
		existingClient.Enabled = true

		if err := existingClient.SetClientSecret(clientSecret); err != nil {
			result.Message = fmt.Sprintf("加密客户端密钥失败: %v", err)
			return result
		}

		if err := db.Save(&existingClient).Error; err != nil {
			result.Message = fmt.Sprintf("更新配置失败: %v", err)
			return result
		}

		result.Status = "updated"
		result.Message = "已更新现有配置"
		result.ClientIDDB = existingClient.ID
		fmt.Printf("  ✅ 已更新现有配置 (ID: %d)\n", existingClient.ID)
	} else if result_db.Error == gorm.ErrRecordNotFound {
		// 不存在，创建新记录
		fmt.Println("  ➕ 创建新配置...")
		client := model.OAuth2Client{
			ProviderID:   int64(MapProviderNameToType(providerName)), // 使用 provider_id 字段
			Name:         name,
			ClientID:     clientID,
			RedirectURI:  fmt.Sprintf("http://localhost:3333/api/v1/auth/%s/callback", mapProviderToAuth(providerName)),
			Enabled:      true,
			IsDefault:    true,
			QuotaDaily:   100,
			QuotaMonthly: 2000,
		}

		if err := client.SetClientSecret(clientSecret); err != nil {
			result.Message = fmt.Sprintf("加密客户端密钥失败: %v", err)
			return result
		}

		if err := db.Create(&client).Error; err != nil {
			result.Message = fmt.Sprintf("创建配置失败: %v", err)
			return result
		}

		result.Status = "created"
		result.Message = "已创建新配置"
		result.ClientIDDB = client.ID
		fmt.Printf("  ✅ 已创建新配置 (ID: %d)\n", client.ID)
	} else {
		result.Message = fmt.Sprintf("查询配置失败: %v", result_db.Error)
		return result
	}

	return result
}

// mapProviderToAuth 将提供商名称映射到认证路径
func mapProviderToAuth(providerName string) string {
	switch providerName {
	case "gmail":
		return "google"
	case "outlook":
		return "microsoft"
	default:
		return providerName
	}
}

// MapProviderNameToType 将提供商名称映射到 provider_type 枚举值
func MapProviderNameToType(providerName string) int {
	switch providerName {
	case "gmail":
		return 1 // ProviderTypeGmail
	case "outlook":
		return 2 // ProviderTypeOutlook
	case "icloud":
		return 3 // ProviderTypeIcloud
	case "qq":
		return 4 // ProviderTypeQQ
	case "163":
		return 5 // ProviderType163
	case "generic":
		return 6 // ProviderTypeGeneric
	default:
		return 0 // 未知类型
	}
}

// syncOAuth2ClientWithProviderID 同步单个 OAuth2 客户端配置（使用provider_id）
func syncOAuth2ClientWithProviderID(db *gorm.DB, providerID int64, name, clientID, clientSecret string) struct {
	Provider   string `json:"provider"`
	Name       string `json:"name"`
	ClientID   string `json:"client_id"`
	Status     string `json:"status"`
	Message    string `json:"message"`
	ClientIDDB int64  `json:"client_id_db"`
} {
	result := struct {
		Provider   string `json:"provider"`
		Name       string `json:"name"`
		ClientID   string `json:"client_id"`
		Status     string `json:"status"`
		Message    string `json:"message"`
		ClientIDDB int64  `json:"client_id_db"`
	}{
		Provider: fmt.Sprintf("provider_id_%d", providerID),
		Name:     name,
		ClientID: clientID,
		Status:   "failed",
	}

	fmt.Printf("  处理 provider_id=%d 配置: %s\n", providerID, name)
	fmt.Printf("    ClientID: %s\n", clientID)

	// 检查是否已存在（使用 provider_id 进行关联）
	var existingClient model.OAuth2Client
	result_db := db.Where("provider_id = ? AND name = ?", providerID, name).First(&existingClient)

	if result_db.Error == nil {
		// 已存在，更新
		fmt.Println("  🔄 更新现有配置...")
		existingClient.ProviderID = providerID
		existingClient.ClientID = clientID
		// 需要根据providerID获取provider名称来生成回调URL
		var provider model.Provider
		if err := db.First(&provider, providerID).Error; err == nil {
			existingClient.RedirectURI = fmt.Sprintf("http://localhost:3333/api/v1/auth/%s/callback", mapProviderToAuth(provider.Name))
		}
		existingClient.Enabled = true

		if err := existingClient.SetClientSecret(clientSecret); err != nil {
			result.Message = fmt.Sprintf("加密客户端密钥失败: %v", err)
			return result
		}

		if err := db.Save(&existingClient).Error; err != nil {
			result.Message = fmt.Sprintf("更新配置失败: %v", err)
			return result
		}

		result.Status = "updated"
		result.Message = "已更新现有配置"
		result.ClientIDDB = existingClient.ID
		fmt.Printf("  ✅ 已更新现有配置 (ID: %d)\n", existingClient.ID)
	} else if result_db.Error == gorm.ErrRecordNotFound {
		// 不存在，创建新记录
		fmt.Println("  ➕ 创建新配置...")
		client := model.OAuth2Client{
			ProviderID:   providerID,
			Name:         name,
			ClientID:     clientID,
			Enabled:      true,
			IsDefault:    true,
			QuotaDaily:   100,
			QuotaMonthly: 2000,
		}

		// 需要根据providerID获取provider名称来生成回调URL
		var provider model.Provider
		if err := db.First(&provider, providerID).Error; err == nil {
			client.RedirectURI = fmt.Sprintf("http://localhost:3333/api/v1/auth/%s/callback", mapProviderToAuth(provider.Name))
		}

		if err := client.SetClientSecret(clientSecret); err != nil {
			result.Message = fmt.Sprintf("加密客户端密钥失败: %v", err)
			return result
		}

		if err := db.Create(&client).Error; err != nil {
			result.Message = fmt.Sprintf("创建配置失败: %v", err)
			return result
		}

		result.Status = "created"
		result.Message = "已创建新配置"
		result.ClientIDDB = client.ID
		fmt.Printf("  ✅ 已创建新配置 (ID: %d)\n", client.ID)
	} else {
		result.Message = fmt.Sprintf("查询配置失败: %v", result_db.Error)
		return result
	}

	return result
}
