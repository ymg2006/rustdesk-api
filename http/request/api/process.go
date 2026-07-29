package api

// ProcessItem is the reported status of one monitored item.
type ProcessItem struct {
	Name    string `json:"name"`    // display name
	Type    string `json:"type"`    // process | port
	Target  string `json:"target"`  // process name / port
	Running bool   `json:"running"` // whether it is running
}

// ProcessStatusForm is the monitoring status reported by the client.
type ProcessStatusForm struct {
	PeerId string        `json:"peer_id"`
	Uuid   string        `json:"uuid"`
	Items  []ProcessItem `json:"items"`
}

// ProcessConfigQuery is used by the client to fetch its monitoring configuration.
type ProcessConfigQuery struct {
	PeerId string `json:"peer_id" form:"peer_id"`
}

// ProcessRuleForm is used by the admin backend to configure monitoring rules.
type ProcessRuleForm struct {
	RowId         uint   `json:"row_id"`
	PeerId        string `json:"peer_id"`
	Name          string `json:"name"`
	Type          string `json:"type"`
	Target        string `json:"target"`
	Interval      int    `json:"interval"`
	DownThreshold int    `json:"down_threshold"`
	AlertConfigId uint   `json:"alert_config_id"`
	Enabled       int    `json:"enabled"`
}
