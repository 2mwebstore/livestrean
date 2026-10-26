package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
)

// Config holds all runtime settings. Everything comes from environment
// variables; see .env.example in the project root.
type Config struct {
	AppEnv string
	// Public origin of the site, e.g. https://stream.example.com.
	AppURL string
	// Origins allowed to make state-changing requests and open WebSockets.
	AllowedOrigins []string

	ServerAddr   string // public API, reached through the reverse proxy
	InternalAddr string // MediaMTX auth hook, never exposed publicly

	DB DBConfig

	SessionCookie  string
	CookieSecure   bool
	SessionTTL     time.Duration
	TrustedProxies []string

	MediaMTXAPI string
	// Shown to streamers as OBS settings.
	PublicRTMPURL   string
	PublicWebRTCURL string
	// Prefixes for playback URLs given to viewers. Empty means same origin.
	MediaHLSBase    string
	MediaWebRTCBase string

	MaxChannelsPerUser int
	RateLimits         RateLimits

	AdminName     string
	AdminEmail    string
	AdminPassword string
}

type DBConfig struct {
	Host     string
	Port     string
	Name     string
	User     string
	Password string
}

// RateLimit allows Burst requests at once, refilled one every Every.
type RateLimit struct {
	Every time.Duration
	Burst int
}

type RateLimits struct {
	API       RateLimit // per IP, all API requests
	Auth      RateLimit // per IP, login and register
	Comments  RateLimit // per user, posting chat messages
	WebSocket RateLimit // per IP, opening WebSocket connections
	Uploads   RateLimit // per user, thumbnail uploads
}

func DefaultRateLimits() RateLimits {
	return RateLimits{
		API:       RateLimit{Every: 50 * time.Millisecond, Burst: 100},
		Auth:      RateLimit{Every: 12 * time.Second, Burst: 5},
		Comments:  RateLimit{Every: time.Second, Burst: 3},
		WebSocket: RateLimit{Every: 2 * time.Second, Burst: 10},
		Uploads:   RateLimit{Every: 10 * time.Second, Burst: 5},
	}
}

func (d DBConfig) DSN(multiStatements bool) string {
	c := mysql.NewConfig()
	c.User = d.User
	c.Passwd = d.Password
	c.Net = "tcp"
	c.Addr = d.Host + ":" + d.Port
	c.DBName = d.Name
	c.ParseTime = true
	c.Loc = time.UTC
	c.MultiStatements = multiStatements
	// Report matched rather than changed rows, so updates that set a value
	// to what it already was don't look like "not found".
	c.ClientFoundRows = true
	c.Params = map[string]string{"charset": "utf8mb4"}
	return c.FormatDSN()
}

func (c *Config) IsProduction() bool { return c.AppEnv == "production" }

func Load() (*Config, error) {
	appURL := strings.TrimRight(env("APP_URL", "http://localhost:8080"), "/")

	cfg := &Config{
		AppEnv:         env("APP_ENV", "development"),
		AppURL:         appURL,
		AllowedOrigins: splitList(env("ALLOWED_ORIGINS", appURL)),
		ServerAddr:     env("SERVER_ADDR", ":8080"),
		InternalAddr:   env("INTERNAL_ADDR", ":8081"),
		DB: DBConfig{
			Host:     env("DB_HOST", "localhost"),
			Port:     env("DB_PORT", "3306"),
			Name:     env("DB_NAME", "livestream"),
			User:     env("DB_USER", "livestream"),
			Password: os.Getenv("DB_PASSWORD"),
		},
		SessionCookie:   env("SESSION_COOKIE", "ls_session"),
		TrustedProxies:  splitList(env("TRUSTED_PROXIES", "10.0.0.0/8,172.16.0.0/12,192.168.0.0/16,100.64.0.0/10,fd00::/8,127.0.0.1/32")),
		MediaMTXAPI:     env("MEDIAMTX_API", "http://localhost:9997"),
		PublicRTMPURL:   strings.TrimRight(env("PUBLIC_RTMP_URL", "rtmp://localhost/live"), "/"),
		PublicWebRTCURL: strings.TrimRight(env("PUBLIC_WEBRTC_URL", appURL), "/"),
		MediaHLSBase:    strings.TrimRight(os.Getenv("MEDIA_HLS_BASE"), "/"),
		MediaWebRTCBase: strings.TrimRight(os.Getenv("MEDIA_WEBRTC_BASE"), "/"),
		RateLimits:      DefaultRateLimits(),
		AdminName:       env("ADMIN_NAME", "Admin"),
		AdminEmail:      os.Getenv("ADMIN_EMAIL"),
		AdminPassword:   os.Getenv("ADMIN_PASSWORD"),
	}

	var err error
	if cfg.CookieSecure, err = envBool("COOKIE_SECURE", strings.HasPrefix(appURL, "https://")); err != nil {
		return nil, err
	}
	if cfg.SessionTTL, err = envDuration("SESSION_TTL", 30*24*time.Hour); err != nil {
		return nil, err
	}
	if cfg.MaxChannelsPerUser, err = envInt("MAX_CHANNELS_PER_USER", 5); err != nil {
		return nil, err
	}
	if cfg.DB.Password == "" {
		return nil, fmt.Errorf("DB_PASSWORD is required")
	}
	return cfg, nil
}

func env(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func envBool(key string, fallback bool) (bool, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return false, fmt.Errorf("%s: %w", key, err)
	}
	return b, nil
}

func envInt(key string, fallback int) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}
	return n, nil
}

func envDuration(key string, fallback time.Duration) (time.Duration, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}
	return d, nil
}

func splitList(v string) []string {
	var out []string
	for _, part := range strings.Split(v, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}
