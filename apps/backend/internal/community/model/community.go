package model

import "time"

type Post struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	GalgameID     *uint     `json:"galgame_id"`
	AuthorID      *uint     `json:"author_id"`
	Title         string    `gorm:"size:255;not null" json:"title"`
	Content       string    `gorm:"not null" json:"content"`
	LikeCount     int64     `gorm:"not null" json:"like_count"`
	CommentCount  int64     `gorm:"not null" json:"comment_count"`
	FavoriteCount int64     `gorm:"not null" json:"favorite_count"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// Comment uses a two-level display structure: parent_id always references a
// top-level comment, and reply_to_user_id marks the replied-to user when the
// comment answers another reply.
type Comment struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	PostID        uint      `gorm:"not null" json:"post_id"`
	AuthorID      *uint     `json:"author_id"`
	ParentID      *uint     `json:"parent_id"`
	ReplyToUserID *uint     `json:"reply_to_user_id"`
	Content       string    `gorm:"not null" json:"content"`
	LikeCount     int64     `gorm:"not null" json:"like_count"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type PostLike struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	PostID    uint      `gorm:"not null" json:"post_id"`
	UserID    uint      `gorm:"not null" json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}

type CommentLike struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CommentID uint      `gorm:"not null" json:"comment_id"`
	UserID    uint      `gorm:"not null" json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}

type PostFavorite struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	PostID    uint      `gorm:"not null" json:"post_id"`
	UserID    uint      `gorm:"not null" json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}
