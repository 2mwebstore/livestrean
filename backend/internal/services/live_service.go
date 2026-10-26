package services

import (
	"context"
	"log/slog"
	"sort"
	"sync"
	"time"

	"livestream/internal/dto"
	"livestream/internal/mediamtx"
	"livestream/internal/realtime"
	"livestream/internal/repositories"
)

// failuresBeforeOffline is how many failed polls in a row (MediaMTX down)
// before every channel is shown as offline.
const failuresBeforeOffline = 3

type PathLister interface {
	ListReadyPaths(ctx context.Context) (map[string]time.Time, error)
}

type LiveState struct {
	Live      bool
	StartedAt *time.Time
}

// LiveService tracks which channels are live by polling MediaMTX, and
// pushes changes to viewers.
type LiveService struct {
	media    PathLister
	hub      *realtime.Hub
	channels repositories.ChannelRepository
	log      *slog.Logger

	mu       sync.RWMutex
	live     map[string]time.Time // slug -> went live at
	failures int
}

func NewLiveService(media PathLister, hub *realtime.Hub, channels repositories.ChannelRepository, log *slog.Logger) *LiveService {
	return &LiveService{media: media, hub: hub, channels: channels, log: log, live: make(map[string]time.Time)}
}

func (s *LiveService) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		s.Poll(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *LiveService) Poll(ctx context.Context) {
	paths, err := s.media.ListReadyPaths(ctx)
	if err != nil {
		s.failures++
		s.log.Warn("could not reach MediaMTX", "error", err, "failures", s.failures)
		if s.failures < failuresBeforeOffline {
			return
		}
		paths = nil
	} else {
		s.failures = 0
	}

	current := make(map[string]time.Time, len(paths))
	for path, started := range paths {
		if slug, ok := mediamtx.SlugFromPath(path); ok {
			current[slug] = started
		}
	}

	s.mu.Lock()
	previous := s.live
	s.live = current
	s.mu.Unlock()

	for slug, started := range current {
		if _, was := previous[slug]; !was {
			s.changed(ctx, slug, LiveState{Live: true, StartedAt: &started})
		}
	}
	for slug := range previous {
		if _, still := current[slug]; !still {
			s.changed(ctx, slug, LiveState{})
		}
	}
}

func (s *LiveService) changed(ctx context.Context, slug string, state LiveState) {
	s.log.Info("channel live status changed", "channel", slug, "live", state.Live)
	s.hub.Broadcast(slug, realtime.Event{Type: realtime.EventStatus, Data: dto.LiveStatus{Live: state.Live, StartedAt: state.StartedAt}})
	if err := s.channels.TouchLastLive(ctx, slug, time.Now().UTC()); err != nil {
		s.log.Error("update last_live_at", "channel", slug, "error", err)
	}
}

func (s *LiveService) Get(slug string) LiveState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	started, ok := s.live[slug]
	if !ok {
		return LiveState{}
	}
	return LiveState{Live: true, StartedAt: &started}
}

func (s *LiveService) LiveSlugs() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]string, 0, len(s.live))
	for slug := range s.live {
		out = append(out, slug)
	}
	sort.Strings(out)
	return out
}
