package services

import (
	"context"
	"fmt"
	"sync"

	"livestream/internal/repositories"
)

type RoleInfo struct {
	ID          uint64
	Slug        string
	Name        string
	Permissions []string
}

// RoleService caches roles and their permissions. They only change through
// migrations, so they're loaded once at startup.
type RoleService struct {
	repo   repositories.RoleRepository
	mu     sync.RWMutex
	byID   map[uint64]RoleInfo
	bySlug map[string]RoleInfo
	all    []RoleInfo
}

func NewRoleService(repo repositories.RoleRepository) *RoleService {
	return &RoleService{repo: repo}
}

func (s *RoleService) Load(ctx context.Context) error {
	roles, err := s.repo.ListWithPermissions(ctx)
	if err != nil {
		return err
	}
	if len(roles) == 0 {
		return fmt.Errorf("no roles found; did migrations run?")
	}
	byID := make(map[uint64]RoleInfo, len(roles))
	bySlug := make(map[string]RoleInfo, len(roles))
	all := make([]RoleInfo, 0, len(roles))
	for _, r := range roles {
		info := RoleInfo{ID: r.ID, Slug: r.Slug, Name: r.Name}
		for _, p := range r.Permissions {
			info.Permissions = append(info.Permissions, p.Slug)
		}
		byID[r.ID] = info
		bySlug[r.Slug] = info
		all = append(all, info)
	}
	s.mu.Lock()
	s.byID, s.bySlug, s.all = byID, bySlug, all
	s.mu.Unlock()
	return nil
}

func (s *RoleService) ByID(id uint64) (RoleInfo, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.byID[id]
	return r, ok
}

func (s *RoleService) BySlug(slug string) (RoleInfo, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.bySlug[slug]
	return r, ok
}

func (s *RoleService) All() []RoleInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]RoleInfo(nil), s.all...)
}
