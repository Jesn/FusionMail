package handler

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// 2FA 登录挑战存储（自 auth.go 同位搬移）：签发/校验一次性挑战 token，
// 绑定 userID + sessionVersion，TTL 见 auth.go 的 twoFactorChallengeTTL。
var twoFactorChallengeStore = newTwoFactorChallengeStore()

type twoFactorLoginChallenge struct {
	userID         int64
	sessionVersion int64
	expiresAt      time.Time
}

type twoFactorChallengeStoreState struct {
	mu         sync.Mutex
	challenges map[string]twoFactorLoginChallenge
}

func newTwoFactorChallengeStore() *twoFactorChallengeStoreState {
	return &twoFactorChallengeStoreState{
		challenges: make(map[string]twoFactorLoginChallenge),
	}
}

func issueTwoFactorLoginChallenge(userID, sessionVersion int64, now time.Time) (string, time.Time, error) {
	return twoFactorChallengeStore.issue(userID, sessionVersion, now)
}

func consumeTwoFactorLoginChallenge(token string, userID, sessionVersion int64, now time.Time) error {
	return twoFactorChallengeStore.consume(token, userID, sessionVersion, now)
}

func (s *twoFactorChallengeStoreState) issue(userID, sessionVersion int64, now time.Time) (string, time.Time, error) {
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", time.Time{}, err
	}

	token := hex.EncodeToString(randomBytes)
	expiresAt := now.Add(twoFactorChallengeTTL)

	s.mu.Lock()
	defer s.mu.Unlock()

	for existingToken, challenge := range s.challenges {
		if !challenge.expiresAt.After(now) {
			delete(s.challenges, existingToken)
		}
	}

	s.challenges[token] = twoFactorLoginChallenge{
		userID:         userID,
		sessionVersion: sessionVersion,
		expiresAt:      expiresAt,
	}

	return token, expiresAt, nil
}

func (s *twoFactorChallengeStoreState) consume(token string, userID, sessionVersion int64, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	challenge, ok := s.challenges[token]
	if !ok {
		return fmt.Errorf("2fa challenge not found")
	}
	delete(s.challenges, token)

	if challenge.userID != userID {
		return fmt.Errorf("2fa challenge user mismatch")
	}
	if challenge.sessionVersion != sessionVersion {
		return fmt.Errorf("2fa challenge session version mismatch")
	}
	if !challenge.expiresAt.After(now) {
		return fmt.Errorf("2fa challenge expired")
	}

	return nil
}
