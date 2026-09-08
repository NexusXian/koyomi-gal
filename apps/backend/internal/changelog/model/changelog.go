package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

const (
	ItemTypeNew     = "new"
	ItemTypeImprove = "improve"
	ItemTypeFix     = "fix"
)

type ChangelogItem struct {
	Type string `json:"type" example:"new"`
	Text string `json:"text" example:"新增站点页脚与更新日志页面"`
}

// SiteChangelog is one published website version entry; items keep the
// typed new/improve/fix structure rendered by the public changelog page.
type SiteChangelog struct {
	ID          uint           `gorm:"primaryKey"`
	Version     string         `gorm:"size:32;not null"`
	Title       string         `gorm:"size:255;not null"`
	Items       ChangelogItems `gorm:"type:jsonb;not null"`
	PublishedAt time.Time      `gorm:"not null"`
	CreatedAt   time.Time      `gorm:"not null"`
	UpdatedAt   time.Time      `gorm:"not null"`
}

type ChangelogItems []ChangelogItem

func (i ChangelogItems) Value() (driver.Value, error) {
	if i == nil {
		return "[]", nil
	}
	raw, err := json.Marshal(i)
	if err != nil {
		return nil, err
	}
	return string(raw), nil
}

func (i *ChangelogItems) Scan(value any) error {
	if value == nil {
		*i = nil
		return nil
	}
	var raw []byte
	switch typed := value.(type) {
	case []byte:
		raw = typed
	case string:
		raw = []byte(typed)
	default:
		return errors.New("unsupported changelog items value")
	}
	return json.Unmarshal(raw, i)
}
