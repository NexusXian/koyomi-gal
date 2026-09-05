package dto

import (
	"time"

	"backend/internal/novel/model"
)

type CreateVolumeRequest struct {
	VolumeNumber  *int   `json:"volume_number" binding:"omitempty,gte=0,lte=9999" example:"1"`
	Title         string `json:"title" binding:"max=255" example:"青春猪头少年不会梦到兔女郎学姐"`
	OriginalTitle string `json:"original_title" binding:"max=255" example:"青春ブタ野郎はバニーガール先輩の夢を見ない"`
	CoverURL      string `json:"cover_url" binding:"max=2048" example:"https://example.com/vol1.jpg"`
	ISBN          string `json:"isbn" binding:"omitempty,max=20" example:"978-4-04-865091-8"`
	ReleaseDate   string `json:"release_date" example:"2014-04-10"`
	Summary       string `json:"summary" example:"第一卷简介"`
	Status        *int16 `json:"status" binding:"omitempty,oneof=0 1" example:"0"`
}

type UpdateVolumeRequest struct {
	VolumeNumber  *int   `json:"volume_number" binding:"omitempty,gte=0,lte=9999" example:"1"`
	Title         string `json:"title" binding:"max=255" example:"青春猪头少年不会梦到兔女郎学姐"`
	OriginalTitle string `json:"original_title" binding:"max=255" example:"青春ブタ野郎はバニーガール先輩の夢を見ない"`
	CoverURL      string `json:"cover_url" binding:"max=2048" example:"https://example.com/vol1.jpg"`
	ISBN          string `json:"isbn" binding:"omitempty,max=20" example:"978-4-04-865091-8"`
	ReleaseDate   string `json:"release_date" example:"2014-04-10"`
	Summary       string `json:"summary" example:"第一卷简介"`
	Status        *int16 `json:"status" binding:"required,oneof=0 1 2 3" example:"1"`
}

// ReviewVolumeRequest approves (1) or rejects (2) a volume submission.
type ReviewVolumeRequest struct {
	Status int16  `json:"status" binding:"required,oneof=1 2" example:"1"`
	Reason string `json:"reason" binding:"omitempty,max=1000" example:"卷号信息有误"`
}

type ReorderVolumesRequest struct {
	IDs []uint `json:"ids" binding:"required,min=1,dive,gt=0" example:"3,1,2"`
}

type VolumeQuery struct {
	Page  int `form:"page" binding:"omitempty,min=1,max=1000000"`
	Limit int `form:"limit" binding:"omitempty,min=1,max=100"`
}

type AdminVolumeQuery struct {
	Status  *int16 `form:"status" binding:"omitempty,oneof=0 1 2 3" example:"0"`
	NovelID *uint  `form:"novel_id" binding:"omitempty,gt=0" example:"1"`
	Page    int    `form:"page" binding:"omitempty,min=1,max=1000000"`
	Limit   int    `form:"limit" binding:"omitempty,min=1,max=100"`
}

type VolumeSummary struct {
	ID            uint    `json:"id" example:"1"`
	VolumeNumber  *int    `json:"volume_number" example:"1"`
	Title         string  `json:"title" example:"青春猪头少年不会梦到兔女郎学姐"`
	OriginalTitle string  `json:"original_title" example:"青春ブタ野郎はバニーガール先輩の夢を見ない"`
	CoverURL      string  `json:"cover_url" example:"https://example.com/vol1.jpg"`
	ISBN          string  `json:"isbn" example:"978-4-04-865091-8"`
	ReleaseDate   *string `json:"release_date" example:"2014-04-10"`
	SortOrder     int     `json:"sort_order" example:"0"`
}

type VolumeData struct {
	ID            uint       `json:"id" example:"1"`
	NovelID       uint       `json:"novel_id" example:"1"`
	VolumeNumber  *int       `json:"volume_number" example:"1"`
	Title         string     `json:"title" example:"青春猪头少年不会梦到兔女郎学姐"`
	OriginalTitle string     `json:"original_title" example:"青春ブタ野郎はバニーガール先輩の夢を見ない"`
	CoverURL      string     `json:"cover_url" example:"https://example.com/vol1.jpg"`
	ISBN          string     `json:"isbn" example:"978-4-04-865091-8"`
	ReleaseDate   *string    `json:"release_date" example:"2014-04-10"`
	Summary       string     `json:"summary" example:"第一卷简介"`
	SortOrder     int        `json:"sort_order" example:"0"`
	Status        int16      `json:"status" example:"1"`
	ReviewedAt    *time.Time `json:"reviewed_at"`
	RejectReason  string     `json:"reject_reason" example:"卷号信息有误"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// AdminVolumeListItem is the admin listing row with the parent novel title.
type AdminVolumeListItem struct {
	ID           uint      `json:"id" example:"1"`
	NovelID      uint      `json:"novel_id" example:"1"`
	NovelTitle   string    `json:"novel_title" example:"青春猪头少年不会梦到兔女郎学姐"`
	VolumeNumber *int      `json:"volume_number" example:"1"`
	Title        string    `json:"title" example:"青春猪头少年不会梦到兔女郎学姐"`
	CoverURL     string    `json:"cover_url" example:"https://example.com/vol1.jpg"`
	ISBN         string    `json:"isbn" example:"978-4-04-865091-8"`
	ReleaseDate  *string   `json:"release_date" example:"2014-04-10"`
	SortOrder    int       `json:"sort_order" example:"0"`
	Status       int16     `json:"status" example:"0"`
	RejectReason string    `json:"reject_reason" example:"卷号信息有误"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type VolumeListData struct {
	Items []VolumeData `json:"items"`
	Total int64        `json:"total" example:"13"`
	Page  int          `json:"page" example:"1"`
	Limit int          `json:"limit" example:"20"`
}

type VolumeListResponse struct {
	Code int            `json:"code" example:"0"`
	Data VolumeListData `json:"data"`
	Msg  string         `json:"msg" example:"success"`
}

type VolumeDataResponse struct {
	Code int        `json:"code" example:"0"`
	Data VolumeData `json:"data"`
	Msg  string     `json:"msg" example:"success"`
}

type AdminVolumeListData struct {
	Items []AdminVolumeListItem `json:"items"`
	Total int64                 `json:"total" example:"13"`
	Page  int                   `json:"page" example:"1"`
	Limit int                   `json:"limit" example:"20"`
}

type AdminVolumeListResponse struct {
	Code int                 `json:"code" example:"0"`
	Data AdminVolumeListData `json:"data"`
	Msg  string              `json:"msg" example:"success"`
}

func NewVolumeSummaries(volumes []model.NovelVolume) []VolumeSummary {
	items := make([]VolumeSummary, 0, len(volumes))
	for _, volume := range volumes {
		items = append(items, VolumeSummary{
			ID:            volume.ID,
			VolumeNumber:  volume.VolumeNumber,
			Title:         volume.Title,
			OriginalTitle: volume.OriginalTitle,
			CoverURL:      volume.CoverURL,
			ISBN:          volume.ISBN,
			ReleaseDate:   formatDate(volume.ReleaseDate),
			SortOrder:     volume.SortOrder,
		})
	}
	return items
}

func NewVolumeData(volume *model.NovelVolume) VolumeData {
	return VolumeData{
		ID:            volume.ID,
		NovelID:       volume.NovelID,
		VolumeNumber:  volume.VolumeNumber,
		Title:         volume.Title,
		OriginalTitle: volume.OriginalTitle,
		CoverURL:      volume.CoverURL,
		ISBN:          volume.ISBN,
		ReleaseDate:   formatDate(volume.ReleaseDate),
		Summary:       volume.Summary,
		SortOrder:     volume.SortOrder,
		Status:        volume.Status,
		ReviewedAt:    volume.ReviewedAt,
		RejectReason:  volume.RejectReason,
		CreatedAt:     volume.CreatedAt,
		UpdatedAt:     volume.UpdatedAt,
	}
}

func NewVolumeListData(volumes []model.NovelVolume) []VolumeData {
	items := make([]VolumeData, 0, len(volumes))
	for i := range volumes {
		items = append(items, NewVolumeData(&volumes[i]))
	}
	return items
}

func NewAdminVolumeListItems(volumes []model.NovelVolume) []AdminVolumeListItem {
	items := make([]AdminVolumeListItem, 0, len(volumes))
	for _, volume := range volumes {
		items = append(items, AdminVolumeListItem{
			ID:           volume.ID,
			NovelID:      volume.NovelID,
			NovelTitle:   volume.NovelTitle,
			VolumeNumber: volume.VolumeNumber,
			Title:        volume.Title,
			CoverURL:     volume.CoverURL,
			ISBN:         volume.ISBN,
			ReleaseDate:  formatDate(volume.ReleaseDate),
			SortOrder:    volume.SortOrder,
			Status:       volume.Status,
			RejectReason: volume.RejectReason,
			CreatedAt:    volume.CreatedAt,
			UpdatedAt:    volume.UpdatedAt,
		})
	}
	return items
}
