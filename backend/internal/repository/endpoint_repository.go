package repository

import (
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/mockhub/mockhub/internal/model"
)

// EndpointRepository persists mock endpoints.
type EndpointRepository struct {
	db *gorm.DB
}

// NewEndpointRepository builds an EndpointRepository.
func NewEndpointRepository(db *gorm.DB) *EndpointRepository {
	return &EndpointRepository{db: db}
}

// Create inserts a mock endpoint.
func (r *EndpointRepository) Create(e *model.MockAPI) error {
	if err := r.db.Create(e).Error; err != nil {
		return fmt.Errorf("create endpoint: %w", err)
	}
	return nil
}

// ListByProject returns all endpoints of a project.
func (r *EndpointRepository) ListByProject(projectID uint) ([]model.MockAPI, error) {
	var endpoints []model.MockAPI
	if err := r.db.Where("project_id = ?", projectID).Order("id ASC").Find(&endpoints).Error; err != nil {
		return nil, fmt.Errorf("list endpoints: %w", err)
	}
	return endpoints, nil
}

// FindByID loads an endpoint by primary key.
func (r *EndpointRepository) FindByID(id uint) (*model.MockAPI, error) {
	var e model.MockAPI
	if err := r.db.First(&e, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find endpoint by id: %w", err)
	}
	return &e, nil
}

// Update persists endpoint changes.
func (r *EndpointRepository) Update(e *model.MockAPI) error {
	if err := r.db.Save(e).Error; err != nil {
		return fmt.Errorf("update endpoint: %w", err)
	}
	return nil
}

// Delete removes an endpoint.
func (r *EndpointRepository) Delete(id uint) error {
	if err := r.db.Delete(&model.MockAPI{}, id).Error; err != nil {
		return fmt.Errorf("delete endpoint: %w", err)
	}
	return nil
}

// Match finds the first endpoint matching method and path (with :param support).
func (r *EndpointRepository) Match(projectID uint, method, path string) (*model.MockAPI, map[string]string, error) {
	endpoints, err := r.ListByProject(projectID)
	if err != nil {
		return nil, nil, err
	}
	for i := range endpoints {
		params, ok := matchPath(endpoints[i].Path, path)
		if ok && equalFold(endpoints[i].Method, method) {
			return &endpoints[i], params, nil
		}
	}
	return nil, nil, ErrNotFound
}
