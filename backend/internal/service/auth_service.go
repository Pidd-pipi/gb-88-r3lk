// Package service implements business logic on top of repositories.
package service

import (
	"errors"
	"fmt"
	"log/slog"

	"golang.org/x/crypto/bcrypt"

	"github.com/mockhub/mockhub/internal/config"
	"github.com/mockhub/mockhub/internal/constants"
	"github.com/mockhub/mockhub/internal/model"
	"github.com/mockhub/mockhub/internal/repository"
	"github.com/mockhub/mockhub/internal/util"
)

// Role constants.
const (
	RoleDev  = "dev"
	RoleLead = "lead"
)

// AuthService handles registration and login.
type AuthService struct {
	cfg    *config.Config
	users  *repository.UserRepository
	logger *slog.Logger
}

// NewAuthService builds an AuthService.
func NewAuthService(cfg *config.Config, users *repository.UserRepository, logger *slog.Logger) *AuthService {
	return &AuthService{cfg: cfg, users: users, logger: logger}
}

// Register creates a new account and returns a token.
func (s *AuthService) Register(username, password string) (*model.User, string, error) {
	exists, err := s.users.ExistsByUsername(username)
	if err != nil {
		return nil, "", fmt.Errorf("register: %w", err)
	}
	if exists {
		return nil, "", constants.NewAppError(constants.CodeConflict, constants.MsgUsernameTaken)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", fmt.Errorf("register: hash password: %w", err)
	}
	user := &model.User{Username: username, PasswordHash: string(hash), Role: RoleDev}
	if err := s.users.Create(user); err != nil {
		return nil, "", fmt.Errorf("register: %w", err)
	}
	token, err := util.GenerateToken(s.cfg.JWTSecret, user.ID, user.Role, s.cfg.JWTExpireDuration())
	if err != nil {
		return nil, "", fmt.Errorf("register: generate token: %w", err)
	}
	s.logger.Info("user registered", "user_id", user.ID, "username", username)
	return user, token, nil
}

// Login authenticates a user and returns a token.
func (s *AuthService) Login(username, password string) (*model.User, string, error) {
	user, err := s.users.FindByUsername(username)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, "", constants.NewAppError(constants.CodeUnauthorized, constants.MsgLoginFailed)
		}
		return nil, "", fmt.Errorf("login: %w", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, "", constants.NewAppError(constants.CodeUnauthorized, constants.MsgLoginFailed)
	}
	token, err := util.GenerateToken(s.cfg.JWTSecret, user.ID, user.Role, s.cfg.JWTExpireDuration())
	if err != nil {
		return nil, "", fmt.Errorf("login: generate token: %w", err)
	}
	s.logger.Info("user logged in", "user_id", user.ID, "username", username)
	return user, token, nil
}

// GetByID loads a user for the /auth/me endpoint.
func (s *AuthService) GetByID(id uint) (*model.User, error) {
	return s.users.FindByID(id)
}
