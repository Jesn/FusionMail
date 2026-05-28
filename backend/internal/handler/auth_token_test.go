package handler

import (
	"testing"
	"time"

	"fusionmail/internal/model"
	"fusionmail/internal/service"
)

func TestIssueSessionTokenIncludesSessionExpiry(t *testing.T) {
	secret := "test-secret"
	now := time.Now().UTC().Truncate(time.Second)
	sessionExpiresAt := now.Add(maxRefreshSessionTTL)
	user := &model.User{
		ID:             42,
		Username:       "alice",
		Role:           "admin",
		SessionVersion: 3,
	}

	tokenString, expiresAt, err := issueSessionToken(secret, user, now, sessionExpiresAt)
	if err != nil {
		t.Fatalf("issueSessionToken returned error: %v", err)
	}

	claims, err := parseSignedTokenClaims(tokenString, secret)
	if err != nil {
		t.Fatalf("parseSignedTokenClaims returned error: %v", err)
	}

	parsedSessionExpiresAt, ok := getRefreshSessionExpiresAt(claims)
	if !ok {
		t.Fatal("expected session_exp claim to exist")
	}
	if !parsedSessionExpiresAt.Equal(sessionExpiresAt) {
		t.Fatalf("expected session_exp %v, got %v", sessionExpiresAt, parsedSessionExpiresAt)
	}
	if claims["sub"] != "42" {
		t.Fatalf("expected sub claim to be string user id, got %#v", claims["sub"])
	}
	if claims[service.SessionVersionClaim] != float64(3) {
		t.Fatalf("expected session version claim 3, got %#v", claims[service.SessionVersionClaim])
	}
	if expiresAt.Sub(now) != accessTokenTTL {
		t.Fatalf("expected access token ttl %v, got %v", accessTokenTTL, expiresAt.Sub(now))
	}
}

func TestTwoFactorLoginChallenge(t *testing.T) {
	originalStore := twoFactorChallengeStore
	twoFactorChallengeStore = newTwoFactorChallengeStore()
	t.Cleanup(func() {
		twoFactorChallengeStore = originalStore
	})

	now := time.Now().UTC().Truncate(time.Second)
	token, expiresAt, err := issueTwoFactorLoginChallenge(42, 7, now)
	if err != nil {
		t.Fatalf("issueTwoFactorLoginChallenge returned error: %v", err)
	}
	if token == "" {
		t.Fatal("expected challenge token")
	}
	if expiresAt.Sub(now) != twoFactorChallengeTTL {
		t.Fatalf("expected challenge ttl %v, got %v", twoFactorChallengeTTL, expiresAt.Sub(now))
	}

	if err := consumeTwoFactorLoginChallenge(token, 42, 7, now.Add(time.Minute)); err != nil {
		t.Fatalf("expected challenge consumption to succeed: %v", err)
	}
	if err := consumeTwoFactorLoginChallenge(token, 42, 7, now.Add(time.Minute)); err == nil {
		t.Fatal("expected consumed challenge to be rejected")
	}
}

func TestTwoFactorLoginChallengeRejectsInvalidUserAndExpiry(t *testing.T) {
	originalStore := twoFactorChallengeStore
	twoFactorChallengeStore = newTwoFactorChallengeStore()
	t.Cleanup(func() {
		twoFactorChallengeStore = originalStore
	})

	now := time.Now().UTC().Truncate(time.Second)
	wrongUserToken, _, err := issueTwoFactorLoginChallenge(42, 7, now)
	if err != nil {
		t.Fatalf("issueTwoFactorLoginChallenge returned error: %v", err)
	}
	if err := consumeTwoFactorLoginChallenge(wrongUserToken, 24, 7, now.Add(time.Minute)); err == nil {
		t.Fatal("expected wrong user to be rejected")
	}

	expiredToken, _, err := issueTwoFactorLoginChallenge(42, 7, now)
	if err != nil {
		t.Fatalf("issueTwoFactorLoginChallenge returned error: %v", err)
	}
	if err := consumeTwoFactorLoginChallenge(expiredToken, 42, 7, now.Add(twoFactorChallengeTTL+time.Second)); err == nil {
		t.Fatal("expected expired challenge to be rejected")
	}
}

func TestTwoFactorLoginChallengeRejectsStaleSessionVersion(t *testing.T) {
	originalStore := twoFactorChallengeStore
	twoFactorChallengeStore = newTwoFactorChallengeStore()
	t.Cleanup(func() {
		twoFactorChallengeStore = originalStore
	})

	now := time.Now().UTC().Truncate(time.Second)
	token, _, err := issueTwoFactorLoginChallenge(42, 7, now)
	if err != nil {
		t.Fatalf("issueTwoFactorLoginChallenge returned error: %v", err)
	}
	if err := consumeTwoFactorLoginChallenge(token, 42, 8, now.Add(time.Minute)); err == nil {
		t.Fatal("expected stale session version to be rejected")
	}
}

func TestValidateRefreshClaims(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)

	tests := []struct {
		name   string
		claims map[string]interface{}
		wantOK bool
	}{
		{
			name: "accepts token near expiry with active session",
			claims: map[string]interface{}{
				"exp":         float64(now.Add(5 * time.Minute).Unix()),
				"session_exp": float64(now.Add(24 * time.Hour).Unix()),
			},
			wantOK: true,
		},
		{
			name: "rejects token refreshed too early",
			claims: map[string]interface{}{
				"exp":         float64(now.Add(30 * time.Minute).Unix()),
				"session_exp": float64(now.Add(24 * time.Hour).Unix()),
			},
			wantOK: false,
		},
		{
			name: "rejects expired token",
			claims: map[string]interface{}{
				"exp":         float64(now.Add(-1 * time.Minute).Unix()),
				"session_exp": float64(now.Add(24 * time.Hour).Unix()),
			},
			wantOK: false,
		},
		{
			name: "accepts legacy token without session claim once",
			claims: map[string]interface{}{
				"exp": float64(now.Add(5 * time.Minute).Unix()),
			},
			wantOK: true,
		},
		{
			name: "rejects expired refresh session",
			claims: map[string]interface{}{
				"exp":         float64(now.Add(5 * time.Minute).Unix()),
				"session_exp": float64(now.Add(-1 * time.Minute).Unix()),
			},
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRefreshClaims(tt.claims, now)
			if tt.wantOK && err != nil {
				t.Fatalf("expected success, got error: %v", err)
			}
			if !tt.wantOK && err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}
