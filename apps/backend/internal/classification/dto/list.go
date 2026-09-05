package dto

import (
	"time"

	"backend/internal/classification/model"
)

// ClassificationListQuery filters the admin classification queue. A blank
// status or keyword means no restriction on that dimension.
type ClassificationListQuery struct {
	Page    int    `form:"page" binding:"omitempty,min=1,max=1000000" example:"1"`
	Limit   int    `form:"limit" binding:"omitempty,min=1,max=100" example:"20"`
	Status  string `form:"status" binding:"omitempty,oneof=queued processing pending_review approved rejected failed cancelled" example:"queued"`
	Keyword string `form:"keyword" binding:"omitempty,max=255" example:"サクラ"`
}

// ClassificationListItem is one game's latest AI run in the queue listing.
type ClassificationListItem struct {
	ID             uint      `json:"id" example:"42"`
	GameID         uint      `json:"game_id" example:"1350"`
	GameTitle      string    `json:"game_title" example:"Sakura no Toki"`
	OriginalTitle  string    `json:"original_title" example:"サクラノ刻"`
	Classification string    `json:"classification" example:"r18"`
	Confidence     float64   `json:"confidence" example:"0.98"`
	Reason         string    `json:"reason" example:"游戏官网明确标注该作品为18岁以上对象作品。"`
	Conflict       bool      `json:"conflict" example:"false"`
	Status         string    `json:"status" example:"queued"`
	Model          string    `json:"model" example:"deepseek-chat"`
	ErrorMessage   string    `json:"error_message" example:""`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// ClassificationListData is the paginated queue envelope.
type ClassificationListData struct {
	Items []ClassificationListItem `json:"items"`
	Total int64                    `json:"total" example:"10"`
	Page  int                      `json:"page" example:"1"`
	Limit int                      `json:"limit" example:"20"`
}

// ClassificationListResponse is the OpenAPI schema for the queue listing.
type ClassificationListResponse struct {
	Code int                    `json:"code" example:"0"`
	Data ClassificationListData `json:"data"`
	Msg  string                 `json:"msg" example:"success"`
}

// NewClassificationListItem maps one latest-run view row to the response item.
func NewClassificationListItem(task *model.ClassificationTask) ClassificationListItem {
	return ClassificationListItem{
		ID:             task.ID,
		GameID:         task.GameID,
		GameTitle:      task.GameTitle,
		OriginalTitle:  task.OriginalTitle,
		Classification: task.Classification,
		Confidence:     task.Confidence,
		Reason:         task.Reason,
		Conflict:       task.Conflict,
		Status:         task.Status,
		Model:          task.Model,
		ErrorMessage:   task.ErrorMessage,
		CreatedAt:      task.CreatedAt,
		UpdatedAt:      task.UpdatedAt,
	}
}
