package router

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"github.com/mockhub/mockhub/internal/config"
	"github.com/mockhub/mockhub/internal/middleware"
)

func registerAuth(api *gin.RouterGroup, h *Handlers, cfg *config.Config, logger *slog.Logger) {
	credentials := api.Group("/auth", middleware.RateLimit(cfg.AuthRateLimit, logger))
	{
		credentials.POST("/register", h.Auth.Register)
		credentials.POST("/login", h.Auth.Login)
	}

	// /auth/me is protected but not subject to the credential rate limit,
	// since authenticated clients may legitimately call it frequently.
	api.GET("/auth/me", middleware.JWTAuth(cfg, logger), h.Auth.Me)
}
