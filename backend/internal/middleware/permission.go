package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"livestream/internal/utils"
)

// RequirePermission must run after RequireAuth.
func RequirePermission(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !CurrentPrincipal(c).Can(permission) {
			utils.Abort(c, http.StatusForbidden, "You don't have permission to do this")
			return
		}
		c.Next()
	}
}
