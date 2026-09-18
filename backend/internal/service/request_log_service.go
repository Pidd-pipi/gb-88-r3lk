package service

import (
	"fmt"
	"log/slog"

	"github.com/mockhub/mockhub/internal/model"
	"github.com/mockhub/mockhub/internal/repository"
)

// RequestLogService exposes paginated request logs.
type RequestLogService struct {
	projects *repository.ProjectRepository
	logs     *repository.RequestLogRepository
	logger   *slog.Logger
}

// NewRequestLogService builds a RequestLogService.
func NewRequestLogService(projects *repository.ProjectRepository, logs *repository.RequestLogRepository, logger *slog.Logger) *RequestLogService {
	return &RequestLogService{projects: projects, logs: logs, logger: logger}
}

// List returns logs for a project with pagination metadata. The caller must
// own the project (or be an administrator) before reading any log.
func (s *RequestLogService) List(projectID, userID uint, role string, page, limit int) ([]model.RequestLog, int64, error) {
	if _, err := checkProjectAccess(s.projects, projectID, userID, role); err != nil {
		return nil, 0, err
	}
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 200 {
		limit = 50
	}
	logs, total, err := s.logs.ListByProject(projectID, page, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("list logs: %w", err)
	}
	return logs, total, nil
}

// Clear removes all logs of a project after an ownership check.
func (s *RequestLogService) Clear(projectID, userID uint, role string) error {
	if _, err := checkProjectAccess(s.projects, projectID, userID, role); err != nil {
		return err
	}
	if err := s.logs.ClearByProject(projectID); err != nil {
		return fmt.Errorf("clear logs: %w", err)
	}
	s.logger.Info("request logs cleared", "project_id", projectID)
	return nil
}
