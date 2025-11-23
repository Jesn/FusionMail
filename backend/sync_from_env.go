package main

import (
	"fmt"
	"log"
	"os"

	"fusionmail/config"
	"fusionmail/internal/model"
	"fusionmail/pkg/database"

	"gorm.io/gorm"
)

func main() {
	fmt.Println("=== 环境变量数据同步到数据库 ===\n")

	// 加载配置
	cfg := config.Load()

	// 初始化数据库连接
	if err := database.Initialize(&cfg.Database); err != nil {
		log.Fatalf("❌ 初始化数据库失败: %v", err)
	}
	fmt.Println("✅ 数据库连接成功\n")

	// 同步 OAuth2 客户端数据
	syncOAuth2Clients()

	// 结束
	fmt.Println("\n=== 同步完成 ===")
}

// syncOAuth2Clients 同步 OAuth2 客户端配置
func syncOAuth2Clients() {
	fmt.Println("📝 开始同步 OAuth2 客户端配置...")

	// 从环境变量获取配置
	gmailClientID := os.Getenv("GMAIL_CLIENT_ID")
	gmailClientSecret := os.Getenv("GMAIL_CLIENT_SECRET")
	microsoftClientID := os.Getenv("MICROSOFT_CLIENT_ID")
	microsoftClientSecret := os.Getenv("MICROSOFT_CLIENT_SECRET")

	// 检查是否有 Gmail 配置
	if gmailClientID != "" && gmailClientSecret != "" {
		if err := upsertOAuth2Client("gmail", "默认配置", gmailClientID, gmailClientSecret); err != nil {
			fmt.Printf("⚠️  同步 Gmail 配置失败: %v\n", err)
		}
	} else {
		fmt.Println("⚠️  未找到 Gmail OAuth2 配置 (GMAIL_CLIENT_ID 或 GMAIL_CLIENT_SECRET)")
	}

	// 检查是否有 Microsoft 配置
	if microsoftClientID != "" && microsoftClientSecret != "" {
		if err := upsertOAuth2Client("outlook", "默认配置", microsoftClientID, microsoftClientSecret); err != nil {
			fmt.Printf("⚠️  同步 Microsoft 配置失败: %v\n", err)
		}
	} else {
		fmt.Println("⚠️  未找到 Microsoft OAuth2 配置 (MICROSOFT_CLIENT_ID 或 MICROSOFT_CLIENT_SECRET)")
	}

	fmt.Println("✅ OAuth2 客户端配置同步完成")
}

// upsertOAuth2Client 插入或更新 OAuth2 客户端配置
func upsertOAuth2Client(providerName, name, clientID, clientSecret string) error {
	db := database.GetDB()

	fmt.Printf("  处理 %s 配置: %s\n", providerName, name)
	fmt.Printf("    ClientID: %s\n", clientID)
	fmt.Printf("    RedirectURI: http://localhost:3333/api/v1/auth/%s/callback\n",
		mapProviderToAuth(providerName))

	// 检查是否已存在
	var existingClient model.OAuth2Client
	result := db.Where("provider_name = ? AND name = ?", providerName, name).First(&existingClient)

	if result.Error == nil {
		// 已存在，更新
		fmt.Println("  🔄 更新现有配置...")
		existingClient.ClientID = clientID
		existingClient.RedirectURI = fmt.Sprintf("http://localhost:3333/api/v1/auth/%s/callback", mapProviderToAuth(providerName))
		existingClient.Enabled = true

		if err := existingClient.SetClientSecret(clientSecret); err != nil {
			return fmt.Errorf("加密客户端密钥失败: %w", err)
		}

		if err := db.Save(&existingClient).Error; err != nil {
			return fmt.Errorf("更新配置失败: %w", err)
		}

		fmt.Printf("  ✅ 已更新现有配置 (ID: %d)\n", existingClient.ID)
	} else if result.Error == gorm.ErrRecordNotFound {
		// 不存在，创建新记录
		fmt.Println("  ➕ 创建新配置...")
		client := model.OAuth2Client{
			ProviderName: providerName,
			Name:         name,
			ClientID:     clientID,
			RedirectURI:  fmt.Sprintf("http://localhost:3333/api/v1/auth/%s/callback", mapProviderToAuth(providerName)),
			Enabled:      true,
			IsDefault:    true,
			QuotaDaily:   100,
			QuotaMonthly: 2000,
		}

		if err := client.SetClientSecret(clientSecret); err != nil {
			return fmt.Errorf("加密客户端密钥失败: %w", err)
		}

		if err := db.Create(&client).Error; err != nil {
			return fmt.Errorf("创建配置失败: %w", err)
		}

		fmt.Printf("  ✅ 已创建新配置 (ID: %d)\n", client.ID)
	} else {
		return fmt.Errorf("查询配置失败: %w", result.Error)
	}

	return nil
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
