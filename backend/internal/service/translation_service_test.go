package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"fusionmail/internal/dto"
)

func TestTranslationServiceTranslatePostsToProvider(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST request, got %s", r.Method)
		}
		if got := r.URL.Query().Get("token"); got != "test-token" {
			t.Fatalf("expected token query test-token, got %q", got)
		}

		var request TranslationRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("failed to decode provider request: %v", err)
		}
		if request.Text != "Hello world" {
			t.Fatalf("expected text %q, got %q", "Hello world", request.Text)
		}
		if request.SourceLang != "auto" {
			t.Fatalf("expected source_lang auto, got %q", request.SourceLang)
		}
		if request.TargetLang != "ZH" {
			t.Fatalf("expected target_lang ZH, got %q", request.TargetLang)
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]string{
			"translated_text": "你好，世界",
		}); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	}))
	t.Cleanup(server.Close)

	translator := NewTranslationService(TranslationServiceConfig{
		APIURL:  server.URL,
		Token:   "test-token",
		Timeout: time.Second,
	})

	result, err := translator.Translate(context.Background(), TranslationRequest{
		Text:       "Hello world",
		SourceLang: "auto",
		TargetLang: "ZH",
	})
	if err != nil {
		t.Fatalf("Translate returned error: %v", err)
	}
	if result.TranslatedText != "你好，世界" {
		t.Fatalf("expected translated text, got %q", result.TranslatedText)
	}
}

func TestTranslationServiceDoesNotSendCustomProviderHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Content-Type"); got != "" {
			t.Fatalf("expected no explicit Content-Type header, got %q", got)
		}
		if got := r.Header.Get("Accept"); got != "" {
			t.Fatalf("expected no explicit Accept header, got %q", got)
		}
		if got := r.UserAgent(); got != "" {
			t.Fatalf("expected no User-Agent header, got %q", got)
		}
		if got := r.Header.Get("Accept-Encoding"); got != "" {
			t.Fatalf("expected no explicit Accept-Encoding header, got %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]string{"data": "无自定义请求头"}); err != nil {
			t.Fatalf("failed to encode response: %v", err)
		}
	}))
	t.Cleanup(server.Close)

	translator := NewTranslationService(TranslationServiceConfig{
		APIURL:  server.URL,
		Token:   "test-token",
		Timeout: time.Second,
	})

	result, err := translator.Translate(context.Background(), TranslationRequest{Text: "Hello"})
	if err != nil {
		t.Fatalf("Translate returned error: %v", err)
	}
	if result.TranslatedText != "无自定义请求头" {
		t.Fatalf("expected translated text from data field, got %q", result.TranslatedText)
	}
}

func TestTranslationServiceDefaultClientDisablesHTTP2(t *testing.T) {
	translator := NewTranslationService(TranslationServiceConfig{
		APIURL: "https://example.com/translate",
		Token:  "test-token",
	})

	service, ok := translator.(*httpTranslationService)
	if !ok {
		t.Fatalf("expected *httpTranslationService, got %T", translator)
	}
	transport, ok := service.client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("expected *http.Transport, got %T", service.client.Transport)
	}
	if transport.ForceAttemptHTTP2 {
		t.Fatal("expected default translator client to disable HTTP/2")
	}
	if !transport.DisableCompression {
		t.Fatal("expected default translator client to disable automatic compression headers")
	}
	if transport.TLSNextProto == nil {
		t.Fatal("expected TLSNextProto to be set so HTTP/2 is not auto-enabled")
	}
}

func TestParseProviderTranslationResponseAcceptsDataString(t *testing.T) {
	translatedText, err := parseProviderTranslationResponse([]byte(`{"code":200,"data":"你好，世界","source_lang":"auto","target_lang":"ZH"}`))
	if err != nil {
		t.Fatalf("parseProviderTranslationResponse returned error: %v", err)
	}
	if translatedText != "你好，世界" {
		t.Fatalf("expected translated text from data field, got %q", translatedText)
	}
}

func TestTranslationServiceRejectsEmptyText(t *testing.T) {
	calledProvider := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calledProvider = true
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	translator := NewTranslationService(TranslationServiceConfig{
		APIURL:  server.URL,
		Token:   "test-token",
		Timeout: time.Second,
	})

	_, err := translator.Translate(context.Background(), TranslationRequest{Text: "   "})
	if err == nil {
		t.Fatal("expected error for empty text")
	}
	apiErr, ok := dto.AsAPIError(err)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.Code != dto.ErrInvalidRequest {
		t.Fatalf("expected ErrInvalidRequest, got %d", apiErr.Code)
	}
	if calledProvider {
		t.Fatal("expected provider not to be called for empty text")
	}
}

func TestTranslationServiceReturnsOperationErrorForProviderFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`provider exploded`))
	}))
	t.Cleanup(server.Close)

	translator := NewTranslationService(TranslationServiceConfig{
		APIURL:  server.URL,
		Token:   "test-token",
		Timeout: time.Second,
	})

	_, err := translator.Translate(context.Background(), TranslationRequest{Text: "Hello"})
	if err == nil {
		t.Fatal("expected provider failure error")
	}
	apiErr, ok := dto.AsAPIError(err)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.Code != dto.ErrOperationFailed {
		t.Fatalf("expected ErrOperationFailed, got %d", apiErr.Code)
	}
	if apiErr.Message == "provider exploded" {
		t.Fatal("expected provider response body not to be exposed")
	}
}
