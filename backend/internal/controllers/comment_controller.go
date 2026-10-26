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

type CommentController struct {
	comments *services.CommentService
}

func NewCommentController(comments *services.CommentService) *CommentController {
	return &CommentController{comments: comments}
}

// List: GET /channels/:slug/comments?before_id=&limit=
func (ctl *CommentController) List(c *gin.Context) {
	beforeID, _ := strconv.ParseUint(c.Query("before_id"), 10, 64)
	pg := utils.ParsePagination(c, 50, 100)
	items, err := ctl.comments.List(c.Request.Context(), c.Param("slug"), beforeID, pg.Limit)
	if err != nil {
		handleError(c, err)
		return
	}
	utils.OK(c, http.StatusOK, "", items)
}

func (ctl *CommentController) Create(c *gin.Context) {
	var req dto.CreateCommentRequest
	if !bindJSON(c, &req) {
		return
	}
	comment, err := ctl.comments.Create(c.Request.Context(), middleware.CurrentPrincipal(c), c.Param("slug"), req)
	if err != nil {
		handleError(c, err)
		return
	}
	utils.OK(c, http.StatusCreated, "", comment)
}

func (ctl *CommentController) Delete(c *gin.Context) {
	id, ok := paramID(c, "id")
	if !ok {
		return
	}
	if err := ctl.comments.Delete(c.Request.Context(), middleware.CurrentPrincipal(c), id, requestMeta(c)); err != nil {
		handleError(c, err)
		return
	}
	utils.OK(c, http.StatusOK, "Comment deleted", nil)
}
