package model

import "time"

const (
	TypeFeedback  = "feedback"
	TypeCopyright = "copyright"
)

type Feedback struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	Type      string     `gorm:"size:32;not null" json:"type"`
	Content   string     `gorm:"not null" json:"content"`
	Contact   string     `gorm:"size:255;not null" json:"contact"`
	UserID    *uint      `json:"user_id"`
	IP        string     `gorm:"size:64;not null" json:"ip"`
	UserAgent string     `gorm:"size:512;not null" json:"user_agent"`
	HandledBy *uint      `json:"handled_by"`
	HandledAt *time.Time `json:"handled_at"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}
