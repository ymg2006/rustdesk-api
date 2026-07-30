package model

import (
	"time"

	"github.com/ymg2006/rustdesk-api/v2/model/custom_types"
)

// ProcessMonitorRule process/port monitoring rules (centralized configuration in the background, distributed to clients by device)
// source_type is empty or peers represents a single device rule; device_group / ab_tags represents a collection of rules

type ProcessMonitorRule struct {
	RowId         uint      `json:"row_id" gorm:"primaryKey"`
	UserId        uint      `json:"user_id" gorm:"default:0;not null;index"`           // Creator
	PeerId        string    `json:"peer_id" gorm:"size:128;not null;default:'';index"` // Target peer id of single device rule
	SourceType    string    `json:"source_type" gorm:"size:16;not null;default:''"`    // '' | peers | device_group | ab_tags
	SourceId      string    `json:"source_id" gorm:"size:128;not null;default:''"`     // Device group ID or address book label
	SourceName    string    `json:"source_name" gorm:"size:128;not null;default:''"`   // Display name (device group name/tag name)
	Name          string    `json:"name" gorm:"size:128;not null;default:''"`          // Monitoring item display name
	Type          string    `json:"type" gorm:"size:16;not null;default:'process'"`    // process | port
	Target        string    `json:"target" gorm:"size:255;not null;default:''"`        // Process name (such as notepad.exe) or port (such as 8080)
	Interval      int       `json:"interval" gorm:"default:30"`                        // Detection interval (seconds)
	DownThreshold int       `json:"down_threshold" gorm:"default:300"`                 // How many seconds after continuous down time will an alarm be triggered?
	AlertConfigId uint      `json:"alert_config_id" gorm:"default:0"`                  // Associated alert rules (reuse alert_config); 0 = no alert
	Enabled       int       `json:"enabled" gorm:"default:1"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (ProcessMonitorRule) TableName() string {
	return "process_monitor_rules"
}

// Association of ProcessMonitorRulePeer collection rules and devices and single device coverage configuration
// When overrides is an empty object {}, the parent rule is completely inherited; non-empty fields override the parent rule.
type ProcessMonitorRulePeer struct {
	RowId     uint                  `json:"row_id" gorm:"primaryKey"`
	RuleId    uint                  `json:"rule_id" gorm:"not null;index"`
	PeerId    string                `json:"peer_id" gorm:"size:128;not null;default:'';index"`
	Overrides custom_types.AutoJson `json:"overrides" gorm:"not null;" swaggertype:"object"`
	CreatedAt time.Time             `json:"created_at"`
	UpdatedAt time.Time             `json:"updated_at"`
}

func (ProcessMonitorRulePeer) TableName() string {
	return "process_monitor_rule_peers"
}

// ProcessMonitorStatus Real-time status of monitoring items reported by the device
type ProcessMonitorStatus struct {
	RowId     uint      `json:"row_id" gorm:"primaryKey"`
	PeerId    string    `json:"peer_id" gorm:"size:128;not null;default:'';index"`
	RuleId    uint      `json:"rule_id" gorm:"default:0;index"`
	Name      string    `json:"name" gorm:"size:128;not null;default:''"`
	Type      string    `json:"type" gorm:"size:16;not null;default:''"`
	Target    string    `json:"target" gorm:"size:255;not null;default:''"`
	Running   int       `json:"running" gorm:"default:0"` // 1=Running 0=Not running
	LastSeen  int64     `json:"last_seen"`                // The timestamp of the last reported running=1
	DownSince int64     `json:"down_since"`               // Timestamp when running=0 was first detected; cleared after recovery
	Alerted   int       `json:"alerted" gorm:"default:0"` // Whether the alarm has been sent (to avoid repeated sending)
	UpdatedAt time.Time `json:"updated_at"`
}

func (ProcessMonitorStatus) TableName() string {
	return "process_monitor_status"
}
