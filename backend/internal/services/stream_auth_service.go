package services

import (
	"context"
	"log/slog"
	"net/url"
	"strings"

	"livestream/internal/auth"
	"livestream/internal/dto"
	"livestream/internal/mediamtx"
	"livestream/internal/policies"
	"livestream/internal/repositories"
)

// StreamAuthService answers MediaMTX's "may this client publish/read?" hook.
type StreamAuthService struct {
	channels repositories.ChannelRepository
	audit    *AuditService
	log      *slog.Logger
}

func NewStreamAuthService(channels repositories.ChannelRepository, audit *AuditService, log *slog.Logger) *StreamAuthService {
	return &StreamAuthService{channels: channels, audit: audit, log: log}
}

func (s *StreamAuthService) Authorize(ctx context.Context, req dto.MediaMTXAuthRequest) bool {
	slug, ok := mediamtx.SlugFromPath(req.Path)
	if !ok {
		return false
	}
	ch, err := s.channels.FindBySlug(ctx, slug)
	if err != nil {
		if err != repositories.ErrNotFound {
			s.log.Error("stream auth: load channel", "channel", slug, "error", err)
		}
		return false
	}
	if !policies.IsPublic(ch) {
		return false
	}

	switch req.Action {
	case "read":
		// Channels are public to watch. Viewer restrictions (paid, private)
		// would go here.
		return true
	case "publish":
		meta := RequestMeta{IP: req.IP}
		for _, key := range PresentedKeys(req) {
			if auth.TokenMatches(key, ch.StreamKeyHash) {
				s.log.Info("publish accepted", "channel", slug, "protocol", req.Protocol, "ip", req.IP)
				s.audit.Record(ctx, AuditEntry{UserID: &ch.OwnerID, Action: AuditStreamPublish, Resource: "channel", ResourceID: idString(ch.ID), Meta: meta, Success: true})
				return true
			}
		}
		s.log.Warn("publish denied: wrong stream key", "channel", slug, "protocol", req.Protocol, "ip", req.IP)
		s.audit.Record(ctx, AuditEntry{Action: AuditStreamPublish, Resource: "channel", ResourceID: idString(ch.ID), Meta: meta})
		return false
	default:
		return false
	}
}

// PresentedKeys lists every place OBS may have put the stream key:
//
//	RTMP: Stream Key "<slug>?user=obs&pass=KEY" -> password
//	WHIP: Bearer Token "KEY"                     -> token (or "user:KEY")
func PresentedKeys(req dto.MediaMTXAuthRequest) []string {
	params, _ := url.ParseQuery(req.Query)
	candidates := []string{req.Password, req.Token, params.Get("key"), params.Get("pass"), params.Get("token")}
	if _, after, ok := strings.Cut(req.Token, ":"); ok {
		candidates = append(candidates, after)
	}
	out := candidates[:0]
	for _, c := range candidates {
		if c != "" {
			out = append(out, c)
		}
	}
	return out
}
