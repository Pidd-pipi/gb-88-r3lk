package repository

import (
	"fmt"

	"gorm.io/gorm"

	"github.com/mockhub/mockhub/internal/model"
)

// RequestLogRepository persists mock request logs.
type RequestLogRepository struct {
	db *gorm.DB
}

// NewRequestLogRepository builds a RequestLogRepository.
func NewRequestLogRepository(db *gorm.DB) *RequestLogRepository {
	return &RequestLogRepository{db: db}
}

// Create inserts a request log.
func (r *RequestLogRepository) Create(l *model.RequestLog) error {
	if err := r.db.Create(l).Error; err != nil {
		return fmt.Errorf("create request log: %w", err)
	}
	return nil
}

// ListByProject returns logs for a project with pagination.
func (r *RequestLogRepository) ListByProject(projectID uint, page, limit int) ([]model.RequestLog, int64, error) {
	var logs []model.RequestLog
	var total int64
	if err := r.db.Model(&model.RequestLog{}).Where("project_id = ?", projectID).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count request logs: %w", err)
	}
	if err := r.db.Where("project_id = ?", projectID).
		Order("created_at DESC").
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&logs).Error; err != nil {
		return nil, 0, fmt.Errorf("list request logs: %w", err)
	}
	return logs, total, nil
}

// ClearByProject removes all logs of a project.
func (r *RequestLogRepository) ClearByProject(projectID uint) error {
	if err := r.db.Where("project_id = ?", projectID).Delete(&model.RequestLog{}).Error; err != nil {
		return fmt.Errorf("clear request logs: %w", err)
	}
	return nil
}
