package middleware

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"livestream/internal/auth"
	"livestream/internal/services"
	"livestream/internal/utils"
)

const principalKey = "principal"

type Authenticator interface {
	Authenticate(ctx context.Context, token string) (*auth.Principal, error)
}

// LoadPrincipal attaches the logged-in user, if the session cookie is valid.
// It never rejects a request; RequireAuth does that.
func LoadPrincipal(a Authenticator, cookieName string, log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie(cookieName)
		if err == nil && token != "" {
			p, err := a.Authenticate(c.Request.Context(), token)
			var svcErr *services.Error
			switch {
			case err == nil:
				c.Set(principalKey, p)
			case !errors.As(err, &svcErr):
				log.Error("authenticate session", "error", err, "request_id", c.GetString(requestIDKey))
			}
		}
		c.Next()
	}
}

// CurrentPrincipal returns the logged-in user, or nil.
func CurrentPrincipal(c *gin.Context) *auth.Principal {
	if v, ok := c.Get(principalKey); ok {
		return v.(*auth.Principal)
	}
	return nil
}

func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if CurrentPrincipal(c) == nil {
			utils.Abort(c, http.StatusUnauthorized, "Please log in")
			return
		}
		c.Next()
	}
}
