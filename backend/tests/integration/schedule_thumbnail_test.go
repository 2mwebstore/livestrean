package integration

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"strings"
	"testing"
	"time"
)

func pngBytes(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for x := 0; x < w; x++ {
		img.Set(x, h/2, color.NRGBA{G: 200, A: 255})
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

type channelMedia struct {
	Slug         string  `json:"slug"`
	ThumbnailURL *string `json:"thumbnail_url"`
	ScheduledAt  *string `json:"scheduled_at"`
}

func publicChannel(t *testing.T, slug string) channelMedia {
	t.Helper()
	return decode[channelMedia](t, newClient(t).do("GET", "/api/v1/channels/"+slug, nil).expect(t, http.StatusOK).Data)
}

func get(t *testing.T, url string, header map[string]string) *http.Response {
	t.Helper()
	req, _ := http.NewRequest("GET", url, nil)
	for k, v := range header {
		req.Header.Set(k, v)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func TestThumbnail(t *testing.T) {
	owner, _ := streamer(t)
	created := createChannel(t, owner)
	slug := created.Channel.Slug
	path := fmt.Sprintf("/api/v1/me/channels/%d/thumbnail", created.Channel.ID)

	own := decode[channelMedia](t, owner.raw("PUT", path, "image/png", pngBytes(t, 1920, 1080)).expect(t, http.StatusOK).Data)
	if own.ThumbnailURL == nil || !strings.HasPrefix(*own.ThumbnailURL, "/api/v1/channels/"+slug+"/thumbnail?v=") {
		t.Fatalf("thumbnail_url = %v", own.ThumbnailURL)
	}
	if pub := publicChannel(t, slug); pub.ThumbnailURL == nil || *pub.ThumbnailURL != *own.ThumbnailURL {
		t.Fatalf("public thumbnail_url = %v", pub.ThumbnailURL)
	}

	// Served as a re-encoded JPEG, cached forever for this version.
	res := get(t, api.URL+*own.ThumbnailURL, nil)
	body := make([]byte, 3)
	res.Body.Read(body)
	res.Body.Close()
	etag := res.Header.Get("ETag")
	if res.StatusCode != 200 || res.Header.Get("Content-Type") != "image/jpeg" || !bytes.Equal(body[:2], []byte{0xFF, 0xD8}) {
		t.Fatalf("thumbnail: %d %s %x", res.StatusCode, res.Header.Get("Content-Type"), body)
	}
	if !strings.Contains(res.Header.Get("Cache-Control"), "immutable") || etag == "" {
		t.Fatalf("caching headers: %q %q", res.Header.Get("Cache-Control"), etag)
	}
	if res := get(t, api.URL+*own.ThumbnailURL, map[string]string{"If-None-Match": etag}); res.StatusCode != http.StatusNotModified {
		t.Fatalf("If-None-Match: %d", res.StatusCode)
	}

	// Only the owner, only images, only non-form bodies.
	other, _ := streamer(t)
	other.raw("PUT", path, "image/png", pngBytes(t, 640, 360)).expect(t, http.StatusNotFound)
	viewer, _ := register(t)
	viewer.raw("PUT", path, "image/png", pngBytes(t, 640, 360)).expect(t, http.StatusForbidden)
	bad := owner.raw("PUT", path, "image/jpeg", []byte("<svg onload=alert(1)>"))
	bad.expect(t, http.StatusUnprocessableEntity)
	if bad.Errors["thumbnail"] == "" {
		t.Fatalf("expected errors.thumbnail: %s", bad.Raw)
	}
	owner.raw("PUT", path, "multipart/form-data; boundary=x", []byte("--x--")).expect(t, http.StatusUnsupportedMediaType)
	owner.raw("PUT", path, "image/png", make([]byte, 6<<20)).expect(t, http.StatusRequestEntityTooLarge)

	owner.do("DELETE", path, nil).expect(t, http.StatusOK)
	if pub := publicChannel(t, slug); pub.ThumbnailURL != nil {
		t.Fatalf("thumbnail_url after delete = %v", *pub.ThumbnailURL)
	}
	if res := get(t, api.URL+*own.ThumbnailURL, nil); res.StatusCode != http.StatusNotFound {
		t.Fatalf("deleted thumbnail still served: %d", res.StatusCode)
	}
}

func TestScheduleCountdown(t *testing.T) {
	owner, _ := streamer(t)
	created := createChannel(t, owner)
	slug := created.Channel.Slug
	path := fmt.Sprintf("/api/v1/me/channels/%d/schedule", created.Channel.ID)

	conn, _, err := dialChannel(t, slug, "")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	nextEvent(t, conn, "viewers", 3*time.Second)

	// Setting a time reaches viewers instantly and shows on the channel.
	at := time.Now().Add(2 * time.Hour).UTC().Truncate(time.Second).Format(time.RFC3339)
	owner.do("PUT", path, map[string]any{"scheduled_at": at}).expect(t, http.StatusOK)
	if got := nextEvent(t, conn, "schedule", 3*time.Second)["scheduled_at"]; got != at {
		t.Fatalf("schedule event = %v, want %s", got, at)
	}
	if pub := publicChannel(t, slug); pub.ScheduledAt == nil || *pub.ScheduledAt != at {
		t.Fatalf("scheduled_at = %v", pub.ScheduledAt)
	}

	// Upcoming channels are listed first among offline channels.
	list := decode[[]channelMedia](t, newClient(t).do("GET", "/api/v1/channels?live=false&limit=100", nil).expect(t, http.StatusOK).Data)
	if len(list) == 0 || list[0].Slug != slug {
		t.Fatalf("upcoming channel not first: %+v", list[:min(3, len(list))])
	}

	// Validation and ownership
	past := time.Now().Add(-time.Minute).UTC().Format(time.RFC3339)
	farFuture := time.Now().Add(400 * 24 * time.Hour).UTC().Format(time.RFC3339)
	owner.do("PUT", path, map[string]any{"scheduled_at": past}).expect(t, http.StatusUnprocessableEntity)
	owner.do("PUT", path, map[string]any{"scheduled_at": farFuture}).expect(t, http.StatusUnprocessableEntity)
	owner.do("PUT", path, map[string]any{"scheduled_at": "tomorrow"}).expect(t, http.StatusUnprocessableEntity)
	other, _ := streamer(t)
	other.do("PUT", path, map[string]any{"scheduled_at": at}).expect(t, http.StatusNotFound)

	// null clears it.
	owner.do("PUT", path, map[string]any{"scheduled_at": nil}).expect(t, http.StatusOK)
	if got := nextEvent(t, conn, "schedule", 3*time.Second)["scheduled_at"]; got != nil {
		t.Fatalf("schedule event after clear = %v", got)
	}

	// Going live uses up a schedule that's due soon.
	soon := time.Now().Add(20 * time.Minute).UTC().Truncate(time.Second).Format(time.RFC3339)
	owner.do("PUT", path, map[string]any{"scheduled_at": soon}).expect(t, http.StatusOK)
	nextEvent(t, conn, "schedule", 3*time.Second)
	media.setLive("live/"+slug, "conn-sched-"+slug)
	defer media.setLive("live/"+slug, "")
	if live := nextEvent(t, conn, "status", 6*time.Second)["live"]; live != true {
		t.Fatalf("live = %v", live)
	}
	if got := nextEvent(t, conn, "schedule", 3*time.Second)["scheduled_at"]; got != nil {
		t.Fatalf("schedule not cleared when going live: %v", got)
	}
	if pub := publicChannel(t, slug); pub.ScheduledAt != nil {
		t.Fatalf("scheduled_at after going live = %v", *pub.ScheduledAt)
	}
}
