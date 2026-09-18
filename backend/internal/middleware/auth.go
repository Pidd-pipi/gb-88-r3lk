// Package middleware provides HTTP middlewares shared by the API.
package middleware

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/mockhub/mockhub/internal/config"
	"github.com/mockhub/mockhub/internal/constants"
	"github.com/mockhub/mockhub/internal/util"
)

const (
	ctxKeyUserID = "auth_user_id"
	ctxKeyRole   = "auth_role"
)

// JWTAuth validates the Bearer token and injects the user identity.
func JWTAuth(cfg *config.Config, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, util.Response{
				Code:    constants.CodeUnauthorized,
				Message: constants.MsgUnauthorized,
			})
			return
		}
		token := strings.TrimPrefix(header, "Bearer ")
		claims, err := util.ParseToken(cfg.JWTSecret, token)
		if err != nil {
			logger.Warn("invalid jwt", "error", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, util.Response{
				Code:    constants.CodeUnauthorized,
				Message: constants.MsgUnauthorized,
			})
			return
		}
		c.Set(ctxKeyUserID, claims.UserID)
		c.Set(ctxKeyRole, claims.Role)
		c.Next()
	}
}

// GetUserID returns the authenticated user id from the context.
func GetUserID(c *gin.Context) uint {
	if v, ok := c.Get(ctxKeyUserID); ok {
		if id, ok := v.(uint); ok {
			return id
		}
	}
	return 0
}

// GetRole returns the authenticated user role from the context.
func GetRole(c *gin.Context) string {
	if v, ok := c.Get(ctxKeyRole); ok {
		if role, ok := v.(string); ok {
			return role
		}
	}
	return ""
}
