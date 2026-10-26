package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"livestream/internal/dto"
	"livestream/internal/services"
	"livestream/internal/utils"
)

// SystemController serves the health check and the internal MediaMTX hook.
type SystemController struct {
	db         *gorm.DB
	streamAuth *services.StreamAuthService
}

func NewSystemController(db *gorm.DB, streamAuth *services.StreamAuthService) *SystemController {
	return &SystemController{db: db, streamAuth: streamAuth}
}

func (ctl *SystemController) Health(c *gin.Context) {
	sqlDB, err := ctl.db.DB()
	if err == nil {
		err = sqlDB.PingContext(c.Request.Context())
	}
	if err != nil {
		utils.Fail(c, http.StatusServiceUnavailable, "Database unavailable", nil)
		return
	}
	utils.OK(c, http.StatusOK, "", gin.H{"status": "ok"})
}

// MediaMTXAuth is called by MediaMTX for every publish and read attempt.
// Any 2xx allows it. Only reachable on the internal listener.
func (ctl *SystemController) MediaMTXAuth(c *gin.Context) {
	var req dto.MediaMTXAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	if ctl.streamAuth.Authorize(c.Request.Context(), req) {
		c.Status(http.StatusOK)
		return
	}
	c.Status(http.StatusUnauthorized)
}
