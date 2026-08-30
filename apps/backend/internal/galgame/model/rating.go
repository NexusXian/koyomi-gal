package model

import "time"

type Rating struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	GalgameID uint      `gorm:"not null" json:"galgame_id"`
	UserID    uint      `gorm:"not null" json:"user_id"`
	Score     int16     `gorm:"not null" json:"score"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Rating) TableName() string {
	return "galgame_ratings"
}
