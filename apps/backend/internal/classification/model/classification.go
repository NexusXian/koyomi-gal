package model

import "time"

type ClassificationValue string

const (
	ClassificationR18     ClassificationValue = "r18"
	ClassificationR17     ClassificationValue = "r17"
	ClassificationR15     ClassificationValue = "r15"
	ClassificationR12     ClassificationValue = "r12"
	ClassificationNonR18  ClassificationValue = "non_r18"
	ClassificationUnknown ClassificationValue = "unknown"
)

func (v ClassificationValue) Valid() bool {
	switch v {
	case ClassificationR18, ClassificationR17, ClassificationR15, ClassificationR12,
		ClassificationNonR18, ClassificationUnknown:
		return true
	default:
		return false
	}
}

type ClassificationStatus string

const (
	StatusQueued     ClassificationStatus = "queued"
	StatusProcessing ClassificationStatus = "processing"
	StatusPending    ClassificationStatus = "pending_review"
	StatusApproved   ClassificationStatus = "approved"
	StatusRejected   ClassificationStatus = "rejected"
	StatusFailed     ClassificationStatus = "failed"
)

func (s ClassificationStatus) Active() bool {
	return s == StatusQueued || s == StatusProcessing
}

type GameClassification struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	GameID         uint       `gorm:"not null;index" json:"game_id"`
	Classification string     `gorm:"size:16;not null;default:''" json:"classification"`
	Confidence     float64    `gorm:"type:numeric(5,4);not null" json:"confidence"`
	Reason         string     `gorm:"not null;default:''" json:"reason"`
	Conflict       bool       `gorm:"not null;default:false" json:"conflict"`
	Status         string     `gorm:"size:24;not null;default:'queued'" json:"status"`
	Model          string     `gorm:"size:128;not null;default:''" json:"model"`
	ErrorMessage   string     `gorm:"not null;default:''" json:"error_message"`
	ReviewerID     *uint      `json:"reviewer_id"`
	ReviewedAt     *time.Time `json:"reviewed_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`

	Evidences []GameClassificationEvidence `gorm:"foreignKey:ClassificationID" json:"evidences,omitempty"`
}

func (GameClassification) TableName() string {
	return "game_classifications"
}

func (c *GameClassification) AsClassificationValue() ClassificationValue {
	value := ClassificationValue(c.Classification)
	if value.Valid() {
		return value
	}
	return ClassificationUnknown
}

func (c *GameClassification) EvidenceCount() int {
	return len(c.Evidences)
}
