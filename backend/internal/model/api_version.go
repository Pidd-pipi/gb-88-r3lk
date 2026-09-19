package model

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// APIVersion is an immutable snapshot of a mock endpoint's configuration.
// Every edit appends a new version row recording the editor and the time;
// version rows are never updated or deleted, so any of them can be previewed
// and published again later (rollback).
type APIVersion struct {
	ID                uint      `gorm:"primaryKey" json:"_id,string"`
	APIID             uint      `gorm:"uniqueIndex:idx_api_version;not null" json:"apiId,string"`
	Version           int       `gorm:"uniqueIndex:idx_api_version;not null" json:"version"`
	Path              string    `gorm:"size:255;not null" json:"path"`
	Method            string    `gorm:"size:16;not null" json:"method"`
	StatusCode        int       `gorm:"not null;default:200" json:"statusCode"`
	ResponseBody      string    `gorm:"type:text" json:"responseBody"`
	ResponseHeadersJS string    `gorm:"column:response_headers;type:text" json:"-"`
	Delay             int       `gorm:"not null;default:0" json:"delay"`
	ConditionsJS      string    `gorm:"column:conditions;type:text" json:"-"`
	EditorID          uint      `gorm:"not null;default:0" json:"editorId,string"`
	EditorName        string    `gorm:"size:64;not null;default:''" json:"editorName"`
	CreatedAt         time.Time `json:"createdAt"`

	// Computed fields (not persisted).
	ResponseHeaders map[string]string `gorm:"-" json:"responseHeaders"`
	Conditions      []ConditionRule   `gorm:"-" json:"conditions"`
}

// BeforeSave serializes computed header/condition fields into JSON columns.
func (v *APIVersion) BeforeSave(_ *gorm.DB) error {
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
func (v *APIVersion) AfterFind(_ *gorm.DB) error {
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
