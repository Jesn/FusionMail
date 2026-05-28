package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"fusionmail/internal/model"
	cryptoutil "fusionmail/pkg/crypto"
	"fusionmail/pkg/database"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestLoginUsesHttpOnlyCookieWithoutReturningToken(t *testing.T) {
	handler, _ := newAuthCookieTestHandler(t)
	router := gin.New()
	router.POST("/auth/login", handler.Login)

	body := bytes.NewBufferString(`{"username":"alice","password":"correct-password"}`)
	req := httptest.NewRequest(http.MethodPost, "/auth/login", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
	assertSessionCookieSet(t, w)
	assertResponseDataHasNoToken(t, w.Body.Bytes())
}

func TestRefreshTokenUsesCookieWithoutReturningToken(t *testing.T) {
	handler, user := newAuthCookieTestHandler(t)
	router := gin.New()
	router.POST("/auth/refresh", handler.RefreshToken)

	now := time.Now().UTC().Truncate(time.Second)
	tokenString, _, err := issueSessionToken(
		"test-secret",
		user,
		now.Add(-accessTokenTTL+5*time.Minute),
		now.Add(maxRefreshSessionTTL),
	)
	if err != nil {
		t.Fatalf("failed to issue test token: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: "fm_session", Value: tokenString})
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
	assertSessionCookieSet(t, w)
	assertResponseDataHasNoToken(t, w.Body.Bytes())
}

func TestVerifyUsesCookieSessionWithoutAuthorizationHeader(t *testing.T) {
	handler, user := newAuthCookieTestHandler(t)
	router := gin.New()
	router.GET("/auth/verify", handler.Verify)

	tokenString, _, err := issueSessionToken("test-secret", user, time.Now(), time.Now().Add(maxRefreshSessionTTL))
	if err != nil {
		t.Fatalf("failed to issue test token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/auth/verify", nil)
	req.AddCookie(&http.Cookie{Name: "fm_session", Value: tokenString})
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), `"token"`) {
		t.Fatalf("verify response must not expose token: %s", w.Body.String())
	}
}

func newAuthCookieTestHandler(t *testing.T) (*DBAuthHandler, *model.User) {
	t.Helper()

	originalMode := gin.Mode()
	gin.SetMode(gin.TestMode)
	t.Cleanup(func() {
		gin.SetMode(originalMode)
	})

	previousDB := database.DB
	db, err := gorm.Open(sqlite.Open("file:auth_cookie_session_test?mode=memory&cache=shared&_foreign_keys=1"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("failed to get sql db: %v", err)
	}
	t.Cleanup(func() {
		database.DB = previousDB
		_ = sqlDB.Close()
	})
	database.DB = db

	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatalf("failed to migrate users table: %v", err)
	}
	passwordHash, err := cryptoutil.HashPassword("correct-password")
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	user := &model.User{
		Username:       "alice",
		Email:          "alice@example.com",
		PasswordHash:   passwordHash,
		Role:           "admin",
		IsActive:       true,
		SessionVersion: 4,
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	return NewDBAuthHandler("test-secret", nil), user
}

func assertSessionCookieSet(t *testing.T, w *httptest.ResponseRecorder) {
	t.Helper()

	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == "fm_session" {
			if cookie.Value == "" {
				t.Fatal("expected fm_session cookie value")
			}
			if !cookie.HttpOnly {
				t.Fatal("expected fm_session cookie to be HttpOnly")
			}
			return
		}
	}

	t.Fatalf("expected fm_session cookie, got Set-Cookie=%q", w.Header().Values("Set-Cookie"))
}

func assertResponseDataHasNoToken(t *testing.T, body []byte) {
	t.Helper()

	if strings.Contains(string(body), `"token"`) {
		t.Fatalf("response body must not expose token: %s", string(body))
	}

	var response struct {
		Success bool           `json:"success"`
		Data    map[string]any `json:"data"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !response.Success {
		t.Fatalf("expected success response: %s", string(body))
	}
	if response.Data["expiresAt"] == "" {
		t.Fatalf("expected expiresAt in response data: %s", string(body))
	}
}
