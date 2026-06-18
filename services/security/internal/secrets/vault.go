package secrets

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hashicorp/vault/api"
)

type VaultManager struct {
	client *api.Client
	mu     sync.RWMutex
}

type Secret struct {
	Path      string
	Key       string
	Value     string
	CreatedAt time.Time
	ExpiresAt *time.Time
	TTL       time.Duration
}

type DynamicSecret struct {
	Type      string
	Lease     string
	LeaseDuration int
	Renewable bool
	Data      map[string]interface{}
}

func NewVaultManager(ctx context.Context, addr string, token string) (*VaultManager, error) {
	config := api.DefaultConfig()
	config.Address = addr

	client, err := api.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Vault client: %w", err)
	}

	if token != "" {
		client.SetToken(token)
	}

	// Verify connection
	_, err = client.Auth().Token().LookupSelf()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Vault: %w", err)
	}

	return &VaultManager{
		client: client,
	}, nil
}

func (vm *VaultManager) WriteSecret(ctx context.Context, path string, data map[string]interface{}) error {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	_, err := vm.client.Logical().WriteWithContext(ctx, path, data)
	if err != nil {
		return fmt.Errorf("failed to write secret to %s: %w", path, err)
	}

	return nil
}

func (vm *VaultManager) ReadSecret(ctx context.Context, path string) (*Secret, error) {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	secret, err := vm.client.Logical().ReadWithContext(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("failed to read secret from %s: %w", path, err)
	}

	if secret == nil {
		return nil, fmt.Errorf("secret not found at %s", path)
	}

	result := &Secret{
		Path:      path,
		CreatedAt: time.Now(),
	}

	if secret.Data != nil {
		if val, ok := secret.Data["value"]; ok {
			result.Value = val.(string)
		}
	}

	return result, nil
}

func (vm *VaultManager) DeleteSecret(ctx context.Context, path string) error {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	_, err := vm.client.Logical().DeleteWithContext(ctx, path)
	if err != nil {
		return fmt.Errorf("failed to delete secret at %s: %w", path, err)
	}

	return nil
}

func (vm *VaultManager) ListSecrets(ctx context.Context, path string) ([]string, error) {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	secret, err := vm.client.Logical().ListWithContext(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("failed to list secrets at %s: %w", path, err)
	}

	if secret == nil || secret.Data == nil {
		return []string{}, nil
	}

	keys, ok := secret.Data["keys"].([]interface{})
	if !ok {
		return []string{}, nil
	}

	result := make([]string, len(keys))
	for i, k := range keys {
		result[i] = k.(string)
	}

	return result, nil
}

func (vm *VaultManager) GenerateDynamicSecret(ctx context.Context, role string) (*DynamicSecret, error) {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	path := fmt.Sprintf("database/creds/%s", role)
	secret, err := vm.client.Logical().ReadWithContext(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("failed to generate dynamic secret: %w", err)
	}

	if secret == nil {
		return nil, fmt.Errorf("no dynamic secret generated")
	}

	return &DynamicSecret{
		Type:           secret.Auth.ClientToken,
		Lease:          secret.LeaseID,
		LeaseDuration:  secret.LeaseDuration,
		Renewable:      secret.Renewable,
		Data:           secret.Data,
	}, nil
}

func (vm *VaultManager) RenewSecret(ctx context.Context, leaseID string) error {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	_, err := vm.client.Logical().RenewWithContext(ctx, leaseID, 0)
	if err != nil {
		return fmt.Errorf("failed to renew secret: %w", err)
	}

	return nil
}

func (vm *VaultManager) RevokeSecret(ctx context.Context, leaseID string) error {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	err := vm.client.Logical().RevokeWithContext(ctx, leaseID)
	if err != nil {
		return fmt.Errorf("failed to revoke secret: %w", err)
	}

	return nil
}

func (vm *VaultManager) RotateSecret(ctx context.Context, path string, newData map[string]interface{}) error {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	// Delete old secret
	_, _ = vm.client.Logical().DeleteWithContext(ctx, path)

	// Write new secret
	_, err := vm.client.Logical().WriteWithContext(ctx, path, newData)
	if err != nil {
		return fmt.Errorf("failed to rotate secret: %w", err)
	}

	return nil
}

func (vm *VaultManager) Close() error {
	return nil
}
