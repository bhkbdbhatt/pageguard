package memory

import (
	"context"
	"sort"

	"github.com/pageguard/pageguard/internal/domain"
)

func (s *Store) CreateAPIKey(_ context.Context, k *domain.APIKey) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *k
	if cp.Roles == nil {
		cp.Roles = []string{}
	}
	s.apiKeys[k.ID] = &cp
	return nil
}

func (s *Store) GetAPIKey(_ context.Context, id domain.ID) (*domain.APIKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.apiKeys[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *v
	return &cp, nil
}

func (s *Store) UpdateAPIKey(_ context.Context, k *domain.APIKey) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.apiKeys[k.ID]; !ok {
		return domain.ErrNotFound
	}
	cp := *k
	s.apiKeys[k.ID] = &cp
	return nil
}

func (s *Store) DeleteAPIKey(_ context.Context, id domain.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.apiKeys[id]; !ok {
		return domain.ErrNotFound
	}
	delete(s.apiKeys, id)
	return nil
}

func (s *Store) ListAPIKeys(_ context.Context, projectID domain.ID) ([]*domain.APIKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []*domain.APIKey{}
	for _, v := range s.apiKeys {
		if projectID != "" && v.ProjectID != projectID {
			continue
		}
		cp := *v
		out = append(out, &cp)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}

func (s *Store) GetAPIKeyByHash(_ context.Context, keyHash string) (*domain.APIKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, v := range s.apiKeys {
		if v.KeyHash == keyHash {
			cp := *v
			return &cp, nil
		}
	}
	return nil, domain.ErrNotFound
}
