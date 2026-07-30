package model

import "time"

// InviteCode authorization code (uniformly used for registration + subscription activation + renewal)
type InviteCode struct {
	IdModel
	// Code 32-bit base62 random string, unique index
	Code string `gorm:"uniqueIndex;size:64" json:"code"`
	// Plan Redeemable package ID
	Plan string `gorm:"size:32;default:'pro'" json:"plan"`
	// ExpireDays Valid days (automatic code issuance = subscription period; manual generation is specified by the administrator)
	ExpireDays int `json:"expire_days"`
	// The expiration time of the ExpireAt code itself
	ExpireAt time.Time `gorm:"index" json:"expire_at"`
	// Status code status: unused / used / revoked
	Status string `gorm:"size:16;index;default:'unused'" json:"status"`
	// UsedBy user user ID
	UsedBy uint `gorm:"index;default:0" json:"used_by"`
	// Merchant order number associated with BoundOrderID
	BoundOrderID string `gorm:"size:64;index" json:"bound_order_id"`
	// Remark Admin Notes
	Remark string `gorm:"size:256" json:"remark"`
	// UsedAt usage time
	UsedAt *time.Time `json:"used_at"`
	// RevokedAt expiration time
	RevokedAt *time.Time `json:"revoked_at"`
	// CreatedAt creation time
	CreatedAt time.Time `json:"created_at"`
}

// TableName specifies the table name
func (InviteCode) TableName() string {
	return "invite_codes"
}
