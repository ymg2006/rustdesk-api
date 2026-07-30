package model

// ClientDownload client download link
type ClientDownload struct {
	IdModel
	Platform  string `json:"platform" gorm:"type:varchar(32);default:'';not null;comment:'Platform ID'"`
	Name      string `json:"name" gorm:"type:varchar(128);default:'';not null;comment:'display name'"`
	Url       string `json:"url" gorm:"type:varchar(512);default:'';not null;comment:'Download link'"`
	SortOrder int    `json:"sort_order" gorm:"type:int;default:0;comment:'sort'"`
	Status    int    `json:"status" gorm:"type:tinyint;default:1;comment:'1=enable 2=disable'"`
	TimeModel
}

func (ClientDownload) TableName() string {
	return "client_downloads"
}
