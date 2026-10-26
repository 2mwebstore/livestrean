package controllers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"livestream/internal/dto"
	"livestream/internal/middleware"
	"livestream/internal/services"
	"livestream/internal/utils"
)

type CookieSettings struct {
	Name   string
	Secure bool
}

type AuthController struct {
	auth   *services.AuthService
	cookie CookieSettings
}

func NewAuthController(auth *services.AuthService, cookie CookieSettings) *AuthController {
	return &AuthController{auth: auth, cookie: cookie}
}

func (ctl *AuthController) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if !bindJSON(c, &req) {
		return
	}
	res, err := ctl.auth.Register(c.Request.Context(), req, requestMeta(c))
	if err != nil {
		handleError(c, err)
		return
	}
	ctl.setSessionCookie(c, res.Token, res.ExpiresAt)
	utils.OK(c, http.StatusCreated, "Account created", dto.NewUserResponse(res.Principal))
}

func (ctl *AuthController) Login(c *gin.Context) {
	var req dto.LoginRequest
	if !bindJSON(c, &req) {
		return
	}
	res, err := ctl.auth.Login(c.Request.Context(), req, requestMeta(c))
	if err != nil {
		handleError(c, err)
		return
	}
	ctl.setSessionCookie(c, res.Token, res.ExpiresAt)
	utils.OK(c, http.StatusOK, "Logged in", dto.NewUserResponse(res.Principal))
}

func (ctl *AuthController) Logout(c *gin.Context) {
	token, _ := c.Cookie(ctl.cookie.Name)
	if err := ctl.auth.Logout(c.Request.Context(), middleware.CurrentPrincipal(c), token, requestMeta(c)); err != nil {
		handleError(c, err)
		return
	}
	ctl.clearSessionCookie(c)
	utils.OK(c, http.StatusOK, "Logged out", nil)
}

func (ctl *AuthController) Me(c *gin.Context) {
	utils.OK(c, http.StatusOK, "", dto.NewUserResponse(middleware.CurrentPrincipal(c)))
}

func (ctl *AuthController) setSessionCookie(c *gin.Context, token string, expires time.Time) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     ctl.cookie.Name,
		Value:    token,
		Path:     "/",
		Expires:  expires,
		MaxAge:   int(time.Until(expires).Seconds()),
		HttpOnly: true,
		Secure:   ctl.cookie.Secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (ctl *AuthController) clearSessionCookie(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     ctl.cookie.Name,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   ctl.cookie.Secure,
		SameSite: http.SameSiteLaxMode,
	})
}
