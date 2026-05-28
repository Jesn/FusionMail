package service

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"fusionmail/internal/dto"
)

const translationResponseLimit = 2 * 1024 * 1024

// TranslationRequest describes the payload sent to the external translator.
type TranslationRequest struct {
	Text       string `json:"text"`
	SourceLang string `json:"source_lang"`
	TargetLang string `json:"target_lang"`
}

// TranslationResult is the normalized response returned to the frontend.
type TranslationResult struct {
	TranslatedText string `json:"translated_text"`
}

// TranslationService translates text through a provider-backed proxy.
type TranslationService interface {
	Translate(ctx context.Context, req TranslationRequest) (*TranslationResult, error)
}

// TranslationServiceConfig configures the provider-backed translator.
type TranslationServiceConfig struct {
	APIURL     string
	Token      string
	Timeout    time.Duration
	HTTPClient *http.Client
}

type httpTranslationService struct {
	apiURL string
	token  string
	client *http.Client
}

// NewTranslationService creates a translator that keeps provider credentials on the server.
func NewTranslationService(cfg TranslationServiceConfig) TranslationService {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	client := cfg.HTTPClient
	if client == nil {
		client = newTranslationHTTPClient(timeout)
	}

	return &httpTranslationService{
		apiURL: strings.TrimSpace(cfg.APIURL),
		token:  strings.TrimSpace(cfg.Token),
		client: client,
	}
}

func newTranslationHTTPClient(timeout time.Duration) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.ForceAttemptHTTP2 = false
	transport.TLSNextProto = map[string]func(string, *tls.Conn) http.RoundTripper{}

	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}
}

func (s *httpTranslationService) Translate(ctx context.Context, req TranslationRequest) (*TranslationResult, error) {
	text := strings.TrimSpace(req.Text)
	if text == "" {
		return nil, dto.NewAPIErrorWithMessage(dto.ErrInvalidRequest, "翻译文本不能为空")
	}
	if s.apiURL == "" || s.token == "" {
		return nil, dto.NewAPIErrorWithMessage(dto.ErrOperationFailed, "翻译服务未配置")
	}

	endpoint, err := url.Parse(s.apiURL)
	if err != nil || endpoint.Scheme == "" || endpoint.Host == "" {
		return nil, dto.NewAPIErrorWithMessage(dto.ErrOperationFailed, "翻译服务配置无效")
	}
	query := endpoint.Query()
	query.Set("token", s.token)
	endpoint.RawQuery = query.Encode()

	sourceLang := strings.TrimSpace(req.SourceLang)
	if sourceLang == "" {
		sourceLang = "auto"
	}
	targetLang := strings.TrimSpace(req.TargetLang)
	if targetLang == "" {
		targetLang = "ZH"
	}

	payload, err := json.Marshal(TranslationRequest{
		Text:       text,
		SourceLang: sourceLang,
		TargetLang: targetLang,
	})
	if err != nil {
		return nil, dto.NewAPIErrorWithMessage(dto.ErrOperationFailed, "翻译请求创建失败")
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), bytes.NewReader(payload))
	if err != nil {
		return nil, dto.NewAPIErrorWithMessage(dto.ErrOperationFailed, "翻译请求创建失败")
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("User-Agent", "FusionMail/1.0")

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return nil, dto.NewAPIErrorWithMessage(dto.ErrOperationFailed, "翻译服务请求失败")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, translationResponseLimit))
	if err != nil {
		return nil, dto.NewAPIErrorWithMessage(dto.ErrOperationFailed, "翻译响应读取失败")
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, dto.NewAPIErrorWithMessage(dto.ErrOperationFailed, "翻译服务请求失败")
	}

	translatedText, err := parseProviderTranslationResponse(body)
	if err != nil {
		return nil, dto.NewAPIErrorWithMessage(dto.ErrOperationFailed, "翻译响应格式错误")
	}

	return &TranslationResult{TranslatedText: translatedText}, nil
}

func parseProviderTranslationResponse(body []byte) (string, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return "", fmt.Errorf("decode translation response: %w", err)
	}

	translatedText := extractTranslatedText(raw)
	if translatedText == "" {
		return "", fmt.Errorf("translation response missing translated text")
	}
	return translatedText, nil
}

func extractTranslatedText(raw map[string]json.RawMessage) string {
	for _, key := range []string{"translated_text", "translatedText", "translation", "text"} {
		value, ok := raw[key]
		if !ok {
			continue
		}
		var translatedText string
		if err := json.Unmarshal(value, &translatedText); err == nil {
			if trimmed := strings.TrimSpace(translatedText); trimmed != "" {
				return trimmed
			}
		}
	}

	if value, ok := raw["data"]; ok {
		var translatedText string
		if err := json.Unmarshal(value, &translatedText); err == nil {
			if trimmed := strings.TrimSpace(translatedText); trimmed != "" {
				return trimmed
			}
		}

		var nested map[string]json.RawMessage
		if err := json.Unmarshal(value, &nested); err == nil {
			return extractTranslatedText(nested)
		}
	}

	return ""
}
