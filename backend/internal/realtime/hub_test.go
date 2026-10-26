package realtime

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func newTestServer(t *testing.T, h *Hub) *httptest.Server {
	up := websocket.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		h.Serve(conn, "room", "1.2.3.4", nil)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func dial(t *testing.T, srv *httptest.Server) *websocket.Conn {
	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

// next reads events until one of the given type arrives.
func next(t *testing.T, conn *websocket.Conn, typ string) map[string]any {
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	for {
		var e struct {
			Type string         `json:"type"`
			Data map[string]any `json:"data"`
		}
		if err := conn.ReadJSON(&e); err != nil {
			t.Fatalf("waiting for %q: %v", typ, err)
		}
		if e.Type == typ {
			return e.Data
		}
	}
}

func waitFor(t *testing.T, cond func() bool) {
	deadline := time.Now().Add(3 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal("condition not met in time")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestHubViewersAndBroadcast(t *testing.T) {
	h := NewHub(slog.New(slog.NewTextHandler(io.Discard, nil)))
	srv := newTestServer(t, h)

	a := dial(t, srv)
	if got := next(t, a, EventViewers)["count"]; got != 1.0 {
		t.Fatalf("first viewer count = %v, want 1", got)
	}
	b := dial(t, srv)
	waitFor(t, func() bool { return h.ViewerCount("room") == 2 })
	h.FlushViewerCounts()
	if got := next(t, a, EventViewers)["count"]; got != 2.0 {
		t.Fatalf("viewer count after second join = %v, want 2", got)
	}

	h.Broadcast("room", Event{Type: EventComment, Data: map[string]string{"body": "hi"}})
	for _, conn := range []*websocket.Conn{a, b} {
		if got := next(t, conn, EventComment)["body"]; got != "hi" {
			t.Fatalf("comment body = %v", got)
		}
	}

	b.Close()
	waitFor(t, func() bool { return h.ViewerCount("room") == 1 })
}

func TestHubLimitsConnectionsPerIP(t *testing.T) {
	h := NewHub(slog.New(slog.NewTextHandler(io.Discard, nil)))
	srv := newTestServer(t, h)
	for i := 0; i < maxConnsPerIP; i++ {
		dial(t, srv)
	}
	waitFor(t, func() bool { return h.ViewerCount("room") == maxConnsPerIP })

	extra := dial(t, srv)
	extra.SetReadDeadline(time.Now().Add(3 * time.Second))
	_, _, err := extra.ReadMessage()
	var closeErr *websocket.CloseError
	if !errors.As(err, &closeErr) || closeErr.Code != websocket.ClosePolicyViolation {
		t.Fatalf("extra connection: got %v, want policy-violation close", err)
	}
	if n := h.ViewerCount("room"); n != maxConnsPerIP {
		t.Fatalf("viewer count = %d, want %d", n, maxConnsPerIP)
	}
}
