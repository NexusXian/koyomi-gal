package dto

import (
	"time"

	"backend/internal/classification/model"
)

type EvidenceResponse struct {
	ID         uint      `json:"id" example:"1"`
	SourceType string    `json:"source_type" example:"official"`
	Title      string    `json:"title" example:"サクラノ刻 公式サイト"`
	URL        string    `json:"url" example:"https://example.com"`
	Evidence   string    `json:"evidence" example:"18歳以上対象"`
	Weight     int       `json:"weight" example:"10"`
	CreatedAt  time.Time `json:"created_at"`
}

// ClassificationResponse is one proposal row including its evidence.
type ClassificationResponse struct {
	ID             uint               `json:"id" example:"1"`
	GameID         uint               `json:"game_id" example:"1350"`
	Classification string             `json:"classification" example:"r18"`
	Confidence     float64            `json:"confidence" example:"0.98"`
	Reason         string             `json:"reason" example:"游戏官网明确标注该作品为18岁以上对象作品。"`
	Conflict       bool               `json:"conflict" example:"false"`
	Status         string             `json:"status" example:"pending_review"`
	Model          string             `json:"model" example:"deepseek-chat"`
	ErrorMessage   string             `json:"error_message" example:""`
	ReviewerID     *uint              `json:"reviewer_id"`
	ReviewedAt     *time.Time         `json:"reviewed_at"`
	CreatedAt      time.Time          `json:"created_at"`
	UpdatedAt      time.Time          `json:"updated_at"`
	Evidences      []EvidenceResponse `json:"evidences"`
}

type GameSummary struct {
	ID             uint   `json:"id" example:"1350"`
	Title          string `json:"title" example:"Sakura no Toki"`
	OriginalTitle  string `json:"original_title" example:"サクラノ刻"`
	CoverURL       string `json:"cover_url" example:"https://img.example.com/cover.jpg"`
	AgeRating      int16  `json:"age_rating" example:"0"`
	CoverSensitive bool   `json:"cover_sensitive" example:"false"`
}

// ClassificationDetailData pairs the game summary with its latest proposal.
type ClassificationDetailData struct {
	Game           GameSummary             `json:"game"`
	Classification *ClassificationResponse `json:"classification"`
}

// ClassificationDetailResponse is the envelope for classification queries.
type ClassificationDetailResponse struct {
	Code int                      `json:"code" example:"0"`
	Data ClassificationDetailData `json:"data"`
	Msg  string                   `json:"msg" example:"success"`
}

// BatchResponse is the envelope for batch run/approve endpoints.
type BatchResponse struct {
	Code int       `json:"code" example:"0"`
	Data BatchData `json:"data"`
	Msg  string    `json:"msg" example:"success"`
}

func NewEvidenceResponses(rows []model.GameClassificationEvidence) []EvidenceResponse {
	items := make([]EvidenceResponse, 0, len(rows))
	for _, row := range rows {
		items = append(items, EvidenceResponse{
			ID:         row.ID,
			SourceType: row.SourceType,
			Title:      row.Title,
			URL:        row.URL,
			Evidence:   row.Evidence,
			Weight:     row.Weight,
			CreatedAt:  row.CreatedAt,
		})
	}
	return items
}

func NewClassificationResponse(row *model.GameClassification) *ClassificationResponse {
	if row == nil {
		return nil
	}
	return &ClassificationResponse{
		ID:             row.ID,
		GameID:         row.GameID,
		Classification: row.Classification,
		Confidence:     row.Confidence,
		Reason:         row.Reason,
		Conflict:       row.Conflict,
		Status:         row.Status,
		Model:          row.Model,
		ErrorMessage:   row.ErrorMessage,
		ReviewerID:     row.ReviewerID,
		ReviewedAt:     row.ReviewedAt,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
		Evidences:      NewEvidenceResponses(row.Evidences),
	}
}
