package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/ChaitanyaJoshi1769/TitanOS/services/security/internal/secrets"
)

type AuthManager struct {
	secretMgr *secrets.VaultManager
	rbac      *RBACManager
	mu        sync.RWMutex
	users     map[string]*User
	sessions  map[string]*Session
}

type User struct {
	ID           string
	Username     string
	PasswordHash string
	Email        string
	CreatedAt    time.Time
	LastLogin    *time.Time
	Active       bool
}

type Session struct {
	ID        string
	UserID    string
	Token     string
	CreatedAt time.Time
	ExpiresAt time.Time
	IP        string
}

type Credentials struct {
	Username string
	Password string
}

func NewAuthManager(secretMgr *secrets.VaultManager) *AuthManager {
	am := &AuthManager{
		secretMgr: secretMgr,
		rbac:      NewRBACManager(),
		users:     make(map[string]*User),
		sessions:  make(map[string]*Session),
	}

	// Initialize default roles
	am.rbac.InitializeDefaultRoles()

	return am
}

func (am *AuthManager) CreateUser(username string, password string, email string) (*User, error) {
	am.mu.Lock()
	defer am.mu.Unlock()

	// Check if user exists
	for _, u := range am.users {
		if u.Username == username {
			return nil, fmt.Errorf("user already exists: %s", username)
		}
	}

	userID := generateID()
	passwordHash := hashPassword(password)

	user := &User{
		ID:           userID,
		Username:     username,
		PasswordHash: passwordHash,
		Email:        email,
		CreatedAt:    time.Now(),
		Active:       true,
	}

	am.users[userID] = user

	// Assign default user role
	am.rbac.AssignRoleToUser(userID, "user")

	return user, nil
}

func (am *AuthManager) AuthenticateUser(creds Credentials) (*Session, error) {
	am.mu.Lock()
	defer am.mu.Unlock()

	var user *User
	for _, u := range am.users {
		if u.Username == creds.Username {
			user = u
			break
		}
	}

	if user == nil {
		return nil, fmt.Errorf("user not found: %s", creds.Username)
	}

	if !user.Active {
		return nil, fmt.Errorf("user account is inactive")
	}

	if !verifyPassword(creds.Password, user.PasswordHash) {
		return nil, fmt.Errorf("invalid password")
	}

	// Create session
	sessionID := generateID()
	token := generateToken()

	session := &Session{
		ID:        sessionID,
		UserID:    user.ID,
		Token:     token,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	am.sessions[sessionID] = session

	// Update last login
	now := time.Now()
	user.LastLogin = &now

	return session, nil
}

func (am *AuthManager) ValidateSession(token string) (*Session, error) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	for _, session := range am.sessions {
		if session.Token == token {
			if time.Now().After(session.ExpiresAt) {
				return nil, fmt.Errorf("session expired")
			}
			return session, nil
		}
	}

	return nil, fmt.Errorf("invalid session")
}

func (am *AuthManager) RevokeSession(token string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	for sessionID, session := range am.sessions {
		if session.Token == token {
			delete(am.sessions, sessionID)
			return nil
		}
	}

	return fmt.Errorf("session not found")
}

func (am *AuthManager) CheckPermission(userID string, resource string, action string) bool {
	return am.rbac.HasPermission(userID, resource, action)
}

func (am *AuthManager) AssignRole(userID string, roleName string) error {
	return am.rbac.AssignRoleToUser(userID, roleName)
}

func (am *AuthManager) RevokeRole(userID string, roleName string) error {
	return am.rbac.RevokeRoleFromUser(userID, roleName)
}

func (am *AuthManager) GetUser(userID string) (*User, error) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	user, exists := am.users[userID]
	if !exists {
		return nil, fmt.Errorf("user not found")
	}

	return user, nil
}

func (am *AuthManager) ListUsers() []*User {
	am.mu.RLock()
	defer am.mu.RUnlock()

	users := make([]*User, 0, len(am.users))
	for _, u := range am.users {
		users = append(users, u)
	}

	return users
}

// Helper functions

func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func generateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func hashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}

func verifyPassword(password string, hash string) bool {
	computedHash := hashPassword(password)
	return computedHash == hash
}
