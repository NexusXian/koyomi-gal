package model

import "time"

const (
	BackgroundSourceNone   = "none"
	BackgroundSourcePreset = "preset"
	BackgroundSourceCustom = "custom"

	BackgroundSizeCover   = "cover"
	BackgroundSizeContain = "contain"

	SensitiveCoverModeBlur = "blur"
	SensitiveCoverModeShow = "show"

	defaultBackgroundOpacity = 0.35
	defaultBackgroundBlur    = 0
	defaultBackgroundPos     = "center center"
)

type UserPreference struct {
	UserID             uint      `gorm:"primaryKey" json:"user_id"`
	BackgroundSource   string    `gorm:"size:16;not null" json:"background_source"`
	BackgroundAssetID  *uint     `gorm:"column:background_asset_id" json:"background_asset_id"`
	BackgroundPreset   *string   `gorm:"size:64" json:"background_preset"`
	BackgroundOpacity  float64   `gorm:"not null" json:"background_opacity"`
	BackgroundBlur     float64   `gorm:"not null" json:"background_blur"`
	BackgroundPosition string    `gorm:"size:64;not null" json:"background_position"`
	BackgroundSize     string    `gorm:"size:16;not null" json:"background_size"`
	SensitiveCoverMode string    `gorm:"size:16;not null;default:'blur'" json:"sensitive_cover_mode"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// DefaultUserPreference returns the defaults returned when no row exists yet.
func DefaultUserPreference(userID uint) *UserPreference {
	return &UserPreference{
		UserID:             userID,
		BackgroundSource:   BackgroundSourceNone,
		BackgroundOpacity:  defaultBackgroundOpacity,
		BackgroundBlur:     defaultBackgroundBlur,
		BackgroundPosition: defaultBackgroundPos,
		BackgroundSize:     BackgroundSizeCover,
		SensitiveCoverMode: SensitiveCoverModeBlur,
	}
}

// ValidSensitiveCoverMode reports whether the value is a known mode.
func ValidSensitiveCoverMode(value string) bool {
	switch value {
	case SensitiveCoverModeBlur, SensitiveCoverModeShow:
		return true
	default:
		return false
	}
}
