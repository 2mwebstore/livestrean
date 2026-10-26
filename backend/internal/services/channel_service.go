package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"livestream/internal/auth"
	"livestream/internal/dto"
	"livestream/internal/mediamtx"
	"livestream/internal/models"
	"livestream/internal/policies"
	"livestream/internal/realtime"
	"livestream/internal/repositories"
	"livestream/internal/validators"
)

type PublisherKicker interface {
	KickPublisher(ctx context.Context, path string) error
}

type ChannelSettings struct {
	MaxPerUser int
	// OBS settings shown to streamers.
	RTMPURL   string // e.g. rtmp://stream.example.com/live
	WebRTCURL string // e.g. https://stream.example.com
	// Prefixes for viewer playback URLs ("" = same origin).
	HLSBase  string
	WHEPBase string
}

type ChannelService struct {
	channels repositories.ChannelRepository
	live     *LiveService
	hub      *realtime.Hub
	media    PublisherKicker
	audit    *AuditService
	settings ChannelSettings
	log      *slog.Logger
}

func NewChannelService(channels repositories.ChannelRepository, live *LiveService, hub *realtime.Hub, media PublisherKicker, audit *AuditService, settings ChannelSettings, log *slog.Logger) *ChannelService {
	return &ChannelService{channels: channels, live: live, hub: hub, media: media, audit: audit, settings: settings, log: log}
}

var errChannelNotFound = notFound("Channel not found")

// --- Public -----------------------------------------------------------------

// ListPublic lists active channels. live: nil = all, true = only live ones,
// false = only those not live right now.
func (s *ChannelService) ListPublic(ctx context.Context, live *bool, page, limit int) ([]dto.ChannelResponse, int64, error) {
	f := repositories.ChannelFilter{PublicOnly: true, Page: page, Limit: limit}
	if live != nil && *live {
		f.FilterSlugs = true
		f.Slugs = s.live.LiveSlugs()
	} else if live != nil {
		f.ExcludeSlugs = s.live.LiveSlugs()
	}
	list, total, err := s.channels.List(ctx, f)
	if err != nil {
		return nil, 0, err
	}
	out := make([]dto.ChannelResponse, 0, len(list))
	for i := range list {
		out = append(out, s.toPublic(&list[i]))
	}
	return out, total, nil
}

func (s *ChannelService) GetPublic(ctx context.Context, slug string) (*dto.ChannelResponse, error) {
	ch, err := s.channels.FindBySlug(ctx, slug)
	if errors.Is(err, repositories.ErrNotFound) {
		return nil, errChannelNotFound
	}
	if err != nil {
		return nil, err
	}
	if !policies.IsPublic(ch) {
		return nil, errChannelNotFound
	}
	resp := s.toPublic(ch)
	return &resp, nil
}

// --- Owner (dashboard) --------------------------------------------------------

func (s *ChannelService) ListOwn(ctx context.Context, p *auth.Principal) ([]dto.OwnChannelResponse, error) {
	list, err := s.channels.ListByOwner(ctx, p.UserID)
	if err != nil {
		return nil, err
	}
	out := make([]dto.OwnChannelResponse, 0, len(list))
	for i := range list {
		out = append(out, s.toOwn(&list[i]))
	}
	return out, nil
}

func (s *ChannelService) Create(ctx context.Context, p *auth.Principal, req dto.CreateChannelRequest, meta RequestMeta) (*dto.StreamKeyResponse, error) {
	slug := strings.TrimSpace(req.Slug)
	if !validators.ValidSlug(slug) {
		return nil, validationError("slug", "Use 3-40 lowercase letters, numbers or hyphens")
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return nil, validationError("title", "This field is required")
	}

	count, err := s.channels.CountByOwner(ctx, p.UserID)
	if err != nil {
		return nil, err
	}
	if int(count) >= s.settings.MaxPerUser {
		return nil, forbidden(fmt.Sprintf("You can have at most %d channels", s.settings.MaxPerUser))
	}

	key, hash := auth.NewToken(auth.StreamKeyPrefix)
	ch := &models.Channel{
		OwnerID:       p.UserID,
		Slug:          slug,
		Title:         title,
		Description:   strings.TrimSpace(req.Description),
		StreamKeyHash: hash,
		Status:        models.StatusActive,
	}
	if err := s.channels.Create(ctx, ch); err != nil {
		if errors.Is(err, repositories.ErrDuplicate) {
			return nil, conflict("slug", "This channel URL is already taken")
		}
		return nil, err
	}
	ch.Owner = models.User{ID: p.UserID, Name: p.Name, Status: models.StatusActive}

	s.audit.Record(ctx, AuditEntry{UserID: &p.UserID, Action: AuditChannelCreate, Resource: "channel", ResourceID: idString(ch.ID), Meta: meta, Success: true})
	return s.keyResponse(ch, key), nil
}

func (s *ChannelService) GetOwn(ctx context.Context, p *auth.Principal, id uint64) (*dto.OwnChannelResponse, error) {
	ch, err := s.findOwned(ctx, p, id)
	if err != nil {
		return nil, err
	}
	resp := s.toOwn(ch)
	return &resp, nil
}

func (s *ChannelService) Update(ctx context.Context, p *auth.Principal, id uint64, req dto.UpdateChannelRequest) (*dto.OwnChannelResponse, error) {
	ch, err := s.findOwned(ctx, p, id)
	if err != nil {
		return nil, err
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return nil, validationError("title", "This field is required")
	}
	ch.Title = title
	ch.Description = strings.TrimSpace(req.Description)
	if err := s.channels.UpdateDetails(ctx, ch.ID, ch.Title, ch.Description); err != nil {
		return nil, err
	}
	resp := s.toOwn(ch)
	return &resp, nil
}

// RegenerateKey replaces the stream key and disconnects anyone streaming
// with the old one.
func (s *ChannelService) RegenerateKey(ctx context.Context, p *auth.Principal, id uint64, meta RequestMeta) (*dto.StreamKeyResponse, error) {
	ch, err := s.findOwned(ctx, p, id)
	if err != nil {
		return nil, err
	}
	key, hash := auth.NewToken(auth.StreamKeyPrefix)
	if err := s.channels.UpdateStreamKeyHash(ctx, ch.ID, hash); err != nil {
		return nil, err
	}
	s.kick(ctx, ch.Slug)
	s.audit.Record(ctx, AuditEntry{UserID: &p.UserID, Action: AuditStreamKeyRegenerate, Resource: "channel", ResourceID: idString(ch.ID), Meta: meta, Success: true})
	return s.keyResponse(ch, key), nil
}

func (s *ChannelService) Delete(ctx context.Context, p *auth.Principal, id uint64, meta RequestMeta) error {
	ch, err := s.findOwned(ctx, p, id)
	if err != nil {
		return err
	}
	if err := s.channels.Delete(ctx, ch.ID); err != nil {
		return err
	}
	s.kick(ctx, ch.Slug)
	s.audit.Record(ctx, AuditEntry{UserID: &p.UserID, Action: AuditChannelDelete, Resource: "channel", ResourceID: idString(ch.ID), Meta: meta, Success: true})
	return nil
}

// findOwned loads a channel and applies the ownership policy. Channels the
// user doesn't own are reported as not found, so IDs can't be probed.
func (s *ChannelService) findOwned(ctx context.Context, p *auth.Principal, id uint64) (*models.Channel, error) {
	ch, err := s.channels.FindByID(ctx, id)
	if errors.Is(err, repositories.ErrNotFound) {
		return nil, errChannelNotFound
	}
	if err != nil {
		return nil, err
	}
	if !policies.CanManageOwnChannel(p, ch) {
		return nil, errChannelNotFound
	}
	return ch, nil
}

// --- Schedule and thumbnail -------------------------------------------------------

// maxScheduleAhead is how far ahead a stream can be scheduled.
const maxScheduleAhead = 366 * 24 * time.Hour

// SetSchedule sets (or with nil, clears) the next stream time. Viewers
// watching the channel see the countdown change immediately.
func (s *ChannelService) SetSchedule(ctx context.Context, p *auth.Principal, id uint64, at *time.Time) (*dto.OwnChannelResponse, error) {
	ch, err := s.findOwned(ctx, p, id)
	if err != nil {
		return nil, err
	}
	if at != nil {
		t := at.UTC().Truncate(time.Second)
		now := time.Now().UTC()
		if !t.After(now) {
			return nil, validationError("scheduled_at", "Pick a time in the future")
		}
		if t.After(now.Add(maxScheduleAhead)) {
			return nil, validationError("scheduled_at", "Pick a time within the next year")
		}
		at = &t
	}
	if err := s.channels.UpdateSchedule(ctx, ch.ID, at); err != nil {
		return nil, err
	}
	ch.ScheduledAt = at
	s.hub.Broadcast(ch.Slug, realtime.Event{Type: realtime.EventSchedule, Data: dto.ScheduleStatus{ScheduledAt: at}})
	resp := s.toOwn(ch)
	return &resp, nil
}

func (s *ChannelService) SetThumbnail(ctx context.Context, p *auth.Principal, id uint64, upload io.Reader) (*dto.OwnChannelResponse, error) {
	ch, err := s.findOwned(ctx, p, id)
	if err != nil {
		return nil, err
	}
	data, version, err := processThumbnail(upload)
	if err != nil {
		return nil, err
	}
	thumb := &models.ChannelThumbnail{ContentType: "image/jpeg", Data: data, UpdatedAt: time.Now().UTC()}
	if err := s.channels.SaveThumbnail(ctx, ch.ID, version, thumb); err != nil {
		return nil, err
	}
	ch.ThumbnailVersion = &version
	resp := s.toOwn(ch)
	return &resp, nil
}

func (s *ChannelService) RemoveThumbnail(ctx context.Context, p *auth.Principal, id uint64) (*dto.OwnChannelResponse, error) {
	ch, err := s.findOwned(ctx, p, id)
	if err != nil {
		return nil, err
	}
	if err := s.channels.DeleteThumbnail(ctx, ch.ID); err != nil {
		return nil, err
	}
	ch.ThumbnailVersion = nil
	resp := s.toOwn(ch)
	return &resp, nil
}

// Thumbnail returns a public channel's image and its current version.
func (s *ChannelService) Thumbnail(ctx context.Context, slug string) (*models.ChannelThumbnail, string, error) {
	ch, err := s.channels.FindBySlug(ctx, slug)
	if errors.Is(err, repositories.ErrNotFound) || (err == nil && (!policies.IsPublic(ch) || ch.ThumbnailVersion == nil)) {
		return nil, "", notFound("No thumbnail")
	}
	if err != nil {
		return nil, "", err
	}
	thumb, err := s.channels.FindThumbnail(ctx, ch.ID)
	if errors.Is(err, repositories.ErrNotFound) {
		return nil, "", notFound("No thumbnail")
	}
	return thumb, *ch.ThumbnailVersion, err
}

// thumbnailURL includes the version, so browsers can cache each image forever.
func thumbnailURL(ch *models.Channel) *string {
	if ch.ThumbnailVersion == nil {
		return nil
	}
	u := "/api/v1/channels/" + ch.Slug + "/thumbnail?v=" + *ch.ThumbnailVersion
	return &u
}

// --- Admin ----------------------------------------------------------------------

func (s *ChannelService) ListAll(ctx context.Context, query string, page, limit int) ([]dto.AdminChannelResponse, int64, error) {
	list, total, err := s.channels.List(ctx, repositories.ChannelFilter{Query: strings.TrimSpace(query), Page: page, Limit: limit})
	if err != nil {
		return nil, 0, err
	}
	out := make([]dto.AdminChannelResponse, 0, len(list))
	for i := range list {
		out = append(out, s.toAdmin(&list[i]))
	}
	return out, total, nil
}

func (s *ChannelService) SetStatus(ctx context.Context, actor *auth.Principal, id uint64, status string, meta RequestMeta) (*dto.AdminChannelResponse, error) {
	ch, err := s.channels.FindByID(ctx, id)
	if errors.Is(err, repositories.ErrNotFound) {
		return nil, errChannelNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := s.channels.UpdateStatus(ctx, ch.ID, status); err != nil {
		return nil, err
	}
	ch.Status = status
	if status == models.StatusDisabled {
		s.kick(ctx, ch.Slug)
	}
	s.audit.Record(ctx, AuditEntry{UserID: &actor.UserID, Action: AuditChannelStatusChange, Resource: "channel", ResourceID: idString(ch.ID), Meta: meta, Success: true})
	resp := s.toAdmin(ch)
	return &resp, nil
}

// KickOwnerStreams disconnects every stream of a user, e.g. when they're disabled.
func (s *ChannelService) KickOwnerStreams(ctx context.Context, ownerID uint64) {
	list, err := s.channels.ListByOwner(ctx, ownerID)
	if err != nil {
		s.log.Error("list channels to kick", "owner", ownerID, "error", err)
		return
	}
	for _, ch := range list {
		s.kick(ctx, ch.Slug)
	}
}

func (s *ChannelService) kick(ctx context.Context, slug string) {
	if err := s.media.KickPublisher(ctx, mediamtx.PathForSlug(slug)); err != nil {
		s.log.Error("kick publisher", "channel", slug, "error", err)
	}
}

// --- Mapping ----------------------------------------------------------------

func (s *ChannelService) toPublic(ch *models.Channel) dto.ChannelResponse {
	state := s.live.Get(ch.Slug)
	path := mediamtx.PathForSlug(ch.Slug)
	return dto.ChannelResponse{
		Slug:          ch.Slug,
		Title:         ch.Title,
		Description:   ch.Description,
		Owner:         dto.ChannelOwner{ID: ch.OwnerID, Name: ch.Owner.Name},
		Live:          state.Live,
		LiveStartedAt: state.StartedAt,
		Viewers:       s.hub.ViewerCount(ch.Slug),
		LastLiveAt:    ch.LastLiveAt,
		ScheduledAt:   ch.ScheduledAt,
		ThumbnailURL:  thumbnailURL(ch),
		HLSURL:        s.settings.HLSBase + "/" + path + "/index.m3u8",
		WHEPURL:       s.settings.WHEPBase + "/" + path + "/whep",
	}
}

func (s *ChannelService) toOwn(ch *models.Channel) dto.OwnChannelResponse {
	state := s.live.Get(ch.Slug)
	return dto.OwnChannelResponse{
		ID:            ch.ID,
		Slug:          ch.Slug,
		Title:         ch.Title,
		Description:   ch.Description,
		Status:        ch.Status,
		Live:          state.Live,
		LiveStartedAt: state.StartedAt,
		Viewers:       s.hub.ViewerCount(ch.Slug),
		LastLiveAt:    ch.LastLiveAt,
		ScheduledAt:   ch.ScheduledAt,
		ThumbnailURL:  thumbnailURL(ch),
		CreatedAt:     ch.CreatedAt,
		OBS: dto.OBSSettings{
			RTMPServer: s.settings.RTMPURL,
			WHIPURL:    s.settings.WebRTCURL + "/" + mediamtx.PathForSlug(ch.Slug) + "/whip",
		},
	}
}

func (s *ChannelService) toAdmin(ch *models.Channel) dto.AdminChannelResponse {
	return dto.AdminChannelResponse{
		ID:        ch.ID,
		Slug:      ch.Slug,
		Title:     ch.Title,
		Status:    ch.Status,
		Owner:     dto.AdminChannelOwner{ID: ch.OwnerID, Name: ch.Owner.Name, Email: ch.Owner.Email},
		Live:      s.live.Get(ch.Slug).Live,
		Viewers:   s.hub.ViewerCount(ch.Slug),
		CreatedAt: ch.CreatedAt,
	}
}

func (s *ChannelService) keyResponse(ch *models.Channel, key string) *dto.StreamKeyResponse {
	return &dto.StreamKeyResponse{
		Channel:   s.toOwn(ch),
		StreamKey: key,
		// OBS: Server = rtmp://host/live, Stream Key = <slug>?user=obs&pass=<key>
		RTMPStreamKey: ch.Slug + "?user=obs&pass=" + key,
	}
}
