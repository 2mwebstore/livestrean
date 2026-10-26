package services

import (
	"context"
	"log/slog"
	"time"

	"livestream/internal/models"
	"livestream/internal/repositories"
)

// Audit actions.
const (
	AuditRegister            = "register"
	AuditLogin               = "login"
	AuditFailedLogin         = "failed_login"
	AuditLogout              = "logout"
	AuditUserStatusChange    = "user_status_change"
	AuditRoleChange          = "role_change"
	AuditChannelCreate       = "channel_create"
	AuditChannelDelete       = "channel_delete"
	AuditChannelStatusChange = "channel_status_change"
	AuditStreamKeyRegenerate = "stream_key_regenerate"
	AuditStreamPublish       = "stream_publish"
	AuditCommentModerate     = "comment_moderate"
)

type AuditEntry struct {
	UserID     *uint64
	Action     string
	Resource   string
	ResourceID string
	Meta       RequestMeta
	Success    bool
}

type AuditService struct {
	repo repositories.AuditRepository
	log  *slog.Logger
}

func NewAuditService(repo repositories.AuditRepository, log *slog.Logger) *AuditService {
	return &AuditService{repo: repo, log: log}
}

// Record never fails the caller: a lost audit entry is logged, not returned.
// Never put passwords, tokens or stream keys in an entry.
func (s *AuditService) Record(ctx context.Context, e AuditEntry) {
	entry := &models.AuditLog{
		UserID:     e.UserID,
		Action:     e.Action,
		Resource:   e.Resource,
		ResourceID: truncate(e.ResourceID, 64),
		IP:         truncate(e.Meta.IP, 45),
		UserAgent:  truncate(e.Meta.UserAgent, 255),
		Success:    e.Success,
		CreatedAt:  time.Now().UTC(),
	}
	if err := s.repo.Create(context.WithoutCancel(ctx), entry); err != nil {
		s.log.Error("audit log write failed", "action", e.Action, "error", err)
	}
}
