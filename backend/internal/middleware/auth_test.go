package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"fusionmail/internal/model"
	"fusionmail/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type fakeSessionUserStore struct {
	user *model.User
	err  error
}

func (s fakeSessionUserStore) GetUserByID(id int64) (*model.User, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.user == nil || s.user.ID != id {
		return nil, errors.New("user not found")
	}
	return s.user, nil
}

func TestRequireAuthRejectsStaleSessionVersion(t *testing.T) {
	originalMode := gin.Mode()
	gin.SetMode(gin.TestMode)
	t.Cleanup(func() {
		gin.SetMode(originalMode)
	})

	secret := "test-secret"
	user := &model.User{
		ID:             42,
		Username:       "alice",
		Role:           "admin",
		IsActive:       true,
		SessionVersion: 5,
	}

	tests := []struct {
		name       string
		token      string
		wantStatus int
	}{
		{
			name:       "匹配版本允许访问",
			token:      signedMiddlewareTestToken(t, secret, user.ID, user.SessionVersion),
			wantStatus: http.StatusOK,
		},
		{
			name:       "旧版本拒绝访问",
			token:      signedMiddlewareTestToken(t, secret, user.ID, user.SessionVersion-1),
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "缺少版本拒绝访问",
			token:      signedMiddlewareTestTokenWithoutSessionVersion(t, secret, user.ID),
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(NewAuthMiddlewareWithUserStore(secret, fakeSessionUserStore{user: user}).RequireAuth())
			router.GET("/private", func(c *gin.Context) {
				if got := c.GetString("userID"); got != strconv.FormatInt(user.ID, 10) {
					t.Fatalf("expected userID %d, got %q", user.ID, got)
				}
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/private", nil)
			req.Header.Set("Authorization", "Bearer "+tt.token)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}

func signedMiddlewareTestToken(t *testing.T, secret string, userID, sessionVersion int64) string {
	t.Helper()
	return signMiddlewareTestClaims(t, secret, jwt.MapClaims{
		"sub":                       strconv.FormatInt(userID, 10),
		"username":                  "alice",
		"role":                      "admin",
		"exp":                       time.Now().Add(time.Hour).Unix(),
		service.SessionVersionClaim: sessionVersion,
	})
}

func signedMiddlewareTestTokenWithoutSessionVersion(t *testing.T, secret string, userID int64) string {
	t.Helper()
	return signMiddlewareTestClaims(t, secret, jwt.MapClaims{
		"sub":      strconv.FormatInt(userID, 10),
		"username": "alice",
		"role":     "admin",
		"exp":      time.Now().Add(time.Hour).Unix(),
	})
}

func signMiddlewareTestClaims(t *testing.T, secret string, claims jwt.MapClaims) string {
	t.Helper()
	tokenString, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}
	return tokenString
}
