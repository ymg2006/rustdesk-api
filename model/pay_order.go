package model

import "time"

// PayOrder subscribes to payment orders (persistent pending payment/paid status)
type PayOrder struct {
	IdModel
	// OutTradeNo merchant order number (globally unique, used as an idempotent key)
	OutTradeNo string `gorm:"uniqueIndex;size:64" json:"out_trade_no"`
	// UserID Order user ID
	UserID uint `gorm:"index" json:"user_id"`
	// Plan package ID
	Plan string `gorm:"size:32;default:'pro'" json:"plan"`
	// PlanKey duration option key, such as 1m / 3m / 6m / 12m
	PlanKey string `gorm:"size:16" json:"plan_key"`
	// AmountCents order amount (cents)
	AmountCents int64 `json:"amount_cents"`
	// Channel payment channel: wechat / alipay
	Channel string `gorm:"size:16" json:"channel"`
	// PeriodDays subscription period days
	PeriodDays int `json:"period_days"`
	// Status order status: pending / paid / failed / closed
	Status string `gorm:"size:16;index;default:'pending'" json:"status"`
	// CashierURL Cashier link
	CashierURL string `gorm:"size:512" json:"cashier_url"`
	// QRPayload optional: the backend directly outputs the QR code content
	QRPayload string `gorm:"size:512" json:"qr_payload"`
	// PaidAt payment completion time
	PaidAt *time.Time `json:"paid_at"`
	// CallbackRaw raw callback message (only used for reconciliation/troubleshooting, not exposed to the outside world)
	CallbackRaw string    `gorm:"type:text" json:"-"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TableName specifies the table name
func (PayOrder) TableName() string {
	return "pay_orders"
}
