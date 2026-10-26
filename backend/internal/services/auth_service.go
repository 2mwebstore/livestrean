package services

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"livestream/internal/auth"
	"livestream/internal/dto"
	"livestream/internal/models"
	"livestream/internal/repositories"
)

// sessionTouchInterval limits last_seen_at writes to one per session per interval.
const sessionTouchInterval = 5 * time.Minute

var errUnauthenticated = unauthorized("Please log in")

type AuthService struct {
	users    repositories.UserRepository
	sessions repositories.SessionRepository
	roles    *RoleService
	audit    *AuditService
	ttl      time.Duration
}

func NewAuthService(users repositories.UserRepository, sessions repositories.SessionRepository, roles *RoleService, audit *AuditService, ttl time.Duration) *AuthService {
	return &AuthService{users: users, sessions: sessions, roles: roles, audit: audit, ttl: ttl}
}

// AuthResult is a new login: the principal plus the session token for the cookie.
type AuthResult struct {
	Principal *auth.Principal
	Token     string
	ExpiresAt time.Time
}

func (s *AuthService) Register(ctx context.Context, req dto.RegisterRequest, meta RequestMeta) (*AuthResult, error) {
	name := strings.TrimSpace(req.Name)
	if len([]rune(name)) < 2 {
		return nil, validationError("name", "Must be at least 2 characters")
	}
	if len(req.Password) > auth.MaxPasswordBytes {
		return nil, validationError("password", "Must be at most 72 bytes")
	}
	email := normalizeEmail(req.Email)

	role, ok := s.roles.BySlug(auth.RoleViewer)
	if !ok {
		return nil, fmt.Errorf("role %q missing", auth.RoleViewer)
	}
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}
	user := &models.User{Name: name, Email: email, PasswordHash: hash, RoleID: role.ID, Status: models.StatusActive}
	if err := s.users.Create(ctx, user); err != nil {
		if errors.Is(err, repositories.ErrDuplicate) {
			return nil, conflict("email", "This email is already registered")
		}
		return nil, err
	}

	s.audit.Record(ctx, AuditEntry{UserID: &user.ID, Action: AuditRegister, Resource: "user", ResourceID: idString(user.ID), Meta: meta, Success: true})
	return s.startSession(ctx, user, meta)
}

func (s *AuthService) Login(ctx context.Context, req dto.LoginRequest, meta RequestMeta) (*AuthResult, error) {
	invalid := unauthorized("Invalid email or password")

	user, err := s.users.FindByEmail(ctx, normalizeEmail(req.Email))
	if errors.Is(err, repositories.ErrNotFound) {
		auth.EqualizeTiming(req.Password)
		s.audit.Record(ctx, AuditEntry{Action: AuditFailedLogin, Resource: "user", Meta: meta})
		return nil, invalid
	}
	if err != nil {
		return nil, err
	}
	if !auth.CheckPassword(user.PasswordHash, req.Password) {
		s.audit.Record(ctx, AuditEntry{UserID: &user.ID, Action: AuditFailedLogin, Resource: "user", ResourceID: idString(user.ID), Meta: meta})
		return nil, invalid
	}
	// Only revealed after a correct password.
	if user.Status != models.StatusActive {
		s.audit.Record(ctx, AuditEntry{UserID: &user.ID, Action: AuditFailedLogin, Resource: "user", ResourceID: idString(user.ID), Meta: meta})
		return nil, forbidden("This account has been disabled")
	}

	s.audit.Record(ctx, AuditEntry{UserID: &user.ID, Action: AuditLogin, Resource: "user", ResourceID: idString(user.ID), Meta: meta, Success: true})
	return s.startSession(ctx, user, meta)
}

func (s *AuthService) Logout(ctx context.Context, p *auth.Principal, token string, meta RequestMeta) error {
	if err := s.sessions.DeleteByTokenHash(ctx, auth.HashToken(token)); err != nil {
		return err
	}
	s.audit.Record(ctx, AuditEntry{UserID: &p.UserID, Action: AuditLogout, Resource: "user", ResourceID: idString(p.UserID), Meta: meta, Success: true})
	return nil
}

// Authenticate resolves a session cookie token to a principal. Disabled
// users and expired sessions are rejected on every request.
func (s *AuthService) Authenticate(ctx context.Context, token string) (*auth.Principal, error) {
	if token == "" {
		return nil, errUnauthenticated
	}
	hash := auth.HashToken(token)
	sess, err := s.sessions.FindByTokenHash(ctx, hash)
	if errors.Is(err, repositories.ErrNotFound) {
		return nil, errUnauthenticated
	}
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	if now.After(sess.ExpiresAt) {
		_ = s.sessions.DeleteByTokenHash(ctx, hash)
		return nil, errUnauthenticated
	}
	user, err := s.users.FindByID(ctx, sess.UserID)
	if errors.Is(err, repositories.ErrNotFound) {
		return nil, errUnauthenticated
	}
	if err != nil {
		return nil, err
	}
	if user.Status != models.StatusActive {
		return nil, errUnauthenticated
	}
	if now.Sub(sess.LastSeenAt) > sessionTouchInterval {
		_ = s.sessions.Touch(ctx, sess.ID, now)
	}
	return s.principal(user, sess.ID)
}

func (s *AuthService) CleanupExpiredSessions(ctx context.Context) (int64, error) {
	return s.sessions.DeleteExpired(ctx, time.Now().UTC())
}

func (s *AuthService) startSession(ctx context.Context, user *models.User, meta RequestMeta) (*AuthResult, error) {
	token, hash := auth.NewToken("")
	now := time.Now().UTC()
	sess := &models.Session{
		UserID:     user.ID,
		TokenHash:  hash,
		IP:         truncate(meta.IP, 45),
		UserAgent:  truncate(meta.UserAgent, 255),
		ExpiresAt:  now.Add(s.ttl),
		LastSeenAt: now,
		CreatedAt:  now,
	}
	if err := s.sessions.Create(ctx, sess); err != nil {
		return nil, err
	}
	p, err := s.principal(user, sess.ID)
	if err != nil {
		return nil, err
	}
	return &AuthResult{Principal: p, Token: token, ExpiresAt: sess.ExpiresAt}, nil
}

func (s *AuthService) principal(user *models.User, sessionID uint64) (*auth.Principal, error) {
	role, ok := s.roles.ByID(user.RoleID)
	if !ok {
		return nil, fmt.Errorf("user %d has unknown role %d", user.ID, user.RoleID)
	}
	return auth.NewPrincipal(user.ID, user.Name, user.Email, role.Slug, sessionID, role.Permissions), nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func idString(id uint64) string { return strconv.FormatUint(id, 10) }
