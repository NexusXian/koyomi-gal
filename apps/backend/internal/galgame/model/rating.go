package model

import "time"

type Rating struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	GalgameID      uint      `gorm:"not null" json:"galgame_id"`
	UserID         uint      `gorm:"not null" json:"user_id"`
	Score          int16     `gorm:"not null" json:"score"`
	Visual         *int16    `json:"visual"`
	Story          *int16    `json:"story"`
	Music          *int16    `json:"music"`
	Character      *int16    `json:"character"`
	Branch         *int16    `json:"branch"`
	System         *int16    `json:"system"`
	Voice          *int16    `json:"voice"`
	Replay         *int16    `json:"replay"`
	Recommendation *int16    `json:"recommendation"`
	ReviewText     *string   `json:"review_text"`
	SpoilerLevel   int16     `gorm:"not null;default:0" json:"spoiler_level"`
	LikeCount      int64     `gorm:"not null;default:0" json:"like_count"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (Rating) TableName() string {
	return "galgame_ratings"
}

type RatingView struct {
	Rating      `gorm:"embedded"`
	Username    string
	DisplayName string
	AvatarURL   string
	PlayStatus  *int16
	Liked       bool
}

type RatingDimensionSummary struct {
	Average *float64 `json:"average"`
	Count   int64    `json:"count"`
}

type RatingSummary struct {
	Count     int64
	Overall   *float64
	Visual    RatingDimensionSummary
	Story     RatingDimensionSummary
	Music     RatingDimensionSummary
	Character RatingDimensionSummary
	Branch    RatingDimensionSummary
	System    RatingDimensionSummary
	Voice     RatingDimensionSummary
	Replay    RatingDimensionSummary
}
