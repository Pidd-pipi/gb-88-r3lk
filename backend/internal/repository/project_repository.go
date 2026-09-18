package repository

import (
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/mockhub/mockhub/internal/model"
)

// ProjectRepository persists mock projects.
type ProjectRepository struct {
	db *gorm.DB
}

// NewProjectRepository builds a ProjectRepository.
func NewProjectRepository(db *gorm.DB) *ProjectRepository {
	return &ProjectRepository{db: db}
}

// Create inserts a project.
func (r *ProjectRepository) Create(p *model.Project) error {
	if err := r.db.Create(p).Error; err != nil {
		return fmt.Errorf("create project: %w", err)
	}
	return nil
}

// List returns all projects, optionally filtered by owner.
func (r *ProjectRepository) List(userID uint, all bool) ([]model.Project, error) {
	var projects []model.Project
	q := r.db.Order("created_at DESC")
	if !all {
		q = q.Where("user_id = ?", userID)
	}
	if err := q.Find(&projects).Error; err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	return projects, nil
}

// FindByID loads a project by primary key.
func (r *ProjectRepository) FindByID(id uint) (*model.Project, error) {
	var p model.Project
	if err := r.db.First(&p, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find project by id: %w", err)
	}
	return &p, nil
}

// Update persists project changes.
func (r *ProjectRepository) Update(p *model.Project) error {
	if err := r.db.Save(p).Error; err != nil {
		return fmt.Errorf("update project: %w", err)
	}
	return nil
}

// Delete removes a project.
func (r *ProjectRepository) Delete(id uint) error {
	if err := r.db.Delete(&model.Project{}, id).Error; err != nil {
		return fmt.Errorf("delete project: %w", err)
	}
	return nil
}

// CountEndpoints returns how many mock endpoints a project contains.
func (r *ProjectRepository) CountEndpoints(projectID uint) (int64, error) {
	var count int64
	if err := r.db.Model(&model.MockAPI{}).Where("project_id = ?", projectID).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("count endpoints: %w", err)
	}
	return count, nil
}
