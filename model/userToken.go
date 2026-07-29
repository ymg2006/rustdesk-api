package model

type UserToken struct {
	IdModel
	UserId     uint   `json:"user_id" gorm:"default:0;not null;index"`
	DeviceUuid string `json:"device_uuid" gorm:"default:'';omitempty;"`
	DeviceId   string `json:"device_id" gorm:"default:'';omitempty;"`
	Token      string `json:"token" gorm:"default:'';not null;index"`
	ExpiredAt  int64  `json:"expired_at" gorm:"default:0;not null;"`
	// Fingerprint binds the client characteristics when the token is issued (IP+User-Agent hash),
	// Used by the backend to verify the source of the request to prevent the token from being used from other environments after being stolen.
	Fingerprint string `json:"fingerprint" gorm:"default:'';not null;"`
	TimeModel
}

type UserTokenList struct {
	UserTokens []UserToken `json:"list"`
	Pagination
}
