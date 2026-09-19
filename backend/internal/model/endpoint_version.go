package model

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// MockAPIVersion is an immutable snapshot of a mock endpoint's content.
// Every edit appends a new row (versions are never updated or deleted while
// the endpoint exists); publishing an old version flips the endpoint's
// current_version pointer back to it, which is how rollback is implemented.
// The (api_id, version) pair is unique so concurrent writers cannot allocate
// the same version number.
type MockAPIVersion struct {
	ID                uint      `gorm:"primaryKey" json:"_id,string"`
	APIID             uint      `gorm:"uniqueIndex:uk_api_version,priority:1;not null" json:"apiId,string"`
	Version           int       `gorm:"uniqueIndex:uk_api_version,priority:2;not null" json:"version"`
	Path              string    `gorm:"size:255;not null" json:"path"`
	Method            string    `gorm:"size:16;not null" json:"method"`
	StatusCode        int       `gorm:"not null;default:200" json:"statusCode"`
	ResponseBody      string    `gorm:"type:text" json:"responseBody"`
	ResponseHeadersJS string    `gorm:"column:response_headers;type:text" json:"-"`
	Delay             int       `gorm:"not null;default:0" json:"delay"`
	ConditionsJS      string    `gorm:"column:conditions;type:text" json:"-"`
	CreatedBy         uint      `gorm:"not null" json:"createdBy,string"`
	CreatedByName     string    `gorm:"size:64;not null;default:''" json:"createdByName"`
	CreatedAt         time.Time `json:"createdAt"`

	// Computed fields (not persisted).
	ResponseHeaders map[string]string `gorm:"-" json:"responseHeaders"`
	Conditions      []ConditionRule   `gorm:"-" json:"conditions"`
}

// SnapshotFrom builds an immutable version row from an endpoint's current
// content. The computed map fields are copied and serialized by BeforeSave.
func SnapshotFrom(a *MockAPI, version int, editorID uint, editorName string) *MockAPIVersion {
	return &MockAPIVersion{
		APIID:           a.ID,
		Version:         version,
		Path:            a.Path,
		Method:          a.Method,
		StatusCode:      a.StatusCode,
		ResponseBody:    a.ResponseBody,
		ResponseHeaders: a.ResponseHeaders,
		Delay:           a.Delay,
		Conditions:      a.Conditions,
		CreatedBy:       editorID,
		CreatedByName:   editorName,
	}
}

// BeforeSave serializes computed header/condition fields into JSON columns.
func (v *MockAPIVersion) BeforeSave(_ *gorm.DB) error {
	if v.ResponseHeaders != nil {
		b, err := json.Marshal(v.ResponseHeaders)
		if err != nil {
			return err
		}
		v.ResponseHeadersJS = string(b)
	}
	if v.Conditions != nil {
		b, err := json.Marshal(v.Conditions)
		if err != nil {
			return err
		}
		v.ConditionsJS = string(b)
	}
	return nil
}

// AfterFind restores computed fields from JSON columns.
func (v *MockAPIVersion) AfterFind(_ *gorm.DB) error {
	v.ResponseHeaders = map[string]string{}
	if v.ResponseHeadersJS != "" {
		_ = json.Unmarshal([]byte(v.ResponseHeadersJS), &v.ResponseHeaders)
	}
	v.Conditions = []ConditionRule{}
	if v.ConditionsJS != "" {
		_ = json.Unmarshal([]byte(v.ConditionsJS), &v.Conditions)
	}
	return nil
}
