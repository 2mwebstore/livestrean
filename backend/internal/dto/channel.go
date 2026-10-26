package dto

import "time"

type CreateChannelRequest struct {
	Slug        string `json:"slug" binding:"required,slug"`
	Title       string `json:"title" binding:"required,max=100"`
	Description string `json:"description" binding:"max=1000"`
}

// UpdateChannelRequest lists the only fields an owner may change. Slug,
// owner, status and stream key are deliberately absent.
type UpdateChannelRequest struct {
	Title       string `json:"title" binding:"required,max=100"`
	Description string `json:"description" binding:"max=1000"`
}

type ChannelOwner struct {
	ID   uint64 `json:"id"`
	Name string `json:"name"`
}

type LiveStatus struct {
	Live      bool       `json:"live"`
	StartedAt *time.Time `json:"started_at"`
}

// ChannelResponse is the public view of a channel.
type ChannelResponse struct {
	Slug          string       `json:"slug"`
	Title         string       `json:"title"`
	Description   string       `json:"description"`
	Owner         ChannelOwner `json:"owner"`
	Live          bool         `json:"live"`
	LiveStartedAt *time.Time   `json:"live_started_at"`
	Viewers       int          `json:"viewers"`
	LastLiveAt    *time.Time   `json:"last_live_at"`
	ScheduledAt   *time.Time   `json:"scheduled_at"`
	ThumbnailURL  *string      `json:"thumbnail_url"`
	HLSURL        string       `json:"hls_url"`
	WHEPURL       string       `json:"whep_url"`
}

// OBSSettings tells a streamer where to point OBS. The stream key itself is
// only ever returned by StreamKeyResponse.
type OBSSettings struct {
	RTMPServer string `json:"rtmp_server"`
	WHIPURL    string `json:"whip_url"`
}

// OwnChannelResponse is the owner's dashboard view of a channel.
type OwnChannelResponse struct {
	ID            uint64      `json:"id"`
	Slug          string      `json:"slug"`
	Title         string      `json:"title"`
	Description   string      `json:"description"`
	Status        string      `json:"status"`
	Live          bool        `json:"live"`
	LiveStartedAt *time.Time  `json:"live_started_at"`
	Viewers       int         `json:"viewers"`
	LastLiveAt    *time.Time  `json:"last_live_at"`
	ScheduledAt   *time.Time  `json:"scheduled_at"`
	ThumbnailURL  *string     `json:"thumbnail_url"`
	CreatedAt     time.Time   `json:"created_at"`
	OBS           OBSSettings `json:"obs"`
}

// StreamKeyResponse is returned once, when a key is created or regenerated.
type StreamKeyResponse struct {
	Channel   OwnChannelResponse `json:"channel"`
	StreamKey string             `json:"stream_key"`
	// What to paste into OBS's "Stream Key" field for RTMP.
	RTMPStreamKey string `json:"rtmp_stream_key"`
}

type AdminChannelOwner struct {
	ID    uint64 `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type AdminChannelResponse struct {
	ID        uint64            `json:"id"`
	Slug      string            `json:"slug"`
	Title     string            `json:"title"`
	Status    string            `json:"status"`
	Owner     AdminChannelOwner `json:"owner"`
	Live      bool              `json:"live"`
	Viewers   int               `json:"viewers"`
	CreatedAt time.Time         `json:"created_at"`
}

type UpdateStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=active disabled"`
}

// UpdateScheduleRequest sets the next stream time (RFC 3339), or clears it with null.
type UpdateScheduleRequest struct {
	ScheduledAt *time.Time `json:"scheduled_at"`
}

// ScheduleStatus is pushed to viewers when the schedule changes.
type ScheduleStatus struct {
	ScheduledAt *time.Time `json:"scheduled_at"`
}
