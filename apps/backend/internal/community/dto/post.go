package dto

import (
	"time"

	"backend/internal/community/model"
)

type CreatePostRequest struct {
	Title     string `json:"title" binding:"required,max=255" example:"千恋＊万花通关感想"`
	Content   string `json:"content" binding:"required" example:"剧情感想……"`
	GalgameID *uint  `json:"galgame_id" binding:"omitempty,gt=0" example:"1"`
}

type UpdatePostRequest struct {
	Title   string `json:"title" binding:"required,max=255" example:"千恋＊万花通关感想（更新）"`
	Content string `json:"content" binding:"required" example:"剧情感想（更新）……"`
}

type PostQuery struct {
	GalgameID *uint `form:"galgame_id" binding:"omitempty,gt=0"`
	Page      int   `form:"page" binding:"omitempty,min=1,max=1000000"`
	Limit     int   `form:"limit" binding:"omitempty,min=1,max=100"`
}

type PostData struct {
	ID            uint      `json:"id" example:"1"`
	GalgameID     *uint     `json:"galgame_id" example:"1"`
	AuthorID      *uint     `json:"author_id" example:"1"`
	Title         string    `json:"title" example:"千恋＊万花通关感想"`
	Content       string    `json:"content" example:"剧情感想……"`
	LikeCount     int64     `json:"like_count" example:"10"`
	CommentCount  int64     `json:"comment_count" example:"3"`
	FavoriteCount int64     `json:"favorite_count" example:"2"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type PostListData struct {
	Items []PostData `json:"items"`
	Total int64      `json:"total" example:"100"`
	Page  int        `json:"page" example:"1"`
	Limit int        `json:"limit" example:"20"`
}

type PostListResponse struct {
	Code int          `json:"code" example:"0"`
	Data PostListData `json:"data"`
	Msg  string       `json:"msg" example:"success"`
}

type PostDataResponse struct {
	Code int      `json:"code" example:"0"`
	Data PostData `json:"data"`
	Msg  string   `json:"msg" example:"success"`
}

func NewPostData(post *model.Post) PostData {
	return PostData{
		ID:            post.ID,
		GalgameID:     post.GalgameID,
		AuthorID:      post.AuthorID,
		Title:         post.Title,
		Content:       post.Content,
		LikeCount:     post.LikeCount,
		CommentCount:  post.CommentCount,
		FavoriteCount: post.FavoriteCount,
		CreatedAt:     post.CreatedAt,
		UpdatedAt:     post.UpdatedAt,
	}
}

func NewPostListItems(posts []model.Post) []PostData {
	items := make([]PostData, 0, len(posts))
	for i := range posts {
		items = append(items, NewPostData(&posts[i]))
	}
	return items
}
