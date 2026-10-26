package services

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"livestream/internal/auth"
	"livestream/internal/dto"
	"livestream/internal/models"
	"livestream/internal/policies"
	"livestream/internal/realtime"
	"livestream/internal/repositories"
)

const maxCommentLength = 500

type CommentService struct {
	comments repositories.CommentRepository
	channels repositories.ChannelRepository
	hub      *realtime.Hub
	audit    *AuditService
}

func NewCommentService(comments repositories.CommentRepository, channels repositories.ChannelRepository, hub *realtime.Hub, audit *AuditService) *CommentService {
	return &CommentService{comments: comments, channels: channels, hub: hub, audit: audit}
}

// List returns comments oldest first, ending just before beforeID (0 = latest).
func (s *CommentService) List(ctx context.Context, slug string, beforeID uint64, limit int) ([]dto.CommentResponse, error) {
	ch, err := s.publicChannel(ctx, slug)
	if err != nil {
		return nil, err
	}
	list, err := s.comments.ListByChannel(ctx, ch.ID, beforeID, limit)
	if err != nil {
		return nil, err
	}
	out := make([]dto.CommentResponse, len(list))
	for i, c := range list {
		out[len(list)-1-i] = toCommentResponse(&c, c.User.Name)
	}
	return out, nil
}

func (s *CommentService) Create(ctx context.Context, p *auth.Principal, slug string, req dto.CreateCommentRequest) (*dto.CommentResponse, error) {
	body := cleanCommentBody(req.Body)
	if body == "" {
		return nil, validationError("body", "Write a message first")
	}
	if utf8.RuneCountInString(body) > maxCommentLength {
		return nil, validationError("body", "Must be at most 500 characters")
	}
	ch, err := s.publicChannel(ctx, slug)
	if err != nil {
		return nil, err
	}

	c := &models.Comment{ChannelID: ch.ID, UserID: p.UserID, Body: body, CreatedAt: time.Now().UTC()}
	if err := s.comments.Create(ctx, c); err != nil {
		return nil, err
	}
	resp := toCommentResponse(c, p.Name)
	s.hub.Broadcast(ch.Slug, realtime.Event{Type: realtime.EventComment, Data: resp})
	return &resp, nil
}

func (s *CommentService) Delete(ctx context.Context, p *auth.Principal, id uint64, meta RequestMeta) error {
	c, err := s.comments.FindByID(ctx, id)
	if errors.Is(err, repositories.ErrNotFound) {
		return notFound("Comment not found")
	}
	if err != nil {
		return err
	}
	ch, err := s.channels.FindByID(ctx, c.ChannelID)
	if err != nil {
		return err
	}
	if !policies.CanDeleteComment(p, c, ch) {
		return forbidden("You can't delete this comment")
	}
	if err := s.comments.Delete(ctx, c.ID); err != nil {
		return err
	}
	s.hub.Broadcast(ch.Slug, realtime.Event{Type: realtime.EventCommentDeleted, Data: dto.CommentDeleted{ID: c.ID}})
	if c.UserID != p.UserID {
		s.audit.Record(ctx, AuditEntry{UserID: &p.UserID, Action: AuditCommentModerate, Resource: "comment", ResourceID: idString(c.ID), Meta: meta, Success: true})
	}
	return nil
}

func (s *CommentService) publicChannel(ctx context.Context, slug string) (*models.Channel, error) {
	ch, err := s.channels.FindBySlug(ctx, slug)
	if errors.Is(err, repositories.ErrNotFound) || (err == nil && !policies.IsPublic(ch)) {
		return nil, errChannelNotFound
	}
	return ch, err
}

// cleanCommentBody keeps chat to one line of printable text.
func cleanCommentBody(s string) string {
	s = strings.Map(func(r rune) rune {
		switch {
		case r == '\n' || r == '\r' || r == '\t':
			return ' '
		case unicode.IsControl(r):
			return -1
		}
		return r
	}, s)
	return strings.TrimSpace(s)
}

func toCommentResponse(c *models.Comment, userName string) dto.CommentResponse {
	return dto.CommentResponse{
		ID:        c.ID,
		Body:      c.Body,
		CreatedAt: c.CreatedAt,
		User:      dto.CommentUser{ID: c.UserID, Name: userName},
	}
}
