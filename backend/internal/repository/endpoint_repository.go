package repository

import (
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/mockhub/mockhub/internal/model"
)

// ErrVersionConflict reports a lost optimistic-concurrency race: the endpoint's
// current version changed between read and write, so the write was rejected
// without touching the live version.
var ErrVersionConflict = errors.New("version conflict")

// EndpointRepository persists mock endpoints and appends immutable versions.
type EndpointRepository struct {
	db *gorm.DB
}

// NewEndpointRepository builds an EndpointRepository.
func NewEndpointRepository(db *gorm.DB) *EndpointRepository {
	return &EndpointRepository{db: db}
}

// CreateWithVersion inserts an endpoint together with its first version snapshot.
func (r *EndpointRepository) CreateWithVersion(e *model.MockAPI, v *model.APIVersion) error {
	if err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(e).Error; err != nil {
			return fmt.Errorf("create endpoint: %w", err)
		}
		v.APIID = e.ID
		if err := tx.Create(v).Error; err != nil {
			return fmt.Errorf("create endpoint version: %w", err)
		}
		return nil
	}); err != nil {
		return err
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

// UpdateWithVersion CAS-updates the endpoint's live content and appends the new
// immutable version in one transaction. The update only applies when the
// endpoint is still at baseVersion; a concurrent writer flips the guard first
// and this call fails with ErrVersionConflict leaving the row untouched.
func (r *EndpointRepository) UpdateWithVersion(e *model.MockAPI, v *model.APIVersion, baseVersion int) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := e.BeforeSave(tx); err != nil {
			return fmt.Errorf("serialize endpoint: %w", err)
		}
		res := tx.Model(&model.MockAPI{}).
			Where("id = ? AND current_version = ?", e.ID, baseVersion).
			Updates(map[string]any{
				"path":             e.Path,
				"method":           e.Method,
				"status_code":      e.StatusCode,
				"response_body":    e.ResponseBody,
				"response_headers": e.ResponseHeadersJS,
				"delay":            e.Delay,
				"conditions":       e.ConditionsJS,
				"current_version":  v.Version,
			})
		if res.Error != nil {
			return fmt.Errorf("update endpoint: %w", res.Error)
		}
		if res.RowsAffected == 0 {
			return ErrVersionConflict
		}
		// Mirror the exact serialized payload onto the snapshot so the version
		// row always matches the live content byte for byte.
		v.ResponseHeadersJS = e.ResponseHeadersJS
		v.ConditionsJS = e.ConditionsJS
		v.APIID = e.ID
		if err := tx.Create(v).Error; err != nil {
			return fmt.Errorf("create endpoint version: %w", err)
		}
		return nil
	})
}

// PublishVersion points the endpoint at an existing immutable version. The CAS
// guard on baseVersion makes concurrent publishes mutually exclusive: exactly
// one transaction wins, the loser gets ErrVersionConflict and changes nothing.
func (r *EndpointRepository) PublishVersion(apiID uint, target *model.APIVersion, baseVersion int) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&model.MockAPI{}).
			Where("id = ? AND current_version = ?", apiID, baseVersion).
			Updates(map[string]any{
				"path":             target.Path,
				"method":           target.Method,
				"status_code":      target.StatusCode,
				"response_body":    target.ResponseBody,
				"response_headers": target.ResponseHeadersJS,
				"delay":            target.Delay,
				"conditions":       target.ConditionsJS,
				"current_version":  target.Version,
			})
		if res.Error != nil {
			return fmt.Errorf("publish endpoint version: %w", res.Error)
		}
		if res.RowsAffected == 0 {
			return ErrVersionConflict
		}
		return nil
	})
}

// Delete removes an endpoint together with its version history.
func (r *EndpointRepository) Delete(id uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("api_id = ?", id).Delete(&model.APIVersion{}).Error; err != nil {
			return fmt.Errorf("delete endpoint versions: %w", err)
		}
		if err := tx.Delete(&model.MockAPI{}, id).Error; err != nil {
			return fmt.Errorf("delete endpoint: %w", err)
		}
		return nil
	})
}

// BackfillVersions snapshots legacy endpoints (created before versioning
// existed, current_version = 0) into an initial v1 so they join the versioned
// workflow. It is idempotent and safe to run on every boot.
func (r *EndpointRepository) BackfillVersions(editorName string) (int, error) {
	var legacy []model.MockAPI
	if err := r.db.Where("current_version = ?", 0).Order("id ASC").Find(&legacy).Error; err != nil {
		return 0, fmt.Errorf("list legacy endpoints: %w", err)
	}
	if len(legacy) == 0 {
		return 0, nil
	}
	migrated := 0
	err := r.db.Transaction(func(tx *gorm.DB) error {
		for i := range legacy {
			e := &legacy[i]
			v := &model.APIVersion{
				APIID:             e.ID,
				Version:           1,
				Path:              e.Path,
				Method:            e.Method,
				StatusCode:        e.StatusCode,
				ResponseBody:      e.ResponseBody,
				ResponseHeadersJS: e.ResponseHeadersJS,
				Delay:             e.Delay,
				ConditionsJS:      e.ConditionsJS,
				EditorName:        editorName,
			}
			if err := tx.Create(v).Error; err != nil {
				return fmt.Errorf("backfill endpoint %d version: %w", e.ID, err)
			}
			res := tx.Model(&model.MockAPI{}).
				Where("id = ? AND current_version = ?", e.ID, 0).
				Update("current_version", 1)
			if res.Error != nil {
				return fmt.Errorf("backfill endpoint %d pointer: %w", e.ID, res.Error)
			}
			migrated++
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return migrated, nil
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
