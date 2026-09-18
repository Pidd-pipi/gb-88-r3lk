// Package router wires up the Gin engine and all routes.
package router

import (
	"log/slog"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/mockhub/mockhub/internal/config"
	"github.com/mockhub/mockhub/internal/docs"
	"github.com/mockhub/mockhub/internal/handler"
	"github.com/mockhub/mockhub/internal/middleware"
)

// Handlers aggregates every HTTP handler for assembly.
type Handlers struct {
	Health     *handler.HealthHandler
	Auth       *handler.AuthHandler
	Project    *handler.ProjectHandler
	Endpoint   *handler.EndpointHandler
	Mock       *handler.MockHandler
	RequestLog *handler.RequestLogHandler
}

// New assembles the Gin engine and registers all routes.
func New(cfg *config.Config, logger *slog.Logger, h *Handlers) *gin.Engine {
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	engine := gin.New()
	engine.Use(middleware.ErrorHandler(logger), middleware.RequestLogger(logger))
	engine.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.AllowedOrigins(),
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-Request-ID"},
		ExposeHeaders:    []string{"X-Request-ID"},
		AllowCredentials: false,
	}))

	engine.GET("/healthz", h.Health.Liveness)
	engine.GET("/api/healthz", h.Health.Liveness)
	engine.GET("/readyz", h.Health.Readiness)
	engine.GET("/api/readyz", h.Health.Readiness)

	// Static API docs.
	engine.GET("/docs", docs.IndexHandler())
	engine.GET("/docs/openapi.json", docs.OpenAPIHandler())

	// Public mock endpoint: /mock/:projectId/*path. Registered at root level so
	// it matches the Nginx reverse-proxy path without duplication.
	engine.Any("/mock/:projectId/*path", middleware.RateLimit(cfg.MockRateLimit, logger), h.Mock.Serve)

	api := engine.Group("/api/v1")
	registerAuth(api, h, cfg, logger)
	registerProject(api, h, cfg, logger)
	registerEndpoint(api, h, cfg, logger)
	registerRequestLog(api, h, cfg, logger)

	return engine
}
