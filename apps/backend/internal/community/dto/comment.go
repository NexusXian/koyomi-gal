package dto

import (
	"time"

	"backend/internal/community/model"
)

// CreateCommentRequest builds a two-level structure: parent_id must reference
// a top-level comment of the same post, and reply_to_comment_id (a reply being
// answered) is resolved server-side to reply_to_user_id; clients never submit
// user IDs.
type CreateCommentRequest struct {
	Content          string `json:"content" binding:"required,max=10000" example:"同感！"`
	ParentID         *uint  `json:"parent_id" binding:"omitempty,gt=0" example:"1"`
	ReplyToCommentID *uint  `json:"reply_to_comment_id" binding:"omitempty,gt=0" example:"5"`
}

type UpdateCommentRequest struct {
	Content string `json:"content" binding:"required,max=10000" example:"同感！（已编辑）"`
}

type CommentQuery struct {
	Page  int `form:"page" binding:"omitempty,min=1,max=1000000"`
	Limit int `form:"limit" binding:"omitempty,min=1,max=100"`
}

type CommentData struct {
	ID            uint      `json:"id" example:"1"`
	PostID        uint      `json:"post_id" example:"1"`
	AuthorID      *uint     `json:"author_id" example:"1"`
	AuthorName    string    `json:"author_name" example:"koyomi"`
	AuthorAvatar  string    `json:"author_avatar" example:"https://img.example.com/avatars/1/2026/09/uuid.png"`
	ParentID      *uint     `json:"parent_id" example:"1"`
	ReplyToUserID *uint     `json:"reply_to_user_id" example:"2"`
	Content       string    `json:"content" example:"同感！"`
	LikeCount     int64     `json:"like_count" example:"5"`
	ReplyCount    int64     `json:"reply_count" example:"3"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type CommentListData struct {
	Items []CommentData `json:"items"`
	Total int64         `json:"total" example:"10"`
	Page  int           `json:"page" example:"1"`
	Limit int           `json:"limit" example:"20"`
}

type CommentListResponse struct {
	Code int             `json:"code" example:"0"`
	Data CommentListData `json:"data"`
	Msg  string          `json:"msg" example:"success"`
}

type CommentDataResponse struct {
	Code int              `json:"code" example:"0"`
	Data CommentData      `json:"data"`
	Msg  string           `json:"msg" example:"success"`
}

type PostLikeData struct {
	Liked     bool  `json:"liked" example:"true"`
	LikeCount int64 `json:"like_count" example:"11"`
}

type PostLikeDataResponse struct {
	Code int          `json:"code" example:"0"`
	Data PostLikeData `json:"data"`
	Msg  string       `json:"msg" example:"success"`
}

type CommentLikeData struct {
	Liked     bool  `json:"liked" example:"true"`
	LikeCount int64 `json:"like_count" example:"6"`
}

type CommentLikeDataResponse struct {
	Code int             `json:"code" example:"0"`
	Data CommentLikeData `json:"data"`
	Msg  string          `json:"msg" example:"success"`
}

type PostFavoriteData struct {
	Favorited     bool  `json:"favorited" example:"true"`
	FavoriteCount int64 `json:"favorite_count" example:"3"`
}

type PostFavoriteDataResponse struct {
	Code int              `json:"code" example:"0"`
	Data PostFavoriteData `json:"data"`
	Msg  string           `json:"msg" example:"success"`
}

func NewCommentData(comment *model.Comment, replyCount int64) CommentData {
	return CommentData{
		ID:            comment.ID,
		PostID:        comment.PostID,
		AuthorID:      comment.AuthorID,
		AuthorName:    comment.AuthorName,
		AuthorAvatar:  comment.AuthorAvatar,
		ParentID:      comment.ParentID,
		ReplyToUserID: comment.ReplyToUserID,
		Content:       comment.Content,
		LikeCount:     comment.LikeCount,
		ReplyCount:    replyCount,
		CreatedAt:     comment.CreatedAt,
		UpdatedAt:     comment.UpdatedAt,
	}
}
