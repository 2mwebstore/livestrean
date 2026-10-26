// Package realtime pushes channel events (live status, viewer count, chat)
// to browsers over WebSocket. Each channel has a room; every open
// connection in a room counts as one viewer.
package realtime

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait     = 10 * time.Second
	pongWait      = 60 * time.Second
	pingPeriod    = 50 * time.Second
	sendBuffer    = 64
	maxConnsPerIP = 10
)

// Event types sent to clients.
const (
	EventStatus         = "status"
	EventViewers        = "viewers"
	EventComment        = "comment"
	EventCommentDeleted = "comment_deleted"
	EventSchedule       = "schedule"
)

type Event struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

type viewersData struct {
	Count int `json:"count"`
}

var ErrTooManyConnections = errors.New("too many connections from this IP")

type Hub struct {
	mu    sync.RWMutex
	rooms map[string]map[*client]struct{}
	perIP map[string]int
	dirty map[string]struct{} // rooms whose viewer count changed
	log   *slog.Logger
}

func NewHub(log *slog.Logger) *Hub {
	return &Hub{
		rooms: make(map[string]map[*client]struct{}),
		perIP: make(map[string]int),
		dirty: make(map[string]struct{}),
		log:   log,
	}
}

// Run sends viewer counts for rooms that changed, at most once per second,
// so a burst of joins doesn't flood every viewer with updates.
func (h *Hub) Run(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			h.FlushViewerCounts()
		}
	}
}

func (h *Hub) FlushViewerCounts() {
	h.mu.Lock()
	counts := make(map[string]int, len(h.dirty))
	for room := range h.dirty {
		counts[room] = len(h.rooms[room])
	}
	h.dirty = make(map[string]struct{})
	h.mu.Unlock()

	for room, n := range counts {
		h.Broadcast(room, Event{Type: EventViewers, Data: viewersData{Count: n}})
	}
}

func (h *Hub) ViewerCount(room string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.rooms[room])
}

func (h *Hub) Broadcast(room string, e Event) {
	msg, err := json.Marshal(e)
	if err != nil {
		h.log.Error("realtime: encode event", "type", e.Type, "error", err)
		return
	}
	h.mu.RLock()
	clients := make([]*client, 0, len(h.rooms[room]))
	for c := range h.rooms[room] {
		clients = append(clients, c)
	}
	h.mu.RUnlock()

	for _, c := range clients {
		c.enqueue(msg)
	}
}

// Serve joins conn to room, sends initial events followed by the current
// viewer count, and blocks until the connection closes.
func (h *Hub) Serve(conn *websocket.Conn, room, ip string, initial []Event) error {
	c := &client{conn: conn, send: make(chan []byte, sendBuffer), done: make(chan struct{})}

	h.mu.Lock()
	if h.perIP[ip] >= maxConnsPerIP {
		h.mu.Unlock()
		conn.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "too many connections"),
			time.Now().Add(writeWait))
		conn.Close()
		return ErrTooManyConnections
	}
	if h.rooms[room] == nil {
		h.rooms[room] = make(map[*client]struct{})
	}
	h.rooms[room][c] = struct{}{}
	h.perIP[ip]++
	h.dirty[room] = struct{}{}
	count := len(h.rooms[room])
	h.mu.Unlock()

	initial = append(initial, Event{Type: EventViewers, Data: viewersData{Count: count}})
	for _, e := range initial {
		if msg, err := json.Marshal(e); err == nil {
			c.enqueue(msg)
		}
	}

	go c.writePump()
	c.readPump()
	h.leave(room, ip, c)
	return nil
}

func (h *Hub) leave(room, ip string, c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.rooms[room], c)
	if len(h.rooms[room]) == 0 {
		delete(h.rooms, room)
	}
	h.perIP[ip]--
	if h.perIP[ip] <= 0 {
		delete(h.perIP, ip)
	}
	h.dirty[room] = struct{}{}
}

type client struct {
	conn      *websocket.Conn
	send      chan []byte
	done      chan struct{}
	closeOnce sync.Once
}

// enqueue never blocks: a client too slow to keep up is disconnected.
func (c *client) enqueue(msg []byte) {
	select {
	case <-c.done:
	case c.send <- msg:
	default:
		c.close()
	}
}

func (c *client) close() {
	c.closeOnce.Do(func() {
		close(c.done)
		c.conn.Close()
	})
}

// readPump only watches for disconnects and pong replies. Viewers never
// send anything; chat messages are posted through the REST API.
func (c *client) readPump() {
	defer c.close()
	c.conn.SetReadLimit(512)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})
	for {
		if _, _, err := c.conn.ReadMessage(); err != nil {
			return
		}
	}
}

func (c *client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.close()
	}()
	for {
		select {
		case <-c.done:
			return
		case msg := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
