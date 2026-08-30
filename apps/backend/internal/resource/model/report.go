package model

import "time"

const (
	ReportReasonInvalidLink int16 = iota
	ReportReasonWrongPassword
	ReportReasonCorrupted
	ReportReasonMalware
	ReportReasonWrongVersion
	ReportReasonDuplicate
	ReportReasonOther
)

const (
	ReportStatusPending int16 = iota
	ReportStatusResolved
	ReportStatusRejected
)

type ResourceReport struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	ResourceID  uint       `gorm:"not null" json:"resource_id"`
	UserID      uint       `gorm:"not null" json:"user_id"`
	Reason      int16      `gorm:"not null" json:"reason"`
	Description string     `gorm:"not null" json:"description"`
	Status      int16      `gorm:"not null" json:"status"`
	HandledBy   *uint      `json:"handled_by"`
	HandledAt   *time.Time `json:"handled_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	Resource    *Resource  `gorm:"foreignKey:ResourceID" json:"resource,omitempty"`
}

func (ResourceReport) TableName() string {
	return "resource_reports"
}
