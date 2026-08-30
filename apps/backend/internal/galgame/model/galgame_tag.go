package model

import "time"

type GalgameTag struct {
	GalgameID uint      `gorm:"not null" json:"galgame_id"`
	TagID     uint      `gorm:"not null" json:"tag_id"`
	CreatedAt time.Time `json:"created_at"`
}
