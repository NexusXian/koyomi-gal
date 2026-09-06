package dto

import "time"

// CharacterRequest replaces shared character metadata. Description must be spoiler-free.
type CharacterRequest struct {
	Name         string `json:"name" binding:"required,max=255"`
	OriginalName string `json:"original_name" binding:"max=255"`
	Description  string `json:"description"`
	ImageURL     string `json:"image_url" binding:"max=2048"`
	Gender       string `json:"gender" binding:"max=50"`
	Birthday     string `json:"birthday" binding:"max=50" example:"03-14"`
	BloodType    string `json:"blood_type" binding:"max=50"`
	Height       int    `json:"height" binding:"min=0,max=2147483647" example:"165"`
	Source       string `json:"source" binding:"max=50" example:"vndb"`
	SourceID     string `json:"source_id" binding:"max=255" example:"c123"`
}

type CharacterResponse struct {
	ID           uint      `json:"id"`
	Name         string    `json:"name"`
	OriginalName string    `json:"original_name"`
	Description  string    `json:"description"`
	ImageURL     string    `json:"image_url"`
	Gender       string    `json:"gender"`
	Birthday     string    `json:"birthday"`
	BloodType    string    `json:"blood_type"`
	Height       int       `json:"height"`
	Source       string    `json:"source"`
	SourceID     string    `json:"source_id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type UpdateGalgameCharacterRequest struct {
	Role               string `json:"role" binding:"omitempty,oneof=other protagonist main supporting guest" enums:"other,protagonist,main,supporting,guest" default:"other"`
	SpoilerLevel       string `json:"spoiler_level" binding:"omitempty,oneof=none minor major" enums:"none,minor,major" default:"none"`
	AppearanceSpoiler  bool   `json:"appearance_spoiler"`
	Description        string `json:"description"`
	SpoilerDescription string `json:"spoiler_description"`
	SortOrder          int    `json:"sort_order" binding:"min=-2147483648,max=2147483647"`
}

type GalgameCharacterRequest struct {
	CharacterID uint `json:"character_id" binding:"required,gt=0"`
	UpdateGalgameCharacterRequest
}

// Concealed fields are omitted, not sent as null or empty placeholders.
type GalgameCharacterResponse struct {
	ID                 uint       `json:"id"`
	Role               string     `json:"role" enums:"other,protagonist,main,supporting,guest"`
	AppearanceSpoiler  bool       `json:"appearance_spoiler"`
	HasSpoiler         bool       `json:"has_spoiler"`
	SortOrder          int        `json:"sort_order"`
	CharacterID        *uint      `json:"character_id,omitempty"`
	Name               *string    `json:"name,omitempty"`
	OriginalName       *string    `json:"original_name,omitempty"`
	ImageURL           *string    `json:"image_url,omitempty"`
	Description        *string    `json:"description,omitempty"`
	PublicDescription  *string    `json:"public_description,omitempty"`
	Gender             *string    `json:"gender,omitempty"`
	Birthday           *string    `json:"birthday,omitempty"`
	BloodType          *string    `json:"blood_type,omitempty"`
	Height             *int       `json:"height,omitempty"`
	Source             *string    `json:"source,omitempty"`
	SourceID           *string    `json:"source_id,omitempty"`
	SpoilerLevel       *string    `json:"spoiler_level,omitempty" enums:"none,minor,major"`
	SpoilerDescription *string    `json:"spoiler_description,omitempty"`
	CreatedAt          *time.Time `json:"created_at,omitempty"`
	UpdatedAt          *time.Time `json:"updated_at,omitempty"`
}

type CharacterSearchQuery struct {
	Q        string `form:"q" binding:"max=255"`
	Page     int    `form:"page" binding:"omitempty,min=1,max=1000000"`
	PageSize int    `form:"page_size" binding:"omitempty,min=1,max=100"`
}

type CharacterSearchData struct {
	Items    []CharacterResponse `json:"items"`
	Total    int64               `json:"total"`
	Page     int                 `json:"page"`
	PageSize int                 `json:"page_size"`
}

type GalgameCharacterListData struct {
	Items []GalgameCharacterResponse `json:"items"`
}

type CharacterDataResponse struct {
	Code int               `json:"code" example:"0"`
	Data CharacterResponse `json:"data"`
	Msg  string            `json:"msg" example:"success"`
}

type CharacterSearchResponse struct {
	Code int                 `json:"code" example:"0"`
	Data CharacterSearchData `json:"data"`
	Msg  string              `json:"msg" example:"success"`
}

type GalgameCharacterDataResponse struct {
	Code int                      `json:"code" example:"0"`
	Data GalgameCharacterResponse `json:"data"`
	Msg  string                   `json:"msg" example:"success"`
}

type GalgameCharacterListResponse struct {
	Code int                      `json:"code" example:"0"`
	Data GalgameCharacterListData `json:"data"`
	Msg  string                   `json:"msg" example:"success"`
}
