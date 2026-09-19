package router

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"github.com/mockhub/mockhub/internal/config"
	"github.com/mockhub/mockhub/internal/middleware"
)

func registerEndpoint(api *gin.RouterGroup, h *Handlers, cfg *config.Config, logger *slog.Logger) {
	apis := api.Group("/projects/:projectId/apis", middleware.JWTAuth(cfg, logger))
	{
		apis.GET("", h.Endpoint.List)
		apis.POST("", h.Endpoint.Create)
		apis.GET("/:id", h.Endpoint.Get)
		apis.PUT("/:id", h.Endpoint.Update)
		apis.DELETE("/:id", h.Endpoint.Delete)
		apis.GET("/:id/versions", h.Endpoint.ListVersions)
		apis.GET("/:id/versions/:version", h.Endpoint.GetVersion)
		apis.POST("/:id/versions/:version/publish", h.Endpoint.PublishVersion)
	}

	api.POST("/projects/:projectId/swagger/import", middleware.JWTAuth(cfg, logger), h.Endpoint.ImportSwagger)
}
