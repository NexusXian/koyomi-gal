package dto

type UpdateMeRequest struct {
	// AvatarAssetID references an image_assets row; null clears the avatar.
	AvatarAssetID *uint `json:"avatar_asset_id" binding:"omitempty,min=1" example:"123"`
}

type MeData struct {
	ID       uint   `json:"id" example:"1"`
	Username string `json:"username" example:"koyomi"`
	Email    string `json:"email" example:"user@example.com"`
	Avatar   string `json:"avatar" example:"https://img.example.com/avatars/1/2026/09/uuid.png"`
}

type MeResponse struct {
	Code int    `json:"code" example:"0"`
	Data MeData `json:"data"`
	Msg  string `json:"msg" example:"success"`
}

type UserPreferencesData struct {
	BackgroundSource   string  `json:"background_source" example:"preset"`
	BackgroundAssetID  *uint   `json:"background_asset_id" example:"100"`
	BackgroundPreset   *string `json:"background_preset" example:"default-01"`
	BackgroundOpacity  float64 `json:"background_opacity" example:"0.35"`
	BackgroundBlur     float64 `json:"background_blur" example:"0"`
	BackgroundPosition string  `json:"background_position" example:"center center"`
	BackgroundSize     string  `json:"background_size" example:"cover"`
	// BackgroundImageURL is the resolved CDN URL of the custom background.
	BackgroundImageURL string `json:"background_image_url"`
}

type UserPreferencesResponse struct {
	Code int                 `json:"code" example:"0"`
	Data UserPreferencesData `json:"data"`
	Msg  string              `json:"msg" example:"success"`
}

type UpdateUserPreferencesRequest struct {
	BackgroundSource   string  `json:"background_source" binding:"required,oneof=none preset custom" example:"custom"`
	BackgroundAssetID  *uint   `json:"background_asset_id" binding:"omitempty,min=1" example:"100"`
	BackgroundPreset   *string `json:"background_preset" binding:"omitempty,max=64" example:"default-01"`
	BackgroundOpacity  float64 `json:"background_opacity" binding:"min=0,max=1" example:"0.35"`
	BackgroundBlur     float64 `json:"background_blur" binding:"min=0,max=20" example:"0"`
	BackgroundPosition string  `json:"background_position" binding:"omitempty,max=64" example:"center center"`
	BackgroundSize     string  `json:"background_size" binding:"omitempty,oneof=cover contain" example:"cover"`
}
