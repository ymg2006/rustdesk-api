package api

import "time"

// OrderResp is the create/query order response.
type OrderResp struct {
	OutTradeNo    string     `json:"out_trade_no"`
	CashierURL    string     `json:"cashier_url"`
	QRPayload     string     `json:"qr_payload"`
	AmountCents   int64      `json:"amount_cents"`
	Plan          string     `json:"plan"`
	PlanKey       string     `json:"plan_key"`
	PeriodDays    int        `json:"period_days"`
	ExpireSeconds int        `json:"expire_seconds"`
	Status        string     `json:"status"`
	PaidAt        *time.Time `json:"paid_at,omitempty"`
	CodeIssued    bool       `json:"code_issued"`
	InviteCode    string     `json:"invite_code,omitempty"`
	ExpireAt      *time.Time `json:"expire_at,omitempty"`
}

// ClaimResp is the claim response.
type ClaimResp struct {
	Code     string    `json:"code"`
	ExpireAt time.Time `json:"expire_at"`
	Plan     string    `json:"plan"`
}

// MineResp is the current user's subscription response.
type MineResp struct {
	Plan                 string     `json:"plan"`
	SubscriptionExpireAt *time.Time `json:"subscription_expire_at"`
	Status               string     `json:"status"`
	DaysLeft             int        `json:"days_left"`
	IsExpiringSoon       bool       `json:"is_expiring_soon"`
}

// RedeemResp is the redemption response.
type RedeemResp struct {
	Plan                 string     `json:"plan"`
	ExpireAt             time.Time  `json:"expire_at"`
	SubscriptionExpireAt *time.Time `json:"subscription_expire_at"`
}

// CodeListItem is an invite-code list item for admin management.
type CodeListItem struct {
	ID           uint      `json:"id"`
	Code         string    `json:"code"`
	Plan         string    `json:"plan"`
	Status       string    `json:"status"`
	UsedBy       uint      `json:"used_by"`
	UsedByName   string    `json:"used_by_name"`
	ExpireAt     time.Time `json:"expire_at"`
	BoundOrderID string    `json:"bound_order_id"`
	CreatedAt    time.Time `json:"created_at"`
}
