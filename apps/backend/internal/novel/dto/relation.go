package dto

import "time"

type CreateRelationRequest struct {
	TargetType   string `json:"target_type" binding:"required,oneof=galgame novel" example:"galgame"`
	TargetID     uint   `json:"target_id" binding:"required,gt=0" example:"3"`
	RelationType string `json:"relation_type" binding:"required,oneof=adaptation original spin_off sequel prequel same_series related" example:"adaptation"`
}

type RelationData struct {
	ID           uint      `json:"id" example:"1"`
	SourceType   string    `json:"source_type" example:"novel"`
	SourceID     uint      `json:"source_id" example:"1"`
	TargetType   string    `json:"target_type" example:"galgame"`
	TargetID     uint      `json:"target_id" example:"3"`
	RelationType string    `json:"relation_type" example:"adaptation"`
	CreatedBy    *uint     `json:"created_by" example:"1001"`
	CreatedAt    time.Time `json:"created_at"`
}

type RelationListData struct {
	Items []RelationData `json:"items"`
	Total int64          `json:"total" example:"2"`
}

type RelationListResponse struct {
	Code int              `json:"code" example:"0"`
	Data RelationListData `json:"data"`
	Msg  string           `json:"msg" example:"success"`
}

type RelationDataResponse struct {
	Code int          `json:"code" example:"0"`
	Data RelationData `json:"data"`
	Msg  string       `json:"msg" example:"success"`
}
