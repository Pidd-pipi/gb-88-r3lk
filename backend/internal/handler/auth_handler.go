// Package handler implements HTTP handlers.
package handler

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"github.com/mockhub/mockhub/internal/dto"
	"github.com/mockhub/mockhub/internal/middleware"
	"github.com/mockhub/mockhub/internal/service"
	"github.com/mockhub/mockhub/internal/util"
)

// AuthHandler exposes auth endpoints.
type AuthHandler struct {
	svc    *service.AuthService
	logger *slog.Logger
}

// NewAuthHandler builds an AuthHandler.
func NewAuthHandler(svc *service.AuthService, logger *slog.Logger) *AuthHandler {
	return &AuthHandler{svc: svc, logger: logger}
}

// Register handles POST /auth/register.
func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if !util.BindAndValidate(c, &req) {
		return
	}
	user, token, err := h.svc.Register(req.Username, req.Password)
	if err != nil {
		util.Fail(c, err)
		return
	}
	util.OK(c, dto.AuthResponse{Token: token, User: user})
}

// Login handles POST /auth/login.
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if !util.BindAndValidate(c, &req) {
		return
	}
	user, token, err := h.svc.Login(req.Username, req.Password)
	if err != nil {
		util.Fail(c, err)
		return
	}
	util.OK(c, dto.AuthResponse{Token: token, User: user})
}

// Me handles GET /auth/me.
func (h *AuthHandler) Me(c *gin.Context) {
	userID := middleware.GetUserID(c)
	user, err := h.svc.GetByID(userID)
	if err != nil {
		util.Fail(c, err)
		return
	}
	util.OK(c, user)
}
