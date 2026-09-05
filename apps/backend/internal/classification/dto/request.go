package dto

// BatchClassificationRequest starts agent runs for many games at once.
type BatchClassificationRequest struct {
	GameIDs []uint `json:"game_ids" binding:"required,min=1,max=500,dive,gt=0" example:"1350,1305"`
}

// OverrideClassificationRequest replaces the agent proposal with a human
// verdict. The record stays pending_review until an admin approves it.
type OverrideClassificationRequest struct {
	Classification string `json:"classification" binding:"required,oneof=r18 non_r18 unknown" example:"r18"`
	Reason         string `json:"reason" binding:"max=4000" example:"管理员人工指定"`
}

type BatchItem struct {
	GameID uint   `json:"game_id" example:"1350"`
	Reason string `json:"reason" example:"置信度低于 95%"`
}

// BatchData summarizes a batch run or batch approval.
type BatchData struct {
	Enqueued       int         `json:"enqueued" example:"2"`
	Approved       []uint      `json:"approved,omitempty" example:"1350"`
	AlreadyRunning []uint      `json:"already_running,omitempty" example:"1305"`
	Skipped        []BatchItem `json:"skipped,omitempty"`
	Failed         []BatchItem `json:"failed,omitempty"`
}
