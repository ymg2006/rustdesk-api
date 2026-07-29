package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ymg2006/rustdesk-api/v2/lib/payverify"
	"github.com/ymg2006/rustdesk-api/v2/model"
	"gorm.io/gorm"
)

// SubscribeService Subscription paid service
type SubscribeService struct{}

// NewSubscribeService creates SubscribeService
func NewSubscribeService() *SubscribeService {
	return &SubscribeService{}
}

// Db returns the DB instance
func (s *SubscribeService) Db() *gorm.DB {
	return DB
}

// generateOutTradeNo generates merchant order number
// Format: SUB + YYYYMMDD + 12hex = about 24 characters
func (s *SubscribeService) generateOutTradeNo() string {
	datePart := time.Now().Format("20060102")
	b := make([]byte, 6)
	rand.Read(b)
	return "SUB" + datePart + hex.EncodeToString(b)
}

// PayConfig returns a shortcut reference to the payment configuration
func (s *SubscribeService) PayConfig() (secretKey string, expireSec int) {
	pc := Config.Payment
	return pc.SecretKey, pc.OrderExpireSec
}

// ExtractAmountFromSMS Extracts the payment amount (yuan) from the text message content
// Supported formats: "10.00 yuan received", "Receive 10 yuan", "10.5 yuan received", etc.
func (s *SubscribeService) ExtractAmountFromSMS(msg string) (string, error) {
	// Matches the pattern "Arrival/receipt+ number. number + yuan"
	re := regexp.MustCompile(`(?:credited|payment received|deposited|received)[:\s]*(\d+\.?\d*)\s*(?:yuan|CNY)`)
	matches := re.FindStringSubmatch(msg)
	if len(matches) < 2 {
		return "", fmt.Errorf("cannot extract amount from SMS: %s", msg)
	}
	amount := matches[1]
	// Complete decimal places
	if !strings.Contains(amount, ".") {
		amount += ".00"
	}
	return amount, nil
}

// MatchOrderByAmount matches the latest unpaid orders by amount (SmsForwarder callback by amount)
// amount: string amount, supports "10.00" or "10" format
// Returns the matched out_trade_no
func (s *SubscribeService) MatchOrderByAmount(amount string) (string, error) {
	// Analyze amount to cents
	amount = strings.TrimSpace(amount)
	var amountCents int64
	if idx := strings.Index(amount, "."); idx >= 0 {
		intPart := amount[:idx]
		decPart := amount[idx+1:]
		if len(decPart) > 2 {
			decPart = decPart[:2]
		}
		for len(decPart) < 2 {
			decPart += "0"
		}
		ai, _ := strconv.ParseInt(intPart, 10, 64)
		ad, _ := strconv.ParseInt(decPart, 10, 64)
		amountCents = ai*100 + ad
	} else {
		ai, _ := strconv.ParseInt(amount, 10, 64)
		amountCents = ai * 100
	}
	if amountCents <= 0 {
		return "", fmt.Errorf("invalid amount: %s", amount)
	}

	// Amount matching window: only match unpaid orders created within 5 minutes
	since := time.Now().Add(-5 * time.Minute)

	order := &model.PayOrder{}
	err := s.Db().
		Where("amount_cents = ? AND status = 'pending' AND created_at >= ?", amountCents, since).
		Order("created_at DESC").
		First(order).Error
	if err != nil {
		return "", fmt.Errorf("no pending order matches amount %s", amount)
	}
	return order.OutTradeNo, nil
}

// CreateOrder creates a subscription order
// channel: alipay / wechat, planKey: 1m / 3m / 6m / 12m
func (s *SubscribeService) CreateOrder(userID uint, channel, planKey string) (*model.PayOrder, error) {
	if channel != "wechat" && channel != "alipay" {
		return nil, fmt.Errorf("unsupported channel: %s", channel)
	}

	// Check duration options
	opt := Config.Subscription.LookupPlan(planKey)
	if opt == nil {
		return nil, fmt.Errorf("invalid plan_key: %s", planKey)
	}
	priceCents := opt.PriceCents
	periodDays := opt.PeriodDays
	if priceCents <= 0 || periodDays <= 0 {
		return nil, fmt.Errorf("invalid plan config for key %s", planKey)
	}

	expireSec := Config.Payment.OrderExpireSec
	if expireSec <= 0 {
		expireSec = 600
	}

	order := &model.PayOrder{
		OutTradeNo:  s.generateOutTradeNo(),
		UserID:      userID,
		Plan:        Config.Subscription.Plan,
		PlanKey:     planKey,
		AmountCents: priceCents,
		Channel:     channel,
		Status:      "pending",
		PeriodDays:  periodDays,
	}

	if err := s.Db().Create(order).Error; err != nil {
		return nil, fmt.Errorf("create order: %w", err)
	}
	return order, nil
}

// NotifyConfig notification signature parameters
type NotifyConfig struct {
	SecretKey string
	PID       string // Merchant ID, fixed when self-fulfilling
}

// BuildNotifyParams build callback notification parameter string (including sign)
// Standard format for code payment: pid, trade_no, out_trade_no, type, name, money, trade_status, sign
func (s *SubscribeService) BuildNotifyParams(order *model.PayOrder) map[string]string {
	params := map[string]string{
		"pid":          "1000",
		"trade_no":     order.OutTradeNo,
		"out_trade_no": order.OutTradeNo,
		"type":         order.Channel,
		"name":         order.Plan + "subscription",
		"money":        fmt.Sprintf("%.2f", float64(order.AmountCents)/100),
		"trade_status": "TRADE_SUCCESS",
	}
	// Sign only if secret_key is available
	sk, _ := s.PayConfig()
	if sk != "" {
		params["sign"] = payverify.Sign(params, sk)
	}
	return params
}

// HandleNotify handles payment callback notifications
// params: callback parameters (including sign)
// Returning true indicates that the processing is successful, false indicates that the signature verification fails and needs to be ignored.
func (s *SubscribeService) HandleNotify(params map[string]string) (bool, error) {
	secretKey, _ := s.PayConfig()
	if secretKey != "" {
		if !payverify.Verify(params, secretKey) {
			Logger.Warnf("codepay notify sign verify failed: %+v", params)
			return false, fmt.Errorf("sign verify failed")
		}
	} else {
		Logger.Warn("codepay secret_key not configured, skip sign verification")
	}

	// Take out_trade_no
	outTradeNo := params["out_trade_no"]
	if outTradeNo == "" {
		return false, fmt.Errorf("out_trade_no empty")
	}

	// Only handle TRADE_SUCCESS
	if params["trade_status"] != "TRADE_SUCCESS" {
		Logger.Infof("notify ignored: status=%s, out_trade_no=%s", params["trade_status"], outTradeNo)
		return true, nil
	}

	// Transaction: Idempotent order modification → Generate invitation code → Activate subscription
	now := time.Now()
	err := s.Db().Transaction(func(tx *gorm.DB) error {
		order := &model.PayOrder{}
		if err := tx.Where("out_trade_no = ?", outTradeNo).First(order).Error; err != nil {
			return fmt.Errorf("order not found: %s", outTradeNo)
		}

		// Idempotent
		if order.Status == "paid" {
			Logger.Infof("notify idempotent: order %s already paid, skip", outTradeNo)
			return nil
		}
		if order.Status != "pending" {
			return fmt.Errorf("order %s status is %s, cannot paid", outTradeNo, order.Status)
		}

		// Change order
		if err := tx.Model(order).Updates(map[string]interface{}{
			"status":       "paid",
			"paid_at":      &now,
			"callback_raw": fmt.Sprintf("%v", params),
		}).Error; err != nil {
			return err
		}
		order.Status = "paid"
		order.PaidAt = &now

		// Generate invitation code and activate
		periodDays := order.PeriodDays
		if periodDays <= 0 {
			periodDays = 30
		}
		plan := order.Plan
		if plan == "" {
			plan = "pro"
		}

		ics := &InviteCodeService{}
		ic, err := ics.GenerateWithDB(tx, plan, order.UserID, outTradeNo, periodDays)
		if err != nil {
			return fmt.Errorf("generate code: %w", err)
		}

		// Activate subscription (postponed)
		user := &model.User{}
		if err := tx.Where("id = ?", order.UserID).First(user).Error; err != nil {
			return err
		}
		periodDuration := time.Duration(periodDays*24) * time.Hour
		var newExpire time.Time
		if user.SubscriptionExpireAt == nil || user.SubscriptionExpireAt.Before(now) {
			newExpire = now.Add(periodDuration)
		} else {
			newExpire = user.SubscriptionExpireAt.Add(periodDuration)
		}
		// Synchronously update expired_at so that the client cannot log in when the membership expires.
		expiredAt := newExpire.Unix()
		if err := tx.Model(&model.User{}).Where("id = ?", order.UserID).
			Updates(map[string]interface{}{
				"subscription_plan":      plan,
				"subscription_expire_at": &newExpire,
				"expired_at":             expiredAt,
			}).Error; err != nil {
			return err
		}

		// Update invitation code status
		icTime := now
		if err := tx.Model(&model.InviteCode{}).Where("id = ?", ic.Id).
			Updates(map[string]interface{}{
				"status":    "used",
				"used_by":   order.UserID,
				"used_at":   &icTime,
				"expire_at": newExpire,
			}).Error; err != nil {
			return err
		}

		Logger.Infof("subscribe activated: user=%d, order=%s, plan=%s, expire=%s",
			order.UserID, outTradeNo, plan, newExpire.Format(time.RFC3339))
		return nil
	})
	if err != nil {
		return false, err
	}
	return true, nil
}

// QueryOrder Query order (visible only to the user who owns the order)
func (s *SubscribeService) QueryOrder(outTradeNo string, userID uint) (*model.PayOrder, error) {
	order := &model.PayOrder{}
	if err := s.Db().Where("out_trade_no = ?", outTradeNo).First(order).Error; err != nil {
		return nil, fmt.Errorf("order not found")
	}
	if order.UserID != userID {
		return nil, fmt.Errorf("order not found")
	}
	return order, nil
}

// ClaimCode order number to claim the invitation code (the code has been paid but not received)
func (s *SubscribeService) ClaimCode(userID uint, outTradeNo string) (*model.InviteCode, error) {
	order := &model.PayOrder{}
	if err := s.Db().Where("out_trade_no = ? AND user_id = ?", outTradeNo, userID).First(order).Error; err != nil {
		return nil, fmt.Errorf("ORDER_NOT_FOUND")
	}
	if order.Status != "paid" {
		return nil, fmt.Errorf("ORDER_NOT_PAID")
	}

	ics := &InviteCodeService{}
	existing := ics.InfoByOrderID(outTradeNo)
	if existing != nil && existing.Id > 0 {
		return existing, nil
	}

	periodDays := order.PeriodDays
	if periodDays <= 0 {
		periodDays = 30
	}
	plan := order.Plan
	if plan == "" {
		plan = "pro"
	}

	ic, err := ics.Generate(plan, userID, outTradeNo, periodDays)
	if err != nil {
		return nil, fmt.Errorf("generate code: %w", err)
	}
	_, err = ics.Activate(ic.Code, userID)
	if err != nil {
		return nil, fmt.Errorf("activate code: %w", err)
	}
	return ic, nil
}

// RedeemCode users redeem invitation codes
func (s *SubscribeService) RedeemCode(userID uint, codeStr string) (*model.InviteCode, error) {
	ics := &InviteCodeService{}
	ic, err := ics.Activate(codeStr, userID)
	if err != nil {
		return nil, err
	}
	return ic, nil
}

// GetMine gets current user subscription information
func (s *SubscribeService) GetMine(userID uint) (*model.User, error) {
	user := &model.User{}
	if err := s.Db().Where("id = ?", userID).First(user).Error; err != nil {
		return nil, fmt.Errorf("user not found")
	}
	return user, nil
}
