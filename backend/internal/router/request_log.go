package router

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"github.com/mockhub/mockhub/internal/config"
	"github.com/mockhub/mockhub/internal/middleware"
)

func registerRequestLog(api *gin.RouterGroup, h *Handlers, cfg *config.Config, logger *slog.Logger) {
	logs := api.Group("/projects/:projectId/logs", middleware.JWTAuth(cfg, logger))
	{
		logs.GET("", h.RequestLog.List)
		logs.DELETE("", h.RequestLog.Clear)
	}
}
