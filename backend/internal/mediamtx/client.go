// Package mediamtx talks to the MediaMTX control API.
package mediamtx

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Channels publish to "live/<slug>", e.g. rtmp://host/live/<slug>.
const pathPrefix = "live/"

func PathForSlug(slug string) string { return pathPrefix + slug }

func SlugFromPath(path string) (string, bool) {
	slug, ok := strings.CutPrefix(path, pathPrefix)
	if !ok || slug == "" || strings.Contains(slug, "/") {
		return "", false
	}
	return slug, true
}

type Client struct {
	base string
	http *http.Client
}

func NewClient(base string) *Client {
	return &Client{base: strings.TrimRight(base, "/"), http: &http.Client{Timeout: 3 * time.Second}}
}

type pathSource struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

type pathItem struct {
	Name      string      `json:"name"`
	Ready     bool        `json:"ready"`
	ReadyTime *time.Time  `json:"readyTime"`
	Source    *pathSource `json:"source"`
}

type pathList struct {
	PageCount int        `json:"pageCount"`
	Items     []pathItem `json:"items"`
}

// ListReadyPaths returns every path that currently has a publisher, with
// the time it went live.
func (c *Client) ListReadyPaths(ctx context.Context) (map[string]time.Time, error) {
	ready := make(map[string]time.Time)
	for page := 0; ; page++ {
		var list pathList
		if err := c.get(ctx, fmt.Sprintf("/v3/paths/list?itemsPerPage=500&page=%d", page), &list); err != nil {
			return nil, err
		}
		for _, item := range list.Items {
			if !item.Ready {
				continue
			}
			started := time.Now().UTC()
			if item.ReadyTime != nil {
				started = item.ReadyTime.UTC()
			}
			ready[item.Name] = started
		}
		if page+1 >= list.PageCount {
			return ready, nil
		}
	}
}

// Kick endpoints per publisher type, as reported in a path's "source".
var kickEndpoints = map[string]string{
	"rtmpConn":      "rtmpconns",
	"webRTCSession": "webrtcsessions",
	"rtspSession":   "rtspsessions",
	"srtConn":       "srtconns",
}

// KickPublisher disconnects whoever is publishing to path. It's a no-op when
// nobody is.
func (c *Client) KickPublisher(ctx context.Context, path string) error {
	var item pathItem
	err := c.get(ctx, "/v3/paths/get/"+escapePath(path), &item)
	if err == errNotFound || (err == nil && item.Source == nil) {
		return nil
	}
	if err != nil {
		return err
	}
	kind, ok := kickEndpoints[item.Source.Type]
	if !ok {
		return fmt.Errorf("don't know how to kick a %q publisher", item.Source.Type)
	}
	return c.post(ctx, fmt.Sprintf("/v3/%s/kick/%s", kind, url.PathEscape(item.Source.ID)))
}

var errNotFound = fmt.Errorf("not found")

func (c *Client) get(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+path, nil)
	if err != nil {
		return err
	}
	res, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusNotFound {
		return errNotFound
	}
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("mediamtx %s: status %d", path, res.StatusCode)
	}
	return json.NewDecoder(res.Body).Decode(out)
}

func (c *Client) post(ctx context.Context, path string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+path, nil)
	if err != nil {
		return err
	}
	res, err := c.http.Do(req)
	if err != nil {
		return err
	}
	res.Body.Close()
	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusNotFound {
		return fmt.Errorf("mediamtx %s: status %d", path, res.StatusCode)
	}
	return nil
}

func escapePath(path string) string {
	parts := strings.Split(path, "/")
	for i, p := range parts {
		parts[i] = url.PathEscape(p)
	}
	return strings.Join(parts, "/")
}
