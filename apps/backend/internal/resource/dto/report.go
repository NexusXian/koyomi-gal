package dto

import (
	"time"

	"backend/internal/resource/model"
)

type CreateResourceReportRequest struct {
	Reason      int16  `json:"reason" binding:"oneof=0 1 2 3 4 5 6" example:"0"`
	Description string `json:"description" binding:"max=1000" example:"链接已失效"`
}

type HandleResourceReportRequest struct {
	Status int16 `json:"status" binding:"required,oneof=1 2" example:"1"`
}

// ReportedResourceData summarizes the reported resource for admin listings
// without exposing its links.
type ReportedResourceData struct {
	ID     uint   `json:"id" example:"1"`
	Title  string `json:"title" example:"千恋＊万花 官方整合包"`
	Type   int16  `json:"type" example:"1"`
	Status int16  `json:"status" example:"1"`
}

type ResourceReportData struct {
	ID          uint                  `json:"id" example:"1"`
	ResourceID  uint                  `json:"resource_id" example:"1"`
	UserID      uint                  `json:"user_id" example:"2"`
	Reason      int16                 `json:"reason" example:"0"`
	Description string                `json:"description" example:"链接已失效"`
	Status      int16                 `json:"status" example:"0"`
	HandledBy   *uint                 `json:"handled_by" example:"3"`
	HandledAt   *time.Time            `json:"handled_at"`
	Resource    *ReportedResourceData `json:"resource"`
	CreatedAt   time.Time             `json:"created_at"`
	UpdatedAt   time.Time             `json:"updated_at"`
}

type ResourceReportListData struct {
	Items []ResourceReportData `json:"items"`
	Total int64                `json:"total" example:"10"`
	Page  int                  `json:"page" example:"1"`
	Limit int                  `json:"limit" example:"20"`
}

type ResourceReportListResponse struct {
	Code int                    `json:"code" example:"0"`
	Data ResourceReportListData `json:"data"`
	Msg  string                 `json:"msg" example:"success"`
}

type ResourceReportDataResponse struct {
	Code int                `json:"code" example:"0"`
	Data ResourceReportData `json:"data"`
	Msg  string             `json:"msg" example:"success"`
}

func NewResourceReportData(report *model.ResourceReport) ResourceReportData {
	data := ResourceReportData{
		ID:          report.ID,
		ResourceID:  report.ResourceID,
		UserID:      report.UserID,
		Reason:      report.Reason,
		Description: report.Description,
		Status:      report.Status,
		HandledBy:   report.HandledBy,
		HandledAt:   report.HandledAt,
		CreatedAt:   report.CreatedAt,
		UpdatedAt:   report.UpdatedAt,
	}
	if report.Resource != nil {
		data.Resource = &ReportedResourceData{
			ID:     report.Resource.ID,
			Title:  report.Resource.Title,
			Type:   report.Resource.Type,
			Status: report.Resource.Status,
		}
	}
	return data
}

func NewResourceReportListItems(reports []model.ResourceReport) []ResourceReportData {
	items := make([]ResourceReportData, 0, len(reports))
	for i := range reports {
		items = append(items, NewResourceReportData(&reports[i]))
	}
	return items
}
