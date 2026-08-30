package model

import "time"

type UserRole struct {
	UserID    uint      `gorm:"primaryKey" json:"user_id"`
	RoleID    int64     `gorm:"primaryKey" json:"role_id"`
	CreatedAt time.Time `json:"created_at"`
}
