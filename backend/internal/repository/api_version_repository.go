package repository

import (
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/mockhub/mockhub/internal/model"
)

// APIVersionRepository queries immutable endpoint versions.
type APIVersionRepository struct {
	db *gorm.DB
}

// NewAPIVersionRepository builds an APIVersionRepository.
func NewAPIVersionRepository(db *gorm.DB) *APIVersionRepository {
	return &APIVersionRepository{db: db}
}

// ListByAPI returns version summaries of an endpoint, newest first. Heavy
// payload columns are skipped; use FindByAPIAndVersion for the full snapshot.
func (r *APIVersionRepository) ListByAPI(apiID uint) ([]model.APIVersion, error) {
	var versions []model.APIVersion
	if err := r.db.
		Select("id", "api_id", "version", "path", "method", "status_code", "delay", "editor_id", "editor_name", "created_at").
		Where("api_id = ?", apiID).
		Order("version DESC").
		Find(&versions).Error; err != nil {
		return nil, fmt.Errorf("list endpoint versions: %w", err)
	}
	return versions, nil
}

// FindByAPIAndVersion loads one full version snapshot.
func (r *APIVersionRepository) FindByAPIAndVersion(apiID uint, version int) (*model.APIVersion, error) {
	var v model.APIVersion
	if err := r.db.Where("api_id = ? AND version = ?", apiID, version).First(&v).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find endpoint version: %w", err)
	}
	return &v, nil
}
