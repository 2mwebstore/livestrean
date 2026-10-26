// Package controllers is the HTTP layer: parse and validate the request,
// call a service, write the response. No business logic or SQL here.
package controllers

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"livestream/internal/services"
	"livestream/internal/utils"
	"livestream/internal/validators"
)

var errorStatus = map[services.ErrorKind]int{
	services.KindValidation:   http.StatusUnprocessableEntity,
	services.KindUnauthorized: http.StatusUnauthorized,
	services.KindForbidden:    http.StatusForbidden,
	services.KindNotFound:     http.StatusNotFound,
	services.KindConflict:     http.StatusConflict,
}

// handleError writes expected service errors as-is and hides everything
// else behind a generic 500.
func handleError(c *gin.Context, err error) {
	var svcErr *services.Error
	if errors.As(err, &svcErr) {
		utils.Fail(c, errorStatus[svcErr.Kind], svcErr.Message, svcErr.Fields)
		return
	}
	slog.Error("request failed", "error", err, "path", c.Request.URL.Path, "request_id", c.GetString("request_id"))
	utils.Fail(c, http.StatusInternalServerError, "Something went wrong, please try again", nil)
}

func bindJSON(c *gin.Context, dst any) bool {
	if err := c.ShouldBindJSON(dst); err != nil {
		utils.Fail(c, http.StatusUnprocessableEntity, "Please check the highlighted fields", validators.FieldErrors(err))
		return false
	}
	return true
}

func paramID(c *gin.Context, name string) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param(name), 10, 64)
	if err != nil || id == 0 {
		utils.Fail(c, http.StatusNotFound, "Not found", nil)
		return 0, false
	}
	return id, true
}

func requestMeta(c *gin.Context) services.RequestMeta {
	return services.RequestMeta{IP: c.ClientIP(), UserAgent: c.Request.UserAgent()}
}
