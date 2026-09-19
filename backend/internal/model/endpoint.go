package model

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// ConditionRule is a conditional response rule evaluated against query/body fields.
type ConditionRule struct {
	Field        string `json:"field"`
	Operator     string `json:"operator"`
	Value        string `json:"value"`
	ResponseBody string `json:"responseBody"`
	StatusCode   int    `json:"statusCode"`
}

// MockAPI is a configurable mock endpoint under a project.
type MockAPI struct {
	ID                uint   `gorm:"primaryKey" json:"_id,string"`
	ProjectID         uint   `gorm:"index;not null" json:"projectId,string"`
	Path              string `gorm:"size:255;not null" json:"path"`
	Method            string `gorm:"size:16;not null" json:"method"`
	StatusCode        int    `gorm:"not null;default:200" json:"statusCode"`
	ResponseBody      string `gorm:"type:text" json:"responseBody"`
	ResponseHeadersJS string `gorm:"column:response_headers;type:text" json:"-"`
	Delay             int    `gorm:"not null;default:0" json:"delay"`
	ConditionsJS      string `gorm:"column:conditions;type:text" json:"-"`
	// CurrentVersion points at the published row in mock_api_versions.
	// Zero means the endpoint predates versioning (legacy) and has no
	// snapshots yet; it is backfilled lazily on the first edit.
	CurrentVersion int       `gorm:"not null;default:0" json:"currentVersion"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"-"`

	// Computed fields (not persisted).
	ResponseHeaders map[string]string `gorm:"-" json:"responseHeaders"`
	Conditions      []ConditionRule   `gorm:"-" json:"conditions"`
}

// BeforeSave serializes computed header/condition fields into JSON columns.
func (a *MockAPI) BeforeSave(_ *gorm.DB) error {
	return a.SerializeJSONFields()
}

// SerializeJSONFields marshals the computed map fields into the JSON string
// columns. It is called by the BeforeSave hook and by repositories that
// update the columns directly.
func (a *MockAPI) SerializeJSONFields() error {
	if a.ResponseHeaders != nil {
		b, err := json.Marshal(a.ResponseHeaders)
		if err != nil {
			return err
		}
		a.ResponseHeadersJS = string(b)
	}
	if a.Conditions != nil {
		b, err := json.Marshal(a.Conditions)
		if err != nil {
			return err
		}
		a.ConditionsJS = string(b)
	}
	return nil
}

// AfterFind restores computed fields from JSON columns.
func (a *MockAPI) AfterFind(_ *gorm.DB) error {
	a.ResponseHeaders = map[string]string{}
	if a.ResponseHeadersJS != "" {
		_ = json.Unmarshal([]byte(a.ResponseHeadersJS), &a.ResponseHeaders)
	}
	a.Conditions = []ConditionRule{}
	if a.ConditionsJS != "" {
		_ = json.Unmarshal([]byte(a.ConditionsJS), &a.Conditions)
	}
	return nil
}
