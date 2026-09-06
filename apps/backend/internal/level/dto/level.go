package leveldto

import (
	"time"

	levelModel "backend/internal/level/model"
)

// UserLevelSummary is the public level badge payload shared by every module
// that displays a user (profiles, comments, posts, contributor lists).
type UserLevelSummary struct {
	Level   int    `json:"level" example:"5"`
	Name    string `json:"name" example:"资深鉴赏家"`
	IconURL string `json:"icon_url,omitempty" example:""`
	Color   string `json:"color,omitempty" example:"#f59e0b"`
}

// UserLevelData is the full level profile with progress information.
type UserLevelData struct {
	Level            int     `json:"level" example:"4"`
	LevelName        string  `json:"level_name" example:"鉴赏家"`
	TotalExp         int64   `json:"total_exp" example:"3480"`
	CurrentLevelExp  int64   `json:"current_level_exp" example:"1500"`
	NextLevel        *int    `json:"next_level" example:"5"`
	NextLevelExp     *int64  `json:"next_level_exp" example:"4000"`
	NextLevelName    *string `json:"next_level_name" example:"资深鉴赏家"`
	RemainingExp     int64   `json:"remaining_exp" example:"520"`
	Progress         float64 `json:"progress" example:"0.792"`
	IconURL          string  `json:"icon_url" example:""`
	Color            string  `json:"color" example:""`
	IsMaxLevel       bool    `json:"is_max_level" example:"false"`
	CheckedInToday   bool    `json:"checked_in_today" example:"true"`
	ConsecutiveDays  int     `json:"consecutive_days" example:"7"`
}

type UserLevelResponse struct {
	Code int            `json:"code" example:"0"`
	Data UserLevelData  `json:"data"`
	Msg  string         `json:"msg" example:"success"`
}

type CheckinStatusData struct {
	CheckedInToday  bool `json:"checked_in_today" example:"false"`
	ConsecutiveDays int  `json:"consecutive_days" example:"3"`
}

type CheckinStatusResponse struct {
	Code int               `json:"code" example:"0"`
	Data CheckinStatusData `json:"data"`
	Msg  string            `json:"msg" example:"success"`
}

type CheckinResultData struct {
	ExpGained       int    `json:"exp_gained" example:"10"`
	ConsecutiveDays int    `json:"consecutive_days" example:"7"`
	TotalExp        int64  `json:"total_exp" example:"1280"`
	Level           int    `json:"level" example:"3"`
	LevelName       string `json:"level_name" example:"爱好者"`
}

type CheckinResultResponse struct {
	Code int              `json:"code" example:"0"`
	Data CheckinResultData `json:"data"`
	Msg  string           `json:"msg" example:"success"`
}

type ExperienceLogData struct {
	ID          uint      `json:"id" example:"1"`
	EventType   string    `json:"event_type" example:"daily_checkin"`
	ExpDelta    int64     `json:"exp_delta" example:"10"`
	SourceType  string    `json:"source_type" example:"checkin"`
	SourceID    *uint     `json:"source_id" example:"1"`
	Description string    `json:"description" example:"每日签到"`
	CreatedAt   time.Time `json:"created_at"`
}

type ExperienceLogListData struct {
	Items []ExperienceLogData `json:"items"`
	Total int64               `json:"total" example:"10"`
	Page  int                 `json:"page" example:"1"`
	Limit int                 `json:"limit" example:"20"`
}

type ExperienceLogListResponse struct {
	Code int                   `json:"code" example:"0"`
	Data ExperienceLogListData `json:"data"`
	Msg  string                `json:"msg" example:"success"`
}

type ExperienceQuery struct {
	Page  int `form:"page" binding:"omitempty,min=1,max=1000000"`
	Limit int `form:"limit" binding:"omitempty,min=1,max=100"`
}

type LevelConfigData struct {
	ID          uint      `json:"id" example:"1"`
	Level       int       `json:"level" example:"4"`
	Name        string    `json:"name" example:"鉴赏家"`
	MinExp      int64     `json:"min_exp" example:"1500"`
	IconURL     string    `json:"icon_url" example:""`
	Color       string    `json:"color" example:"#f59e0b"`
	Description string    `json:"description" example:"对 Galgame 有自己的见解"`
	IsEnabled   bool      `json:"is_enabled" example:"true"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type LevelConfigListData struct {
	Items []LevelConfigData `json:"items"`
	Total int64             `json:"total" example:"8"`
}

type LevelConfigListResponse struct {
	Code int                 `json:"code" example:"0"`
	Data LevelConfigListData `json:"data"`
	Msg  string              `json:"msg" example:"success"`
}

type LevelConfigDataResponse struct {
	Code int            `json:"code" example:"0"`
	Data LevelConfigData `json:"data"`
	Msg  string         `json:"msg" example:"success"`
}

type CreateLevelConfigRequest struct {
	Level       int    `json:"level" binding:"required,min=1,max=1000" example:"9"`
	Name        string `json:"name" binding:"required,max=50" example:"传奇"`
	MinExp      int64  `json:"min_exp" binding:"required,min=0" example:"150000"`
	IconURL     string `json:"icon_url" binding:"omitempty,max=2048" example:""`
	Color       string `json:"color" binding:"omitempty,max=32" example:"#dc2626"`
	Description string `json:"description" binding:"omitempty,max=255" example:"社区的传奇贡献者"`
	IsEnabled   *bool  `json:"is_enabled" example:"true"`
}

type UpdateLevelConfigRequest struct {
	Name        string `json:"name" binding:"required,max=50" example:"鉴赏家"`
	MinExp      int64  `json:"min_exp" binding:"required,min=0" example:"1500"`
	IconURL     string `json:"icon_url" binding:"omitempty,max=2048" example:""`
	Color       string `json:"color" binding:"omitempty,max=32" example:"#f59e0b"`
	Description string `json:"description" binding:"omitempty,max=255" example:"对 Galgame 有自己的见解"`
	IsEnabled   *bool  `json:"is_enabled" binding:"required" example:"true"`
}

type ExperienceRuleData struct {
	ID              uint      `json:"id" example:"1"`
	EventType       string    `json:"event_type" example:"daily_checkin"`
	Name            string    `json:"name" example:"每日签到"`
	Exp             int       `json:"exp" example:"10"`
	DailyLimit      int       `json:"daily_limit" example:"1"`
	DailyExpLimit   int       `json:"daily_exp_limit" example:"0"`
	CooldownSeconds int       `json:"cooldown_seconds" example:"0"`
	Enabled         bool      `json:"enabled" example:"true"`
	Description     string    `json:"description" example:"每天首次签到获得经验"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type ExperienceRuleListData struct {
	Items []ExperienceRuleData `json:"items"`
	Total int64                `json:"total" example:"11"`
}

type ExperienceRuleListResponse struct {
	Code int                    `json:"code" example:"0"`
	Data ExperienceRuleListData `json:"data"`
	Msg  string                 `json:"msg" example:"success"`
}

type ExperienceRuleDataResponse struct {
	Code int                `json:"code" example:"0"`
	Data ExperienceRuleData `json:"data"`
	Msg  string             `json:"msg" example:"success"`
}

type UpdateExperienceRuleRequest struct {
	Name            *string `json:"name" binding:"omitempty,max=100" example:"每日签到"`
	Exp             *int    `json:"exp" binding:"omitempty,min=0" example:"10"`
	DailyLimit      *int    `json:"daily_limit" binding:"omitempty,min=0" example:"1"`
	DailyExpLimit   *int    `json:"daily_exp_limit" binding:"omitempty,min=0" example:"0"`
	CooldownSeconds *int    `json:"cooldown_seconds" binding:"omitempty,min=0" example:"0"`
	Enabled         *bool   `json:"enabled" example:"true"`
	Description     *string `json:"description" binding:"omitempty,max=255" example:"每天首次签到获得经验"`
}

type AdjustExperienceRequest struct {
	Exp    int64  `json:"exp" binding:"required,ne=0" example:"500"`
	Reason string `json:"reason" binding:"required,max=255" example:"夏季资源贡献活动奖励"`
}

func NewLevelConfigData(config *levelModel.LevelConfig) LevelConfigData {
	return LevelConfigData{
		ID: config.ID, Level: config.Level, Name: config.Name, MinExp: config.MinExp,
		IconURL: config.IconURL, Color: config.Color, Description: config.Description,
		IsEnabled: config.IsEnabled, CreatedAt: config.CreatedAt, UpdatedAt: config.UpdatedAt,
	}
}

func NewLevelConfigList(configs []levelModel.LevelConfig) []LevelConfigData {
	items := make([]LevelConfigData, 0, len(configs))
	for i := range configs {
		items = append(items, NewLevelConfigData(&configs[i]))
	}
	return items
}

func NewExperienceRuleData(rule *levelModel.ExperienceRule) ExperienceRuleData {
	return ExperienceRuleData{
		ID: rule.ID, EventType: string(rule.EventType), Name: rule.Name, Exp: rule.Exp,
		DailyLimit: rule.DailyLimit, DailyExpLimit: rule.DailyExpLimit,
		CooldownSeconds: rule.CooldownSeconds, Enabled: rule.Enabled,
		Description: rule.Description, CreatedAt: rule.CreatedAt, UpdatedAt: rule.UpdatedAt,
	}
}

func NewExperienceRuleList(rules []levelModel.ExperienceRule) []ExperienceRuleData {
	items := make([]ExperienceRuleData, 0, len(rules))
	for i := range rules {
		items = append(items, NewExperienceRuleData(&rules[i]))
	}
	return items
}
