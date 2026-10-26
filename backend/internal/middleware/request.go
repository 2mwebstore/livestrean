package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"regexp"
	"time"

	"github.com/gin-gonic/gin"

	"livestream/internal/utils"
)

const requestIDKey = "request_id"

var validRequestID = regexp.MustCompile(`^[A-Za-z0-9-]{8,64}$`)

// RequestID tags each request with an ID (reusing the proxy's if valid),
// returned in X-Request-ID and included in logs.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Request-ID")
		if !validRequestID.MatchString(id) {
			b := make([]byte, 8)
			rand.Read(b)
			id = hex.EncodeToString(b)
		}
		c.Set(requestIDKey, id)
		c.Header("X-Request-ID", id)
		c.Next()
	}
}

// RequestLogger logs one line per request. Query strings are left out on
// purpose: they could contain secrets.
func RequestLogger(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		attrs := []any{
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"duration_ms", time.Since(start).Milliseconds(),
			"ip", c.ClientIP(),
			"request_id", c.GetString(requestIDKey),
		}
		if p := CurrentPrincipal(c); p != nil {
			attrs = append(attrs, "user_id", p.UserID)
		}
		log.Info("request", attrs...)
	}
}

func Recovery(log *slog.Logger) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, err any) {
		log.Error("panic", "error", err, "path", c.Request.URL.Path, "request_id", c.GetString(requestIDKey))
		utils.Abort(c, http.StatusInternalServerError, "Something went wrong")
	})
}
