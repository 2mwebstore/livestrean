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

func TestBlockFormBodies(t *testing.T) {
	post := func(contentType, body string) int {
		req := httptest.NewRequest("POST", "/x", strings.NewReader(body))
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}
		return serve(BlockFormBodies(), req)
	}
	cases := []struct {
		contentType string
		want        int
	}{
		{"application/json", 200},
		{"application/json; charset=utf-8", 200},
		{"image/jpeg", 200},
		{"application/x-www-form-urlencoded", 415},
		{"multipart/form-data; boundary=x", 415},
		{"text/plain", 415},
		{"text/plain; charset=utf-8", 415},
		{"", 415},
	}
	for _, tc := range cases {
		if got := post(tc.contentType, "x=1"); got != tc.want {
			t.Errorf("%q: got %d, want %d", tc.contentType, got, tc.want)
		}
	}
	if got := serve(BlockFormBodies(), httptest.NewRequest("POST", "/x", nil)); got != 200 {
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
