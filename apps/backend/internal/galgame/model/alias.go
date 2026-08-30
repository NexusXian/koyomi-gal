package model

import "time"

type Alias struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	GalgameID uint      `gorm:"not null" json:"galgame_id"`
	Alias     string    `gorm:"size:255;not null" json:"alias"`
	CreatedAt time.Time `json:"created_at"`
}

func (Alias) TableName() string {
	return "galgame_aliases"
}
