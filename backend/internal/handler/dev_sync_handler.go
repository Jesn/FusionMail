package handler

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	"fusionmail/internal/dto"
	"fusionmail/internal/dto/response"
	"fusionmail/internal/model"
	"fusionmail/internal/repository"

	"github.com/gin-gonic/gin"
)

// DevSyncHandler 开发环境数据同步处理器。
//
// 依赖通过构造函数注入：此前它直接调用 database.GetDB() 并在 handler 内
// 用 gorm 拼查询、写库，是项目里唯一一处 handler 直连 ORM 的越层点。
type DevSyncHandler struct {
	oauth2Repo   repository.OAuth2ClientRepository
	providerRepo repository.ProviderRepository
}

// NewDevSyncHandler 创建设备同步处理器
func NewDevSyncHandler(oauth2Repo repository.OAuth2ClientRepository, providerRepo repository.ProviderRepository) *DevSyncHandler {
	return &DevSyncHandler{oauth2Repo: oauth2Repo, providerRepo: providerRepo}
}

// SyncFromEnv 从环境变量同步数据到数据库
// 注意：此端点仅用于开发环境，生产环境应删除
// @Summary 从环境变量同步数据到数据库
// @Description 同步 OAuth2 客户端配置从环境变量
// @Tags 开发工具
// @Accept json
// @Produce json
// @Success 200 {object} dto.Response{data=response.DevSyncResponse}
// @Router /api/v1/dev/sync-from-env [post]
func (h *DevSyncHandler) SyncFromEnv(c *gin.Context) {
	log.Println("开始从环境变量同步数据...")

	var results []response.DevSyncResult

	ctx := c.Request.Context()

	// 同步 Gmail 配置
	if gmailClientID := os.Getenv("GMAIL_CLIENT_ID"); gmailClientID != "" {
		if gmailClientSecret := os.Getenv("GMAIL_CLIENT_SECRET"); gmailClientSecret != "" {
			// Gmail provider ID = 1 (根据提供商表中的ID)
			result := h.syncOAuth2ClientWithProviderID(ctx, 1, "默认配��", gmailClientID, gmailClientSecret)
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
				Provider: "gmail",
				Name:     "默认配置",
				ClientID: gmailClientID,
				Status:   "failed",
				Message:  "未找到 GMAIL_CLIENT_SECRET",
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
			result := h.syncOAuth2ClientWithProviderID(ctx, 2, "默认配置", microsoftClientID, microsoftClientSecret)
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
				Provider: "outlook",
				Name:     "默认配置",
				ClientID: microsoftClientID,
				Status:   "failed",
				Message:  "未找到 MICROSOFT_CLIENT_SECRET",
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
	dto.SuccessResponse(c, response.DevSyncResponse{
		Message: "同步完成",
		Results: results,
	})
}

// syncOAuth2Client 同步单个 OAuth2 客户端配置
func (h *DevSyncHandler) syncOAuth2Client(ctx context.Context, providerName, name, clientID, clientSecret string) struct {
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
	existingClientPtr, findErr := h.oauth2Repo.FindByProviderAndName(ctx, int64(MapProviderNameToType(providerName)), name)
	existingClient := model.OAuth2Client{}
	if existingClientPtr != nil {
		existingClient = *existingClientPtr
	}

	if findErr == nil {
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

		if err := h.oauth2Repo.Update(ctx, &existingClient); err != nil {
			result.Message = fmt.Sprintf("更新配置失败: %v", err)
			return result
		}

		result.Status = "updated"
		result.Message = "已更新现有配置"
		result.ClientIDDB = existingClient.ID
		fmt.Printf("  ✅ 已更新现有配置 (ID: %d)\n", existingClient.ID)
	} else if errors.Is(findErr, repository.ErrNotFound) {
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

		if err := h.oauth2Repo.Create(ctx, &client); err != nil {
			result.Message = fmt.Sprintf("创建配置失败: %v", err)
			return result
		}

		result.Status = "created"
		result.Message = "已创建新配置"
		result.ClientIDDB = client.ID
		fmt.Printf("  ✅ 已创建新配置 (ID: %d)\n", client.ID)
	} else {
		result.Message = fmt.Sprintf("查询配置失败: %v", findErr)
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
func (h *DevSyncHandler) syncOAuth2ClientWithProviderID(ctx context.Context, providerID int64, name, clientID, clientSecret string) struct {
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
	existingClientPtr, findErr := h.oauth2Repo.FindByProviderAndName(ctx, providerID, name)
	existingClient := model.OAuth2Client{}
	if existingClientPtr != nil {
		existingClient = *existingClientPtr
	}

	if findErr == nil {
		// 已存在，更新
		fmt.Println("  🔄 更新现有配置...")
		existingClient.ProviderID = providerID
		existingClient.ClientID = clientID
		// 需要根据providerID获取provider名称来生成回调URL
		if provider, err := h.providerRepo.FindByID(ctx, providerID); err == nil && provider != nil {
			existingClient.RedirectURI = fmt.Sprintf("http://localhost:3333/api/v1/auth/%s/callback", mapProviderToAuth(provider.Name))
		}
		existingClient.Enabled = true

		if err := existingClient.SetClientSecret(clientSecret); err != nil {
			result.Message = fmt.Sprintf("加密客户端密钥失败: %v", err)
			return result
		}

		if err := h.oauth2Repo.Update(ctx, &existingClient); err != nil {
			result.Message = fmt.Sprintf("更新配置失败: %v", err)
			return result
		}

		result.Status = "updated"
		result.Message = "已更新现有配置"
		result.ClientIDDB = existingClient.ID
		fmt.Printf("  ✅ 已更新现有配置 (ID: %d)\n", existingClient.ID)
	} else if errors.Is(findErr, repository.ErrNotFound) {
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
		if provider, err := h.providerRepo.FindByID(ctx, providerID); err == nil && provider != nil {
			client.RedirectURI = fmt.Sprintf("http://localhost:3333/api/v1/auth/%s/callback", mapProviderToAuth(provider.Name))
		}

		if err := client.SetClientSecret(clientSecret); err != nil {
			result.Message = fmt.Sprintf("加密客户端密钥失败: %v", err)
			return result
		}

		if err := h.oauth2Repo.Create(ctx, &client); err != nil {
			result.Message = fmt.Sprintf("创建配置失败: %v", err)
			return result
		}

		result.Status = "created"
		result.Message = "已创建新配置"
		result.ClientIDDB = client.ID
		fmt.Printf("  ✅ 已创建新配置 (ID: %d)\n", client.ID)
	} else {
		result.Message = fmt.Sprintf("查询配置失败: %v", findErr)
		return result
	}

	return result
}
