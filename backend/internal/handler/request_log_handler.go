package handler

import (
	"log/slog"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/mockhub/mockhub/internal/middleware"
	"github.com/mockhub/mockhub/internal/service"
	"github.com/mockhub/mockhub/internal/util"
)

// RequestLogHandler exposes request log endpoints.
type RequestLogHandler struct {
	svc    *service.RequestLogService
	logger *slog.Logger
}

// NewRequestLogHandler builds a RequestLogHandler.
func NewRequestLogHandler(svc *service.RequestLogService, logger *slog.Logger) *RequestLogHandler {
	return &RequestLogHandler{svc: svc, logger: logger}
}

// List handles GET /projects/:projectId/logs?page=&page_size=.
func (h *RequestLogHandler) List(c *gin.Context) {
	projectID, ok := parseID(c)
	if !ok {
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("page_size", c.DefaultQuery("limit", "50")))
	logs, total, err := h.svc.List(projectID, middleware.GetUserID(c), middleware.GetRole(c), page, limit)
	if err != nil {
		util.Fail(c, err)
		return
	}
	util.OK(c, gin.H{
		"logs": logs,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
			"total": total,
		},
	})
}

// Clear handles DELETE /projects/:projectId/logs.
func (h *RequestLogHandler) Clear(c *gin.Context) {
	projectID, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.svc.Clear(projectID, middleware.GetUserID(c), middleware.GetRole(c)); err != nil {
		util.Fail(c, err)
		return
	}
	util.OK(c, gin.H{"cleared": true})
}
