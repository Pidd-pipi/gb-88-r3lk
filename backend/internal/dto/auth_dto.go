// Package dto defines request/response payloads with validator rules.
package dto

import "github.com/mockhub/mockhub/internal/model"

// LoginRequest is the login payload.
type LoginRequest struct {
	Username string `json:"username" validate:"required,min=2,max=64"`
	Password string `json:"password" validate:"required,min=6,max=128"`
}

// RegisterRequest is the registration payload.
type RegisterRequest struct {
	Username string `json:"username" validate:"required,min=2,max=64"`
	Password string `json:"password" validate:"required,min=6,max=128"`
}

// AuthResponse carries the JWT token and the authenticated user.
type AuthResponse struct {
	Token string     `json:"token"`
	User  *model.User `json:"user"`
}
