package dto

import (
	"encoding/json"
	"time"
)

type PublicUserSummary struct {
	ID          uint   `json:"id" example:"1001"`
	Username    string `json:"username" example:"koyomi"`
	DisplayName string `json:"display_name" example:"Koyomi"`
	AvatarURL   string `json:"avatar_url" example:"https://img.example.com/avatars/1001/avatar.webp"`
}

type ProfileAccess struct {
	CanViewProfile   bool `json:"can_view_profile"`
	CanViewPosts     bool `json:"can_view_posts"`
	CanViewComments  bool `json:"can_view_comments"`
	CanViewRatings   bool `json:"can_view_ratings"`
	CanViewFavorites bool `json:"can_view_favorites"`
	CanViewActivity  bool `json:"can_view_activity"`
	CanViewLocation  bool `json:"can_view_location"`
	CanViewBirthday  bool `json:"can_view_birthday"`
}

type PublicUserProfile struct {
	PublicUserSummary
	BannerURL     string        `json:"banner_url,omitempty"`
	Bio           string        `json:"bio,omitempty"`
	Gender        string        `json:"gender,omitempty"`
	Location      string        `json:"location,omitempty"`
	Birthday      *string       `json:"birthday,omitempty" example:"2000-01-02"`
	WebsiteURL    string        `json:"website_url,omitempty"`
	RegisteredAt  *time.Time    `json:"registered_at,omitempty"`
	PostCount     *int64        `json:"post_count,omitempty"`
	CommentCount  *int64        `json:"comment_count,omitempty"`
	RatingCount   *int64        `json:"rating_count,omitempty"`
	FavoriteCount *int64        `json:"favorite_count,omitempty"`
	IsSelf        bool          `json:"is_self"`
	IsPrivate     bool          `json:"is_private"`
	IsRestricted  bool          `json:"is_restricted"`
	Access        ProfileAccess `json:"access"`
}

type PublicUserProfileResponse struct {
	Code int               `json:"code" example:"0"`
	Data PublicUserProfile `json:"data"`
	Msg  string            `json:"msg" example:"success"`
}

type UpdateProfileRequest struct {
	DisplayName   *string `json:"display_name" binding:"omitempty,max=100"`
	Bio           *string `json:"bio" binding:"omitempty,max=1000"`
	Gender        *string `json:"gender" binding:"omitempty,oneof=male female non_binary undisclosed"`
	Location      *string `json:"location" binding:"omitempty,max=100"`
	Birthday      *string `json:"birthday" extensions:"x-nullable" example:"2000-01-02"`
	WebsiteURL    *string `json:"website_url" binding:"omitempty,max=2048"`
	AvatarAssetID *uint   `json:"avatar_asset_id" binding:"omitempty,min=1" extensions:"x-nullable"`
	BannerAssetID *uint   `json:"banner_asset_id" binding:"omitempty,min=1" extensions:"x-nullable"`
	AvatarSet     bool    `json:"-" swaggerignore:"true"`
	BannerSet     bool    `json:"-" swaggerignore:"true"`
	BirthdaySet   bool    `json:"-" swaggerignore:"true"`
}

func (r *UpdateProfileRequest) UnmarshalJSON(data []byte) error {
	type requestAlias UpdateProfileRequest
	var value requestAlias
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	*r = UpdateProfileRequest(value)
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	_, r.AvatarSet = fields["avatar_asset_id"]
	_, r.BannerSet = fields["banner_asset_id"]
	_, r.BirthdaySet = fields["birthday"]
	return nil
}

type UpdatePrivacyRequest struct {
	ProfileVisibility *string `json:"profile_visibility" binding:"omitempty,oneof=public registered private"`
	ShowLocation      *bool   `json:"show_location"`
	ShowBirthday      *bool   `json:"show_birthday"`
	ShowPosts         *bool   `json:"show_posts"`
	ShowComments      *bool   `json:"show_comments"`
	ShowRatings       *bool   `json:"show_ratings"`
	ShowFavorites     *bool   `json:"show_favorites"`
	ShowActivity      *bool   `json:"show_activity"`
}

type PrivacySettingsData struct {
	ProfileVisibility string `json:"profile_visibility" example:"public"`
	ShowLocation      bool   `json:"show_location"`
	ShowBirthday      bool   `json:"show_birthday"`
	ShowPosts         bool   `json:"show_posts"`
	ShowComments      bool   `json:"show_comments"`
	ShowRatings       bool   `json:"show_ratings"`
	ShowFavorites     bool   `json:"show_favorites"`
	ShowActivity      bool   `json:"show_activity"`
}

type PrivacySettingsResponse struct {
	Code int                 `json:"code" example:"0"`
	Data PrivacySettingsData `json:"data"`
	Msg  string              `json:"msg" example:"success"`
}

type ProfileListQuery struct {
	Page  int `form:"page" binding:"omitempty,min=1,max=1000000"`
	Limit int `form:"limit" binding:"omitempty,min=1,max=100"`
}

type ProfilePostData struct {
	ID            uint      `json:"id"`
	GalgameID     *uint     `json:"galgame_id"`
	GalgameTitle  string    `json:"galgame_title"`
	Title         string    `json:"title"`
	Content       string    `json:"content"`
	EditorMode    string    `json:"editor_mode"`
	LikeCount     int64     `json:"like_count"`
	CommentCount  int64     `json:"comment_count"`
	FavoriteCount int64     `json:"favorite_count"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type ProfileCommentData struct {
	ID        uint      `json:"id"`
	PostID    uint      `json:"post_id"`
	PostTitle string    `json:"post_title"`
	ParentID  *uint     `json:"parent_id"`
	Content   string    `json:"content"`
	LikeCount int64     `json:"like_count"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ProfileGalgameData struct {
	ID        uint      `json:"id"`
	Title     string    `json:"title"`
	Slug      string    `json:"slug"`
	CoverURL  string    `json:"cover_url"`
	Score     *int16    `json:"score,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UserActivityData struct {
	ID         uint           `json:"id"`
	Type       string         `json:"type" example:"post_created"`
	TargetType string         `json:"target_type" example:"post"`
	TargetID   *uint          `json:"target_id"`
	Metadata   map[string]any `json:"metadata" swaggertype:"object"`
	CreatedAt  time.Time      `json:"created_at"`
}

type ProfilePostListData struct {
	Items []ProfilePostData `json:"items"`
	Total int64             `json:"total"`
	Page  int               `json:"page"`
	Limit int               `json:"limit"`
}
type ProfileCommentListData struct {
	Items []ProfileCommentData `json:"items"`
	Total int64                `json:"total"`
	Page  int                  `json:"page"`
	Limit int                  `json:"limit"`
}
type ProfileGalgameListData struct {
	Items []ProfileGalgameData `json:"items"`
	Total int64                `json:"total"`
	Page  int                  `json:"page"`
	Limit int                  `json:"limit"`
}
type UserActivityListData struct {
	Items []UserActivityData `json:"items"`
	Total int64              `json:"total"`
	Page  int                `json:"page"`
	Limit int                `json:"limit"`
}
type ProfilePostListResponse struct {
	Code int                 `json:"code"`
	Data ProfilePostListData `json:"data"`
	Msg  string              `json:"msg"`
}
type ProfileCommentListResponse struct {
	Code int                    `json:"code"`
	Data ProfileCommentListData `json:"data"`
	Msg  string                 `json:"msg"`
}
type ProfileGalgameListResponse struct {
	Code int                    `json:"code"`
	Data ProfileGalgameListData `json:"data"`
	Msg  string                 `json:"msg"`
}
type UserActivityListResponse struct {
	Code int                  `json:"code"`
	Data UserActivityListData `json:"data"`
	Msg  string               `json:"msg"`
}
