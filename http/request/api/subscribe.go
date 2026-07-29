package api

// CreateOrderReq is the create-order request.
type CreateOrderReq struct {
	Channel string `json:"channel" binding:"required,oneof=wechat alipay"` // payment channel
	// PlanKey is the duration option key (1m / 3m / 6m / 12m). The server uses it to look up pricing.
	PlanKey string `json:"plan_key" binding:"required,max=16"`
}

// ClaimReq is the order-number claim request.
type ClaimReq struct {
	OutTradeNo string `json:"out_trade_no" binding:"required,max=64"` // merchant order number
}

// RedeemReq is the invite-code redemption request.
type RedeemReq struct {
	Code string `json:"code" binding:"required,max=64"` // invite code
}

// AdminCreateCodeReq is the admin request to manually generate invite codes.
type AdminCreateCodeReq struct {
	Plan       string `json:"plan"`        // plan identifier, defaults to "pro"
	ExpireDays int    `json:"expire_days"` // validity period in days, defaults to 30
	Remark     string `json:"remark"`      // admin remark
}
