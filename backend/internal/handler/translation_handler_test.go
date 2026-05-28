package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"fusionmail/internal/service"

	"github.com/gin-gonic/gin"
)

type fakeTranslationService struct {
	request service.TranslationRequest
	result  *service.TranslationResult
	err     error
}

func (s *fakeTranslationService) Translate(ctx context.Context, req service.TranslationRequest) (*service.TranslationResult, error) {
	s.request = req
	if s.err != nil {
		return nil, s.err
	}
	return s.result, nil
}

func TestTranslationHandlerTranslateReturnsTranslatedText(t *testing.T) {
	originalMode := gin.Mode()
	gin.SetMode(gin.TestMode)
	t.Cleanup(func() { gin.SetMode(originalMode) })

	fakeService := &fakeTranslationService{
		result: &service.TranslationResult{TranslatedText: "你好"},
	}
	handler := NewTranslationHandler(fakeService)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/translate", strings.NewReader(`{"text":"Hello","source_lang":"auto","target_lang":"ZH"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	handler.Translate(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if fakeService.request.Text != "Hello" {
		t.Fatalf("expected request text Hello, got %q", fakeService.request.Text)
	}

	var response struct {
		Success bool `json:"success"`
		Data    struct {
			TranslatedText string `json:"translated_text"`
		} `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !response.Success {
		t.Fatal("expected success response")
	}
	if response.Data.TranslatedText != "你好" {
		t.Fatalf("expected translated text, got %q", response.Data.TranslatedText)
	}
}

func TestTranslationHandlerTranslateRejectsMalformedJSON(t *testing.T) {
	originalMode := gin.Mode()
	gin.SetMode(gin.TestMode)
	t.Cleanup(func() { gin.SetMode(originalMode) })

	handler := NewTranslationHandler(&fakeTranslationService{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/translate", strings.NewReader(`{"text"`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	handler.Translate(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}
