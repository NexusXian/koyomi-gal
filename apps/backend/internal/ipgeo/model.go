package ipgeo

import "time"

type Location struct {
	Country       string `json:"country"`
	Region        string `json:"region"`
	City          string `json:"city"`
	ISP           string `json:"isp"`
	DisplayRegion string `json:"display_region"`
}

type UserIPLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"not null" json:"user_id"`
	IPAddress string    `gorm:"column:ip_address;size:45;not null" json:"ip"`
	Country   string    `gorm:"size:64;not null" json:"country"`
	Region    string    `gorm:"size:64;not null" json:"region"`
	City      string    `gorm:"size:64;not null" json:"city"`
	ISP       string    `gorm:"size:128;not null" json:"isp"`
	Action    string    `gorm:"size:32;not null" json:"action"`
	EntityID  *uint     `json:"entity_id"`
	CreatedAt time.Time `json:"created_at"`
}

func (UserIPLog) TableName() string {
	return "user_ip_logs"
}
