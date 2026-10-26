package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"livestream/internal/dto"
	"livestream/internal/middleware"
	"livestream/internal/services"
	"livestream/internal/utils"
)

type AdminController struct {
	users    *services.UserService
	channels *services.ChannelService
}

func NewAdminController(users *services.UserService, channels *services.ChannelService) *AdminController {
	return &AdminController{users: users, channels: channels}
}

func (ctl *AdminController) ListUsers(c *gin.Context) {
	pg := utils.ParsePagination(c, 20, 100)
	items, total, err := ctl.users.List(c.Request.Context(), c.Query("q"), pg.Page, pg.Limit)
	if err != nil {
		handleError(c, err)
		return
	}
	utils.Paginated(c, items, utils.NewPageMeta(pg, total))
}

func (ctl *AdminController) ListRoles(c *gin.Context) {
	utils.OK(c, http.StatusOK, "", ctl.users.Roles())
}

func (ctl *AdminController) UpdateUserStatus(c *gin.Context) {
	id, ok := paramID(c, "id")
	if !ok {
		return
	}
	var req dto.UpdateStatusRequest
	if !bindJSON(c, &req) {
		return
	}
	user, err := ctl.users.UpdateStatus(c.Request.Context(), middleware.CurrentPrincipal(c), id, req.Status, requestMeta(c))
	if err != nil {
		handleError(c, err)
		return
	}
	utils.OK(c, http.StatusOK, "User updated", user)
}

func (ctl *AdminController) UpdateUserRole(c *gin.Context) {
	id, ok := paramID(c, "id")
	if !ok {
		return
	}
	var req dto.UpdateRoleRequest
	if !bindJSON(c, &req) {
		return
	}
	user, err := ctl.users.UpdateRole(c.Request.Context(), middleware.CurrentPrincipal(c), id, req.Role, requestMeta(c))
	if err != nil {
		handleError(c, err)
		return
	}
	utils.OK(c, http.StatusOK, "Role updated", user)
}

func (ctl *AdminController) ListChannels(c *gin.Context) {
	pg := utils.ParsePagination(c, 20, 100)
	items, total, err := ctl.channels.ListAll(c.Request.Context(), c.Query("q"), pg.Page, pg.Limit)
	if err != nil {
		handleError(c, err)
		return
	}
	utils.Paginated(c, items, utils.NewPageMeta(pg, total))
}

func (ctl *AdminController) UpdateChannelStatus(c *gin.Context) {
	id, ok := paramID(c, "id")
	if !ok {
		return
	}
	var req dto.UpdateStatusRequest
	if !bindJSON(c, &req) {
		return
	}
	ch, err := ctl.channels.SetStatus(c.Request.Context(), middleware.CurrentPrincipal(c), id, req.Status, requestMeta(c))
	if err != nil {
		handleError(c, err)
		return
	}
	utils.OK(c, http.StatusOK, "Channel updated", ch)
}
