package auth

import (
	"crypto/sha256"
	"fmt"
	"sync"
	"time"
)

type AuthManager struct {
	tokens map[string]*Token
	mu     sync.RWMutex
}

type Token struct {
	Value     string
	ExpiresAt time.Time
	UserID    string
	Scopes    []string
}

func NewAuthManager() *AuthManager {
	return &AuthManager{
		tokens: make(map[string]*Token),
	}
}

func (am *AuthManager) GenerateToken(userID string, scopes []string) string {
	token := fmt.Sprintf("%x", sha256.Sum256([]byte(userID+time.Now().String())))

	am.mu.Lock()
	am.tokens[token] = &Token{
		Value:     token,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		UserID:    userID,
		Scopes:    scopes,
	}
	am.mu.Unlock()

	return token
}

func (am *AuthManager) ValidateToken(token string) bool {
	am.mu.RLock()
	defer am.mu.RUnlock()

	t, exists := am.tokens[token]
	if !exists {
		return false
	}

	return time.Now().Before(t.ExpiresAt)
}

func (am *AuthManager) RevokeToken(token string) {
	am.mu.Lock()
	delete(am.tokens, token)
	am.mu.Unlock()
}

func (am *AuthManager) GetToken(token string) *Token {
	am.mu.RLock()
	defer am.mu.RUnlock()

	return am.tokens[token]
}
