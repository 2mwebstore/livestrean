package integration

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"livestream/config"
	"livestream/internal/app"
)

// Every test here bypasses the frontend and calls the API directly.

func TestAnonymousCannotPostComments(t *testing.T) {
	owner, _ := streamer(t)
	slug := createChannel(t, owner).Channel.Slug
	newClient(t).do("POST", "/api/v1/channels/"+slug+"/comments", map[string]string{"body": "hi"}).expect(t, http.StatusUnauthorized)
}

func TestForgedSessionCookieRejected(t *testing.T) {
	c := newClient(t)
	req, _ := http.NewRequest("GET", api.URL+"/api/v1/auth/me", nil)
	req.AddCookie(&http.Cookie{Name: "ls_session", Value: "forged-token"})
	res, err := c.http.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("forged cookie: %d", res.StatusCode)
	}
}

func TestLoggedOutSessionCannotBeReused(t *testing.T) {
	c, _ := register(t)
	var token string
	for _, ck := range c.http.Jar.Cookies(mustURL(api.URL)) {
		if ck.Name == "ls_session" {
			token = ck.Value
		}
	}
	if token == "" {
		t.Fatal("no session cookie after register")
	}
	c.do("POST", "/api/v1/auth/logout", nil).expect(t, http.StatusOK)

	req, _ := http.NewRequest("GET", api.URL+"/api/v1/auth/me", nil)
	req.AddCookie(&http.Cookie{Name: "ls_session", Value: token})
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("old session after logout: %d", res.StatusCode)
	}
}

func TestViewerCannotCreateChannels(t *testing.T) {
	viewer, _ := register(t)
	viewer.do("POST", "/api/v1/me/channels", map[string]string{"slug": unique("ch-"), "title": "x"}).expect(t, http.StatusForbidden)
	viewer.do("GET", "/api/v1/me/channels", nil).expect(t, http.StatusForbidden)
}

func TestChannelIDOR(t *testing.T) {
	a, _ := streamer(t)
	b, _ := streamer(t)
	id := createChannel(t, a).Channel.ID
	base := fmt.Sprintf("/api/v1/me/channels/%d", id)

	b.do("GET", base, nil).expect(t, http.StatusNotFound)
	b.do("PUT", base, map[string]string{"title": "hijacked"}).expect(t, http.StatusNotFound)
	b.do("POST", base+"/stream-key", nil).expect(t, http.StatusNotFound)
	b.do("DELETE", base, nil).expect(t, http.StatusNotFound)
	a.do("GET", base, nil).expect(t, http.StatusOK)
}

func TestChannelMassAssignmentIgnored(t *testing.T) {
	a, _ := streamer(t)
	_, bu := streamer(t)
	created := createChannel(t, a)

	a.do("PUT", fmt.Sprintf("/api/v1/me/channels/%d", created.Channel.ID), map[string]any{
		"title": "New title", "description": "d",
		"owner_id": bu.ID, "status": "disabled", "slug": "hacked-slug", "stream_key_hash": strings.Repeat("0", 64),
	}).expect(t, http.StatusOK)

	got := decode[ownChannel](t, a.do("GET", fmt.Sprintf("/api/v1/me/channels/%d", created.Channel.ID), nil).expect(t, http.StatusOK).Data)
	if got.Title != "New title" || got.Slug != created.Channel.Slug || got.Status != "active" {
		t.Fatalf("protected fields changed: %+v", got)
	}
	if publish(t, created.Channel.Slug, created.StreamKey) != http.StatusOK {
		t.Fatal("stream key changed through mass assignment")
	}
}

func TestAdminEndpointsRequirePermission(t *testing.T) {
	viewer, vu := register(t)
	streamerClient, _ := streamer(t)

	newClient(t).do("GET", "/api/v1/admin/users", nil).expect(t, http.StatusUnauthorized)
	viewer.do("GET", "/api/v1/admin/users", nil).expect(t, http.StatusForbidden)
	streamerClient.do("GET", "/api/v1/admin/channels", nil).expect(t, http.StatusForbidden)

	// Privilege escalation attempt
	viewer.do("PATCH", fmt.Sprintf("/api/v1/admin/users/%d/role", vu.ID), map[string]string{"role": "admin"}).expect(t, http.StatusForbidden)
	me := decode[user](t, viewer.do("GET", "/api/v1/auth/me", nil).Data)
	if me.Role != "viewer" {
		t.Fatalf("role changed to %q", me.Role)
	}

	admin := loginAdmin(t)
	res := admin.do("GET", "/api/v1/admin/users?limit=5", nil).expect(t, http.StatusOK)
	if !strings.Contains(string(res.Meta), `"total"`) || strings.Contains(res.Raw, "password") {
		t.Fatalf("admin user list: %s", res.Raw)
	}
}

func TestAdminCannotLockThemselvesOut(t *testing.T) {
	admin := loginAdmin(t)
	me := decode[user](t, admin.do("GET", "/api/v1/auth/me", nil).Data)
	admin.do("PATCH", fmt.Sprintf("/api/v1/admin/users/%d/status", me.ID), map[string]string{"status": "disabled"}).expect(t, http.StatusForbidden)
	admin.do("PATCH", fmt.Sprintf("/api/v1/admin/users/%d/role", me.ID), map[string]string{"role": "viewer"}).expect(t, http.StatusForbidden)
}

func TestDisabledUserLosesAccess(t *testing.T) {
	c, u := streamer(t)
	created := createChannel(t, c)
	admin := loginAdmin(t)

	admin.do("PATCH", fmt.Sprintf("/api/v1/admin/users/%d/status", u.ID), map[string]string{"status": "disabled"}).expect(t, http.StatusOK)
	c.do("GET", "/api/v1/auth/me", nil).expect(t, http.StatusUnauthorized)
	c.do("POST", "/api/v1/auth/login", map[string]string{"email": u.Email, "password": "password-123"}).expect(t, http.StatusForbidden)
	if publish(t, created.Channel.Slug, created.StreamKey) != http.StatusUnauthorized {
		t.Fatal("disabled user's channel must not accept streams")
	}
	newClient(t).do("GET", "/api/v1/channels/"+created.Channel.Slug, nil).expect(t, http.StatusNotFound)

	admin.do("PATCH", fmt.Sprintf("/api/v1/admin/users/%d/status", u.ID), map[string]string{"status": "active"}).expect(t, http.StatusOK)
	c.do("POST", "/api/v1/auth/login", map[string]string{"email": u.Email, "password": "password-123"}).expect(t, http.StatusOK)
}

func TestDisabledChannel(t *testing.T) {
	c, _ := streamer(t)
	created := createChannel(t, c)
	admin := loginAdmin(t)
	path := fmt.Sprintf("/api/v1/admin/channels/%d/status", created.Channel.ID)

	admin.do("PATCH", path, map[string]string{"status": "disabled"}).expect(t, http.StatusOK)
	if publish(t, created.Channel.Slug, created.StreamKey) != http.StatusUnauthorized {
		t.Fatal("disabled channel must not accept streams")
	}
	newClient(t).do("GET", "/api/v1/channels/"+created.Channel.Slug, nil).expect(t, http.StatusNotFound)
	admin.do("PATCH", path, map[string]string{"status": "bogus"}).expect(t, http.StatusUnprocessableEntity)
	admin.do("PATCH", path, map[string]string{"status": "active"}).expect(t, http.StatusOK)
}

func TestCrossSiteRequestsBlocked(t *testing.T) {
	evil := newClient(t)
	evil.origin = "https://evil.example"
	evil.do("POST", "/api/v1/auth/login", map[string]string{"email": adminEmail, "password": adminPassword}).expect(t, http.StatusForbidden)

	ok := newClient(t)
	ok.origin = "http://test.local"
	ok.do("POST", "/api/v1/auth/login", map[string]string{"email": adminEmail, "password": adminPassword}).expect(t, http.StatusOK)

	// An HTML form can only send form encoding; the API refuses it.
	res, err := http.Post(api.URL+"/api/v1/auth/login", "application/x-www-form-urlencoded", strings.NewReader("email=a&password=b"))
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusUnsupportedMediaType {
		t.Fatalf("form post: %d", res.StatusCode)
	}
}

func TestWebSocketChecks(t *testing.T) {
	if _, res, err := dialChannel(t, "no-such-channel", ""); err == nil || res == nil || res.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown channel: err=%v res=%v", err, res)
	}
	owner, _ := streamer(t)
	slug := createChannel(t, owner).Channel.Slug
	if _, res, err := dialChannel(t, slug, "https://evil.example"); err == nil || res == nil || res.StatusCode != http.StatusForbidden {
		t.Fatalf("foreign origin: err=%v res=%v", err, res)
	}
}

func TestLoginRateLimited(t *testing.T) {
	// A second API instance on the same database with strict login limits.
	cfg := *testCfg
	cfg.RateLimits = unlimited()
	cfg.RateLimits.Auth = config.RateLimit{Every: time.Hour, Burst: 3}
	db, err := config.OpenDatabase(&cfg)
	if err != nil {
		t.Fatal(err)
	}
	a, err := app.New(context.Background(), &cfg, db, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(a.Router)
	defer srv.Close()

	statuses := make([]int, 0, 4)
	for i := 0; i < 4; i++ {
		res, err := http.Post(srv.URL+"/api/v1/auth/login", "application/json", strings.NewReader(`{"email":"x@test.local","password":"wrong-password"}`))
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		statuses = append(statuses, res.StatusCode)
	}
	want := []int{401, 401, 401, 429}
	if fmt.Sprint(statuses) != fmt.Sprint(want) {
		t.Fatalf("statuses = %v, want %v", statuses, want)
	}
}
