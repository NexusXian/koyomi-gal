package model

import "time"

const (
	UserStateWish      int16 = 1
	UserStatePlaying   int16 = 2
	UserStateCompleted int16 = 3
	UserStatePaused    int16 = 4
	UserStateDropped   int16 = 5
)

type UserState struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	GalgameID       uint      `gorm:"not null" json:"galgame_id"`
	UserID          uint      `gorm:"not null" json:"user_id"`
	State           int16     `gorm:"not null" json:"state"`
	PlayTimeMinutes int64     `gorm:"not null" json:"play_time_minutes"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (UserState) TableName() string {
	return "user_galgames"
}
