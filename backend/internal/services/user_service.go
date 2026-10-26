package services

import (
	"context"
	"errors"
	"strings"

	"livestream/internal/auth"
	"livestream/internal/dto"
	"livestream/internal/models"
	"livestream/internal/repositories"
)

// UserService is user management for admins.
type UserService struct {
	users    repositories.UserRepository
	sessions repositories.SessionRepository
	roles    *RoleService
	channels *ChannelService
	audit    *AuditService
}

func NewUserService(users repositories.UserRepository, sessions repositories.SessionRepository, roles *RoleService, channels *ChannelService, audit *AuditService) *UserService {
	return &UserService{users: users, sessions: sessions, roles: roles, channels: channels, audit: audit}
}

var errUserNotFound = notFound("User not found")

func (s *UserService) List(ctx context.Context, query string, page, limit int) ([]dto.AdminUserResponse, int64, error) {
	list, total, err := s.users.List(ctx, repositories.UserFilter{Query: strings.TrimSpace(query), Page: page, Limit: limit})
	if err != nil {
		return nil, 0, err
	}
	out := make([]dto.AdminUserResponse, 0, len(list))
	for i := range list {
		out = append(out, s.toResponse(&list[i]))
	}
	return out, total, nil
}

func (s *UserService) Roles() []dto.RoleResponse {
	roles := s.roles.All()
	out := make([]dto.RoleResponse, 0, len(roles))
	for _, r := range roles {
		out = append(out, dto.RoleResponse{Slug: r.Slug, Name: r.Name, Permissions: r.Permissions})
	}
	return out
}

// UpdateStatus enables or disables a user. Disabling signs them out
// everywhere and stops their live streams.
func (s *UserService) UpdateStatus(ctx context.Context, actor *auth.Principal, id uint64, status string, meta RequestMeta) (*dto.AdminUserResponse, error) {
	if id == actor.UserID {
		return nil, forbidden("You can't change your own status")
	}
	user, err := s.find(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.users.UpdateStatus(ctx, id, status); err != nil {
		return nil, err
	}
	user.Status = status
	if status == models.StatusDisabled {
		if err := s.sessions.DeleteByUserID(ctx, id); err != nil {
			return nil, err
		}
		s.channels.KickOwnerStreams(ctx, id)
	}
	s.audit.Record(ctx, AuditEntry{UserID: &actor.UserID, Action: AuditUserStatusChange, Resource: "user", ResourceID: idString(id), Meta: meta, Success: true})
	resp := s.toResponse(user)
	return &resp, nil
}

func (s *UserService) UpdateRole(ctx context.Context, actor *auth.Principal, id uint64, roleSlug string, meta RequestMeta) (*dto.AdminUserResponse, error) {
	if id == actor.UserID {
		return nil, forbidden("You can't change your own role")
	}
	role, ok := s.roles.BySlug(roleSlug)
	if !ok {
		return nil, validationError("role", "Unknown role")
	}
	user, err := s.find(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.users.UpdateRole(ctx, id, role.ID); err != nil {
		return nil, err
	}
	user.RoleID = role.ID
	s.audit.Record(ctx, AuditEntry{UserID: &actor.UserID, Action: AuditRoleChange, Resource: "user", ResourceID: idString(id), Meta: meta, Success: true})
	resp := s.toResponse(user)
	return &resp, nil
}

func (s *UserService) find(ctx context.Context, id uint64) (*models.User, error) {
	user, err := s.users.FindByID(ctx, id)
	if errors.Is(err, repositories.ErrNotFound) {
		return nil, errUserNotFound
	}
	return user, err
}

func (s *UserService) toResponse(u *models.User) dto.AdminUserResponse {
	role, _ := s.roles.ByID(u.RoleID)
	return dto.AdminUserResponse{ID: u.ID, Name: u.Name, Email: u.Email, Role: role.Slug, Status: u.Status, CreatedAt: u.CreatedAt}
}
