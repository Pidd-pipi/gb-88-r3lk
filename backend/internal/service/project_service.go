package service

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/mockhub/mockhub/internal/model"
	"github.com/mockhub/mockhub/internal/repository"
)

// ProjectService manages mock projects.
type ProjectService struct {
	projects *repository.ProjectRepository
	logger   *slog.Logger
}

// NewProjectService builds a ProjectService.
func NewProjectService(projects *repository.ProjectRepository, logger *slog.Logger) *ProjectService {
	return &ProjectService{projects: projects, logger: logger}
}

// List returns projects visible to the caller.
func (s *ProjectService) List(userID uint, role string) ([]model.Project, error) {
	return s.projects.List(userID, role == RoleLead)
}

// Get loads a project with ownership check.
func (s *ProjectService) Get(id, userID uint, role string) (*model.Project, error) {
	return checkProjectAccess(s.projects, id, userID, role)
}

// Create creates a project and computes its public mock base URL.
func (s *ProjectService) Create(name, description string, userID uint, baseURL string) (*model.Project, error) {
	p := &model.Project{
		Name:        name,
		Description: description,
		UserID:      userID,
	}
	if err := s.projects.Create(p); err != nil {
		return nil, err
	}
	p.BaseURL = fmt.Sprintf("%s/mock/%d", baseURL, p.ID)
	if err := s.projects.Update(p); err != nil {
		return nil, fmt.Errorf("set project base url: %w", err)
	}
	s.logger.Info("project created", "project_id", p.ID, "user_id", userID)
	return p, nil
}

// Update edits a project after checking ownership.
func (s *ProjectService) Update(id, userID uint, role, name, description string) (*model.Project, error) {
	p, err := s.Get(id, userID, role)
	if err != nil {
		return nil, err
	}
	p.Name = name
	p.Description = description
	if err := s.projects.Update(p); err != nil {
		return nil, err
	}
	return p, nil
}

// Delete removes a project after checking ownership.
func (s *ProjectService) Delete(id, userID uint, role string) error {
	if _, err := s.Get(id, userID, role); err != nil {
		return err
	}
	if err := s.projects.Delete(id); err != nil {
		return err
	}
	s.logger.Info("project deleted", "project_id", id, "user_id", userID)
	return nil
}

// EnsureBaseURL backfills the base URL for seed projects.
func (s *ProjectService) EnsureBaseURL(id uint, baseURL string) error {
	p, err := s.projects.FindByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil
		}
		return err
	}
	if p.BaseURL == "" {
		p.BaseURL = fmt.Sprintf("%s/mock/%d", baseURL, p.ID)
		if err := s.projects.Update(p); err != nil {
			return err
		}
	}
	return nil
}
