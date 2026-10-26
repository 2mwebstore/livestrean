// Package integration exercises the full API (router, middleware,
// controllers, services, repositories) against a real MySQL database and
// a fake MediaMTX. Run with: docker compose --profile test run --rm backend-test
package integration

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"livestream/config"
	"livestream/internal/app"
	"livestream/migrations"
)

const (
	adminEmail    = "admin@test.local"
	adminPassword = "admin-password-123"
	maxChannels   = 3
)

var (
	api      *httptest.Server // public API
	internal *httptest.Server // MediaMTX auth hook
	media    *fakeMediaMTX
	testCfg  *config.Config
)

func TestMain(m *testing.M) {
	if os.Getenv("TEST_DB_HOST") == "" {
		fmt.Println("TEST_DB_HOST not set; skipping integration tests")
		os.Exit(0)
	}
	gin.SetMode(gin.TestMode)
	code, err := setup(m)
	if err != nil {
		fmt.Fprintln(os.Stderr, "integration setup:", err)
		os.Exit(1)
	}
	os.Exit(code)
}

func setup(m *testing.M) (int, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	media = newFakeMediaMTX()
	mediaSrv := httptest.NewServer(media)
	defer mediaSrv.Close()

	testCfg = &config.Config{
		AppEnv:         "test",
		AppURL:         "http://test.local",
		AllowedOrigins: []string{"http://test.local"},
		DB: config.DBConfig{
			Host:     os.Getenv("TEST_DB_HOST"),
			Port:     envOr("TEST_DB_PORT", "3306"),
			Name:     os.Getenv("TEST_DB_NAME"),
			User:     os.Getenv("TEST_DB_USER"),
			Password: os.Getenv("TEST_DB_PASSWORD"),
		},
		SessionCookie:      "ls_session",
		SessionTTL:         time.Hour,
		MediaMTXAPI:        mediaSrv.URL,
		PublicRTMPURL:      "rtmp://test.local/live",
		PublicWebRTCURL:    "http://test.local",
		MaxChannelsPerUser: maxChannels,
		RateLimits:         unlimited(),
		AdminName:          "Admin",
		AdminEmail:         adminEmail,
		AdminPassword:      adminPassword,
	}

	if err := resetDatabase(ctx, testCfg); err != nil {
		return 0, err
	}
	if err := config.RunMigrations(ctx, testCfg, migrations.Files, log); err != nil {
		return 0, err
	}
	db, err := config.OpenDatabase(testCfg)
	if err != nil {
		return 0, err
	}
	a, err := app.New(ctx, testCfg, db, log)
	if err != nil {
		return 0, err
	}
	if err := a.EnsureAdmin(ctx); err != nil {
		return 0, err
	}
	a.StartBackground(ctx)

	api = httptest.NewServer(a.Router)
	defer api.Close()
	internal = httptest.NewServer(a.Internal)
	defer internal.Close()

	return m.Run(), nil
}

func unlimited() config.RateLimits {
	free := config.RateLimit{Every: time.Microsecond, Burst: 100000}
	return config.RateLimits{API: free, Auth: free, Comments: free, WebSocket: free}
}

// resetDatabase recreates the test database so each run starts from the
// migrations. The test runner connects as MySQL root (test profile only).
func resetDatabase(ctx context.Context, cfg *config.Config) error {
	server := cfg.DB
	server.Name = ""
	db, err := sql.Open("mysql", server.DSN(false))
	if err != nil {
		return err
	}
	defer db.Close()
	name := "`" + strings.ReplaceAll(cfg.DB.Name, "`", "") + "`"
	if _, err := db.ExecContext(ctx, "DROP DATABASE IF EXISTS "+name); err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, "CREATE DATABASE "+name)
	return err
}

func mustURL(raw string) *url.URL {
	u, err := url.Parse(raw)
	if err != nil {
		panic(err)
	}
	return u
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// --- Fake MediaMTX API -----------------------------------------------------------

type fakeMediaMTX struct {
	mu     sync.Mutex
	live   map[string]string // path -> publisher connection id
	kicked []string
}

func newFakeMediaMTX() *fakeMediaMTX { return &fakeMediaMTX{live: map[string]string{}} }

func (f *fakeMediaMTX) setLive(path, connID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if connID == "" {
		delete(f.live, path)
	} else {
		f.live[path] = connID
	}
}

func (f *fakeMediaMTX) wasKicked(connID string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, k := range f.kicked {
		if k == connID {
			return true
		}
	}
	return false
}

func (f *fakeMediaMTX) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	switch {
	case r.URL.Path == "/v3/paths/list":
		items := []map[string]any{}
		for path, id := range f.live {
			items = append(items, map[string]any{
				"name": path, "ready": true, "readyTime": time.Now().UTC().Format(time.RFC3339Nano),
				"source": map[string]string{"type": "rtmpConn", "id": id},
			})
		}
		json.NewEncoder(w).Encode(map[string]any{"pageCount": 1, "items": items})
	case strings.HasPrefix(r.URL.Path, "/v3/paths/get/"):
		path := strings.TrimPrefix(r.URL.Path, "/v3/paths/get/")
		id, ok := f.live[path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"name": path, "ready": true, "source": map[string]string{"type": "rtmpConn", "id": id}})
	case strings.HasPrefix(r.URL.Path, "/v3/rtmpconns/kick/"):
		f.kicked = append(f.kicked, strings.TrimPrefix(r.URL.Path, "/v3/rtmpconns/kick/"))
	default:
		http.NotFound(w, r)
	}
}

// --- API client helpers -------------------------------------------------------------

type client struct {
	t      *testing.T
	http   *http.Client
	origin string
}

func newClient(t *testing.T) *client {
	jar, _ := cookiejar.New(nil)
	return &client{t: t, http: &http.Client{Jar: jar}}
}

type response struct {
	Status  int
	Success bool              `json:"success"`
	Message string            `json:"message"`
	Data    json.RawMessage   `json:"data"`
	Errors  map[string]string `json:"errors"`
	Meta    json.RawMessage   `json:"meta"`
	Raw     string
}

func (c *client) do(method, path string, body any) response {
	c.t.Helper()
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			c.t.Fatal(err)
		}
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, api.URL+path, r)
	if err != nil {
		c.t.Fatal(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.origin != "" {
		req.Header.Set("Origin", c.origin)
	}
	res, err := c.http.Do(req)
	if err != nil {
		c.t.Fatal(err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	out := response{Status: res.StatusCode, Raw: string(raw)}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			c.t.Fatalf("%s %s: invalid JSON %q", method, path, raw)
		}
	}
	return out
}

// expect fails the test unless the status matches.
func (r response) expect(t *testing.T, status int) response {
	t.Helper()
	if r.Status != status {
		t.Fatalf("status = %d, want %d; body: %s", r.Status, status, r.Raw)
	}
	return r
}

func decode[T any](t *testing.T, raw json.RawMessage) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatalf("decode %s: %v", raw, err)
	}
	return v
}

var seq atomic.Int64

func unique(prefix string) string {
	return fmt.Sprintf("%s%d%d", prefix, time.Now().UnixNano()%100000, seq.Add(1))
}

type user struct {
	ID          uint64   `json:"id"`
	Name        string   `json:"name"`
	Email       string   `json:"email"`
	Role        string   `json:"role"`
	Permissions []string `json:"permissions"`
}

// register signs up a new viewer and returns a logged-in client.
func register(t *testing.T) (*client, user) {
	t.Helper()
	c := newClient(t)
	res := c.do("POST", "/api/v1/auth/register", map[string]string{
		"name": "User " + unique(""), "email": unique("user") + "@test.local", "password": "password-123",
	}).expect(t, http.StatusCreated)
	return c, decode[user](t, res.Data)
}

func loginAdmin(t *testing.T) *client {
	t.Helper()
	c := newClient(t)
	c.do("POST", "/api/v1/auth/login", map[string]string{"email": adminEmail, "password": adminPassword}).expect(t, http.StatusOK)
	return c
}

// streamer signs up a user and has the admin promote them.
func streamer(t *testing.T) (*client, user) {
	t.Helper()
	c, u := register(t)
	loginAdmin(t).do("PATCH", fmt.Sprintf("/api/v1/admin/users/%d/role", u.ID), map[string]string{"role": "streamer"}).expect(t, http.StatusOK)
	return c, u
}

type ownChannel struct {
	ID     uint64 `json:"id"`
	Slug   string `json:"slug"`
	Title  string `json:"title"`
	Status string `json:"status"`
	Live   bool   `json:"live"`
	OBS    struct {
		RTMPServer string `json:"rtmp_server"`
		WHIPURL    string `json:"whip_url"`
	} `json:"obs"`
}

type streamKeyResponse struct {
	Channel       ownChannel `json:"channel"`
	StreamKey     string     `json:"stream_key"`
	RTMPStreamKey string     `json:"rtmp_stream_key"`
}

func createChannel(t *testing.T, c *client) streamKeyResponse {
	t.Helper()
	res := c.do("POST", "/api/v1/me/channels", map[string]string{
		"slug": unique("ch-"), "title": "Test channel", "description": "desc",
	}).expect(t, http.StatusCreated)
	return decode[streamKeyResponse](t, res.Data)
}

// hook calls the MediaMTX auth endpoint and returns the status code.
func hook(t *testing.T, body map[string]string) int {
	t.Helper()
	b, _ := json.Marshal(body)
	res, err := http.Post(internal.URL+"/mediamtx/auth", "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	return res.StatusCode
}

func publish(t *testing.T, slug, key string) int {
	return hook(t, map[string]string{"action": "publish", "path": "live/" + slug, "protocol": "rtmp", "user": "obs", "password": key})
}
