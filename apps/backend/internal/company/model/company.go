package model

import "time"

const (
	StatusActive    = "active"
	StatusInactive  = "inactive"
	StatusClosed    = "closed"
	RoleDeveloper   = "developer"
	RolePublisher   = "publisher"
	RoleBrand       = "brand"
	RoleCoDeveloper = "co_developer"
)

type Company struct {
	ID                 uint `gorm:"primaryKey"`
	Name               string
	NameJa             string
	NameZh             string
	NameEn             string
	Slug               string
	Description        string
	LogoURL            string
	BannerURL          string
	WebsiteURL         string
	Country            string
	FoundedDate        *time.Time `gorm:"type:date"`
	ClosedDate         *time.Time `gorm:"type:date"`
	Status             string
	ParentCompanyID    *uint
	CreatedAt          time.Time
	UpdatedAt          time.Time
	Aliases            []CompanyAlias `gorm:"foreignKey:CompanyID"`
	ParentCompany      *Company       `gorm:"foreignKey:ParentCompanyID"`
	Children           []Company      `gorm:"foreignKey:ParentCompanyID"`
	GameCount          int64          `gorm:"->;-:migration"`
	DeveloperGameCount int64          `gorm:"->;-:migration"`
	PublisherGameCount int64          `gorm:"->;-:migration"`
}

type CompanyAlias struct {
	ID        uint `gorm:"primaryKey"`
	CompanyID uint
	Alias     string
	CreatedAt time.Time
}

type GameCompany struct {
	GameID    uint   `gorm:"primaryKey"`
	CompanyID uint   `gorm:"primaryKey"`
	Role      string `gorm:"primaryKey"`
	CreatedAt time.Time
	Company   Company `gorm:"foreignKey:CompanyID"`
}
