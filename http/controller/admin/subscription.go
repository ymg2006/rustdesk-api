package admin

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ymg2006/rustdesk-api/v2/global"
	"github.com/ymg2006/rustdesk-api/v2/http/response"
	"github.com/ymg2006/rustdesk-api/v2/model"
	"github.com/ymg2006/rustdesk-api/v2/service"
)

// SubscriptionCtl background subscription management
type SubscriptionCtl struct{}

// NewSubscriptionCtl creates a controller
func NewSubscriptionCtl() *SubscriptionCtl {
	return &SubscriptionCtl{}
}

// List Subscriber list
func (sc *SubscriptionCtl) List(c *gin.Context) {
	status := c.Query("status")   // active / expired / none
	keyword := c.Query("keyword") // Search by username or ID
	pageStr := c.Query("page")
	pageSizeStr := c.Query("size")

	page, _ := strconv.Atoi(pageStr)
	if page <= 0 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(pageSizeStr)
	if pageSize <= 0 {
		pageSize = 20
	}

	db := global.DB.Model(&model.User{})

	// Subscription status filter
	now := time.Now()
	switch status {
	case "active":
		db = db.Where("subscription_expire_at IS NOT NULL AND subscription_expire_at > ? AND subscription_expire_at < ?", now, time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC))
	case "expired":
		db = db.Where("subscription_expire_at IS NOT NULL AND subscription_expire_at <= ?", now)
	case "none":
		db = db.Where("subscription_expire_at IS NULL")
	case "permanent":
		db = db.Where("subscription_expire_at IS NOT NULL AND subscription_expire_at >= ?", time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC))
	}

	// keyword search
	if keyword != "" {
		db = db.Where("id = ? OR username LIKE ?", parseUint(keyword), "%"+keyword+"%")
	}

	var total int64
	db.Count(&total)

	var users []model.User
	db.Order("id ASC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&users)

	type subItem struct {
		ID                   uint       `json:"id"`
		Username             string     `json:"username"`
		SubscriptionPlan     string     `json:"subscription_plan"`
		SubscriptionExpireAt *time.Time `json:"subscription_expire_at"`
		Status               string     `json:"status"`
		DaysLeft             int        `json:"days_left"`
	}

	items := make([]subItem, 0, len(users))
	for _, u := range users {
		items = append(items, subItem{
			ID:                   u.Id,
			Username:             u.Username,
			SubscriptionPlan:     u.SubscriptionPlan,
			SubscriptionExpireAt: u.SubscriptionExpireAt,
			Status:               u.SubscriptionStatus(),
			DaysLeft:             u.SubscriptionDaysLeft(),
		})
	}

	response.Success(c, map[string]interface{}{
		"list":     items,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// ExtendReq Extend membership request
type ExtendReq struct {
	UserID  uint   `json:"user_id" binding:"required"`
	Plan    string `json:"plan"`                        // Package ID, default "pro"
	PlanKey string `json:"plan_key" binding:"required"` // Duration key: 1m / 3m / 6m / 12m / forever
}

// Extend Extend user membership (paid monthly)
func (sc *SubscriptionCtl) Extend(c *gin.Context) {
	req := &ExtendReq{}
	if err := c.ShouldBindJSON(req); err != nil {
		response.Fail(c, 400, "param error")
		return
	}

	if req.Plan == "" {
		req.Plan = "pro"
	}

	user := service.AllService.UserService.InfoById(req.UserID)
	if user.Id == 0 {
		response.Fail(c, 404, "user not found")
		return
	}

	now := time.Now()
	var newExpire time.Time

	// Special treatment for permanent packages: set to 9999
	if req.PlanKey == "forever" {
		newExpire = time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC)
		// Permanent membership: expired_at is set to 0 (never expires)
		if err := global.DB.Model(&model.User{}).Where("id = ?", req.UserID).
			Updates(map[string]interface{}{
				"subscription_plan":      req.Plan + "-forever",
				"subscription_expire_at": &newExpire,
				"expired_at":             0,
			}).Error; err != nil {
			response.Fail(c, 500, err.Error())
			return
		}
		response.Success(c, map[string]interface{}{
			"user_id":                req.UserID,
			"plan":                   req.Plan + "-forever",
			"subscription_expire_at": &newExpire,
		})
		return
	}

	// Check the configuration to obtain period_days
	opt := global.Config.Subscription.LookupPlan(req.PlanKey)
	if opt == nil || opt.PeriodDays <= 0 {
		response.Fail(c, 400, "invalid plan_key")
		return
	}

	periodDuration := time.Duration(opt.PeriodDays*24) * time.Hour
	if user.SubscriptionExpireAt == nil || user.SubscriptionExpireAt.Before(now) {
		newExpire = now.Add(periodDuration)
	} else {
		newExpire = user.SubscriptionExpireAt.Add(periodDuration)
	}

	expiredAt := newExpire.Unix()
	if err := global.DB.Model(&model.User{}).Where("id = ?", req.UserID).
		Updates(map[string]interface{}{
			"subscription_plan":      req.Plan,
			"subscription_expire_at": &newExpire,
			"expired_at":             expiredAt,
		}).Error; err != nil {
		response.Fail(c, 500, err.Error())
		return
	}

	response.Success(c, map[string]interface{}{
		"user_id":                req.UserID,
		"plan":                   req.Plan,
		"subscription_expire_at": &newExpire,
	})
}

func parseUint(s string) uint {
	n, _ := strconv.ParseUint(s, 10, 64)
	return uint(n)
}
