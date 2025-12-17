package oauth2config

import (
	"context"
	"fmt"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/microsoft"
	"google.golang.org/api/gmail/v1"

	"fusionmail/internal/model"
	"fusionmail/internal/repository"
	"fusionmail/pkg/crypto"
	"fusionmail/pkg/logger"
)

// Provider OAuth2配置提供者
type Provider struct {
	clientRepo    repository.OAuth2ClientRepository
	providerRepo  repository.ProviderRepository
	cryptoService *crypto.Service
	logger        *logger.Logger
}

// NewProvider 创建OAuth2配置提供者
func NewProvider(
	clientRepo repository.OAuth2ClientRepository,
	providerRepo repository.ProviderRepository,
	cryptoService *crypto.Service,
	logger *logger.Logger,
) *Provider {
	return &Provider{
		clientRepo:    clientRepo,
		providerRepo:  providerRepo,
		cryptoService: cryptoService,
		logger:        logger,
	}
}

// OAuth2ConfigResult OAuth2配置结果（包含客户端ID用于计数）
type OAuth2ConfigResult struct {
	Config   *oauth2.Config
	ClientID int64
}

// GetOAuth2Config 获取OAuth2配置（使用provider_type，保留用于向后兼容）
// 内部会将 provider_type 转换为 provider name 进行查询
func (p *Provider) GetOAuth2Config(ctx context.Context, providerType int) (*oauth2.Config, error) {
	// 将 provider_type 转换为 provider name
	providerName := providerTypeToName(providerType)
	p.logger.Info("Getting OAuth2 config from database", "provider_type", providerType, "provider_name", providerName)

	return p.GetOAuth2ConfigByName(ctx, providerName)
}

// GetOAuth2ConfigByName 根据提供商名称获取OAuth2配置
func (p *Provider) GetOAuth2ConfigByName(ctx context.Context, providerName string) (*oauth2.Config, error) {
	p.logger.Info("Getting OAuth2 config by provider name", "provider_name", providerName)

	// 查找提供商
	provider, err := p.providerRepo.FindByName(ctx, providerName)
	if err != nil {
		p.logger.Error("Failed to find provider", "provider_name", providerName, "error", err)
		return nil, fmt.Errorf("provider not found: %s", providerName)
	}

	// 获取该提供商的所有客户端
	clients, err := p.clientRepo.FindByProvider(ctx, provider.ID)
	if err != nil {
		p.logger.Error("Failed to find OAuth2 clients", "provider_id", provider.ID, "error", err)
		return nil, fmt.Errorf("failed to find OAuth2 clients: %w", err)
	}

	// 过滤启用的客户端
	var enabledClients []model.OAuth2Client
	for _, client := range clients {
		if client.Enabled {
			enabledClients = append(enabledClients, client)
		}
	}

	if len(enabledClients) == 0 {
		p.logger.Error("No enabled OAuth2 clients found", "provider_name", providerName)
		return nil, fmt.Errorf("no enabled OAuth2 clients for provider: %s", providerName)
	}

	// 选择默认客户端或第一个可用客户端
	var selectedClient *model.OAuth2Client
	for i := range enabledClients {
		if enabledClients[i].IsDefault {
			selectedClient = &enabledClients[i]
			break
		}
	}
	if selectedClient == nil {
		selectedClient = &enabledClients[0]
	}

	p.logger.Info("Selected OAuth2 client for usage",
		"client_id", selectedClient.ID,
		"client_name", selectedClient.Name,
		"is_default", selectedClient.IsDefault,
		"provider_name", providerName)

	// 增加使用计数
	if err := p.clientRepo.IncrementUsage(ctx, selectedClient.ID); err != nil {
		p.logger.Error("Failed to increment OAuth2 client usage",
			"client_id", selectedClient.ID,
			"error", err)
		// 不返回错误，继续执行
	} else {
		p.logger.Info("OAuth2 client usage incremented",
			"client_id", selectedClient.ID,
			"client_name", selectedClient.Name)
	}

	// 解密客户端密钥
	clientSecret, err := p.decryptClientSecret(selectedClient.ClientSecretEncrypted)
	if err != nil {
		p.logger.Error("Failed to decrypt client secret", "client_id", selectedClient.ID, "error", err)
		return nil, fmt.Errorf("failed to decrypt client secret: %w", err)
	}

	// 根据提供商名称创建OAuth2配置
	return p.createOAuth2ConfigByName(providerName, selectedClient.ClientID, clientSecret, selectedClient.RedirectURI)
}

// createOAuth2ConfigByName 根据提供商名称创建OAuth2配置
func (p *Provider) createOAuth2ConfigByName(providerName, clientID, clientSecret, redirectURI string) (*oauth2.Config, error) {
	switch providerName {
	case "gmail":
		return p.createGoogleOAuth2Config(clientID, clientSecret, redirectURI), nil
	case "outlook":
		return p.createMicrosoftOAuth2Config(clientID, clientSecret, redirectURI), nil
	default:
		return nil, fmt.Errorf("unsupported OAuth2 provider: %s", providerName)
	}
}

// providerTypeToName 将 provider_type 转换为 provider name（用于向后兼容）
func providerTypeToName(providerType int) string {
	switch providerType {
	case int(model.ProviderTypeGmail):
		return "gmail"
	case int(model.ProviderTypeOutlook):
		return "outlook"
	case int(model.ProviderTypeIcloud):
		return "icloud"
	case int(model.ProviderTypeQQ):
		return "qq"
	case int(model.ProviderType163):
		return "163"
	default:
		return "generic"
	}
}

// GetOAuth2ConfigByProviderID 根据提供商ID获取OAuth2配置
// 这是推荐的方法，避免硬编码 provider 名称
func (p *Provider) GetOAuth2ConfigByProviderID(ctx context.Context, providerID int64) (*oauth2.Config, error) {
	p.logger.Info("Getting OAuth2 config by provider ID", "provider_id", providerID)

	// 获取提供商信息
	provider, err := p.providerRepo.FindByID(ctx, providerID)
	if err != nil {
		p.logger.Error("Failed to find provider", "provider_id", providerID, "error", err)
		return nil, fmt.Errorf("provider not found: %d", providerID)
	}

	// 使用 provider name 获取配置
	return p.GetOAuth2ConfigByName(ctx, provider.Name)
}

// GetOAuth2ConfigForClient 获取指定客户端的OAuth2配置
func (p *Provider) GetOAuth2ConfigForClient(ctx context.Context, clientID int64) (*oauth2.Config, error) {
	p.logger.Info("Getting OAuth2 config for specific client", "client_id", clientID)

	// 获取客户端配置
	client, err := p.clientRepo.FindByID(ctx, clientID)
	if err != nil {
		p.logger.Error("Failed to find OAuth2 client", "client_id", clientID, "error", err)
		return nil, fmt.Errorf("OAuth2 client not found: %d", clientID)
	}

	// 获取提供商信息
	provider, err := p.providerRepo.FindByID(ctx, client.ProviderID)
	if err != nil {
		p.logger.Error("Failed to find provider", "provider_id", client.ProviderID, "error", err)
		return nil, fmt.Errorf("provider not found: %d", client.ProviderID)
	}

	p.logger.Info("Using specific OAuth2 client",
		"client_id", client.ID,
		"client_name", client.Name,
		"provider_name", provider.Name)

	// 增加使用计数
	if err := p.clientRepo.IncrementUsage(ctx, client.ID); err != nil {
		p.logger.Error("Failed to increment OAuth2 client usage",
			"client_id", client.ID,
			"error", err)
		// 不返回错误，继续执行
	} else {
		p.logger.Info("OAuth2 client usage incremented",
			"client_id", client.ID,
			"client_name", client.Name)
	}

	// 解密客户端密钥
	clientSecret, err := p.decryptClientSecret(client.ClientSecretEncrypted)
	if err != nil {
		p.logger.Error("Failed to decrypt client secret", "client_id", client.ID, "error", err)
		return nil, fmt.Errorf("failed to decrypt client secret: %w", err)
	}

	// 根据提供商名称创建OAuth2配置
	return p.createOAuth2ConfigByName(provider.Name, client.ClientID, clientSecret, client.RedirectURI)
}

// createGoogleOAuth2Config 创建Google OAuth2配置
func (p *Provider) createGoogleOAuth2Config(clientID, clientSecret, redirectURI string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURI,
		Scopes: []string{
			gmail.GmailReadonlyScope,
			gmail.GmailModifyScope,
			"https://www.googleapis.com/auth/userinfo.email",
		},
		Endpoint: google.Endpoint,
	}
}

// createMicrosoftOAuth2Config 创建Microsoft OAuth2配置
func (p *Provider) createMicrosoftOAuth2Config(clientID, clientSecret, redirectURI string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURI,
		Scopes: []string{
			"https://graph.microsoft.com/Mail.ReadWrite",
			"https://graph.microsoft.com/User.Read",
			"offline_access",
		},
		Endpoint: microsoft.AzureADEndpoint("common"),
	}
}

// decryptClientSecret 解密客户端密钥
func (p *Provider) decryptClientSecret(encryptedSecret string) (string, error) {
	if encryptedSecret == "" {
		return "", fmt.Errorf("encrypted secret is empty")
	}

	decryptedData, err := p.cryptoService.Decrypt(encryptedSecret)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt client secret: %w", err)
	}

	return string(decryptedData), nil
}
