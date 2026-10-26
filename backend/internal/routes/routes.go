// Package routes registers every endpoint with its middleware. Read this
// file to see who can call what.
package routes

import (
	"github.com/gin-gonic/gin"

	"livestream/internal/auth"
	"livestream/internal/controllers"
	mw "livestream/internal/middleware"
)

type Controllers struct {
	Auth     *controllers.AuthController
	Channels *controllers.ChannelController
	Comments *controllers.CommentController
	Admin    *controllers.AdminController
	Realtime *controllers.RealtimeController
	System   *controllers.SystemController
}

type Limits struct {
	API       *mw.RateLimiter
	Auth      *mw.RateLimiter
	Comments  *mw.RateLimiter
	WebSocket *mw.RateLimiter
}

// RegisterPublic: the API served through the reverse proxy.
func RegisterPublic(r *gin.Engine, c Controllers, l Limits, loadPrincipal gin.HandlerFunc) {
	requireAuth := mw.RequireAuth()
	can := mw.RequirePermission

	r.GET("/ws/channels/:slug", mw.RateLimitByIP(l.WebSocket), c.Realtime.Channel)

	api := r.Group("/api/v1", mw.RateLimitByIP(l.API), loadPrincipal)
	api.GET("/health", c.System.Health)

	// Auth
	authLimit := mw.RateLimitByIP(l.Auth)
	api.POST("/auth/register", authLimit, c.Auth.Register)
	api.POST("/auth/login", authLimit, c.Auth.Login)
	api.POST("/auth/logout", requireAuth, c.Auth.Logout)
	api.GET("/auth/me", requireAuth, c.Auth.Me)

	// Public channels and chat (reading is open; posting needs a login)
	api.GET("/channels", c.Channels.List)
	api.GET("/channels/:slug", c.Channels.Get)
	api.GET("/channels/:slug/comments", c.Comments.List)
	api.POST("/channels/:slug/comments", requireAuth, can(auth.PermCommentsCreate), mw.RateLimitByUser(l.Comments), c.Comments.Create)
	api.DELETE("/comments/:id", requireAuth, c.Comments.Delete) // policy checked in the service

	// The logged-in streamer's own channels (ownership checked in the service)
	me := api.Group("/me/channels", requireAuth)
	me.GET("", can(auth.PermChannelsManageOwn), c.Channels.ListOwn)
	me.POST("", can(auth.PermChannelsCreate), c.Channels.Create)
	me.GET("/:id", can(auth.PermChannelsManageOwn), c.Channels.GetOwn)
	me.PUT("/:id", can(auth.PermChannelsManageOwn), c.Channels.Update)
	me.POST("/:id/stream-key", can(auth.PermChannelsManageOwn), c.Channels.RegenerateKey)
	me.DELETE("/:id", can(auth.PermChannelsManageOwn), c.Channels.Delete)

	// Admin
	admin := api.Group("/admin", requireAuth)
	admin.GET("/users", can(auth.PermUsersView), c.Admin.ListUsers)
	admin.GET("/roles", can(auth.PermUsersView), c.Admin.ListRoles)
	admin.PATCH("/users/:id/status", can(auth.PermUsersUpdate), c.Admin.UpdateUserStatus)
	admin.PATCH("/users/:id/role", can(auth.PermUsersUpdate), c.Admin.UpdateUserRole)
	admin.GET("/channels", can(auth.PermChannelsViewAll), c.Admin.ListChannels)
	admin.PATCH("/channels/:id/status", can(auth.PermChannelsUpdateAll), c.Admin.UpdateChannelStatus)
}

// RegisterInternal: endpoints only MediaMTX can reach (not proxied).
func RegisterInternal(r *gin.Engine, c Controllers) {
	r.POST("/mediamtx/auth", c.System.MediaMTXAuth)
}
