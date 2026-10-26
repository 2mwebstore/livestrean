package mediamtx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSlugFromPath(t *testing.T) {
	cases := map[string]string{"live/my-show": "my-show", "live/": "", "other/x": "", "live/a/b": "", "my-show": ""}
	for path, want := range cases {
		got, ok := SlugFromPath(path)
		if got != want || ok != (want != "") {
			t.Errorf("SlugFromPath(%q) = %q, %v", path, got, ok)
		}
	}
	if PathForSlug("my-show") != "live/my-show" {
		t.Error("PathForSlug")
	}
}

func TestKickPublisher(t *testing.T) {
	var kicked string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v3/paths/get/live/on-air":
			w.Write([]byte(`{"name":"live/on-air","ready":true,"source":{"type":"rtmpConn","id":"abc"}}`))
		case "/v3/rtmpconns/kick/abc":
			kicked = "abc"
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	c := NewClient(srv.URL)

	if err := c.KickPublisher(context.Background(), "live/on-air"); err != nil || kicked != "abc" {
		t.Fatalf("kick live path: err=%v kicked=%q", err, kicked)
	}
	if err := c.KickPublisher(context.Background(), "live/offline"); err != nil {
		t.Fatalf("kick offline path should be a no-op, got %v", err)
	}
}
