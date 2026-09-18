package handler

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// HealthHandler exposes liveness and readiness probes.
type HealthHandler struct {
	db     *gorm.DB
	logger *slog.Logger
}

// NewHealthHandler builds a HealthHandler.
func NewHealthHandler(db *gorm.DB, logger *slog.Logger) *HealthHandler {
	return &HealthHandler{db: db, logger: logger}
}

// Liveness reports that the process is up. It intentionally does not
// depend on any external system so Kubernetes won't restart a healthy
// process during a transient database outage.
func (h *HealthHandler) Liveness(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "mockhub"})
}

// Readiness reports whether the service can serve traffic by pinging MySQL.
func (h *HealthHandler) Readiness(c *gin.Context) {
	sqlDB, err := h.db.DB()
	if err != nil {
		h.logger.Error("readiness: get sql db", "error", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable", "error": "database unavailable"})
		return
	}
	if err := sqlDB.Ping(); err != nil {
		h.logger.Error("readiness: ping database", "error", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable", "error": "database unavailable"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "mockhub"})
}
