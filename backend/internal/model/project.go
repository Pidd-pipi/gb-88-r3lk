package model

import "time"

// Project groups a set of mock endpoints and their request logs.
type Project struct {
	ID          uint      `gorm:"primaryKey" json:"_id,string"`
	Name        string    `gorm:"size:128;not null" json:"name"`
	Description string    `gorm:"size:512" json:"description"`
	BaseURL     string    `gorm:"size:255" json:"baseUrl"`
	UserID      uint      `gorm:"index;not null" json:"userId,string"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"-"`
}
