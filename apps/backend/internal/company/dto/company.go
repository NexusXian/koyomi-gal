package dto

import (
	"time"

	"backend/internal/company/model"
)

type CompanyRequest struct {
	Name            string   `json:"name" binding:"required,max=255"`
	NameJa          string   `json:"name_ja" binding:"max=255"`
	NameZh          string   `json:"name_zh" binding:"max=255"`
	NameEn          string   `json:"name_en" binding:"max=255"`
	Slug            string   `json:"slug" binding:"required,max=255"`
	Description     string   `json:"description"`
	LogoURL         string   `json:"logo_url"`
	BannerURL       string   `json:"banner_url"`
	WebsiteURL      string   `json:"website_url"`
	Country         string   `json:"country" binding:"max=100"`
	FoundedDate     *string  `json:"founded_date" format:"date" extensions:"x-nullable" example:"1998-01-01"`
	ClosedDate      *string  `json:"closed_date" format:"date" extensions:"x-nullable" example:"2020-01-01"`
	Status          string   `json:"status" binding:"required,oneof=active inactive closed" enums:"active,inactive,closed"`
	ParentCompanyID *uint    `json:"parent_company_id" binding:"omitempty,gt=0" extensions:"x-nullable"`
	Aliases         []string `json:"aliases" binding:"max=100,dive,max=255"`
}

type CompanyQuery struct {
	Keyword  string `form:"keyword" binding:"max=255"`
	Page     int    `form:"page" binding:"omitempty,min=1,max=1000000"`
	PageSize int    `form:"page_size" binding:"omitempty,min=1,max=100"`
	Status   string `form:"status" binding:"omitempty,oneof=active inactive closed"`
	Country  string `form:"country" binding:"max=100"`
	Sort     string `form:"sort" binding:"omitempty,oneof=created_at name game_count"`
}

type CompanyGamesQuery struct {
	Page     int    `form:"page" binding:"omitempty,min=1,max=1000000"`
	PageSize int    `form:"page_size" binding:"omitempty,min=1,max=100"`
	Role     string `form:"role" binding:"omitempty,oneof=developer publisher brand co_developer"`
	Keyword  string `form:"keyword" binding:"max=255"`
	Sort     string `form:"sort" binding:"omitempty,oneof=latest oldest rating favorite popular"`
}

type GameCompanyInput struct {
	CompanyID uint   `json:"company_id" binding:"required,gt=0"`
	Role      string `json:"role" binding:"required,oneof=developer publisher brand co_developer" enums:"developer,publisher,brand,co_developer"`
}

type ReplaceGameCompaniesRequest struct {
	Companies []GameCompanyInput `json:"companies" binding:"required,max=100,dive"`
}

type CompanySummary struct {
	ID      uint   `json:"id"`
	Name    string `json:"name"`
	Slug    string `json:"slug"`
	LogoURL string `json:"logo_url"`
}

type GameCompanyResponse struct {
	ID      uint   `json:"id"`
	Name    string `json:"name"`
	Slug    string `json:"slug"`
	LogoURL string `json:"logo_url"`
	Role    string `json:"role" enums:"developer,publisher,brand,co_developer"`
}

type CompanyResponse struct {
	ID                 uint             `json:"id"`
	Name               string           `json:"name"`
	NameJa             string           `json:"name_ja"`
	NameZh             string           `json:"name_zh"`
	NameEn             string           `json:"name_en"`
	Slug               string           `json:"slug"`
	Description        string           `json:"description"`
	LogoURL            string           `json:"logo_url"`
	BannerURL          string           `json:"banner_url"`
	WebsiteURL         string           `json:"website_url"`
	Country            string           `json:"country"`
	FoundedDate        *string          `json:"founded_date" format:"date" extensions:"x-nullable" example:"1998-01-01"`
	ClosedDate         *string          `json:"closed_date" format:"date" extensions:"x-nullable" example:"2020-01-01"`
	Status             string           `json:"status" enums:"active,inactive,closed"`
	ParentCompanyID    *uint            `json:"parent_company_id" extensions:"x-nullable"`
	CreatedAt          time.Time        `json:"created_at"`
	UpdatedAt          time.Time        `json:"updated_at"`
	Aliases            []string         `json:"aliases"`
	ParentCompany      *CompanySummary  `json:"parent_company" extensions:"x-nullable"`
	Children           []CompanySummary `json:"children"`
	GameCount          int64            `json:"game_count"`
	DeveloperGameCount int64            `json:"developer_game_count"`
	PublisherGameCount int64            `json:"publisher_game_count"`
}

type CompanyListData struct {
	Items    []CompanyResponse `json:"items"`
	Total    int64             `json:"total"`
	Page     int               `json:"page"`
	PageSize int               `json:"page_size"`
}

type CompanyListResponse struct {
	Code int             `json:"code"`
	Data CompanyListData `json:"data"`
	Msg  string          `json:"msg"`
}

type CompanyDataResponse struct {
	Code int             `json:"code"`
	Data CompanyResponse `json:"data"`
	Msg  string          `json:"msg"`
}

type GameCompaniesDataResponse struct {
	Code int                   `json:"code"`
	Data []GameCompanyResponse `json:"data"`
	Msg  string                `json:"msg"`
}

func summary(c *model.Company) CompanySummary {
	return CompanySummary{ID: c.ID, Name: c.Name, Slug: c.Slug, LogoURL: c.LogoURL}
}

func date(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format("2006-01-02")
	return &s
}

func NewCompanyResponse(c *model.Company) CompanyResponse {
	r := CompanyResponse{
		ID: c.ID, Name: c.Name, NameJa: c.NameJa, NameZh: c.NameZh, NameEn: c.NameEn,
		Slug: c.Slug, Description: c.Description, LogoURL: c.LogoURL, BannerURL: c.BannerURL,
		WebsiteURL: c.WebsiteURL, Country: c.Country, FoundedDate: date(c.FoundedDate), ClosedDate: date(c.ClosedDate),
		Status: c.Status, ParentCompanyID: c.ParentCompanyID, CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt,
		Aliases: make([]string, 0, len(c.Aliases)), Children: make([]CompanySummary, 0, len(c.Children)),
		GameCount: c.GameCount, DeveloperGameCount: c.DeveloperGameCount, PublisherGameCount: c.PublisherGameCount,
	}
	for _, a := range c.Aliases {
		r.Aliases = append(r.Aliases, a.Alias)
	}
	for i := range c.Children {
		r.Children = append(r.Children, summary(&c.Children[i]))
	}
	if c.ParentCompany != nil {
		p := summary(c.ParentCompany)
		r.ParentCompany = &p
	}
	return r
}

func NewGameCompanyResponses(bindings []model.GameCompany) []GameCompanyResponse {
	r := make([]GameCompanyResponse, 0, len(bindings))
	for _, b := range bindings {
		r = append(r, GameCompanyResponse{ID: b.CompanyID, Name: b.Company.Name, Slug: b.Company.Slug, LogoURL: b.Company.LogoURL, Role: b.Role})
	}
	return r
}
