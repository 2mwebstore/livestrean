package middleware

import (
	"mime"
	"net/http"

	"github.com/gin-gonic/gin"

	"livestream/internal/utils"
)

func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		// API responses can contain account data; never cache them.
		h.Set("Cache-Control", "no-store")
		c.Next()
	}
}

func isSafeMethod(m string) bool {
	return m == http.MethodGet || m == http.MethodHead || m == http.MethodOptions
}

// OriginCheck blocks state-changing requests from other sites (CSRF).
// Browsers always send Origin on such requests; non-browser clients that
// omit it don't carry a victim's cookies anyway.
func OriginCheck(allowed []string) gin.HandlerFunc {
	set := make(map[string]bool, len(allowed))
	for _, o := range allowed {
		set[o] = true
	}
	return func(c *gin.Context) {
		if origin := c.GetHeader("Origin"); !isSafeMethod(c.Request.Method) && origin != "" && !set[origin] {
			utils.Abort(c, http.StatusForbidden, "Cross-site request blocked")
			return
		}
		c.Next()
	}
}

// BlockFormBodies rejects the body types an HTML form on another site can
// send (urlencoded, multipart, text/plain, or none). Any other type, like
// JSON or an image upload, needs a CORS preflight the API never approves,
// so other sites can't send it. Part of the CSRF protection.
func BlockFormBodies() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !isSafeMethod(c.Request.Method) && c.Request.ContentLength != 0 {
			mediaType, _, _ := mime.ParseMediaType(c.GetHeader("Content-Type"))
			switch mediaType {
			case "", "application/x-www-form-urlencoded", "multipart/form-data", "text/plain":
				utils.Abort(c, http.StatusUnsupportedMediaType, "Send JSON, or an image for uploads")
				return
			}
		}
		c.Next()
	}
}
