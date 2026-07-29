package model

type AlertConfig struct {
	RowId                  uint   `json:"row_id" gorm:"primaryKey"`
	UserId                 uint   `json:"user_id" gorm:"default:0;not null;index"`
	ChannelId              uint   `json:"channel_id" gorm:"default:0;not null;index"` // FK -> AlertChannel.row_id
	Channel                string `json:"channel" gorm:"size:32;not null;default:''"` // Redundant, easy to query
	Name                   string `json:"name" gorm:"size:100;not null;default:''"`   // Rule name
	OfflineMin             int    `json:"offline_min" gorm:"default:5"`
	Enabled                int    `json:"enabled" gorm:"default:1"`
	MonitorAll             int    `json:"monitor_all" gorm:"default:1"`
	Recipients             string `json:"recipients" gorm:"size:500; not null;default:''"` // Recipients (SMTP recipient email addresses, comma separated); determined by sending configuration
	LastNotifiedAt         int64  `json:"last_notified_at" gorm:"default:0"`
	NotifiedPeers          string `json:"notified_peers" gorm:"type:text"`           // JSON map (including number of notifications per device, weight, etc.)
	ConsecutiveTriggerDays int    `json:"consecutive_trigger_days" gorm:"default:0"` // The number of consecutive days that offline alarms are triggered
	LastTriggerDay         int    `json:"last_trigger_day" gorm:"default:0"`         // The date when the alarm was last triggered (YYYYMMDD), used for calculation of consecutive days
	CreatedAt              int64  `json:"created_at" gorm:"autoCreateTime"`          // Alarm creation timestamp, used to determine whether the offline event occurred before creation
}

func (AlertConfig) TableName() string {
	return "alert_configs"
}
