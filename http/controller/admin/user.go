package admin

import (
	"encoding/base64"
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pquerna/otp/totp"
	"github.com/skip2/go-qrcode"
	"github.com/ymg2006/rustdesk-api/v2/global"
	"github.com/ymg2006/rustdesk-api/v2/http/request/admin"
	"github.com/ymg2006/rustdesk-api/v2/http/response"
	adResp "github.com/ymg2006/rustdesk-api/v2/http/response/admin"
	"github.com/ymg2006/rustdesk-api/v2/model"
	"github.com/ymg2006/rustdesk-api/v2/service"
	"github.com/ymg2006/rustdesk-api/v2/utils"
	"gorm.io/gorm"
	"strconv"
)

type User struct {
}

// Detail Administrator
// @Tags user
// @Summary Admin details
// @Description Administrator details
// @Accept  json
// @Produce  json
// @Param id path int true "ID"
// @Success 200 {object} response.Response{data=model.User}
// @Failure 500 {object} response.Response
// @Router /admin/user/detail/{id} [get]
// @Security token
func (ct *User) Detail(c *gin.Context) {
	id := c.Param("id")
	iid, _ := strconv.Atoi(id)
	u := service.AllService.UserService.InfoById(uint(iid))
	if u.Id > 0 {
		response.Success(c, u)
		return
	}
	response.Fail(c, 101, response.TranslateMsg(c, "ItemNotFound"))
}

// Create Administrator
// @Tags user
// @Summary Create administrator
// @Description Create administrator
// @Accept  json
// @Produce  json
// @Param body body admin.UserForm true "Administrator information"
// @Success 200 {object} response.Response{data=model.User}
// @Failure 500 {object} response.Response
// @Router /admin/user/create [post]
// @Security token
func (ct *User) Create(c *gin.Context) {
	f := &admin.UserForm{}
	if err := c.ShouldBindJSON(f); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	errList := global.Validator.ValidStruct(c, f)
	if len(errList) > 0 {
		response.Fail(c, 101, errList[0])
		return
	}
	u := f.ToUser()
	// Compatible with old fields: synchronization when is_admin=true is set but role is not passed
	if u.Role == "" && u.IsAdmin != nil && *u.IsAdmin {
		u.Role = "admin"
	} else if u.Role == "" {
		u.Role = "user"
	}
	err := service.AllService.UserService.Create(u)
	if err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "OperationFailed")+err.Error())
		return
	}
	response.Success(c, nil)
}

// List list
// @Tags user
// @Summary Admin List
// @Description Administrator list
// @Accept  json
// @Produce  json
// @Param page query int false "page number"
// @Param page_size query int false "page size"
// @Param username query int false "Account"
// @Success 200 {object} response.Response{data=model.UserList}
// @Failure 500 {object} response.Response
// @Router /admin/user/list [get]
// @Security token
func (ct *User) List(c *gin.Context) {
	query := &admin.UserQuery{}
	if err := c.ShouldBindQuery(query); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	res := service.AllService.UserService.List(query.Page, query.PageSize, func(tx *gorm.DB) {
		if query.Username != "" {
			tx.Where("username like ?", "%"+query.Username+"%")
		}
		// When filtering by department, users under all sub-departments are included.
		if query.GroupId > 0 {
			ids := service.AllService.GroupService.DescendantIds(query.GroupId)
			ids = append(ids, query.GroupId)
			tx.Where("group_id in ?", ids)
		}
	})
	response.Success(c, res)
}

// Update Edit
// @Tags user
// @Summary Admin edit
// @Description Admin edit
// @Accept  json
// @Produce  json
// @Param body body admin.UserForm true "User information"
// @Success 200 {object} response.Response{data=model.User}
// @Failure 500 {object} response.Response
// @Router /admin/user/update [post]
// @Security token
func (ct *User) Update(c *gin.Context) {
	f := &admin.UserForm{}
	if err := c.ShouldBindJSON(f); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	if f.Id == 0 {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError"))
		return
	}
	errList := global.Validator.ValidStruct(c, f)
	if len(errList) > 0 {
		response.Fail(c, 101, errList[0])
		return
	}
	u := f.ToUser()
	// Compatible with old fields: synchronization when is_admin=true is set but role is not passed
	if u.Role == "" && u.IsAdmin != nil && *u.IsAdmin {
		u.Role = "admin"
	} else if u.Role == "" {
		u.Role = "user"
	}
	err := service.AllService.UserService.Update(u)
	if err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "OperationFailed")+err.Error())
		return
	}
	response.Success(c, nil)
}

// Delete Delete
// @Tags user
// @Summary Admin deleted
// @Description Administrator edited and deleted
// @Accept  json
// @Produce  json
// @Param body body admin.UserForm true "User information"
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /admin/user/delete [post]
// @Security token
func (ct *User) Delete(c *gin.Context) {
	f := &admin.UserForm{}
	if err := c.ShouldBindJSON(f); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	id := f.Id
	errList := global.Validator.ValidVar(c, id, "required,gt=0")
	if len(errList) > 0 {
		response.Fail(c, 101, errList[0])
		return
	}
	u := service.AllService.UserService.InfoById(f.Id)
	if u.Id > 0 {
		err := service.AllService.UserService.Delete(u)
		if err == nil {
			response.Success(c, nil)
			return
		}
		response.Fail(c, 101, err.Error())
		return
	}
	response.Fail(c, 101, response.TranslateMsg(c, "ItemNotFound"))
}

// UpdatePassword Change password
// @Tags user
// @Summary Change password
// @Description change password
// @Accept  json
// @Produce  json
// @Param body body admin.UserPasswordForm true "User information"
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /admin/user/updatePassword [post]
// @Security token
func (ct *User) UpdatePassword(c *gin.Context) {
	f := &admin.UserPasswordForm{}
	if err := c.ShouldBindJSON(f); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	errList := global.Validator.ValidStruct(c, f)
	if len(errList) > 0 {
		response.Fail(c, 101, errList[0])
		return
	}
	u := service.AllService.UserService.InfoById(f.Id)
	if u.Id == 0 {
		response.Fail(c, 101, response.TranslateMsg(c, "ItemNotFound"))
		return
	}
	err := service.AllService.UserService.UpdatePassword(u, f.Password)
	if err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "OperationFailed")+err.Error())
		return
	}
	// After changing the password, clear all session tokens for the user and force a re-login to prevent old tokens from being used fraudulently.
	_ = service.AllService.UserService.FlushToken(u)
	response.Success(c, nil)
}

// Current current user
// @Tags user
// @Summary current user
// @Description current user
// @Accept  json
// @Produce  json
// @Success 200 {object} response.Response{data=admin.LoginPayload}
// @Failure 500 {object} response.Response
// @Router /admin/user/current [get]
// @Security token
func (ct *User) Current(c *gin.Context) {
	u := service.AllService.UserService.CurUser(c)
	lp := &adResp.LoginPayload{}
	lp.FromUser(u)
	lp.Token = "" // Do not expose the token to the front end (has been issued through HttpOnly Cookie)
	lp.RouteNames = service.AllService.UserService.RouteNames(u)
	response.Success(c, lp)
}

// ChangeCurPwd changes the current user password
// @Tags user
// @Summary Modify the current user password
// @Description Modify the current user password
// @Accept  json
// @Produce  json
// @Param body body admin.ChangeCurPasswordForm true "User information"
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /admin/user/changeCurPwd [post]
// @Security token
func (ct *User) ChangeCurPwd(c *gin.Context) {
	f := &admin.ChangeCurPasswordForm{}
	if err := c.ShouldBindJSON(f); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}

	errList := global.Validator.ValidStruct(c, f)
	if len(errList) > 0 {
		response.Fail(c, 101, errList[0])
		return
	}
	u := service.AllService.UserService.CurUser(c)
	// Verify the old password only when the account already has one set
	if !service.AllService.UserService.IsPasswordEmptyByUser(u) {
		ok, _, err := utils.VerifyPassword(u.Password, f.OldPassword)
		if err != nil || !ok {
			response.Fail(c, 101, response.TranslateMsg(c, "OldPasswordError"))
			return
		}
	}
	err := service.AllService.UserService.UpdatePassword(u, f.NewPassword)
	if err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "OperationFailed")+err.Error())
		return
	}
	// After changing the password, clear all session tokens of the current user and force a re-login to prevent old tokens from being used fraudulently.
	_ = service.AllService.UserService.FlushToken(u)
	response.Success(c, nil)
}

// MyOauth
// @Tags user
// @Summary My authorization
// @Description My authorization
// @Accept  json
// @Produce  json
// @Success 200 {object} response.Response{data=[]admin.UserOauthItem}
// @Failure 500 {object} response.Response
// @Router /admin/user/myOauth [get]
// @Security token
func (ct *User) MyOauth(c *gin.Context) {
	u := service.AllService.UserService.CurUser(c)
	oal := service.AllService.OauthService.List(1, 100, nil)
	ops := make([]string, 0)
	for _, oa := range oal.Oauths {
		ops = append(ops, oa.Op)
	}
	uts := service.AllService.UserService.UserThirdsByUserId(u.Id)
	var res []*adResp.UserOauthItem
	for _, oa := range oal.Oauths {
		item := &adResp.UserOauthItem{
			Op: oa.Op,
		}
		for _, ut := range uts {
			if ut.Op == oa.Op {
				item.Status = 1
				break
			}
		}
		res = append(res, item)
	}
	response.Success(c, res)
}

// ===================== MFA (TOTP) =====================

// MfaSetup generates TOTP keys and QR codes, and temporarily stores keys (not yet enabled)
// @Tags user
// @Summary MFA initialization
// @Router /admin/user/mfa/setup [post]
// @Security token
func (ct *User) MfaSetup(c *gin.Context) {
	u := service.AllService.UserService.CurUser(c)
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "RustDesk",
		AccountName: u.Username,
	})
	if err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "OperationFailed")+err.Error())
		return
	}
	if err := service.AllService.UserService.SetMfaSecret(u, key.Secret()); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "OperationFailed")+err.Error())
		return
	}
	qrPng, qerr := qrcode.Encode(key.URL(), qrcode.Medium, 256)
	qr := ""
	if qerr == nil {
		qr = "data:image/png;base64," + base64.StdEncoding.EncodeToString(qrPng)
	}
	response.Success(c, gin.H{
		"secret":      key.Secret(),
		"otpauth_url": key.URL(),
		"qr":          qr,
	})
}

// MfaEnable enables MFA after verifying the dynamic code and returns a one-time recovery code
// @Tags user
// @Summary MFA enabled
// @Router /admin/user/mfa/enable [post]
// @Security token
func (ct *User) MfaEnable(c *gin.Context) {
	f := &admin.MfaEnableForm{}
	if err := c.ShouldBindJSON(f); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	errList := global.Validator.ValidStruct(c, f)
	if len(errList) > 0 {
		response.Fail(c, 101, errList[0])
		return
	}
	u := service.AllService.UserService.CurUser(c)
	if u.MfaSecret == "" {
		response.Fail(c, 101, response.TranslateMsg(c, "MfaSecretEmpty"))
		return
	}
	if !service.AllService.UserService.VerifyMfaCode(u, f.Code) {
		response.Fail(c, 101, response.TranslateMsg(c, "MfaCodeError"))
		return
	}
	codes, err := service.AllService.UserService.EnableMfa(u)
	if err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "OperationFailed")+err.Error())
		return
	}
	response.Success(c, gin.H{"recovery_codes": codes})
}

// MfaDisable closes MFA after verifying the login password
// @Tags user
// @Summary MFA Close
// @Router /admin/user/mfa/disable [post]
// @Security token
func (ct *User) MfaDisable(c *gin.Context) {
	f := &admin.MfaDisableForm{}
	if err := c.ShouldBindJSON(f); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	errList := global.Validator.ValidStruct(c, f)
	if len(errList) > 0 {
		response.Fail(c, 101, errList[0])
		return
	}
	u := service.AllService.UserService.CurUser(c)
	checked := service.AllService.UserService.InfoByUsernamePassword(u.Username, f.Password)
	if checked.Id == 0 {
		response.Fail(c, 101, response.TranslateMsg(c, "PasswordError"))
		return
	}
	if err := service.AllService.UserService.DisableMfa(u); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "OperationFailed")+err.Error())
		return
	}
	// After turning off MFA, clear all session tokens of the current user and force a re-login to prevent old tokens from being used fraudulently.
	_ = service.AllService.UserService.FlushToken(u)
	response.Success(c, nil)
}

// MfaStatus returns the current user MFA enablement status
// @Tags user
// @Summary MFA Status
// @Router /admin/user/mfa/status [get]
// @Security token
func (ct *User) MfaStatus(c *gin.Context) {
	u := service.AllService.UserService.CurUser(c)
	response.Success(c, gin.H{"mfa_enabled": u.MfaEnabled})
}

// MfaReset allows the administrator to forcefully close the specified user's MFA (rescue method when the user loses the authenticator/recovery code)
// @Tags user
// @Summary Administrator resets user MFA
// @Router /admin/user/mfa/reset [post]
// @Security token
func (ct *User) MfaReset(c *gin.Context) {
	f := &admin.MfaResetForm{}
	if err := c.ShouldBindJSON(f); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	errList := global.Validator.ValidStruct(c, f)
	if len(errList) > 0 {
		response.Fail(c, 101, errList[0])
		return
	}
	u := service.AllService.UserService.InfoById(f.UserId)
	if u.Id == 0 {
		response.Fail(c, 101, response.TranslateMsg(c, "ItemNotFound"))
		return
	}
	if err := service.AllService.UserService.DisableMfa(u); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "OperationFailed")+err.Error())
		return
	}
	// After resetting MFA, clear all session tokens of the user and force a re-login to prevent old tokens from being used fraudulently.
	_ = service.AllService.UserService.FlushToken(u)
	response.Success(c, nil)
}

// groupUsers
func (ct *User) GroupUsers(c *gin.Context) {
	aG := service.AllService.GroupService.List(1, 999, nil)
	aU := service.AllService.UserService.List(1, 9999, nil)
	response.Success(c, gin.H{
		"groups": aG.Groups,
		"users":  aU.Users,
	})
}

// Register
func (ct *User) Register(c *gin.Context) {
	if !global.Config.App.Register {
		response.Fail(c, 101, response.TranslateMsg(c, "RegisterClosed"))
		return
	}
	f := &admin.RegisterForm{}
	if err := c.ShouldBindJSON(f); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	errList := global.Validator.ValidStruct(c, f)
	if len(errList) > 0 {
		response.Fail(c, 101, errList[0])
		return
	}

	// Authorization code verification: When invitation mode is turned on, a valid authorization code must be provided
	// The authorization code controls registration qualification + subscription activation at the same time
	var userExpiredAt int64
	if global.Config.App.InviteOnly {
		if f.InviteCode == "" {
			response.Fail(c, 101, response.TranslateMsg(c, "InviteCodeRequired"))
			return
		}
		// Replace old Invitation with InviteCode (authorization code)
		ics := &service.InviteCodeService{}
		ic, err := ics.InfoByCode(f.InviteCode)
		if errors.Is(err, service.ErrInviteCodeNotFound) {
			response.Fail(c, 101, response.TranslateMsg(c, "InviteCodeInvalid"))
			return
		}
		if err != nil {
			response.ServerError(c)
			return
		}
		if ic.Status != "unused" {
			response.Fail(c, 101, response.TranslateMsg(c, "InviteCodeInvalid"))
			return
		}
		if ic.ExpireAt.Before(time.Now()) {
			response.Fail(c, 101, response.TranslateMsg(c, "InviteCodeInvalid"))
			return
		}
	}

	regStatus := model.StatusCode(global.Config.App.RegisterStatus)
	// Registration status may not be configured and is enabled by default
	if regStatus != model.COMMON_STATUS_DISABLED && regStatus != model.COMMON_STATUS_ENABLE {
		regStatus = model.COMMON_STATUS_ENABLE
	}

	u := service.AllService.UserService.Register(f.Username, f.Email, f.Password, regStatus, userExpiredAt)
	if u == nil || u.Id == 0 {
		response.Fail(c, 101, response.TranslateMsg(c, "OperationFailed"))
		return
	}

	// After successful registration, consume the authorization code + activate the subscription (atomic update prevents concurrent reuse)
	if global.Config.App.InviteOnly && f.InviteCode != "" {
		ics := &service.InviteCodeService{}
		ic, err := ics.InfoByCode(f.InviteCode)
		if err != nil {
			global.DB.Delete(&model.User{}, u.Id)
			if errors.Is(err, service.ErrInviteCodeNotFound) {
				response.Fail(c, 101, response.TranslateMsg(c, "InviteCodeInvalid"))
			} else {
				response.ServerError(c)
			}
			return
		}
		if ic.Status == "unused" {
			now := time.Now()
			// Atomic update: only update records with status="unused"
			result := global.DB.Model(&model.InviteCode{}).
				Where("id = ? AND status = ?", ic.Id, "unused").
				Updates(map[string]interface{}{
					"status":  "used",
					"used_by": u.Id,
					"used_at": &now,
				})
			if result.Error != nil || result.RowsAffected == 0 {
				// Atomic update failed (already used by someone else), deleted the created user
				global.DB.Delete(&model.User{}, u.Id)
				response.Fail(c, 101, response.TranslateMsg(c, "InviteCodeInvalid"))
				return
			}
			periodDuration := time.Duration(ic.ExpireDays*24) * time.Hour
			newExpire := now.Add(periodDuration)
			expiredAt := newExpire.Unix()
			global.DB.Model(&model.User{}).Where("id = ?", u.Id).
				Updates(map[string]interface{}{
					"subscription_plan":      ic.Plan,
					"subscription_expire_at": &newExpire,
					"expired_at":             expiredAt,
				})
		}
	}

	if regStatus == model.COMMON_STATUS_DISABLED {
		// Requires administrator review
		response.Fail(c, 101, response.TranslateMsg(c, "RegisterSuccessWaitAdminConfirm"))
		return
	}
	// Automatically log in after successful registration
	ut := service.AllService.UserService.Login(u, &model.LoginLog{
		UserId:    u.Id,
		Client:    model.LoginLogClientWebAdmin,
		Uuid:      "",
		Ip:        c.ClientIP(),
		UserAgent: c.GetHeader("User-Agent"),
		Type:      model.LoginLogTypeAccount,
	})
	responseLoginSuccess(c, u, ut)
}
