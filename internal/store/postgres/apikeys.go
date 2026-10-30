package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pageguard/pageguard/internal/domain"
)

const apiKeyColumns = `id::text, name, coalesce(project_id::text,''), prefix, key_hash, roles,
	enabled, expires_at, last_used_at, created_at`

// optionalUUID encodes a project id, treating the empty id as NULL.
func optionalUUID(id domain.ID) any {
	if id.IsEmpty() {
		return nil
	}
	return toUUID(id)
}

func (s *Store) CreateAPIKey(ctx context.Context, k *domain.APIKey) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO pageguard_api_keys (id, name, project_id, prefix, key_hash, roles,
		                                enabled, expires_at, last_used_at, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		toUUID(k.ID), k.Name, optionalUUID(k.ProjectID), k.Prefix, k.KeyHash, k.Roles,
		k.Enabled, k.ExpiresAt, k.LastUsedAt, k.CreatedAt)
	return err
}

func (s *Store) GetAPIKey(ctx context.Context, id domain.ID) (*domain.APIKey, error) {
	return s.scanAPIKey(s.pool.QueryRow(ctx, `
		SELECT `+apiKeyColumns+` FROM pageguard_api_keys WHERE id = $1`, toUUID(id)))
}

func (s *Store) UpdateAPIKey(ctx context.Context, k *domain.APIKey) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE pageguard_api_keys
		   SET name=$2, project_id=$3, prefix=$4, key_hash=$5, roles=$6,
		       enabled=$7, expires_at=$8, last_used_at=$9
		 WHERE id=$1`,
		toUUID(k.ID), k.Name, optionalUUID(k.ProjectID), k.Prefix, k.KeyHash, k.Roles,
		k.Enabled, k.ExpiresAt, k.LastUsedAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *Store) DeleteAPIKey(ctx context.Context, id domain.ID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM pageguard_api_keys WHERE id=$1`, toUUID(id))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *Store) ListAPIKeys(ctx context.Context, projectID domain.ID) ([]*domain.APIKey, error) {
	q := `SELECT ` + apiKeyColumns + ` FROM pageguard_api_keys WHERE 1=1`
	args := []any{}
	if projectID != "" {
		args = append(args, toUUID(projectID))
		q += ` AND project_id = $1`
	}
	q += ` ORDER BY created_at, name`
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.APIKey{}
	for rows.Next() {
		k, err := s.scanAPIKey(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

func (s *Store) GetAPIKeyByHash(ctx context.Context, keyHash string) (*domain.APIKey, error) {
	return s.scanAPIKey(s.pool.QueryRow(ctx, `
		SELECT `+apiKeyColumns+` FROM pageguard_api_keys WHERE key_hash=$1 AND enabled`, keyHash))
}

func (s *Store) scanAPIKey(row pgx.Row) (*domain.APIKey, error) {
	var (
		id, projectID string
		lastUsedAt    *time.Time
		k             domain.APIKey
	)
	if err := row.Scan(&id, &k.Name, &projectID, &k.Prefix, &k.KeyHash, &k.Roles,
		&k.Enabled, &k.ExpiresAt, &lastUsedAt, &k.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	k.ID = toID(id)
	if projectID != "" {
		k.ProjectID = toID(projectID)
	}
	k.LastUsedAt = lastUsedAt
	return &k, nil
}
