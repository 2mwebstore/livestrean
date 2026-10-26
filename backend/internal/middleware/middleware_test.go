package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func init() { gin.SetMode(gin.TestMode) }

func serve(h gin.HandlerFunc, req *http.Request) int {
	r := gin.New()
	r.Use(h)
	r.Any("/x", func(c *gin.Context) { c.Status(http.StatusOK) })
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w.Code
}

func TestOriginCheck(t *testing.T) {
	h := OriginCheck([]string{"https://stream.example.com"})
	cases := []struct {
		method, origin string
		want           int
	}{
		{"POST", "https://stream.example.com", 200},
		{"POST", "https://evil.example", 403},
		{"POST", "null", 403},
		{"POST", "", 200}, // non-browser client
		{"GET", "https://evil.example", 200},
		{"DELETE", "https://evil.example", 403},
	}
	for _, tc := range cases {
		req := httptest.NewRequest(tc.method, "/x", nil)
		if tc.origin != "" {
			req.Header.Set("Origin", tc.origin)
		}
		if got := serve(h, req); got != tc.want {
			t.Errorf("%s from %q: got %d, want %d", tc.method, tc.origin, got, tc.want)
		}
	}
}

func TestRequireJSON(t *testing.T) {
	post := func(contentType, body string) int {
		req := httptest.NewRequest("POST", "/x", strings.NewReader(body))
		req.Header.Set("Content-Type", contentType)
		return serve(RequireJSON(), req)
	}
	if got := post("application/json", `{}`); got != 200 {
		t.Errorf("json: got %d", got)
	}
	if got := post("application/json; charset=utf-8", `{}`); got != 200 {
		t.Errorf("json with charset: got %d", got)
	}
	if got := post("application/x-www-form-urlencoded", "a=b"); got != 415 {
		t.Errorf("form: got %d, want 415", got)
	}
	if got := post("text/plain", `{}`); got != 415 {
		t.Errorf("text/plain: got %d, want 415", got)
	}
	if got := serve(RequireJSON(), httptest.NewRequest("POST", "/x", nil)); got != 200 {
		t.Errorf("empty body: got %d", got)
	}
}

func TestRateLimiter(t *testing.T) {
	rl := NewRateLimiter(time.Hour, 3)
	for i := 0; i < 3; i++ {
		if !rl.Allow("a") {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}
	if rl.Allow("a") {
		t.Fatal("4th request should be limited")
	}
	if !rl.Allow("b") {
		t.Fatal("other keys have their own bucket")
	}
}
