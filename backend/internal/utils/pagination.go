package utils

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

type Pagination struct {
	Page  int
	Limit int
}

type PageMeta struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

// ParsePagination reads ?page and ?limit, clamped to sane values.
func ParsePagination(c *gin.Context, defaultLimit, maxLimit int) Pagination {
	page, _ := strconv.Atoi(c.Query("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(c.Query("limit"))
	if limit < 1 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	return Pagination{Page: page, Limit: limit}
}

func NewPageMeta(p Pagination, total int64) PageMeta {
	pages := int((total + int64(p.Limit) - 1) / int64(p.Limit))
	return PageMeta{Page: p.Page, Limit: p.Limit, Total: total, TotalPages: pages}
}
