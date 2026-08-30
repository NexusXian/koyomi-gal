package dto

import (
	"time"

	"backend/internal/galgame/model"
)

type UpsertRatingRequest struct {
	Score int16 `json:"score" binding:"required,oneof=1 2 3 4 5 6 7 8 9 10" example:"8"`
}

type RatingData struct {
	Score     int16     `json:"score" example:"8"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
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
		Score:     rating.Score,
		CreatedAt: rating.CreatedAt,
		UpdatedAt: rating.UpdatedAt,
	}
}
