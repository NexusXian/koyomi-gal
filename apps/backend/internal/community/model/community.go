package model

import "time"

// EditorMode marks how a post's content should be interpreted and rendered.
type EditorMode string

const (
	EditorModePlain    EditorMode = "plain"
	EditorModeMarkdown EditorMode = "markdown"
)

// IsValidEditorMode reports whether the value is a known editor mode.
func IsValidEditorMode(mode EditorMode) bool {
	switch mode {
	case EditorModePlain, EditorModeMarkdown:
		return true
	default:
		return false
	}
}

type Post struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	GalgameID     *uint      `json:"galgame_id"`
	AuthorID      *uint      `json:"author_id"`
	Title         string     `gorm:"size:255;not null" json:"title"`
	Content       string     `gorm:"not null" json:"content"`
	EditorMode    EditorMode `gorm:"type:varchar(20);not null;default:'plain'" json:"editor_mode"`
	LikeCount     int64      `gorm:"not null" json:"like_count"`
	CommentCount  int64      `gorm:"not null" json:"comment_count"`
	FavoriteCount int64      `gorm:"not null" json:"favorite_count"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	// Filled by repository joins, never written to the database.
	AuthorName        string `gorm:"->" json:"author_name"`
	AuthorDisplayName string `gorm:"->" json:"-"`
	AuthorAvatar      string `gorm:"->" json:"author_avatar"`
	GalgameTitle      string `gorm:"->" json:"galgame_title"`
	GalgameCoverURL   string `gorm:"->" json:"galgame_cover_url"`
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
	// Filled by repository joins, never written to the database.
	AuthorName         string `gorm:"->" json:"author_name"`
	AuthorDisplayName  string `gorm:"->" json:"-"`
	AuthorAvatar       string `gorm:"->" json:"author_avatar"`
	ReplyToName        string `gorm:"->" json:"-"`
	ReplyToDisplayName string `gorm:"->" json:"-"`
	ReplyToAvatar      string `gorm:"->" json:"-"`
	PostTitle          string `gorm:"->" json:"post_title"`
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
