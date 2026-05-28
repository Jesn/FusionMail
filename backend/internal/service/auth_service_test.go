package service

import (
	"context"
	"testing"
	"time"

	"fusionmail/internal/model"
)

type authServiceTestUserRepo struct {
	user *model.User
}

func (r *authServiceTestUserRepo) Create(ctx context.Context, user *model.User) error {
	return nil
}

func (r *authServiceTestUserRepo) FindByID(ctx context.Context, id int64) (*model.User, error) {
	if r.user == nil || r.user.ID != id {
		return nil, ErrUserNotFound
	}
	return r.user, nil
}

func (r *authServiceTestUserRepo) FindByUsername(ctx context.Context, username string) (*model.User, error) {
	return r.user, nil
}

func (r *authServiceTestUserRepo) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	return r.user, nil
}

func (r *authServiceTestUserRepo) Update(ctx context.Context, user *model.User) error {
	r.user = user
	return nil
}

func (r *authServiceTestUserRepo) Delete(ctx context.Context, id int64) error {
	return nil
}

func (r *authServiceTestUserRepo) List(ctx context.Context, offset, limit int) ([]*model.User, int64, error) {
	return nil, 0, nil
}

func (r *authServiceTestUserRepo) IncrementFailedAttempts(ctx context.Context, id int64) error {
	return nil
}

func (r *authServiceTestUserRepo) ResetFailedAttempts(ctx context.Context, id int64) error {
	return nil
}

func (r *authServiceTestUserRepo) UpdateLastLogin(ctx context.Context, id int64, ip string) error {
	return nil
}

func TestAuthServiceIssueTokenIncludesSessionVersion(t *testing.T) {
	service := NewAuthService(nil, nil, "test-secret", time.Hour)
	now := time.Now().UTC().Truncate(time.Second)
	user := &model.User{
		ID:             42,
		Username:       "alice",
		Email:          "alice@example.com",
		SessionVersion: 9,
	}

	tokenString, _, err := service.issueToken(user, now, now.Add(maxRefreshSessionTTL))
	if err != nil {
		t.Fatalf("issueToken returned error: %v", err)
	}

	claims, err := service.parseSignedTokenClaims(tokenString)
	if err != nil {
		t.Fatalf("parseSignedTokenClaims returned error: %v", err)
	}
	if claims[SessionVersionClaim] != float64(9) {
		t.Fatalf("expected session version claim 9, got %#v", claims[SessionVersionClaim])
	}
}

func TestAuthServiceRefreshTokenRejectsStaleSessionVersion(t *testing.T) {
	secret := "test-secret"
	repo := &authServiceTestUserRepo{user: &model.User{
		ID:             42,
		Username:       "alice",
		Email:          "alice@example.com",
		IsActive:       true,
		SessionVersion: 2,
	}}
	service := NewAuthService(repo, nil, secret, 5*time.Minute)
	issuedUser := *repo.user
	issuedUser.SessionVersion = 1

	tokenString, _, err := service.issueToken(&issuedUser, time.Now().Add(-4*time.Minute), time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("issueToken returned error: %v", err)
	}

	if _, _, err := service.RefreshToken(context.Background(), tokenString); err == nil {
		t.Fatal("expected stale session token to be rejected")
	}
}

func TestAuthServiceRefreshTokenAcceptsMatchingSessionVersion(t *testing.T) {
	secret := "test-secret"
	repo := &authServiceTestUserRepo{user: &model.User{
		ID:             42,
		Username:       "alice",
		Email:          "alice@example.com",
		IsActive:       true,
		SessionVersion: 2,
	}}
	service := NewAuthService(repo, nil, secret, 5*time.Minute)

	tokenString, _, err := service.issueToken(repo.user, time.Now().Add(-4*time.Minute), time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("issueToken returned error: %v", err)
	}

	refreshedToken, _, err := service.RefreshToken(context.Background(), tokenString)
	if err != nil {
		t.Fatalf("RefreshToken returned error: %v", err)
	}
	claims, err := service.parseSignedTokenClaims(refreshedToken)
	if err != nil {
		t.Fatalf("parseSignedTokenClaims returned error: %v", err)
	}
	if claims[SessionVersionClaim] != float64(repo.user.SessionVersion) {
		t.Fatalf("expected refreshed token session version %d, got %#v", repo.user.SessionVersion, claims[SessionVersionClaim])
	}
}
