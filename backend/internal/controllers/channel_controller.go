package controllers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"livestream/internal/dto"
	"livestream/internal/middleware"
	"livestream/internal/services"
	"livestream/internal/utils"
)

// ChannelController serves public channel pages and the owner's dashboard.
type ChannelController struct {
	channels *services.ChannelService
}

func NewChannelController(channels *services.ChannelService) *ChannelController {
	return &ChannelController{channels: channels}
}

// --- Public -----------------------------------------------------------------

func (ctl *ChannelController) List(c *gin.Context) {
	pg := utils.ParsePagination(c, 24, 100)
	var live *bool // ?live=true: only live channels, ?live=false: only offline ones
	if v, err := strconv.ParseBool(c.Query("live")); err == nil {
		live = &v
	}
	items, total, err := ctl.channels.ListPublic(c.Request.Context(), live, pg.Page, pg.Limit)
	if err != nil {
		handleError(c, err)
		return
	}
	utils.Paginated(c, items, utils.NewPageMeta(pg, total))
}

func (ctl *ChannelController) Get(c *gin.Context) {
	ch, err := ctl.channels.GetPublic(c.Request.Context(), c.Param("slug"))
	if err != nil {
		handleError(c, err)
		return
	}
	utils.OK(c, http.StatusOK, "", ch)
}

// --- Owner ------------------------------------------------------------------

func (ctl *ChannelController) ListOwn(c *gin.Context) {
	items, err := ctl.channels.ListOwn(c.Request.Context(), middleware.CurrentPrincipal(c))
	if err != nil {
		handleError(c, err)
		return
	}
	utils.OK(c, http.StatusOK, "", items)
}

func (ctl *ChannelController) Create(c *gin.Context) {
	var req dto.CreateChannelRequest
	if !bindJSON(c, &req) {
		return
	}
	res, err := ctl.channels.Create(c.Request.Context(), middleware.CurrentPrincipal(c), req, requestMeta(c))
	if err != nil {
		handleError(c, err)
		return
	}
	utils.OK(c, http.StatusCreated, "Channel created", res)
}

func (ctl *ChannelController) GetOwn(c *gin.Context) {
	id, ok := paramID(c, "id")
	if !ok {
		return
	}
	ch, err := ctl.channels.GetOwn(c.Request.Context(), middleware.CurrentPrincipal(c), id)
	if err != nil {
		handleError(c, err)
		return
	}
	utils.OK(c, http.StatusOK, "", ch)
}

func (ctl *ChannelController) Update(c *gin.Context) {
	id, ok := paramID(c, "id")
	if !ok {
		return
	}
	var req dto.UpdateChannelRequest
	if !bindJSON(c, &req) {
		return
	}
	ch, err := ctl.channels.Update(c.Request.Context(), middleware.CurrentPrincipal(c), id, req)
	if err != nil {
		handleError(c, err)
		return
	}
	utils.OK(c, http.StatusOK, "Channel updated", ch)
}

func (ctl *ChannelController) RegenerateKey(c *gin.Context) {
	id, ok := paramID(c, "id")
	if !ok {
		return
	}
	res, err := ctl.channels.RegenerateKey(c.Request.Context(), middleware.CurrentPrincipal(c), id, requestMeta(c))
	if err != nil {
		handleError(c, err)
		return
	}
	utils.OK(c, http.StatusOK, "New stream key created", res)
}

func (ctl *ChannelController) Delete(c *gin.Context) {
	id, ok := paramID(c, "id")
	if !ok {
		return
	}
	if err := ctl.channels.Delete(c.Request.Context(), middleware.CurrentPrincipal(c), id, requestMeta(c)); err != nil {
		handleError(c, err)
		return
	}
	utils.OK(c, http.StatusOK, "Channel deleted", nil)
}

// --- Schedule and thumbnail ---------------------------------------------------

// Thumbnail serves a channel's image. With the current ?v= version the
// response is cached for a year; uploads change the version.
func (ctl *ChannelController) Thumbnail(c *gin.Context) {
	thumb, version, err := ctl.channels.Thumbnail(c.Request.Context(), c.Param("slug"))
	if err != nil {
		handleError(c, err)
		return
	}
	etag := `"` + version + `"`
	c.Header("ETag", etag)
	if c.Query("v") == version {
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		c.Header("Cache-Control", "public, max-age=60")
	}
	if c.GetHeader("If-None-Match") == etag {
		c.Status(http.StatusNotModified)
		return
	}
	c.Data(http.StatusOK, thumb.ContentType, thumb.Data)
}

// UploadThumbnail: PUT /me/channels/:id/thumbnail with the image as the raw
// body (Content-Type image/jpeg, image/png, image/webp or image/gif).
func (ctl *ChannelController) UploadThumbnail(c *gin.Context) {
	id, ok := paramID(c, "id")
	if !ok {
		return
	}
	if c.Request.ContentLength > services.MaxThumbnailBytes {
		msg := "The image must be 5 MB or smaller"
		utils.Fail(c, http.StatusRequestEntityTooLarge, msg, map[string]string{"thumbnail": msg})
		return
	}
	body := http.MaxBytesReader(c.Writer, c.Request.Body, services.MaxThumbnailBytes+1)
	ch, err := ctl.channels.SetThumbnail(c.Request.Context(), middleware.CurrentPrincipal(c), id, body)
	if err != nil {
		handleError(c, err)
		return
	}
	utils.OK(c, http.StatusOK, "Thumbnail updated", ch)
}

func (ctl *ChannelController) DeleteThumbnail(c *gin.Context) {
	id, ok := paramID(c, "id")
	if !ok {
		return
	}
	ch, err := ctl.channels.RemoveThumbnail(c.Request.Context(), middleware.CurrentPrincipal(c), id)
	if err != nil {
		handleError(c, err)
		return
	}
	utils.OK(c, http.StatusOK, "Thumbnail removed", ch)
}

// UpdateSchedule: PUT /me/channels/:id/schedule {"scheduled_at": "2026-11-01T13:00:00Z" | null}
func (ctl *ChannelController) UpdateSchedule(c *gin.Context) {
	id, ok := paramID(c, "id")
	if !ok {
		return
	}
	var req dto.UpdateScheduleRequest
	if !bindJSON(c, &req) {
		return
	}
	ch, err := ctl.channels.SetSchedule(c.Request.Context(), middleware.CurrentPrincipal(c), id, req.ScheduledAt)
	if err != nil {
		handleError(c, err)
		return
	}
	utils.OK(c, http.StatusOK, "Schedule updated", ch)
}
