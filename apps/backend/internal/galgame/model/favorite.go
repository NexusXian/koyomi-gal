package model

import "time"

type Favorite struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	GalgameID uint      `gorm:"not null" json:"galgame_id"`
	UserID    uint      `gorm:"not null" json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}

func (Favorite) TableName() string {
	return "galgame_favorites"
}
