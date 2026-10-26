package controllers

import (
	"net/http"
	"slices"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"livestream/internal/dto"
	"livestream/internal/realtime"
	"livestream/internal/services"
)

type RealtimeController struct {
	hub      *realtime.Hub
	channels *services.ChannelService
	upgrader websocket.Upgrader
}

func NewRealtimeController(hub *realtime.Hub, channels *services.ChannelService, allowedOrigins []string) *RealtimeController {
	return &RealtimeController{
		hub:      hub,
		channels: channels,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  512,
			WriteBufferSize: 4096,
			CheckOrigin: func(r *http.Request) bool {
				origin := r.Header.Get("Origin")
				return origin == "" || slices.Contains(allowedOrigins, origin)
			},
		},
	}
}

// Channel: GET /ws/channels/:slug. Read-only event stream for one channel:
// live status, viewer count and chat. Open to everyone; posting chat
// messages goes through the authenticated REST API.
func (ctl *RealtimeController) Channel(c *gin.Context) {
	ch, err := ctl.channels.GetPublic(c.Request.Context(), c.Param("slug"))
	if err != nil {
		handleError(c, err)
		return
	}
	conn, err := ctl.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return // the upgrader already replied
	}
	status := realtime.Event{Type: realtime.EventStatus, Data: dto.LiveStatus{Live: ch.Live, StartedAt: ch.LiveStartedAt}}
	ctl.hub.Serve(conn, ch.Slug, c.ClientIP(), []realtime.Event{status})
}
