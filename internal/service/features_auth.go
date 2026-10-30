package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pageguard/pageguard/internal/domain"
)

// APIKeyInput is the dashboard's create-input for API keys.
type APIKeyInput struct {
	Name      string     `json:"name"`
	ProjectID domain.ID  `json:"projectId"`
	Roles     []string   `json:"roles"`
	ExpiresAt *time.Time `json:"expiresAt"`
}

// APIKeySecret is returned when a key is created; it contains the plaintext
// secret and is never persisted.
type APIKeySecret struct {
	Key    *domain.APIKey `json:"key"`
	Secret string         `json:"secret"`
}

// CreateAPIKey mints a key. The plaintext secret is returned exactly once.
func (s *Service) CreateAPIKey(ctx context.Context, input APIKeyInput) (*APIKeySecret, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return nil, fmt.Errorf("key name is required")
	}
	roles := []string{}
	seen := map[string]bool{}
	for _, r := range input.Roles {
		r = strings.TrimSpace(r)
		if r == "" {
			continue
		}
		if !domain.ValidRole(r) {
			return nil, fmt.Errorf("unsupported role %q", r)
		}
		if !seen[r] {
			seen[r] = true
			roles = append(roles, r)
		}
	}
	if len(roles) == 0 {
		return nil, fmt.Errorf("at least one role is required")
	}
	// Reader-only keys are scoped by definition; a project pin or admin role
	// is required so keys never silently span projects.
	if len(roles) == 1 && roles[0] != domain.RoleAdmin && input.ProjectID.IsEmpty() {
		return nil, fmt.Errorf("reader/writer keys must be scoped to a project, or grant the admin role")
	}
	if !input.ProjectID.IsEmpty() {
		if _, err := s.store.GetProject(ctx, input.ProjectID); err != nil {
			return nil, err
		}
	}
	// Non-admin callers may only mint keys inside their own scope.
	if sc, ok := authScope(ctx); ok && !sc.HasRole(domain.RoleAdmin) && input.ProjectID != sc.ProjectID {
		return nil, fmt.Errorf("project %s is outside this API key's scope", input.ProjectID)
	}

	secret, err := domain.NewAPIKeySecret()
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	key := &domain.APIKey{
		ID:        domain.NewID(),
		Name:      input.Name,
		ProjectID: input.ProjectID,
		Prefix:    prefixOf(secret),
		KeyHash:   domain.HashAPIKey(secret),
		Roles:     roles,
		Enabled:   true,
		ExpiresAt: input.ExpiresAt,
		CreatedAt: now,
	}
	if err := s.store.CreateAPIKey(ctx, key); err != nil {
		return nil, err
	}
	key.KeyHash = ""
	return &APIKeySecret{Key: key, Secret: secret}, nil
}

// UpdateAPIKey toggles or renames an existing key.
func (s *Service) UpdateAPIKey(ctx context.Context, id domain.ID, patch APIKeyPatch) (*domain.APIKey, error) {
	k, err := s.store.GetAPIKey(ctx, id)
	if err != nil {
		return nil, err
	}
	if patch.Name != nil {
		if strings.TrimSpace(*patch.Name) == "" {
			return nil, fmt.Errorf("key name cannot be empty")
		}
		k.Name = *patch.Name
	}
	if patch.Enabled != nil {
		if sc, ok := authScope(ctx); ok && !sc.HasRole(domain.RoleAdmin) {
			return nil, fmt.Errorf("only admin keys may re-enable disabled keys")
		}
		k.Enabled = *patch.Enabled
	}
	if err := s.store.UpdateAPIKey(ctx, k); err != nil {
		return nil, err
	}
	k.KeyHash = ""
	return k, nil
}

// DeleteAPIKey removes a key.
func (s *Service) DeleteAPIKey(ctx context.Context, id domain.ID) error {
	return s.store.DeleteAPIKey(ctx, id)
}

// ListAPIKeys returns keys visible to the caller with hashes scrubbed.
func (s *Service) ListAPIKeys(ctx context.Context, projectID domain.ID) ([]*domain.APIKey, error) {
	var (
		keys []*domain.APIKey
		err  error
	)
	if sc, ok := authScope(ctx); ok && !sc.HasRole(domain.RoleAdmin) {
		projectID = sc.ProjectID
	}
	keys, err = s.store.ListAPIKeys(ctx, projectID)
	if err != nil {
		return nil, err
	}
	for _, k := range keys {
		k.KeyHash = ""
	}
	return keys, nil
}

// TouchAPIKey records usage (best-effort, called by the auth middleware).
func (s *Service) TouchAPIKey(ctx context.Context, id domain.ID) {
	k, err := s.store.GetAPIKey(ctx, id)
	if err != nil {
		return
	}
	now := time.Now().UTC()
	k.LastUsedAt = &now
	_ = s.store.UpdateAPIKey(ctx, k)
}

// APIKeyPatch contains optional key fields for partial updates.
type APIKeyPatch struct {
	Name    *string `json:"name"`
	Enabled *bool   `json:"enabled"`
}

func prefixOf(secret string) string {
	if len(secret) > 11 {
		return secret[:11]
	}
	return secret
}
