package model

import "time"

type Permission struct {
	ID          int64     `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"size:64;not null" json:"name"`
	Code        string    `gorm:"size:64;uniqueIndex;not null" json:"code"`
	Resource    string    `gorm:"size:64;not null" json:"resource"`
	Action      string    `gorm:"size:64;not null" json:"action"`
	Description string    `gorm:"size:255;not null" json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
