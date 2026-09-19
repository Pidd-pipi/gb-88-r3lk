package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/mockhub/mockhub/internal/model"
)

// ErrVersionConflict is returned when a compare-and-swap on an endpoint's
// current version loses a race against another writer. The losing request
// changes nothing.
var ErrVersionConflict = errors.New("version conflict")

// EndpointVersionRepository persists immutable endpoint version snapshots and
// runs the version-aware transactional workflows (create / edit / publish /
// delete) that must update the endpoints and versions tables atomically.
type EndpointVersionRepository struct {
	db *gorm.DB
}

// NewEndpointVersionRepository builds an EndpointVersionRepository.
func NewEndpointVersionRepository(db *gorm.DB) *EndpointVersionRepository {
	return &EndpointVersionRepository{db: db}
}

// ListByAPI returns every version of an endpoint, newest first.
func (r *EndpointVersionRepository) ListByAPI(apiID uint) ([]model.MockAPIVersion, error) {
	var versions []model.MockAPIVersion
	if err := r.db.Where("api_id = ?", apiID).Order("version DESC").Find(&versions).Error; err != nil {
		return nil, fmt.Errorf("list endpoint versions: %w", err)
	}
	return versions, nil
}

// FindByAPIAndVersion loads one immutable version snapshot.
func (r *EndpointVersionRepository) FindByAPIAndVersion(apiID uint, version int) (*model.MockAPIVersion, error) {
	var v model.MockAPIVersion
	if err := r.db.Where("api_id = ? AND version = ?", apiID, version).First(&v).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find endpoint version: %w", err)
	}
	return &v, nil
}

// CreateWithFirstVersion inserts a new endpoint together with its v1 snapshot
// in one transaction.
func (r *EndpointVersionRepository) CreateWithFirstVersion(e *model.MockAPI, editorID uint, editorName string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		e.CurrentVersion = 1
		if err := tx.Create(e).Error; err != nil {
			return fmt.Errorf("create endpoint: %w", err)
		}
		if err := tx.Create(model.SnapshotFrom(e, 1, editorID, editorName)).Error; err != nil {
			return fmt.Errorf("create first endpoint version: %w", err)
		}
		return nil
	})
}

// ApplyEdit appends a new immutable version holding next's content and points
// the endpoint at it, atomically. Endpoints without any version history yet
// (created before versioning existed) first get a v1 snapshot of their
// previous content so the pre-edit state stays recoverable.
//
// expectedCurrent is the compare-and-swap token: the endpoint row is only
// updated while its current_version still equals expectedCurrent. A mismatch
// (or a duplicate version number allocated by a racing writer) yields
// ErrVersionConflict and rolls the whole transaction back.
func (r *EndpointVersionRepository) ApplyEdit(prev, next *model.MockAPI, editorID uint, editorName string, expectedCurrent int) (*model.MockAPIVersion, error) {
	// Serialize the computed map fields exactly like the GORM hooks would, so
	// the plain column update below carries the same semantics as a Save.
	if err := next.SerializeJSONFields(); err != nil {
		return nil, fmt.Errorf("serialize endpoint fields: %w", err)
	}

	var created *model.MockAPIVersion
	err := r.db.Transaction(func(tx *gorm.DB) error {
		maxVersion, err := maxEndpointVersion(tx, prev.ID)
		if err != nil {
			return err
		}
		if maxVersion == 0 {
			// Legacy endpoint: preserve the pre-edit content as v1.
			if err := tx.Create(model.SnapshotFrom(prev, 1, editorID, editorName)).Error; err != nil {
				return fmt.Errorf("backfill legacy endpoint version: %w", err)
			}
			maxVersion = 1
		}
		newVersion := maxVersion + 1

		res := tx.Model(&model.MockAPI{}).
			Where("id = ? AND current_version = ?", next.ID, expectedCurrent).
			Updates(map[string]any{
				"path":             next.Path,
				"method":           next.Method,
				"status_code":      next.StatusCode,
				"response_body":    next.ResponseBody,
				"response_headers": next.ResponseHeadersJS,
				"delay":            next.Delay,
				"conditions":       next.ConditionsJS,
				"current_version":  newVersion,
				"updated_at":       time.Now(),
			})
		if res.Error != nil {
			return fmt.Errorf("update endpoint: %w", res.Error)
		}
		if res.RowsAffected == 0 {
			return ErrVersionConflict
		}

		created = model.SnapshotFrom(next, newVersion, editorID, editorName)
		if err := tx.Create(created).Error; err != nil {
			if isDuplicateKey(err) {
				return ErrVersionConflict
			}
			return fmt.Errorf("create endpoint version: %w", err)
		}
		next.CurrentVersion = newVersion
		return nil
	})
	if err != nil {
		return nil, err
	}
	return created, nil
}

// ApplyPublish points the endpoint at an existing immutable version snapshot
// (rollback), guarded by a compare-and-swap on expectedCurrent. Exactly one of
// several concurrent publishers wins; losers get ErrVersionConflict and the
// current version is left untouched.
func (r *EndpointVersionRepository) ApplyPublish(endpointID uint, v *model.MockAPIVersion, expectedCurrent int) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&model.MockAPI{}).
			Where("id = ? AND current_version = ?", endpointID, expectedCurrent).
			Updates(map[string]any{
				"path":             v.Path,
				"method":           v.Method,
				"status_code":      v.StatusCode,
				"response_body":    v.ResponseBody,
				"response_headers": v.ResponseHeadersJS,
				"delay":            v.Delay,
				"conditions":       v.ConditionsJS,
				"current_version":  v.Version,
				"updated_at":       time.Now(),
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

// DeleteWithVersions removes an endpoint and all of its version snapshots.
func (r *EndpointVersionRepository) DeleteWithVersions(endpointID uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("api_id = ?", endpointID).Delete(&model.MockAPIVersion{}).Error; err != nil {
			return fmt.Errorf("delete endpoint versions: %w", err)
		}
		if err := tx.Delete(&model.MockAPI{}, endpointID).Error; err != nil {
			return fmt.Errorf("delete endpoint: %w", err)
		}
		return nil
	})
}

// maxEndpointVersion returns the highest allocated version number of an
// endpoint (0 when it has no snapshots yet).
func maxEndpointVersion(tx *gorm.DB, apiID uint) (int, error) {
	var max sql.NullInt64
	if err := tx.Model(&model.MockAPIVersion{}).
		Where("api_id = ?", apiID).
		Select("MAX(version)").
		Scan(&max).Error; err != nil {
		return 0, fmt.Errorf("max endpoint version: %w", err)
	}
	if !max.Valid {
		return 0, nil
	}
	return int(max.Int64), nil
}

// isDuplicateKey reports a unique-constraint violation across MySQL and
// SQLite (used by tests).
func isDuplicateKey(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "Duplicate entry") ||
		strings.Contains(msg, "UNIQUE constraint failed") ||
		strings.Contains(msg, "duplicate key")
}
