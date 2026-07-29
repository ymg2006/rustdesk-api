package model

import "time"

// ServerStatusMonitor Customize server detection entries (users can fill in their own addresses on the page and can create multiple)
type ServerStatusMonitor struct {
	RowId     uint      `json:"row_id" gorm:"primaryKey"`
	UserId    uint      `json:"user_id" gorm:"default:0;not null;index"`        // Creator; multi-user isolation
	Name      string    `json:"name" gorm:"size:128;not null;default:''"`       // display name
	Host      string    `json:"host" gorm:"size:255;not null;default:''"`       // Host/IP
	Port      int       `json:"port" gorm:"default:0"`                          // Port; 0 means only detecting host connectivity
	Protocol  string    `json:"protocol" gorm:"size:16;not null;default:'tcp'"` // tcp
	Enabled   int       `json:"enabled" gorm:"default:1"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (ServerStatusMonitor) TableName() string {
	return "server_status_monitors"
}
