// Package model defines the GORM entities of the API Mock service.
package model

import "time"

// User is a platform account (dev or lead/admin).
type User struct {
	ID           uint      `gorm:"primaryKey" json:"_id,string"`
	Username     string    `gorm:"size:64;uniqueIndex;not null" json:"username"`
	PasswordHash string    `gorm:"size:255;not null" json:"-"`
	Role         string    `gorm:"size:32;not null;default:dev" json:"role"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"-"`
}
