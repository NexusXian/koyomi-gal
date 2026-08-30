package model

import "time"

type RolePermission struct {
	RoleID       int64     `gorm:"primaryKey" json:"role_id"`
	PermissionID int64     `gorm:"primaryKey" json:"permission_id"`
	CreatedAt    time.Time `json:"created_at"`
}
