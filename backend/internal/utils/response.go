// Package utils holds the standard API response format and request helpers.
package utils

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Success: {"success": true, "message": "...", "data": ...}
func OK(c *gin.Context, status int, message string, data any) {
	c.JSON(status, gin.H{"success": true, "message": message, "data": data})
}

// Paginated: {"success": true, "data": [...], "meta": {...}}
func Paginated(c *gin.Context, data any, meta PageMeta) {
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data, "meta": meta})
}

// Fail: {"success": false, "message": "...", "errors": {...}}
func Fail(c *gin.Context, status int, message string, errors map[string]string) {
	body := gin.H{"success": false, "message": message}
	if len(errors) > 0 {
		body["errors"] = errors
	}
	c.JSON(status, body)
}

// Abort is Fail for middleware: it also stops the handler chain.
func Abort(c *gin.Context, status int, message string) {
	Fail(c, status, message, nil)
	c.Abort()
}
