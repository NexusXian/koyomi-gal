package model

import "time"

type CharacterRole int16

const (
	CharacterRoleOther CharacterRole = iota
	CharacterRoleProtagonist
	CharacterRoleMain
	CharacterRoleSupporting
	CharacterRoleGuest
)

func (r CharacterRole) String() string {
	switch r {
	case CharacterRoleProtagonist:
		return "protagonist"
	case CharacterRoleMain:
		return "main"
	case CharacterRoleSupporting:
		return "supporting"
	case CharacterRoleGuest:
		return "guest"
	default:
		return "other"
	}
}

type SpoilerLevel int16

const (
	SpoilerLevelNone SpoilerLevel = iota
	SpoilerLevelMinor
	SpoilerLevelMajor
)

func (s SpoilerLevel) String() string {
	switch s {
	case SpoilerLevelMinor:
		return "minor"
	case SpoilerLevelMajor:
		return "major"
	default:
		return "none"
	}
}

type Character struct {
	ID           uint   `gorm:"primaryKey"`
	Name         string `gorm:"size:255;not null"`
	OriginalName string `gorm:"size:255;not null;default:''"`
	Description  string `gorm:"type:text;not null;default:''"`
	ImageURL     string `gorm:"size:2048;not null;default:''"`
	Gender       string `gorm:"size:50;not null;default:''"`
	Birthday     string `gorm:"size:50;not null;default:''"`
	BloodType    string `gorm:"size:50;not null;default:''"`
	Height       int    `gorm:"not null;default:0"`
	Source       string `gorm:"size:50;not null;default:''"`
	SourceID     string `gorm:"size:255;not null;default:''"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type GalgameCharacter struct {
	ID                 uint          `gorm:"primaryKey"`
	GalgameID          uint          `gorm:"not null"`
	CharacterID        uint          `gorm:"not null"`
	Role               CharacterRole `gorm:"type:smallint;not null;default:0"`
	SpoilerLevel       SpoilerLevel  `gorm:"type:smallint;not null;default:0"`
	AppearanceSpoiler  bool          `gorm:"not null;default:false"`
	Description        string        `gorm:"type:text;not null;default:''"`
	SpoilerDescription string        `gorm:"type:text;not null;default:''"`
	SortOrder          int           `gorm:"not null;default:0"`
	CreatedAt          time.Time
	UpdatedAt          time.Time
	Character          *Character `gorm:"foreignKey:CharacterID"`
}
