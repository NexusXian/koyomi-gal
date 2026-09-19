package dto

import (
	"time"

	"backend/internal/galgame/model"
	userdto "backend/internal/user/dto"
)

type UpsertRatingRequest struct {
	Score int16 `json:"score" binding:"required,oneof=1 2 3 4 5 6 7 8 9 10" example:"8"`
}

type RatingDimensions struct {
	Visual    *int16 `json:"visual" binding:"omitempty,min=1,max=10" extensions:"x-nullable" example:"9"`
	Story     *int16 `json:"story" binding:"omitempty,min=1,max=10" extensions:"x-nullable" example:"8"`
	Music     *int16 `json:"music" binding:"omitempty,min=1,max=10" extensions:"x-nullable" example:"10"`
	Character *int16 `json:"character" binding:"omitempty,min=1,max=10" extensions:"x-nullable" example:"8"`
	Branch    *int16 `json:"branch" binding:"omitempty,min=1,max=10" extensions:"x-nullable" example:"7"`
	System    *int16 `json:"system" binding:"omitempty,min=1,max=10" extensions:"x-nullable" example:"8"`
	Voice     *int16 `json:"voice" binding:"omitempty,min=1,max=10" extensions:"x-nullable" example:"9"`
	Replay    *int16 `json:"replay" binding:"omitempty,min=1,max=10" extensions:"x-nullable" example:"7"`
}

type PutRatingRequest struct {
	Overall        int16   `json:"overall" binding:"required,min=1,max=10" example:"8"`
	Visual         *int16  `json:"visual" binding:"omitempty,min=1,max=10" extensions:"x-nullable" example:"9"`
	Story          *int16  `json:"story" binding:"omitempty,min=1,max=10" extensions:"x-nullable" example:"8"`
	Music          *int16  `json:"music" binding:"omitempty,min=1,max=10" extensions:"x-nullable" example:"10"`
	Character      *int16  `json:"character" binding:"omitempty,min=1,max=10" extensions:"x-nullable" example:"8"`
	Branch         *int16  `json:"branch" binding:"omitempty,min=1,max=10" extensions:"x-nullable" example:"7"`
	System         *int16  `json:"system" binding:"omitempty,min=1,max=10" extensions:"x-nullable" example:"8"`
	Voice          *int16  `json:"voice" binding:"omitempty,min=1,max=10" extensions:"x-nullable" example:"9"`
	Replay         *int16  `json:"replay" binding:"omitempty,min=1,max=10" extensions:"x-nullable" example:"7"`
	Recommendation *int16  `json:"recommendation" binding:"omitempty,oneof=-1 0 1 2" extensions:"x-nullable" enums:"-1,0,1,2" example:"1"`
	ReviewText     *string `json:"review_text" extensions:"x-nullable" example:"A strong story with excellent music."`
	SpoilerLevel   int16   `json:"spoiler_level" binding:"oneof=0 1 2" enums:"0,1,2" example:"0"`
}

type RatingData struct {
	Score          int16            `json:"score" example:"8"`
	Overall        int16            `json:"overall" example:"8"`
	Dimensions     RatingDimensions `json:"dimensions"`
	Recommendation *int16           `json:"recommendation" extensions:"x-nullable" enums:"-1,0,1,2"`
	ReviewText     *string          `json:"review_text" extensions:"x-nullable"`
	SpoilerLevel   int16            `json:"spoiler_level" enums:"0,1,2" example:"0"`
	LikeCount      int64            `json:"like_count" example:"12"`
	CreatedAt      time.Time        `json:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at"`
}

type RatingDataResponse struct {
	Code int        `json:"code" example:"0"`
	Data RatingData `json:"data"`
	Msg  string     `json:"msg" example:"success"`
}

func NewRatingData(rating *model.Rating) *RatingData {
	if rating == nil {
		return nil
	}
	return &RatingData{
		Score:          rating.Score,
		Overall:        rating.Score,
		Dimensions:     NewRatingDimensions(rating),
		Recommendation: rating.Recommendation,
		ReviewText:     rating.ReviewText,
		SpoilerLevel:   rating.SpoilerLevel,
		LikeCount:      rating.LikeCount,
		CreatedAt:      rating.CreatedAt,
		UpdatedAt:      rating.UpdatedAt,
	}
}

func NewRatingDimensions(rating *model.Rating) RatingDimensions {
	return RatingDimensions{
		Visual: rating.Visual, Story: rating.Story, Music: rating.Music, Character: rating.Character,
		Branch: rating.Branch, System: rating.System, Voice: rating.Voice, Replay: rating.Replay,
	}
}

type RatingRecordData struct {
	ID             uint                      `json:"id" example:"101"`
	GalgameID      uint                      `json:"galgame_id" example:"10"`
	User           userdto.PublicUserSummary `json:"user"`
	Overall        int16                     `json:"overall" example:"8"`
	Dimensions     RatingDimensions          `json:"dimensions"`
	Recommendation *int16                    `json:"recommendation" extensions:"x-nullable" enums:"-1,0,1,2"`
	ReviewText     *string                   `json:"review_text" extensions:"x-nullable"`
	SpoilerLevel   int16                     `json:"spoiler_level" enums:"0,1,2" example:"0"`
	PlayStatus     *int16                    `json:"play_status" extensions:"x-nullable" enums:"1,2,3,4,5"`
	LikeCount      int64                     `json:"like_count" example:"12"`
	Liked          bool                      `json:"liked" example:"true"`
	CreatedAt      time.Time                 `json:"created_at"`
	UpdatedAt      time.Time                 `json:"updated_at"`
}

func NewRatingRecordData(record *model.RatingView) RatingRecordData {
	return RatingRecordData{
		ID: record.ID, GalgameID: record.GalgameID,
		User:    userdto.PublicUserSummary{ID: record.UserID, Username: record.Username, DisplayName: record.DisplayName, AvatarURL: record.AvatarURL},
		Overall: record.Score, Dimensions: NewRatingDimensions(&record.Rating), Recommendation: record.Recommendation,
		ReviewText: record.ReviewText, SpoilerLevel: record.SpoilerLevel, PlayStatus: record.PlayStatus,
		LikeCount: record.LikeCount, Liked: record.Liked, CreatedAt: record.CreatedAt, UpdatedAt: record.UpdatedAt,
	}
}

type RatingListData struct {
	Items    []RatingRecordData `json:"items"`
	Total    int64              `json:"total" example:"42"`
	Page     int                `json:"page" example:"1"`
	PageSize int                `json:"page_size" example:"20"`
}

type RatingListResponse struct {
	Code int            `json:"code" example:"0"`
	Data RatingListData `json:"data"`
	Msg  string         `json:"msg" example:"success"`
}

type RatingRecordResponse struct {
	Code int               `json:"code" example:"0"`
	Data *RatingRecordData `json:"data" extensions:"x-nullable"`
	Msg  string            `json:"msg" example:"success"`
}

type RatingDimensionSummaryData struct {
	Average *float64 `json:"average" extensions:"x-nullable" example:"8.25"`
	Count   int64    `json:"count" example:"12"`
}

type RatingSummaryDimensionsData struct {
	Visual    RatingDimensionSummaryData `json:"visual"`
	Story     RatingDimensionSummaryData `json:"story"`
	Music     RatingDimensionSummaryData `json:"music"`
	Character RatingDimensionSummaryData `json:"character"`
	Branch    RatingDimensionSummaryData `json:"branch"`
	System    RatingDimensionSummaryData `json:"system"`
	Voice     RatingDimensionSummaryData `json:"voice"`
	Replay    RatingDimensionSummaryData `json:"replay"`
}

type RatingSummaryData struct {
	Count      int64                       `json:"count" example:"42"`
	Overall    *float64                    `json:"overall" extensions:"x-nullable" example:"8.32"`
	Dimensions RatingSummaryDimensionsData `json:"dimensions"`
}

func NewRatingSummaryData(summary *model.RatingSummary) RatingSummaryData {
	dimension := func(value model.RatingDimensionSummary) RatingDimensionSummaryData {
		return RatingDimensionSummaryData{Average: value.Average, Count: value.Count}
	}
	return RatingSummaryData{
		Count: summary.Count, Overall: summary.Overall,
		Dimensions: RatingSummaryDimensionsData{
			Visual: dimension(summary.Visual), Story: dimension(summary.Story), Music: dimension(summary.Music),
			Character: dimension(summary.Character), Branch: dimension(summary.Branch), System: dimension(summary.System),
			Voice: dimension(summary.Voice), Replay: dimension(summary.Replay),
		},
	}
}

type RatingSummaryResponse struct {
	Code int               `json:"code" example:"0"`
	Data RatingSummaryData `json:"data"`
	Msg  string            `json:"msg" example:"success"`
}

type RatingLikeData struct {
	RatingID  uint  `json:"rating_id" example:"101"`
	LikeCount int64 `json:"like_count" example:"12"`
	Liked     bool  `json:"liked" example:"true"`
}

type RatingLikeResponse struct {
	Code int            `json:"code" example:"0"`
	Data RatingLikeData `json:"data"`
	Msg  string         `json:"msg" example:"success"`
}

type RatingListQuery struct {
	Page     int    `form:"page" binding:"omitempty,min=1,max=1000000"`
	PageSize int    `form:"page_size" binding:"omitempty,min=1,max=100"`
	Limit    int    `form:"limit" binding:"omitempty,min=1,max=100"`
	Sort     string `form:"sort" binding:"omitempty,oneof=newest highest lowest popular" enums:"newest,highest,lowest,popular"`
}
