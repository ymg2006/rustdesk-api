package api

import (
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ymg2006/rustdesk-api/v2/global"
	"github.com/ymg2006/rustdesk-api/v2/http/request/api"
	"github.com/ymg2006/rustdesk-api/v2/http/response"
	respApi "github.com/ymg2006/rustdesk-api/v2/http/response/api"
	"github.com/ymg2006/rustdesk-api/v2/lib/payverify"
	"github.com/ymg2006/rustdesk-api/v2/service"
)

// SubscribeController subscription controller
type SubscribeController struct{}

// NewSubscribeController creates a controller
func NewSubscribeController() *SubscribeController {
	return &SubscribeController{}
}

// getQRURL reads the filename from the configuration based on the channel and returns the full URL
func (sc *SubscribeController) getQRURL(c *gin.Context, channel string) string {
	cc := global.Config.Payment.Cashier
	qrPath := cc.AlipayQR
	if channel == "wechat" {
		qrPath = cc.WechatQR
	}
	if qrPath == "" {
		return ""
	}

	// Already a complete URL, return directly
	if len(qrPath) > 4 && qrPath[:4] == "http" {
		return qrPath
	}

	// Relative paths: Construct the URL as configured (the filename and extension are entirely determined by the configuration)
	// It is preferred to use the rustdesk.api-server configuration as the base URL (compatible with reverse proxy scenarios)
	base := filepath.Base(qrPath)
	apiServer := strings.TrimRight(global.Config.Rustdesk.ApiServer, "/")
	if apiServer != "" {
		return fmt.Sprintf("%s/static/qr/%s", apiServer, base)
	}
	// Fallback: constructed from request headers (TLS termination proxy scenario check X-Forwarded-Proto)
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	if f := c.Request.Header.Get("X-Forwarded-Proto"); f != "" {
		scheme = f
	}
	return fmt.Sprintf("%s://%s/static/qr/%s", scheme, c.Request.Host, base)
}

// Plans returns a list of optional durations
func (sc *SubscribeController) Plans(c *gin.Context) {
	sc.SubscriptionPlans(c)
}

// SubscriptionPlans returns a list of optional durations
func (sc *SubscribeController) SubscriptionPlans(c *gin.Context) {
	// No need to inject into global.Config, read directly from service
	subCfg := global.Config.Subscription
	type planItem struct {
		Key        string `json:"key"`
		Name       string `json:"name"`
		PriceCents int64  `json:"price_cents"`
		PeriodDays int    `json:"period_days"`
	}
	var items []planItem
	for _, p := range subCfg.Plans {
		items = append(items, planItem{
			Key:        p.Key,
			Name:       p.Name,
			PriceCents: p.PriceCents,
			PeriodDays: p.PeriodDays,
		})
	}
	response.Success(c, items)
}

// CreateOrder creates an order
func (sc *SubscribeController) CreateOrder(c *gin.Context) {
	req := &api.CreateOrderReq{}
	if err := c.ShouldBindJSON(req); err != nil {
		response.Fail(c, 400, response.TranslateMsg(c, "ParamError"))
		return
	}

	user := service.AllService.UserService.CurUser(c)
	if user.Id == 0 {
		response.Fail(c, 403, response.TranslateMsg(c, "NeedLogin"))
		return
	}

	order, err := service.AllService.SubscribeService.CreateOrder(user.Id, req.Channel, req.PlanKey)
	if err != nil {
		response.Fail(c, 500, err.Error())
		return
	}

	expireSec := global.Config.Payment.OrderExpireSec
	if expireSec <= 0 {
		expireSec = 600
	}

	resp := &respApi.OrderResp{
		OutTradeNo:    order.OutTradeNo,
		AmountCents:   order.AmountCents,
		Plan:          order.Plan,
		PlanKey:       order.PlanKey,
		PeriodDays:    order.PeriodDays,
		ExpireSeconds: expireSec,
		Status:        order.Status,
		QRPayload:     sc.getQRURL(c, req.Channel),
		CodeIssued:    false,
	}
	response.Success(c, resp)
}

// WebhookReq simply confirms the webhook request
type WebhookReq struct {
	// Secret pre-shared key (must equal config payment.secret_key)
	Secret string `json:"secret" binding:"required"`
	// OutTradeNo Optional: directly specify the order number
	OutTradeNo string `json:"out_trade_no"`
	// Amount Optional: If order_no is not passed, the amount is passed, and the system will automatically match the latest unpaid order (yuan, such as "10.00")
	Amount string `json:"amount"`
}

// Webhook SmsForwarder and other monitoring tools direct callback interface
// Two calling methods:
//  1. Pass order_no → directly confirm the order
//  2. Only pass amount → match the latest pending orders by amount
func (sc *SubscribeController) Webhook(c *gin.Context) {
	req := &WebhookReq{}
	if err := c.ShouldBindJSON(req); err != nil {
		c.String(http.StatusOK, "fail")
		return
	}

	sk := global.Config.Payment.SecretKey
	if sk == "" || req.Secret != sk {
		global.Logger.Warnf("webhook secret mismatch: got %q", req.Secret)
		c.String(http.StatusOK, "fail")
		return
	}

	var outTradeNo string

	if req.OutTradeNo != "" {
		// Method 1: Confirm directly by order number
		outTradeNo = req.OutTradeNo
	} else if req.Amount != "" {
		// Method 2: Match by amount
		no, err := service.AllService.SubscribeService.MatchOrderByAmount(req.Amount)
		if err != nil {
			global.Logger.Warnf("webhook match amount %s failed: %v", req.Amount, err)
			c.String(http.StatusOK, "fail")
			return
		}
		outTradeNo = no
	} else {
		c.String(http.StatusOK, "fail")
		return
	}

	// Construct signed parameters and pass them to HandleNotify
	params := map[string]string{
		"out_trade_no": outTradeNo,
		"trade_status": "TRADE_SUCCESS",
		"money":        "0",
	}
	params["sign"] = payverify.Sign(params, sk)

	ok, err := service.AllService.SubscribeService.HandleNotify(params)
	if err != nil {
		global.Logger.Warnf("webhook failed: %v", err)
		c.String(http.StatusOK, "fail")
		return
	}
	if ok {
		c.String(http.StatusOK, "success")
	} else {
		c.String(http.StatusOK, "fail")
	}
}

// SmsForwarderReq SmsForwarder native webhook callback format
// Template: {"pid":"1001","aid":"46","uid":"{{UID}}","title":"{{TITLE}}","msg":"{{MSG}}","time":"
type SmsForwarderReq struct {
	PID    string `json:"pid"`
	AID    string `json:"aid"`
	UID    string `json:"uid"`
	Title  string `json:"title"` // Sender, such as "Alipay"
	Msg    string `json:"msg"`   // Content, such as "Alipay received 10.00 yuan"
	Time   string `json:"time"`
	Device string `json:"divice"`
}

// SmsWebhook receives the SmsForwarder native format callback and automatically extracts the amount to match the order.
// Configuration: URL=http://host/api/subscribe/sms-webhook?secret=your-key
func (sc *SubscribeController) SmsWebhook(c *gin.Context) {
	secret := c.Query("secret")
	sk := global.Config.Payment.SecretKey
	if sk == "" || secret != sk {
		global.Logger.Warnf("sms-webhook secret mismatch")
		c.String(http.StatusOK, "fail")
		return
	}

	req := &SmsForwarderReq{}
	if err := c.ShouldBindJSON(req); err != nil || req.Msg == "" {
		c.String(http.StatusOK, "fail")
		return
	}

	amount, err := service.AllService.SubscribeService.ExtractAmountFromSMS(req.Msg)
	if err != nil {
		global.Logger.Warnf("sms-webhook extract amount failed: msg=%q, err=%v", req.Msg, err)
		c.String(http.StatusOK, "fail")
		return
	}

	no, err := service.AllService.SubscribeService.MatchOrderByAmount(amount)
	if err != nil {
		global.Logger.Warnf("sms-webhook match failed: amount=%s, msg=%q, err=%v", amount, req.Msg, err)
		c.String(http.StatusOK, "fail")
		return
	}

	params := map[string]string{
		"out_trade_no": no,
		"trade_status": "TRADE_SUCCESS",
		"money":        amount,
	}
	params["sign"] = payverify.Sign(params, sk)

	ok, err := service.AllService.SubscribeService.HandleNotify(params)
	if err != nil {
		global.Logger.Warnf("sms-webhook notify failed: %v", err)
		c.String(http.StatusOK, "fail")
		return
	}
	if ok {
		global.Logger.Infof("sms-webhook success: order=%s, amount=%s, from=%s", no, amount, req.Title)
		c.String(http.StatusOK, "success")
	} else {
		c.String(http.StatusOK, "fail")
	}
}

// Notify payment callback notification (public interface, no authentication)
func (sc *SubscribeController) Notify(c *gin.Context) {
	params := make(map[string]string)
	c.Request.ParseForm()
	for k, v := range c.Request.Form {
		if len(v) > 0 {
			params[k] = v[0]
		}
	}
	if len(params) == 0 {
		body := make(map[string]interface{})
		if err := c.ShouldBindJSON(&body); err == nil {
			for k, v := range body {
				params[k] = fmt.Sprintf("%v", v)
			}
		}
	}

	ok, err := service.AllService.SubscribeService.HandleNotify(params)
	if err != nil {
		global.Logger.Warnf("notify failed: %v", err)
		c.String(http.StatusOK, "fail")
		return
	}
	if ok {
		c.String(http.StatusOK, "success")
	} else {
		c.String(http.StatusOK, "fail")
	}
}

// QueryOrder Query order status
func (sc *SubscribeController) QueryOrder(c *gin.Context) {
	outTradeNo := c.Param("out_trade_no")
	if outTradeNo == "" {
		response.Fail(c, 400, response.TranslateMsg(c, "MissingOutTradeNo"))
		return
	}
	user := service.AllService.UserService.CurUser(c)
	if user.Id == 0 {
		response.Fail(c, 403, response.TranslateMsg(c, "NeedLogin"))
		return
	}

	order, err := service.AllService.SubscribeService.QueryOrder(outTradeNo, user.Id)
	if err != nil {
		response.Fail(c, 4101, response.TranslateMsg(c, "OrderNotFound"))
		return
	}

	ics := &service.InviteCodeService{}
	ic, err := ics.InfoByOrderID(outTradeNo)
	codeIssued := false
	switch {
	case err == nil:
		codeIssued = true
	case errors.Is(err, service.ErrInviteCodeNotFound):
		codeIssued = false
	default:
		response.ServerError(c)
		return
	}

	resp := &respApi.OrderResp{
		OutTradeNo:  order.OutTradeNo,
		Status:      order.Status,
		Plan:        order.Plan,
		PlanKey:     order.PlanKey,
		PeriodDays:  order.PeriodDays,
		AmountCents: order.AmountCents,
		PaidAt:      order.PaidAt,
		CodeIssued:  codeIssued,
	}
	if codeIssued {
		resp.InviteCode = ic.Code
		resp.ExpireAt = &ic.ExpireAt
	}
	response.Success(c, resp)
}

// Claim order number to claim invitation code (cover the bottom line)
func (sc *SubscribeController) Claim(c *gin.Context) {
	req := &api.ClaimReq{}
	if err := c.ShouldBindJSON(req); err != nil {
		response.Fail(c, 400, response.TranslateMsg(c, "ParamError"))
		return
	}
	user := service.AllService.UserService.CurUser(c)
	if user.Id == 0 {
		response.Fail(c, 403, response.TranslateMsg(c, "NeedLogin"))
		return
	}

	ic, err := service.AllService.SubscribeService.ClaimCode(user.Id, req.OutTradeNo)
	if err != nil {
		errMsg := err.Error()
		switch {
		case contains(errMsg, "ORDER_NOT_FOUND"):
			response.Fail(c, 4101, response.TranslateMsg(c, "OrderNotFound"))
		case contains(errMsg, "ORDER_NOT_PAID"):
			response.Fail(c, 4102, response.TranslateMsg(c, "OrderNotPaid"))
		default:
			response.ServerError(c)
		}
		return
	}

	resp := &respApi.ClaimResp{
		Code:     ic.Code,
		ExpireAt: ic.ExpireAt,
		Plan:     ic.Plan,
	}
	response.Success(c, resp)
}

// Redeem redeem invitation code
func (sc *SubscribeController) Redeem(c *gin.Context) {
	req := &api.RedeemReq{}
	if err := c.ShouldBindJSON(req); err != nil {
		response.Fail(c, 400, response.TranslateMsg(c, "ParamError"))
		return
	}
	user := service.AllService.UserService.CurUser(c)
	if user.Id == 0 {
		response.Fail(c, 403, response.TranslateMsg(c, "NeedLogin"))
		return
	}

	ic, err := service.AllService.SubscribeService.RedeemCode(user.Id, req.Code)
	if err != nil {
		errMsg := err.Error()
		switch {
		case errors.Is(err, service.ErrInviteCodeNotFound):
			response.Fail(c, 4201, response.TranslateMsg(c, "CodeNotFound"))
		case errors.Is(err, service.ErrInviteCodeAlreadyConsumed):
			response.Fail(c, 4202, response.TranslateMsg(c, "CodeUsed"))
		case contains(errMsg, "code revoked"):
			response.Fail(c, 4203, response.TranslateMsg(c, "CodeRevoked"))
		case contains(errMsg, "code expired"):
			response.Fail(c, 4204, response.TranslateMsg(c, "CodeExpired"))
		default:
			response.ServerError(c)
		}
		return
	}

	user = service.AllService.UserService.InfoById(user.Id)
	resp := &respApi.RedeemResp{
		Plan:                 ic.Plan,
		ExpireAt:             ic.ExpireAt,
		SubscriptionExpireAt: user.SubscriptionExpireAt,
	}
	response.Success(c, resp)
}

// Mine gets the current user’s subscription information
func (sc *SubscribeController) Mine(c *gin.Context) {
	user := service.AllService.UserService.CurUser(c)
	if user.Id == 0 {
		response.Fail(c, 403, response.TranslateMsg(c, "NeedLogin"))
		return
	}

	u, err := service.AllService.SubscribeService.GetMine(user.Id)
	if err != nil {
		response.Fail(c, 500, err.Error())
		return
	}

	status := u.SubscriptionStatus()
	daysLeft := u.SubscriptionDaysLeft()
	isExpiringSoon := false
	if status == "active" && daysLeft > 0 && daysLeft <= 7 {
		isExpiringSoon = true
	}

	resp := &respApi.MineResp{
		Plan:                 u.SubscriptionPlan,
		SubscriptionExpireAt: u.SubscriptionExpireAt,
		Status:               status,
		DaysLeft:             daysLeft,
		IsExpiringSoon:       isExpiringSoon,
	}
	response.Success(c, resp)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || contains(s[1:], substr)))
}
