package auth

import (
	"fmt"
	"sync"
)

type Role struct {
	Name        string
	Permissions []string
	Description string
}

type Permission struct {
	Resource string
	Action   string
	Condition string
}

type RBACManager struct {
	roles       map[string]*Role
	userRoles   map[string][]string
	permissions map[string][]Permission
	mu          sync.RWMutex
}

func NewRBACManager() *RBACManager {
	return &RBACManager{
		roles:       make(map[string]*Role),
		userRoles:   make(map[string][]string),
		permissions: make(map[string][]Permission),
	}
}

func (rm *RBACManager) CreateRole(name string, permissions []string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if _, exists := rm.roles[name]; exists {
		return fmt.Errorf("role already exists: %s", name)
	}

	rm.roles[name] = &Role{
		Name:        name,
		Permissions: permissions,
	}

	return nil
}

func (rm *RBACManager) DeleteRole(name string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if _, exists := rm.roles[name]; !exists {
		return fmt.Errorf("role not found: %s", name)
	}

	delete(rm.roles, name)
	return nil
}

func (rm *RBACManager) AssignRoleToUser(userID string, roleName string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if _, exists := rm.roles[roleName]; !exists {
		return fmt.Errorf("role not found: %s", roleName)
	}

	rm.userRoles[userID] = append(rm.userRoles[userID], roleName)
	return nil
}

func (rm *RBACManager) RevokeRoleFromUser(userID string, roleName string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	roles, exists := rm.userRoles[userID]
	if !exists {
		return fmt.Errorf("user has no roles: %s", userID)
	}

	newRoles := make([]string, 0)
	for _, r := range roles {
		if r != roleName {
			newRoles = append(newRoles, r)
		}
	}

	rm.userRoles[userID] = newRoles
	return nil
}

func (rm *RBACManager) HasPermission(userID string, resource string, action string) bool {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	roles, exists := rm.userRoles[userID]
	if !exists {
		return false
	}

	requiredPerm := fmt.Sprintf("%s:%s", resource, action)

	for _, roleName := range roles {
		role, exists := rm.roles[roleName]
		if !exists {
			continue
		}

		for _, perm := range role.Permissions {
			if perm == requiredPerm || perm == "*" {
				return true
			}
		}
	}

	return false
}

func (rm *RBACManager) GetUserRoles(userID string) []string {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	roles, exists := rm.userRoles[userID]
	if !exists {
		return []string{}
	}

	return roles
}

func (rm *RBACManager) GetRolePermissions(roleName string) []string {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	role, exists := rm.roles[roleName]
	if !exists {
		return []string{}
	}

	return role.Permissions
}

// Built-in roles
func (rm *RBACManager) InitializeDefaultRoles() error {
	adminPermissions := []string{
		"tasks:*",
		"workflows:*",
		"agents:*",
		"nodes:*",
		"secrets:*",
		"users:*",
		"audit:read",
	}

	userPermissions := []string{
		"tasks:submit",
		"tasks:read",
		"workflows:create",
		"workflows:read",
		"workflows:execute",
		"agents:read",
	}

	viewerPermissions := []string{
		"tasks:read",
		"workflows:read",
		"agents:read",
		"nodes:read",
	}

	if err := rm.CreateRole("admin", adminPermissions); err != nil {
		return err
	}

	if err := rm.CreateRole("user", userPermissions); err != nil {
		return err
	}

	if err := rm.CreateRole("viewer", viewerPermissions); err != nil {
		return err
	}

	return nil
}
