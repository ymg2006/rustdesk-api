package admin

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ymg2006/rustdesk-api/v2/global"
	"github.com/ymg2006/rustdesk-api/v2/http/response"
	"github.com/ymg2006/rustdesk-api/v2/lib/payverify"
	"github.com/ymg2006/rustdesk-api/v2/model"
	"github.com/ymg2006/rustdesk-api/v2/service"
)

// OrderCtl background order management
type OrderCtl struct{}

// NewOrderCtl creates a controller
func NewOrderCtl() *OrderCtl {
	return &OrderCtl{}
}

// List order list
func (oc *OrderCtl) List(c *gin.Context) {
	status := c.Query("status")   // pending / paid / closed
	keyword := c.Query("keyword") // Search by order number or username
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

	db := service.AllService.SubscribeService.Db().Model(&model.PayOrder{})

	// status filter
	if status != "" {
		db = db.Where("status = ?", status)
	}

	// keyword search
	if keyword != "" {
		db = db.Where("out_trade_no LIKE ? OR user_id IN (SELECT id FROM users WHERE username LIKE ?)",
			"%"+keyword+"%", "%"+keyword+"%")
	}

	var total int64
	db.Count(&total)

	var orders []model.PayOrder
	db.Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&orders)

	type orderItem struct {
		ID          uint       `json:"id"`
		OutTradeNo  string     `json:"out_trade_no"`
		UserID      uint       `json:"user_id"`
		Username    string     `json:"username"`
		Plan        string     `json:"plan"`
		PlanKey     string     `json:"plan_key"`
		PeriodDays  int        `json:"period_days"`
		AmountCents int64      `json:"amount_cents"`
		Channel     string     `json:"channel"`
		Status      string     `json:"status"`
		PaidAt      *time.Time `json:"paid_at"`
		CreatedAt   time.Time  `json:"created_at"`
	}

	items := make([]orderItem, 0, len(orders))
	for _, o := range orders {
		username := ""
		u := service.AllService.UserService.InfoById(o.UserID)
		if u != nil {
			username = u.Username
		}
		items = append(items, orderItem{
			ID:          o.Id,
			OutTradeNo:  o.OutTradeNo,
			UserID:      o.UserID,
			Username:    username,
			Plan:        o.Plan,
			PlanKey:     o.PlanKey,
			PeriodDays:  o.PeriodDays,
			AmountCents: o.AmountCents,
			Channel:     o.Channel,
			Status:      o.Status,
			PaidAt:      o.PaidAt,
			CreatedAt:   o.CreatedAt,
		})
	}

	response.Success(c, map[string]interface{}{
		"list":     items,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// Detail order details
func (oc *OrderCtl) Detail(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.ParseUint(idStr, 10, 64)
	if id == 0 {
		response.Fail(c, 400, response.TranslateMsg(c, "InvalidId"))
		return
	}

	order := &model.PayOrder{}
	if err := service.AllService.SubscribeService.Db().First(order, id).Error; err != nil {
		response.Fail(c, 404, "order not found")
		return
	}

	response.Success(c, order)
}

// Confirm Manually confirm the arrival (mark the pending order as paid)
func (oc *OrderCtl) Confirm(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.ParseUint(idStr, 10, 64)
	if id == 0 {
		response.Fail(c, 400, response.TranslateMsg(c, "InvalidId"))
		return
	}

	order := &model.PayOrder{}
	if err := service.AllService.SubscribeService.Db().First(order, id).Error; err != nil {
		response.Fail(c, 404, "order not found")
		return
	}

	if order.Status != "pending" {
		response.Fail(c, 400, fmt.Sprintf("order status is %s, cannot confirm", order.Status))
		return
	}

	// Construct callback parameters to call HandleNotify (bring signature, otherwise signature verification fails)
	params := map[string]string{
		"out_trade_no": order.OutTradeNo,
		"trade_status": "TRADE_SUCCESS",
		"money":        fmt.Sprintf("%.2f", float64(order.AmountCents)/100),
	}
	if sk := global.Config.Payment.SecretKey; sk != "" {
		params["sign"] = payverify.Sign(params, sk)
	}
	ok, err := service.AllService.SubscribeService.HandleNotify(params)
	if err != nil || !ok {
		response.Fail(c, 500, fmt.Sprintf("confirm failed: %v", err))
		return
	}

	response.Success(c, nil)
}

// Close close order
func (oc *OrderCtl) Close(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.ParseUint(idStr, 10, 64)
	if id == 0 {
		response.Fail(c, 400, response.TranslateMsg(c, "InvalidId"))
		return
	}

	db := service.AllService.SubscribeService.Db()
	order := &model.PayOrder{}
	if err := db.First(order, id).Error; err != nil {
		response.Fail(c, 404, "order not found")
		return
	}

	if order.Status != "pending" {
		response.Fail(c, 400, fmt.Sprintf("order status is %s, cannot close", order.Status))
		return
	}

	if err := db.Model(order).Update("status", "closed").Error; err != nil {
		response.Fail(c, 500, err.Error())
		return
	}

	response.Success(c, nil)
}
