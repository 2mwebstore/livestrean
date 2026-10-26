package controllers

import (
	"net/http"

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
	items, total, err := ctl.channels.ListPublic(c.Request.Context(), c.Query("live") == "true", pg.Page, pg.Limit)
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
