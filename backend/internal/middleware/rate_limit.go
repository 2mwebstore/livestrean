package middleware

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"

	"livestream/internal/utils"
)

// RateLimiter is an in-memory token bucket per key (IP or user). Fine for
// a single API instance; several instances would need a shared store.
type RateLimiter struct {
	mu       sync.Mutex
	limit    rate.Limit
	burst    int
	visitors map[string]*visitor
}

type visitor struct {
	limiter *rate.Limiter
	seen    time.Time
}

func NewRateLimiter(every time.Duration, burst int) *RateLimiter {
	return &RateLimiter{limit: rate.Every(every), burst: burst, visitors: make(map[string]*visitor)}
}

func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	v, ok := rl.visitors[key]
	if !ok {
		v = &visitor{limiter: rate.NewLimiter(rl.limit, rl.burst)}
		rl.visitors[key] = v
	}
	v.seen = time.Now()
	rl.mu.Unlock()
	return v.limiter.Allow()
}

// Cleanup forgets keys idle for 10 minutes, until ctx is cancelled.
func (rl *RateLimiter) Cleanup(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cutoff := time.Now().Add(-10 * time.Minute)
			rl.mu.Lock()
			for key, v := range rl.visitors {
				if v.seen.Before(cutoff) {
					delete(rl.visitors, key)
				}
			}
			rl.mu.Unlock()
		}
	}
}

func RateLimitByIP(rl *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !rl.Allow(c.ClientIP()) {
			tooManyRequests(c)
			return
		}
		c.Next()
	}
}

// RateLimitByUser must run after RequireAuth.
func RateLimitByUser(rl *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !rl.Allow("user:" + strconv.FormatUint(CurrentPrincipal(c).UserID, 10)) {
			tooManyRequests(c)
			return
		}
		c.Next()
	}
}

func tooManyRequests(c *gin.Context) {
	c.Header("Retry-After", "5")
	utils.Abort(c, http.StatusTooManyRequests, "Too many requests, please slow down")
}
