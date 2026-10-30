package domain

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// API key role names. A reader may read, a writer may read and mutate, an
// admin may additionally manage API keys. An admin key without a project
// restriction is unrestricted (super admin).
const (
	RoleReader = "reader"
	RoleWriter = "writer"
	RoleAdmin  = "admin"
)

// Roles is the canonical set of API key roles.
var Roles = []string{RoleReader, RoleWriter, RoleAdmin}

// ValidRole reports whether role is one of the defined roles.
func ValidRole(role string) bool {
	for _, r := range Roles {
		if r == role {
			return true
		}
	}
	return false
}

// APIKey is a bearer token granting API access. The plaintext secret is shown
// exactly once at creation ("pg_<64 hex chars>"); only its SHA-256 hash is
// persisted. KeyHash is therefore abbreviated to a short salt-free digest.
type APIKey struct {
	ID         ID         `json:"id"`
	Name       string     `json:"name"`
	ProjectID  ID         `json:"projectId"`
	Prefix     string     `json:"prefix"`
	KeyHash    string     `json:"keyHash,omitempty"`
	Roles      []string   `json:"roles"`
	Enabled    bool       `json:"enabled"`
	ExpiresAt  *time.Time `json:"expiresAt,omitempty"`
	LastUsedAt *time.Time `json:"lastUsedAt,omitempty"`
	CreatedAt  time.Time  `json:"createdAt"`
}

// HasRole reports whether the key carries role (admins implicitly carry all
// roles).
func (k *APIKey) HasRole(role string) bool {
	for _, r := range k.Roles {
		if r == RoleAdmin || r == role {
			return true
		}
	}
	return false
}

// Unrestricted reports whether the key may scope to any project.
func (k *APIKey) Unrestricted() bool {
	return k.ProjectID.IsEmpty() && k.HasRole(RoleAdmin)
}

// Active reports whether the key is enabled and not expired.
func (k *APIKey) Active() bool {
	if !k.Enabled {
		return false
	}
	if k.ExpiresAt != nil && !k.ExpiresAt.IsZero() && k.ExpiresAt.Before(time.Now().UTC()) {
		return false
	}
	return true
}

// HashAPIKeys derives the storage digest from a plaintext secret.
func HashAPIKey(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

// NewAPIKeySecret generates the plaintext bearer secret for a key.
// The generated string is `pg_` + 64 hex chars.
func NewAPIKeySecret() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("pageguard: crypto/rand unavailable: %w", err)
	}
	secret := "pg_" + hex.EncodeToString(raw)
	return secret, nil
}
