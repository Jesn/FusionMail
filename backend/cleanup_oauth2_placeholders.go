package main

import (
	"fmt"
	"log"

	"fusionmail/config"
	"fusionmail/pkg/database"

	"gorm.io/gorm"
)

func main() {
	// 加载配置
	cfg := config.Load()

	// 初始化数据库连接
	if err := database.Initialize(&cfg.Database); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	db := database.GetDB()

	// 查找占位符数据
	var clients []struct {
		ID            int64
		ProviderName  string
		Name          string
		ClientID      string
		RedirectURI   string
		Enabled       bool
		IsDefault     bool
		CreatedAt     string
	}

	// 查询占位符数据
	result := db.Table("oauth2_clients").
		Select("id, provider_name, name, client_id, redirect_uri, enabled, is_default, created_at").
		Where("client_id LIKE ?", "your-%-client-id").
		Scan(&clients)

	if result.Error != nil && result.Error != gorm.ErrRecordNotFound {
		log.Fatalf("Failed to query placeholder data: %v", result.Error)
	}

	if len(clients) == 0 {
		log.Println("✅ No placeholder OAuth2 clients found.")
		return
	}

	// 显示要删除的数据
	log.Println("Found the following placeholder OAuth2 clients:")
	for _, client := range clients {
		log.Printf("  ID: %d, Provider: %s, Name: %s, ClientID: %s",
			client.ID, client.ProviderName, client.Name, client.ClientID)
	}

	// 删除占位符数据
	deleteResult := db.Where("client_id LIKE ?", "your-%-client-id").Delete(&struct{}{})
	if deleteResult.Error != nil {
		log.Fatalf("Failed to delete placeholder data: %v", deleteResult.Error)
	}

	log.Printf("✅ Successfully deleted %d placeholder OAuth2 client(s)", len(clients))
}
