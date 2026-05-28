package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"fusionmail/internal/model"
	"fusionmail/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type fakeSSEUserStore struct {
	user *model.User
	err  error
}

func (s fakeSSEUserStore) GetUserByID(id int64) (*model.User, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.user == nil || s.user.ID != id {
		return nil, errors.New("user not found")
	}
	return s.user, nil
}

type mutableSSEUserStore struct {
	mu   sync.Mutex
	user model.User
}

func (s *mutableSSEUserStore) GetUserByID(id int64) (*model.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.user.ID != id {
		return nil, errors.New("user not found")
	}
	user := s.user
	return &user, nil
}

func (s *mutableSSEUserStore) setSessionVersion(sessionVersion int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.user.SessionVersion = sessionVersion
}

type closeNotifyRecorder struct {
	*httptest.ResponseRecorder
	closeCh chan bool
}

func (r *closeNotifyRecorder) CloseNotify() <-chan bool {
	return r.closeCh
}

type deadlineRecorder struct {
	*closeNotifyRecorder
	mu       sync.Mutex
	called   bool
	deadline time.Time
}

func (r *deadlineRecorder) SetWriteDeadline(deadline time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.called = true
	r.deadline = deadline
	return nil
}

func (r *deadlineRecorder) deadlineState() (bool, time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.called, r.deadline
}

func TestSSEDisablesWriteDeadline(t *testing.T) {
	originalMode := gin.Mode()
	gin.SetMode(gin.TestMode)
	t.Cleanup(func() {
		gin.SetMode(originalMode)
	})

	w := &deadlineRecorder{
		closeNotifyRecorder: &closeNotifyRecorder{ResponseRecorder: httptest.NewRecorder(), closeCh: make(chan bool)},
	}
	c, _ := gin.CreateTestContext(w)

	NewSSEHandlerWithUserStore("test-secret", nil).disableWriteDeadline(c.Writer)

	called, deadline := w.deadlineState()
	if !called {
		t.Fatal("expected write deadline to be disabled on underlying writer")
	}
	if !deadline.IsZero() {
		t.Fatalf("expected zero write deadline, got %v", deadline)
	}
}

func TestSSERejectsStaleSessionVersion(t *testing.T) {
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
		SessionVersion: 3,
	}
	handler := NewSSEHandlerWithUserStore(secret, fakeSSEUserStore{user: user})
	tokenString := signedSSETestToken(t, secret, user.ID, user.SessionVersion-1)

	req := httptest.NewRequest(http.MethodGet, "/events", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	handler.Stream(c)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestSSERejectsQueryToken(t *testing.T) {
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
		SessionVersion: 3,
	}
	handler := NewSSEHandlerWithUserStore(secret, fakeSSEUserStore{user: user})
	tokenString := signedSSETestToken(t, secret, user.ID, user.SessionVersion)

	req := httptest.NewRequest(http.MethodGet, "/events?token="+tokenString, nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	handler.Stream(c)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestSSEStopsAfterSessionVersionChanges(t *testing.T) {
	originalMode := gin.Mode()
	gin.SetMode(gin.TestMode)
	t.Cleanup(func() {
		gin.SetMode(originalMode)
	})

	originalInterval := sseHeartbeatInterval
	sseHeartbeatInterval = 5 * time.Millisecond
	t.Cleanup(func() {
		sseHeartbeatInterval = originalInterval
	})

	secret := "test-secret"
	store := &mutableSSEUserStore{user: model.User{
		ID:             42,
		Username:       "alice",
		Role:           "admin",
		IsActive:       true,
		SessionVersion: 3,
	}}
	handler := NewSSEHandlerWithUserStore(secret, store)
	tokenString := signedSSETestToken(t, secret, store.user.ID, store.user.SessionVersion)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req := httptest.NewRequest(http.MethodGet, "/events", nil).WithContext(ctx)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	w := &closeNotifyRecorder{ResponseRecorder: httptest.NewRecorder(), closeCh: make(chan bool)}
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	done := make(chan struct{})
	go func() {
		handler.Stream(c)
		close(done)
	}()

	select {
	case <-done:
		t.Fatal("expected stream to remain open before revocation")
	case <-time.After(20 * time.Millisecond):
	}

	store.setSessionVersion(4)

	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected stream to stop after session version changes")
	}
}

func signedSSETestToken(t *testing.T, secret string, userID, sessionVersion int64) string {
	t.Helper()
	tokenString, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":                       strconv.FormatInt(userID, 10),
		"username":                  "alice",
		"role":                      "admin",
		"exp":                       time.Now().Add(time.Hour).Unix(),
		service.SessionVersionClaim: sessionVersion,
	}).SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}
	return tokenString
}
