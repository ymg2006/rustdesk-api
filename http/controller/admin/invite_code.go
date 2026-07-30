package admin

import (
	"encoding/csv"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ymg2006/rustdesk-api/v2/http/request/api"
	"github.com/ymg2006/rustdesk-api/v2/http/response"
	respApi "github.com/ymg2006/rustdesk-api/v2/http/response/api"
	"github.com/ymg2006/rustdesk-api/v2/service"
)

// AdminInviteCodeController background invitation code management
type AdminInviteCodeController struct {
}

// NewAdminInviteCodeController creates a controller
func NewAdminInviteCodeController() *AdminInviteCodeController {
	return &AdminInviteCodeController{}
}

// List Query invitation code list by page
func (ac *AdminInviteCodeController) List(c *gin.Context) {
	status := c.Query("status")
	plan := c.Query("plan")
	usedByStr := c.Query("used_by")
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
	var usedBy uint
	if usedByStr != "" {
		uid, err := strconv.ParseUint(usedByStr, 10, 64)
		if err == nil {
			usedBy = uint(uid)
		}
	}

	filter := service.InviteCodeFilter{
		Status:   status,
		Plan:     plan,
		UsedBy:   usedBy,
		Page:     page,
		PageSize: pageSize,
	}

	ics := &service.InviteCodeService{}
	list, total, err := ics.List(filter)
	if err != nil {
		response.Fail(c, 500, err.Error())
		return
	}

	// Collect all user IDs that have used authorization codes and check user names in batches
	userIdSet := make(map[uint]bool)
	for _, ic := range list {
		if ic.UsedBy > 0 {
			userIdSet[ic.UsedBy] = true
		}
	}
	userNameMap := make(map[uint]string)
	if len(userIdSet) > 0 {
		us := &service.UserService{}
		for uid := range userIdSet {
			u := us.InfoById(uid)
			if u != nil && u.Id > 0 {
				userNameMap[uid] = u.Username
			}
		}
	}

	items := make([]respApi.CodeListItem, 0, len(list))
	for _, ic := range list {
		usedByName := ""
		if ic.UsedBy > 0 {
			usedByName = userNameMap[ic.UsedBy]
		}
		items = append(items, respApi.CodeListItem{
			ID:           ic.Id,
			Code:         ic.Code,
			Plan:         ic.Plan,
			Status:       ic.Status,
			UsedBy:       ic.UsedBy,
			UsedByName:   usedByName,
			ExpireAt:     ic.ExpireAt,
			BoundOrderID: ic.BoundOrderID,
			CreatedAt:    ic.CreatedAt,
		})
	}

	response.Success(c, response.PageData{
		Page:  page,
		Total: int(total),
		List:  items,
	})
}

// Create manually generate invitation code
func (ac *AdminInviteCodeController) Create(c *gin.Context) {
	req := &api.AdminCreateCodeReq{}
	if err := c.ShouldBindJSON(req); err != nil {
		response.Fail(c, 400, response.TranslateMsg(c, "ParamError"))
		return
	}

	if req.Plan == "" {
		req.Plan = "pro"
	}
	if req.ExpireDays <= 0 {
		req.ExpireDays = 30
	}

	ics := &service.InviteCodeService{}
	ic, err := ics.Generate(req.Plan, 0, "", req.ExpireDays)
	if err != nil {
		response.Fail(c, 500, err.Error())
		return
	}

	response.Success(c, gin.H{
		"code":      ic.Code,
		"plan":      ic.Plan,
		"expire_at": ic.ExpireAt,
	})
}

// Revoke invalid invitation code
func (ac *AdminInviteCodeController) Revoke(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		response.Fail(c, 400, response.TranslateMsg(c, "InvalidId"))
		return
	}

	ics := &service.InviteCodeService{}
	if err := ics.Revoke(uint(id)); err != nil {
		response.Fail(c, 500, err.Error())
		return
	}
	response.Success(c, nil)
}

// Delete permanently deletes an unused or revoked invitation code.
func (ac *AdminInviteCodeController) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.Fail(c, 400, response.TranslateMsg(c, "InvalidId"))
		return
	}

	ics := &service.InviteCodeService{}
	if err := ics.Delete(uint(id)); err != nil {
		switch {
		case errors.Is(err, service.ErrInviteCodeNotFound):
			response.Fail(c, 404, response.TranslateMsg(c, "CodeNotFound"))
		case errors.Is(err, service.ErrInviteCodeUsed):
			response.Fail(c, 409, err.Error())
		default:
			response.Fail(c, 500, err.Error())
		}
		return
	}
	response.Success(c, nil)
}

// Export export invitation code CSV (for reconciliation)
func (ac *AdminInviteCodeController) Export(c *gin.Context) {
	status := c.Query("status")
	plan := c.Query("plan")

	filter := service.InviteCodeFilter{
		Status:   status,
		Plan:     plan,
		Page:     1,
		PageSize: 100000, // One-time export
	}

	ics := &service.InviteCodeService{}
	list, _, err := ics.List(filter)
	if err != nil {
		response.Fail(c, 500, err.Error())
		return
	}

	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=invite_codes_%s.csv", time.Now().Format("20060102150405")))

	// Write BOM to make Excel correctly recognize UTF-8
	c.Writer.Write([]byte{0xEF, 0xBB, 0xBF})

	// Check usernames in batches
	userIdSet := make(map[uint]bool)
	for _, ic := range list {
		if ic.UsedBy > 0 {
			userIdSet[ic.UsedBy] = true
		}
	}
	userNameMap := make(map[uint]string)
	if len(userIdSet) > 0 {
		us := &service.UserService{}
		for uid := range userIdSet {
			u := us.InfoById(uid)
			if u != nil && u.Id > 0 {
				userNameMap[uid] = u.Username
			}
		}
	}

	writer := csv.NewWriter(c.Writer)
	writer.Write([]string{"ID", "Code", "Plan", "Status", "UsedBy", "UsedByName", "BoundOrderID", "ExpireAt", "CreatedAt"})

	for _, ic := range list {
		usedByName := ""
		if ic.UsedBy > 0 {
			usedByName = userNameMap[ic.UsedBy]
		}
		writer.Write([]string{
			strconv.Itoa(int(ic.Id)),
			ic.Code,
			ic.Plan,
			ic.Status,
			strconv.Itoa(int(ic.UsedBy)),
			usedByName,
			ic.BoundOrderID,
			ic.ExpireAt.Format("2006-01-02 15:04:05"),
			ic.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	writer.Flush()
}
