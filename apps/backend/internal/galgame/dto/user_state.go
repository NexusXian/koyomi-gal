package dto

import (
	"time"

	"backend/internal/galgame/model"
)

type UpsertUserStateRequest struct {
	State           int16 `json:"state" binding:"required,oneof=1 2 3 4 5" example:"2"`
	PlayTimeMinutes int64 `json:"play_time_minutes" binding:"min=0" example:"120"`
}

type UserStateData struct {
	State           int16     `json:"state" example:"2"`
	PlayTimeMinutes int64     `json:"play_time_minutes" example:"120"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type UserStateDataResponse struct {
	Code int           `json:"code" example:"0"`
	Data UserStateData `json:"data"`
	Msg  string        `json:"msg" example:"success"`
}

func NewUserStateData(state *model.UserState) *UserStateData {
	if state == nil {
		return nil
	}
	return &UserStateData{
		State:           state.State,
		PlayTimeMinutes: state.PlayTimeMinutes,
		CreatedAt:       state.CreatedAt,
		UpdatedAt:       state.UpdatedAt,
	}
}

type FavoriteData struct {
	Favorited bool      `json:"favorited" example:"true"`
	CreatedAt time.Time `json:"created_at"`
}

type FavoriteDataResponse struct {
	Code int          `json:"code" example:"0"`
	Data FavoriteData `json:"data"`
	Msg  string       `json:"msg" example:"success"`
}

func NewFavoriteData(favorite *model.Favorite) *FavoriteData {
	if favorite == nil {
		return nil
	}
	return &FavoriteData{Favorited: true, CreatedAt: favorite.CreatedAt}
}

type GalgameUserRelationData struct {
	GalgameID uint           `json:"galgame_id" example:"1"`
	Rating    *RatingData    `json:"rating"`
	Favorite  *FavoriteData  `json:"favorite"`
	State     *UserStateData `json:"state"`
}

type GalgameUserRelationResponse struct {
	Code int                     `json:"code" example:"0"`
	Data GalgameUserRelationData `json:"data"`
	Msg  string                  `json:"msg" example:"success"`
}
