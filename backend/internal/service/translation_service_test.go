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
