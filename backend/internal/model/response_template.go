package model

import "time"

// ResponseTemplate stores a reusable named response body template.
// It is provided as a reference entity: mock endpoints may optionally use a
// template by referencing it from the endpoint service.
type ResponseTemplate struct {
	ID        uint      `gorm:"primaryKey" json:"_id,string"`
	ProjectID uint      `gorm:"index;not null" json:"projectId,string"`
	Name      string    `gorm:"size:128;not null" json:"name"`
	Body      string    `gorm:"type:text" json:"body"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"-"`
}
