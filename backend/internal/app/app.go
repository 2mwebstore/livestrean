// Package app wires repositories, services, controllers and routes
// together. main.go and the integration tests both build the API here.
package app

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"livestream/config"
	"livestream/internal/controllers"
	"livestream/internal/mediamtx"
	mw "livestream/internal/middleware"
	"livestream/internal/realtime"
	"livestream/internal/repositories"
	"livestream/internal/routes"
	"livestream/internal/services"
	"livestream/internal/utils"
	"livestream/internal/validators"
	"livestream/seeds"
)

type App struct {
	Router   *gin.Engine // public API, behind the reverse proxy
	Internal *gin.Engine // MediaMTX auth hook

	cfg    *config.Config
	log    *slog.Logger
	hub    *realtime.Hub
	live   *services.LiveService
	auth   *services.AuthService
	roles  *services.RoleService
	users  repositories.UserRepository
	limits routes.Limits
}

func New(ctx context.Context, cfg *config.Config, db *gorm.DB, log *slog.Logger) (*App, error) {
	if err := validators.Register(); err != nil {
		return nil, err
	}

	// Repositories
	userRepo := repositories.NewUserRepository(db)
	sessionRepo := repositories.NewSessionRepository(db)
	channelRepo := repositories.NewChannelRepository(db)
	commentRepo := repositories.NewCommentRepository(db)
	roleRepo := repositories.NewRoleRepository(db)
	auditRepo := repositories.NewAuditRepository(db)

	// Services
	media := mediamtx.NewClient(cfg.MediaMTXAPI)
	hub := realtime.NewHub(log)
	audit := services.NewAuditService(auditRepo, log)
	roles := services.NewRoleService(roleRepo)
	if err := roles.Load(ctx); err != nil {
		return nil, err
	}
	authSvc := services.NewAuthService(userRepo, sessionRepo, roles, audit, cfg.SessionTTL)
	live := services.NewLiveService(media, hub, channelRepo, log)
	channelSvc := services.NewChannelService(channelRepo, live, hub, media, audit, services.ChannelSettings{
		MaxPerUser: cfg.MaxChannelsPerUser,
		RTMPURL:    cfg.PublicRTMPURL,
		WebRTCURL:  cfg.PublicWebRTCURL,
		HLSBase:    cfg.MediaHLSBase,
		WHEPBase:   cfg.MediaWebRTCBase,
	}, log)
	commentSvc := services.NewCommentService(commentRepo, channelRepo, hub, audit)
	userSvc := services.NewUserService(userRepo, sessionRepo, roles, channelSvc, audit)
	streamAuth := services.NewStreamAuthService(channelRepo, audit, log)

	// Controllers
	ctls := routes.Controllers{
		Auth:     controllers.NewAuthController(authSvc, controllers.CookieSettings{Name: cfg.SessionCookie, Secure: cfg.CookieSecure}),
		Channels: controllers.NewChannelController(channelSvc),
		Comments: controllers.NewCommentController(commentSvc),
		Admin:    controllers.NewAdminController(userSvc, channelSvc),
		Realtime: controllers.NewRealtimeController(hub, channelSvc, cfg.AllowedOrigins),
		System:   controllers.NewSystemController(db, streamAuth),
	}
	rl := cfg.RateLimits
	limits := routes.Limits{
		API:       mw.NewRateLimiter(rl.API.Every, rl.API.Burst),
		Auth:      mw.NewRateLimiter(rl.Auth.Every, rl.Auth.Burst),
		Comments:  mw.NewRateLimiter(rl.Comments.Every, rl.Comments.Burst),
		WebSocket: mw.NewRateLimiter(rl.WebSocket.Every, rl.WebSocket.Burst),
	}

	// Public router
	router := gin.New()
	if err := router.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		return nil, err
	}
	// Caddy sends the visitor's IP in X-Real-IP; only believed from trusted proxies.
	router.RemoteIPHeaders = []string{"X-Real-IP", "X-Forwarded-For"}
	router.Use(
		mw.RequestID(),
		mw.RequestLogger(log),
		mw.Recovery(log),
		mw.SecurityHeaders(),
		mw.OriginCheck(cfg.AllowedOrigins),
		mw.RequireJSON(),
	)
	router.NoRoute(func(c *gin.Context) { utils.Fail(c, http.StatusNotFound, "Not found", nil) })
	routes.RegisterPublic(router, ctls, limits, mw.LoadPrincipal(authSvc, cfg.SessionCookie, log))

	// Internal router
	internal := gin.New()
	internal.Use(mw.Recovery(log))
	routes.RegisterInternal(internal, ctls)

	return &App{
		Router:   router,
		Internal: internal,
		cfg:      cfg,
		log:      log,
		hub:      hub,
		live:     live,
		auth:     authSvc,
		roles:    roles,
		users:    userRepo,
		limits:   limits,
	}, nil
}

// EnsureAdmin creates the first admin from ADMIN_EMAIL / ADMIN_PASSWORD.
func (a *App) EnsureAdmin(ctx context.Context) error {
	return seeds.EnsureAdmin(ctx, a.users, a.roles, a.cfg.AdminName, a.cfg.AdminEmail, a.cfg.AdminPassword, a.log)
}

// StartBackground runs the loops the API needs until ctx is cancelled.
func (a *App) StartBackground(ctx context.Context) {
	go a.hub.Run(ctx)
	go a.live.Run(ctx, 2*time.Second)
	for _, rl := range []*mw.RateLimiter{a.limits.API, a.limits.Auth, a.limits.Comments, a.limits.WebSocket} {
		go rl.Cleanup(ctx)
	}
	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if n, err := a.auth.CleanupExpiredSessions(ctx); err != nil {
					a.log.Error("cleanup expired sessions", "error", err)
				} else if n > 0 {
					a.log.Info("expired sessions removed", "count", n)
				}
			}
		}
	}()
}
