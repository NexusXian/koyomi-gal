package model

import (
	"encoding/json"
	"time"
)

const (
	ImportJobStatusPending int16 = iota
	ImportJobStatusRunning
	ImportJobStatusSucceeded
	ImportJobStatusFailed
	ImportJobStatusCancelled
)

const ImportJobTypeBatch = "batch"

type ImportJob struct {
	ID             uint            `gorm:"primaryKey" json:"id"`
	Provider       string          `gorm:"size:32;not null" json:"provider"`
	JobType        string          `gorm:"size:32;not null" json:"job_type"`
	Status         int16           `gorm:"not null" json:"status"`
	TotalCount     int             `gorm:"not null" json:"total_count"`
	ProcessedCount int             `gorm:"not null" json:"processed_count"`
	CreatedCount   int             `gorm:"not null" json:"created_count"`
	SkippedCount   int             `gorm:"not null" json:"skipped_count"`
	FailedCount    int             `gorm:"not null" json:"failed_count"`
	Params         json.RawMessage `gorm:"type:jsonb" json:"params,omitempty"`
	ErrorMessage   string          `gorm:"not null" json:"error_message"`
	CreatedBy      *uint           `json:"created_by"`
	CreatedAt      time.Time       `json:"created_at"`
	StartedAt      *time.Time      `json:"started_at"`
	FinishedAt     *time.Time      `json:"finished_at"`
}

func (ImportJob) TableName() string {
	return "import_jobs"
}
