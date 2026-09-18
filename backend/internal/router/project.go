package router

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"github.com/mockhub/mockhub/internal/config"
	"github.com/mockhub/mockhub/internal/middleware"
)

func registerProject(api *gin.RouterGroup, h *Handlers, cfg *config.Config, logger *slog.Logger) {
	projects := api.Group("/projects", middleware.JWTAuth(cfg, logger))
	{
		projects.GET("", h.Project.List)
		projects.POST("", h.Project.Create)
		projects.GET("/:projectId", h.Project.Get)
		projects.PUT("/:projectId", h.Project.Update)
		projects.DELETE("/:projectId", h.Project.Delete)
	}
}
