package dto

import (
	"time"

	"backend/internal/feedback/model"
)

type CreateFeedbackRequest struct {
	Type    string `json:"type" binding:"required,oneof=feedback copyright" example:"feedback"`
	Content string `json:"content" binding:"required,min=5,max=2000" example:"希望增加深色模式的邮件模板"`
	Contact string `json:"contact" binding:"max=255" example:"someone@example.com"`
}

type AdminFeedbackQuery struct {
	Page    int    `form:"page" binding:"omitempty,min=1,max=1000000"`
	Limit   int    `form:"limit" binding:"omitempty,min=1,max=100"`
	Type    string `form:"type" binding:"omitempty,oneof=feedback copyright" example:"feedback"`
	Handled *bool  `form:"handled" example:"false"`
}

type HandleFeedbackRequest struct {
	Handled bool `json:"handled" binding:"required" example:"true"`
}

type FeedbackData struct {
	ID        uint       `json:"id" example:"1"`
	Type      string     `json:"type" example:"feedback"`
	Content   string     `json:"content" example:"希望增加深色模式的邮件模板"`
	Contact   string     `json:"contact" example:"someone@example.com"`
	UserID    *uint      `json:"user_id"`
	IP        string     `json:"ip" example:"203.0.113.10"`
	UserAgent string     `json:"user_agent" example:"Mozilla/5.0"`
	HandledBy *uint      `json:"handled_by" example:"3"`
	HandledAt *time.Time `json:"handled_at"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type FeedbackListData struct {
	Items []FeedbackData `json:"items"`
	Total int64          `json:"total" example:"10"`
	Page  int            `json:"page" example:"1"`
	Limit int            `json:"limit" example:"20"`
}

type FeedbackListResponse struct {
	Code int      `json:"code" example:"0"`
	Data []FeedbackData `json:"data"`
	Msg  string   `json:"msg" example:"success"`
}

type AdminFeedbackListResponse struct {
	Code int              `json:"code" example:"0"`
	Data FeedbackListData `json:"data"`
	Msg  string           `json:"msg" example:"success"`
}

type FeedbackDataResponse struct {
	Code int          `json:"code" example:"0"`
	Data FeedbackData `json:"data"`
	Msg  string       `json:"msg" example:"success"`
}

func NewFeedbackData(feedback *model.Feedback) FeedbackData {
	return FeedbackData{
		ID: feedback.ID, Type: feedback.Type, Content: feedback.Content, Contact: feedback.Contact,
		UserID: feedback.UserID, IP: feedback.IP, UserAgent: feedback.UserAgent,
		HandledBy: feedback.HandledBy, HandledAt: feedback.HandledAt,
		CreatedAt: feedback.CreatedAt, UpdatedAt: feedback.UpdatedAt,
	}
}

func NewFeedbackList(items []model.Feedback) []FeedbackData {
	result := make([]FeedbackData, 0, len(items))
	for i := range items {
		result = append(result, NewFeedbackData(&items[i]))
	}
	return result
}
