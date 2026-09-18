package handler

import (
	"log/slog"
	"github.com/gin-gonic/gin"

	"github.com/mockhub/mockhub/internal/config"
	"github.com/mockhub/mockhub/internal/dto"
	"github.com/mockhub/mockhub/internal/middleware"
	"github.com/mockhub/mockhub/internal/service"
	"github.com/mockhub/mockhub/internal/util"
)

// ProjectHandler exposes project endpoints.
type ProjectHandler struct {
	svc    *service.ProjectService
	cfg    *config.Config
	logger *slog.Logger
}

// NewProjectHandler builds a ProjectHandler.
func NewProjectHandler(svc *service.ProjectService, cfg *config.Config, logger *slog.Logger) *ProjectHandler {
	return &ProjectHandler{svc: svc, cfg: cfg, logger: logger}
}

// List handles GET /projects.
func (h *ProjectHandler) List(c *gin.Context) {
	projects, err := h.svc.List(middleware.GetUserID(c), middleware.GetRole(c))
	if err != nil {
		util.Fail(c, err)
		return
	}
	util.OK(c, projects)
}

// Get handles GET /projects/:id.
func (h *ProjectHandler) Get(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	p, err := h.svc.Get(id, middleware.GetUserID(c), middleware.GetRole(c))
	if err != nil {
		util.Fail(c, err)
		return
	}
	util.OK(c, p)
}

// Create handles POST /projects.
func (h *ProjectHandler) Create(c *gin.Context) {
	var req dto.CreateProjectRequest
	if !util.BindAndValidate(c, &req) {
		return
	}
	p, err := h.svc.Create(req.Name, req.Description, middleware.GetUserID(c), h.cfg.BaseURL)
	if err != nil {
		util.Fail(c, err)
		return
	}
	util.OK(c, p)
}

// Update handles PUT /projects/:id.
func (h *ProjectHandler) Update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req dto.UpdateProjectRequest
	if !util.BindAndValidate(c, &req) {
		return
	}
	p, err := h.svc.Update(id, middleware.GetUserID(c), middleware.GetRole(c), req.Name, req.Description)
	if err != nil {
		util.Fail(c, err)
		return
	}
	util.OK(c, p)
}

// Delete handles DELETE /projects/:id.
func (h *ProjectHandler) Delete(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.svc.Delete(id, middleware.GetUserID(c), middleware.GetRole(c)); err != nil {
		util.Fail(c, err)
		return
	}
	util.OK(c, gin.H{"deleted": true})
}
