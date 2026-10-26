package integration

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestRegisterLoginLogout(t *testing.T) {
	c, u := register(t)
	if u.Role != "viewer" || !slices.Contains(u.Permissions, "comments.create") || slices.Contains(u.Permissions, "channels.create") {
		t.Fatalf("new user should be a viewer who can chat but not stream: %+v", u)
	}
	c.do("GET", "/api/v1/auth/me", nil).expect(t, http.StatusOK)

	dup := newClient(t).do("POST", "/api/v1/auth/register", map[string]string{"name": "Dup", "email": strings.ToUpper(u.Email), "password": "password-123"})
	dup.expect(t, http.StatusConflict)
	if dup.Errors["email"] == "" {
		t.Errorf("duplicate email should report errors.email: %s", dup.Raw)
	}

	c.do("POST", "/api/v1/auth/logout", nil).expect(t, http.StatusOK)
	c.do("GET", "/api/v1/auth/me", nil).expect(t, http.StatusUnauthorized)

	c.do("POST", "/api/v1/auth/login", map[string]string{"email": u.Email, "password": "wrong-password"}).expect(t, http.StatusUnauthorized)
	c.do("POST", "/api/v1/auth/login", map[string]string{"email": u.Email, "password": "password-123"}).expect(t, http.StatusOK)
	c.do("GET", "/api/v1/auth/me", nil).expect(t, http.StatusOK)
}

func TestRegisterValidation(t *testing.T) {
	res := newClient(t).do("POST", "/api/v1/auth/register", map[string]string{"name": "A", "email": "not-an-email", "password": "short"})
	res.expect(t, http.StatusUnprocessableEntity)
	for _, field := range []string{"name", "email", "password"} {
		if res.Errors[field] == "" {
			t.Errorf("missing error for %s: %s", field, res.Raw)
		}
	}
}

func TestChannelLifecycle(t *testing.T) {
	c, u := streamer(t)
	created := createChannel(t, c)

	if !strings.HasPrefix(created.StreamKey, "sk_") {
		t.Fatalf("stream key %q", created.StreamKey)
	}
	if created.RTMPStreamKey != created.Channel.Slug+"?user=obs&pass="+created.StreamKey {
		t.Fatalf("rtmp stream key %q", created.RTMPStreamKey)
	}
	if created.Channel.OBS.RTMPServer != "rtmp://test.local/live" || created.Channel.OBS.WHIPURL != "http://test.local/live/"+created.Channel.Slug+"/whip" {
		t.Fatalf("obs settings %+v", created.Channel.OBS)
	}

	// The key is never shown again.
	own := c.do("GET", fmt.Sprintf("/api/v1/me/channels/%d", created.Channel.ID), nil).expect(t, http.StatusOK)
	if strings.Contains(own.Raw, created.StreamKey) || strings.Contains(own.Raw, "stream_key") {
		t.Fatalf("stream key leaked: %s", own.Raw)
	}

	// Public view
	pub := newClient(t).do("GET", "/api/v1/channels/"+created.Channel.Slug, nil).expect(t, http.StatusOK)
	ch := decode[struct {
		Owner  struct{ ID uint64 } `json:"owner"`
		HLSURL string              `json:"hls_url"`
	}](t, pub.Data)
	if ch.Owner.ID != u.ID || ch.HLSURL != "/live/"+created.Channel.Slug+"/index.m3u8" {
		t.Fatalf("public channel %s", pub.Raw)
	}
	if strings.Contains(pub.Raw, "stream_key") || strings.Contains(pub.Raw, u.Email) {
		t.Fatalf("public channel leaks private data: %s", pub.Raw)
	}

	c.do("PUT", fmt.Sprintf("/api/v1/me/channels/%d", created.Channel.ID), map[string]string{"title": "Renamed", "description": ""}).expect(t, http.StatusOK)
	list := decode[[]ownChannel](t, c.do("GET", "/api/v1/me/channels", nil).expect(t, http.StatusOK).Data)
	if len(list) != 1 || list[0].Title != "Renamed" {
		t.Fatalf("own channels %+v", list)
	}

	c.do("DELETE", fmt.Sprintf("/api/v1/me/channels/%d", created.Channel.ID), nil).expect(t, http.StatusOK)
	newClient(t).do("GET", "/api/v1/channels/"+created.Channel.Slug, nil).expect(t, http.StatusNotFound)
}

func TestChannelSlugRulesAndLimit(t *testing.T) {
	c, _ := streamer(t)
	first := createChannel(t, c)

	c.do("POST", "/api/v1/me/channels", map[string]string{"slug": first.Channel.Slug, "title": "x"}).expect(t, http.StatusConflict)
	c.do("POST", "/api/v1/me/channels", map[string]string{"slug": "Bad Slug!", "title": "x"}).expect(t, http.StatusUnprocessableEntity)
	c.do("POST", "/api/v1/me/channels", map[string]string{"slug": "admin", "title": "x"}).expect(t, http.StatusUnprocessableEntity)

	for i := 1; i < maxChannels; i++ {
		createChannel(t, c)
	}
	c.do("POST", "/api/v1/me/channels", map[string]string{"slug": unique("ch-"), "title": "x"}).expect(t, http.StatusForbidden)
}

func TestStreamKeyAuthorization(t *testing.T) {
	c, _ := streamer(t)
	created := createChannel(t, c)
	slug, key := created.Channel.Slug, created.StreamKey

	if got := publish(t, slug, key); got != http.StatusOK {
		t.Fatalf("publish with key: %d", got)
	}
	if got := publish(t, slug, "sk_wrong"); got != http.StatusUnauthorized {
		t.Fatalf("publish with wrong key: %d", got)
	}
	if got := publish(t, slug, ""); got != http.StatusUnauthorized {
		t.Fatalf("publish without key: %d", got)
	}
	// OBS WHIP sends the key as a Bearer token.
	if got := hook(t, map[string]string{"action": "publish", "path": "live/" + slug, "protocol": "webrtc", "token": key}); got != http.StatusOK {
		t.Fatalf("whip publish: %d", got)
	}
	// Another channel's key doesn't work here.
	other := createChannel(t, c)
	if got := publish(t, slug, other.StreamKey); got != http.StatusUnauthorized {
		t.Fatalf("publish with another channel's key: %d", got)
	}
	if got := hook(t, map[string]string{"action": "read", "path": "live/" + slug, "protocol": "hls"}); got != http.StatusOK {
		t.Fatalf("read: %d", got)
	}
	for _, path := range []string{"live/does-not-exist", "live/" + slug + "/x", slug, "other/" + slug} {
		if got := hook(t, map[string]string{"action": "read", "path": path}); got != http.StatusUnauthorized {
			t.Errorf("read %q: %d, want 401", path, got)
		}
	}
	if got := hook(t, map[string]string{"action": "playback", "path": "live/" + slug, "password": key}); got != http.StatusUnauthorized {
		t.Errorf("playback: %d, want 401", got)
	}

	// Regenerating the key disconnects the current stream and invalidates the old key.
	media.setLive("live/"+slug, "conn-"+slug)
	defer media.setLive("live/"+slug, "")
	regen := decode[streamKeyResponse](t, c.do("POST", fmt.Sprintf("/api/v1/me/channels/%d/stream-key", created.Channel.ID), nil).expect(t, http.StatusOK).Data)
	if !media.wasKicked("conn-" + slug) {
		t.Error("regenerating the key should kick the live publisher")
	}
	if publish(t, slug, key) != http.StatusUnauthorized || publish(t, slug, regen.StreamKey) != http.StatusOK {
		t.Fatal("old key must stop working and the new one must work")
	}
}

func TestComments(t *testing.T) {
	owner, _ := streamer(t)
	slug := createChannel(t, owner).Channel.Slug
	path := "/api/v1/channels/" + slug + "/comments"
	viewer, vu := register(t)

	posted := decode[struct {
		ID   uint64              `json:"id"`
		Body string              `json:"body"`
		User struct{ ID uint64 } `json:"user"`
	}](t, viewer.do("POST", path, map[string]string{"body": "  hello\nworld  "}).expect(t, http.StatusCreated).Data)
	if posted.Body != "hello world" || posted.User.ID != vu.ID {
		t.Fatalf("posted %+v", posted)
	}

	viewer.do("POST", path, map[string]string{"body": "   "}).expect(t, http.StatusUnprocessableEntity)
	viewer.do("POST", path, map[string]string{"body": strings.Repeat("a", 501)}).expect(t, http.StatusUnprocessableEntity)
	viewer.do("POST", "/api/v1/channels/no-such-channel/comments", map[string]string{"body": "hi"}).expect(t, http.StatusNotFound)

	list := decode[[]struct {
		Body string `json:"body"`
		User struct {
			Name string `json:"name"`
		} `json:"user"`
	}](t, newClient(t).do("GET", path, nil).expect(t, http.StatusOK).Data)
	if len(list) != 1 || list[0].Body != "hello world" || list[0].User.Name == "" {
		t.Fatalf("comments %+v", list)
	}

	// Another viewer can't delete it; the channel owner can.
	other, _ := register(t)
	other.do("DELETE", fmt.Sprintf("/api/v1/comments/%d", posted.ID), nil).expect(t, http.StatusForbidden)
	owner.do("DELETE", fmt.Sprintf("/api/v1/comments/%d", posted.ID), nil).expect(t, http.StatusOK)
	if got := decode[[]any](t, newClient(t).do("GET", path, nil).Data); len(got) != 0 {
		t.Fatalf("deleted comment still listed: %v", got)
	}
}

type wsEvent struct {
	Type string         `json:"type"`
	Data map[string]any `json:"data"`
}

func dialChannel(t *testing.T, slug string, origin string) (*websocket.Conn, *http.Response, error) {
	header := http.Header{}
	if origin != "" {
		header.Set("Origin", origin)
	}
	return websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(api.URL, "http")+"/ws/channels/"+slug, header)
}

// nextEvent waits for an event of the given type, skipping others.
func nextEvent(t *testing.T, conn *websocket.Conn, typ string, timeout time.Duration) map[string]any {
	t.Helper()
	conn.SetReadDeadline(time.Now().Add(timeout))
	for {
		var e wsEvent
		if err := conn.ReadJSON(&e); err != nil {
			t.Fatalf("waiting for %q event: %v", typ, err)
		}
		if e.Type == typ {
			return e.Data
		}
	}
}

func TestRealtimeViewersStatusAndChat(t *testing.T) {
	owner, _ := streamer(t)
	slug := createChannel(t, owner).Channel.Slug

	a, _, err := dialChannel(t, slug, "http://test.local")
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	if live := nextEvent(t, a, "status", 3*time.Second)["live"]; live != false {
		t.Fatalf("initial live = %v", live)
	}
	if n := nextEvent(t, a, "viewers", 3*time.Second)["count"]; n != 1.0 {
		t.Fatalf("initial viewers = %v", n)
	}

	b, _, err := dialChannel(t, slug, "")
	if err != nil {
		t.Fatal(err)
	}
	if n := nextEvent(t, a, "viewers", 3*time.Second)["count"]; n != 2.0 {
		t.Fatalf("viewers after second join = %v", n)
	}

	// Going live in MediaMTX reaches viewers and the API.
	media.setLive("live/"+slug, "conn-rt")
	if live := nextEvent(t, a, "status", 6*time.Second)["live"]; live != true {
		t.Fatalf("live = %v", live)
	}
	pub := decode[struct {
		Live    bool `json:"live"`
		Viewers int  `json:"viewers"`
	}](t, newClient(t).do("GET", "/api/v1/channels/"+slug, nil).Data)
	if !pub.Live || pub.Viewers != 2 {
		t.Fatalf("channel %+v, want live with 2 viewers", pub)
	}
	liveList := newClient(t).do("GET", "/api/v1/channels?live=true", nil).expect(t, http.StatusOK)
	if !strings.Contains(liveList.Raw, `"slug":"`+slug+`"`) {
		t.Fatalf("live list missing channel: %s", liveList.Raw)
	}

	// Chat messages are pushed to everyone in the room.
	viewer, _ := register(t)
	viewer.do("POST", "/api/v1/channels/"+slug+"/comments", map[string]string{"body": "realtime!"}).expect(t, http.StatusCreated)
	if body := nextEvent(t, b, "comment", 3*time.Second)["body"]; body != "realtime!" {
		t.Fatalf("comment body = %v", body)
	}

	media.setLive("live/"+slug, "")
	if live := nextEvent(t, a, "status", 6*time.Second)["live"]; live != false {
		t.Fatalf("live after stop = %v", live)
	}

	b.Close()
	if n := nextEvent(t, a, "viewers", 3*time.Second)["count"]; n != 1.0 {
		t.Fatalf("viewers after leave = %v", n)
	}
}
