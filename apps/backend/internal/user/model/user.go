package model

import "time"

type User struct {
	Id        int       `gorm:"primaryKey" json:"id"`
	Username  string    `json:"name"`
	Email     string    `json:"email"`
	Password  string    `json:"password"`
	Avatar    string    `json:"avatar"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}


