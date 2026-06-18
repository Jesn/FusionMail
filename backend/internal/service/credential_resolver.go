package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"fusionmail/internal/adapter"
	"fusionmail/internal/model"
	"fusionmail/pkg/crypto"
	"fusionmail/pkg/oauth2config"
)

type credentialDecryptor interface {
	Decrypt(ciphertext string) (string, error)
}

type CredentialResolver struct {
	cryptoService        *crypto.Service
	encryptor            credentialDecryptor
	oauth2ConfigProvider *oauth2config.Provider
}

func NewCredentialResolver(cryptoService *crypto.Service, oauth2ConfigProvider *oauth2config.Provider) *CredentialResolver {
	return &CredentialResolver{
		cryptoService:        cryptoService,
		oauth2ConfigProvider: oauth2ConfigProvider,
	}
}

func NewCredentialResolverWithEncryptor(encryptor credentialDecryptor, oauth2ConfigProvider *oauth2config.Provider) *CredentialResolver {
	return &CredentialResolver{
		encryptor:            encryptor,
		oauth2ConfigProvider: oauth2ConfigProvider,
	}
}

func (r *CredentialResolver) Resolve(account *model.EmailAccount) (*adapter.Credentials, error) {
	if account == nil {
		return nil, fmt.Errorf("account is required")
	}

	decryptedData, err := r.decrypt(account.EncryptedCredentials)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt credentials: %w", err)
	}

	authType := resolveAccountAuthType(account, decryptedData)
	credentials := &adapter.Credentials{
		Email:    account.Email,
		AuthType: authType,
	}

	switch authType {
	case "quick":
		if err := applyQuickCredentials(credentials, decryptedData); err != nil {
			return nil, err
		}
	case "oauth2":
		if err := r.applyOAuth2Credentials(account, credentials, decryptedData); err != nil {
			return nil, err
		}
	default:
		credentials.Password = string(decryptedData)
	}

	applyProtocolServerConfig(account, credentials)
	if credentials.Host == "mail.linuxdo.org" {
		credentials.Host = "mail.linux.do"
	}

	protocol := account.GetProtocol()
	if protocol == "imap" || protocol == "pop3" {
		if credentials.Host == "" || credentials.Port == 0 {
			return nil, fmt.Errorf("server configuration missing: host=%s, port=%d (provider=%s, protocol=%s)",
				credentials.Host, credentials.Port, account.GetProviderName(), protocol)
		}
	}

	return credentials, nil
}

func (r *CredentialResolver) decrypt(encrypted string) ([]byte, error) {
	if r == nil {
		return nil, fmt.Errorf("credential resolver is nil")
	}
	if r.cryptoService != nil {
		return r.cryptoService.Decrypt(encrypted)
	}
	if r.encryptor != nil {
		decrypted, err := r.encryptor.Decrypt(encrypted)
		if err != nil {
			return nil, err
		}
		return []byte(decrypted), nil
	}
	return nil, fmt.Errorf("credential decryptor is not configured")
}

func resolveAccountAuthType(account *model.EmailAccount, decryptedData []byte) string {
	var credAuthType struct {
		AuthType string `json:"auth_type"`
	}
	authType := account.GetAuthType()
	if json.Unmarshal(decryptedData, &credAuthType) == nil && credAuthType.AuthType != "" {
		authType = credAuthType.AuthType
	}
	return authType
}

func applyQuickCredentials(credentials *adapter.Credentials, decryptedData []byte) error {
	var quickCreds struct {
		RefreshToken string `json:"refresh_token"`
		ClientID     string `json:"client_id"`
	}

	if err := json.Unmarshal(decryptedData, &quickCreds); err != nil {
		return fmt.Errorf("failed to parse quick credentials: %w", err)
	}

	credentials.RefreshToken = quickCreds.RefreshToken
	credentials.ClientID = quickCreds.ClientID
	credentials.ClientSecret = ""
	return nil
}

func (r *CredentialResolver) applyOAuth2Credentials(account *model.EmailAccount, credentials *adapter.Credentials, decryptedData []byte) error {
	var oauthCreds struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		TokenExpiry  any    `json:"token_expiry"`
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
	}

	if err := json.Unmarshal(decryptedData, &oauthCreds); err != nil {
		return fmt.Errorf("failed to parse OAuth2 credentials: %w", err)
	}

	credentials.AccessToken = oauthCreds.AccessToken
	credentials.RefreshToken = oauthCreds.RefreshToken
	credentials.ClientID = oauthCreds.ClientID
	credentials.ClientSecret = oauthCreds.ClientSecret
	credentials.TokenExpiry = parseTokenExpiry(oauthCreds.TokenExpiry)

	if credentials.ClientID != "" && credentials.ClientSecret != "" {
		return nil
	}
	if r.oauth2ConfigProvider == nil {
		return nil
	}

	if account.ProviderID > 0 {
		oauth2Config, err := r.oauth2ConfigProvider.GetOAuth2ConfigByProviderID(context.Background(), account.ProviderID)
		if err != nil {
			return fmt.Errorf("failed to get OAuth2 config for provider_id %d: %w", account.ProviderID, err)
		}
		credentials.ClientID = oauth2Config.ClientID
		credentials.ClientSecret = oauth2Config.ClientSecret
		return nil
	}

	providerName := account.GetProviderName()
	oauth2Config, err := r.oauth2ConfigProvider.GetOAuth2ConfigByName(context.Background(), providerName)
	if err != nil {
		return fmt.Errorf("failed to get OAuth2 config for provider %s: %w", providerName, err)
	}
	credentials.ClientID = oauth2Config.ClientID
	credentials.ClientSecret = oauth2Config.ClientSecret
	return nil
}

func parseTokenExpiry(value any) time.Time {
	switch v := value.(type) {
	case string:
		if v == "" {
			return time.Time{}
		}
		parsed, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return time.Time{}
		}
		return parsed
	default:
		return time.Time{}
	}
}

func applyProtocolServerConfig(account *model.EmailAccount, credentials *adapter.Credentials) {
	switch account.GetProtocol() {
	case "imap":
		host, port, encryption := account.GetIMAPConfig()
		applyServerConfig(credentials, host, port, encryption)
	case "pop3":
		host, port, encryption := account.GetPOP3Config()
		applyServerConfig(credentials, host, port, encryption)
	}
}

func applyServerConfig(credentials *adapter.Credentials, host string, port int, encryption string) {
	credentials.Host = host
	credentials.Port = port

	switch encryption {
	case "ssl", "":
		credentials.TLS = true
	case "starttls":
		credentials.StartTLS = true
	case "none":
		credentials.TLS = false
		credentials.StartTLS = false
	default:
		credentials.TLS = true
	}
}
