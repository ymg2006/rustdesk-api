package model

// Strategy remote strategy configuration
// bind_type: user=bind user, group=bind device group, tag=bind tag, global=global
type Strategy struct {
	IdModel
	Name        string `json:"name" gorm:"type:varchar(100);default:'';not null;comment:'Strategy name'"`
	ConfigItems string `json:"config_items" gorm:"type:text;comment:'key=value multi-line configuration'"`
	Priority    int    `json:"priority" gorm:"type:int;default:0;comment:'Priority, the larger the number, the priority'"`
	Status      int    `json:"status" gorm:"type:tinyint;default:1;comment:'1=enable 2=disable'"`
	BindType    string `json:"bind_type" gorm:"type:varchar(16);default:'global';comment:'user/group/tag/global'"`
	BindId      uint   `json:"bind_id" gorm:"default:0;not null;index;comment:'User/device group/tag ID bound to the policy, 0 for global'"`
	TimeModel
}

func (Strategy) TableName() string {
	return "strategies"
}

type StrategyList struct {
	Strategies []*Strategy `json:"list"`
	Pagination
}

// StrategyOptions The strategy configuration for heartbeat delivery, corresponding to the client StrategyOptions
type StrategyOptions struct {
	ConfigOptions map[string]string `json:"config_options"`
	Extra         map[string]string `json:"extra,omitempty"`
}
