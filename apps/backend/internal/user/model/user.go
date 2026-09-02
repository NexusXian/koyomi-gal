package model

import (
	"time"

	rbacModel "backend/internal/rbac/model"
)

type User struct {
	ID            uint             `gorm:"primaryKey" json:"id"`
	Username      string           `gorm:"size:50;not null" json:"username"`
	Email         string           `gorm:"size:255;uniqueIndex;not null" json:"email"`
	PasswordHash  string           `gorm:"not null" json:"-"`
	IsBanned      bool             `gorm:"not null" json:"-"`
	Avatar        string           `json:"avatar"`
	AvatarAssetID *uint            `gorm:"column:avatar_asset_id" json:"-"`
	CreatedAt     time.Time        `json:"created_at"`
	UpdatedAt     time.Time        `json:"updated_at"`
	Roles         []rbacModel.Role `gorm:"many2many:user_roles;foreignKey:ID;joinForeignKey:UserID;references:ID;joinReferences:RoleID" json:"-"`
}
