package admin

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ymg2006/rustdesk-api/v2/global"
	"github.com/ymg2006/rustdesk-api/v2/http/controller/api"
	"github.com/ymg2006/rustdesk-api/v2/http/request/admin"
	apiReq "github.com/ymg2006/rustdesk-api/v2/http/request/api"
	"github.com/ymg2006/rustdesk-api/v2/http/response"
	adResp "github.com/ymg2006/rustdesk-api/v2/http/response/admin"
	"github.com/ymg2006/rustdesk-api/v2/model"
	"github.com/ymg2006/rustdesk-api/v2/service"
)

type Login struct {
}

// Login Login
// @Tags login
// @Summary Login
// @Description Login
// @Accept  json
// @Produce  json
// @Param body body admin.Login true "Login information"
// @Success 200 {object} response.Response{data=admin.LoginPayload}
// @Failure 500 {object} response.Response
// @Router /admin/login [post]
// @Security token
func (ct *Login) Login(c *gin.Context) {
	if global.Config.App.DisablePwdLogin {
		response.Fail(c, 101, response.TranslateMsg(c, "PwdLoginDisabled"))
		return
	}

	// Check login restrictions
	loginLimiter := global.LoginLimiter
	clientIp := c.ClientIP()
	_, needCaptcha := loginLimiter.CheckSecurityStatus(clientIp)

	f := &admin.Login{}
	err := c.ShouldBindJSON(f)
	if err != nil {
		loginLimiter.RecordFailedAttempt(clientIp)
		global.Logger.Warn(fmt.Sprintf("Login Fail: %s %s %s", "ParamsError", c.RemoteIP(), clientIp))
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}

	errList := global.Validator.ValidStruct(c, f)
	if len(errList) > 0 {
		loginLimiter.RecordFailedAttempt(clientIp)
		global.Logger.Warn(fmt.Sprintf("Login Fail: %s %s %s", "ParamsError", c.RemoteIP(), clientIp))
		response.Fail(c, 101, errList[0])
		return
	}

	// Check if a verification code is required
	if needCaptcha {
		if f.CaptchaId == "" || f.Captcha == "" || !loginLimiter.VerifyCaptcha(f.CaptchaId, f.Captcha) {
			response.Fail(c, 101, response.TranslateMsg(c, "CaptchaError"))
			return
		}
	}

	u := service.AllService.UserService.InfoByUsernamePassword(f.Username, f.Password)

	if u.Id == 0 {
		global.Logger.Warn(fmt.Sprintf("Login Fail: %s %s %s", "UsernameOrPasswordError", c.RemoteIP(), clientIp))
		loginLimiter.RecordFailedAttempt(clientIp)
		if _, needCaptcha = loginLimiter.CheckSecurityStatus(clientIp); needCaptcha {
			response.Fail(c, 110, response.TranslateMsg(c, "UsernameOrPasswordError"))
		} else {
			response.Fail(c, 101, response.TranslateMsg(c, "UsernameOrPasswordError"))
		}
		return
	}

	if !service.AllService.UserService.CheckUserEnable(u) {
		if needCaptcha {
			response.Fail(c, 110, response.TranslateMsg(c, "UserDisabled"))
			return
		}
		response.Fail(c, 101, response.TranslateMsg(c, "UserDisabled"))
		return
	}

	// MFA two-step verification: If enabled, a temporary token will be issued, and the front end will enter the dynamic code input step.
	if u.MfaEnabled {
		mfaToken := global.Jwt.GenerateMfaToken(u.Id)
		global.Logger.Infof("[MFA] Login() uid=%d mfa_token_len=%d", u.Id, len(mfaToken))
		response.SendResponse(c, 113, response.TranslateMsg(c, "MfaRequired"), gin.H{"mfa_token": mfaToken})
		return
	}

	ut := service.AllService.UserService.Login(u, &model.LoginLog{
		UserId:    u.Id,
		Client:    model.LoginLogClientWebAdmin,
		Uuid:      "", //must be empty
		Ip:        clientIp,
		UserAgent: c.GetHeader("User-Agent"),
		Type:      model.LoginLogTypeAccount,
		Platform:  f.Platform,
	})

	// Login successful, clear login restrictions
	loginLimiter.RemoveAttempts(clientIp)
	responseLoginSuccess(c, u, ut)
}

// MfaLogin MFA issues official token after secondary verification
// @Tags login
// @Summary MFA two-step verification login
// @Description Use the mfa_token returned when logging in and the dynamic code/recovery code to exchange for the official login token
// @Accept  json
// @Produce  json
// @Param body body admin.MfaLogin true "MFA verification information"
// @Success 200 {object} response.Response{data=admin.LoginPayload}
// @Failure 500 {object} response.Response
// @Router /admin/login/mfa [post]
func (ct *Login) MfaLogin(c *gin.Context) {
	f := &admin.MfaLogin{}
	if err := c.ShouldBindJSON(f); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	// Record parsed key fields (excluding sensitive values) to troubleshoot MFA token and one-time-code issues.
	// Note: c.ShouldBindJSON cannot be called again here because the request body has been consumed by ShouldBindJSON(f) above.
	// Secondary reading will result in empty bodies, causing log distortion and possibly interfering with subsequent processing.
	global.Logger.Infof("[MFA] MfaLogin parsed={MfaToken_len=%d HasCode=%t HasRecoveryCode=%t Platform=%q}",
		len(f.MfaToken), f.Code != "", f.RecoveryCode != "", f.Platform)

	errList := global.Validator.ValidStruct(c, f)
	if len(errList) > 0 {
		// mfa_token is completely missing: return special error code 114 to guide the front end to re-login process
		if f.MfaToken == "" {
			global.Logger.Warnf("[MFA] MfaLogin: mfa_token missing (body and header both empty)")
			response.Fail(c, 114, response.TranslateMsg(c, "MfaTokenMissing"))
			return
		}
		global.Logger.Warnf("[MFA] MfaLogin: validation failed: %v", errList)
		response.Fail(c, 101, errList[0])
		return
	}
	uid, err := global.Jwt.ParseMfaToken(f.MfaToken)
	if err != nil {
		// SECURITY: MFA temporary tokens are short-lived (5 minutes) and sensitive. Only the length is recorded in the log to avoid leaking tokens that can be replayed.
		global.Logger.Warnf("[MFA] MfaLogin: ParseMfaToken failed, mfa_token_len=%d err=%v", len(f.MfaToken), err)
		response.Fail(c, 101, response.TranslateMsg(c, "MfaTokenInvalid"))
		return
	}
	u := service.AllService.UserService.InfoById(uid)
	if u.Id == 0 || !u.MfaEnabled {
		response.Fail(c, 101, response.TranslateMsg(c, "MfaTokenInvalid"))
		return
	}
	// Verify dynamic code or recovery code
	ok := false
	if f.RecoveryCode != "" {
		ok = service.AllService.UserService.VerifyMfaRecovery(u, f.RecoveryCode)
	} else {
		ok = service.AllService.UserService.VerifyMfaCode(u, f.Code)
	}
	if !ok {
		response.Fail(c, 101, response.TranslateMsg(c, "MfaCodeError"))
		return
	}
	ut := service.AllService.UserService.Login(u, &model.LoginLog{
		UserId:    u.Id,
		Client:    model.LoginLogClientWebAdmin,
		Uuid:      "", //must be empty
		Ip:        c.ClientIP(),
		UserAgent: c.GetHeader("User-Agent"),
		Type:      model.LoginLogTypeAccount,
		Platform:  f.Platform,
	})
	responseLoginSuccess(c, u, ut)
}

func (ct *Login) Captcha(c *gin.Context) {
	loginLimiter := global.LoginLimiter
	clientIp := c.ClientIP()
	banned, needCaptcha := loginLimiter.CheckSecurityStatus(clientIp)
	if banned {
		response.Fail(c, 101, response.TranslateMsg(c, "LoginBanned"))
		return
	}
	if !needCaptcha {
		response.Fail(c, 101, response.TranslateMsg(c, "NoCaptchaRequired"))
		return
	}
	err, captcha := loginLimiter.RequireCaptcha()
	if err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "CaptchaError")+err.Error())
		return
	}
	err, b64 := loginLimiter.DrawCaptcha(captcha.Content)
	if err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "CaptchaError")+err.Error())
		return
	}
	response.Success(c, gin.H{
		"captcha": gin.H{
			"id":  captcha.Id,
			"b64": b64,
		},
	})
}

// Logout
// @Tags login
// @Summary Sign out
// @Description log out
// @Accept  json
// @Produce  json
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /admin/logout [post]
func (ct *Login) Logout(c *gin.Context) {
	// Read token from HttpOnly Cookie (front-end JS cannot read/delete this cookie and can only be cleared by the back-end)
	token, err := c.Cookie("access_token")
	if err == nil && token != "" {
		// Passing an empty fingerprint skips source verification, ensuring that you can log out normally and clear records even if the source changes.
		u, _ := service.AllService.UserService.InfoByAccessToken(token, "")
		if u != nil && u.Id != 0 {
			service.AllService.UserService.Logout(u, token)
		}
	}
	clearAuthCookie(c)
	response.Success(c, nil)
}

// LoginOptions
// @Tags login
// @Summary Login options
// @Description Login options
// @Accept  json
// @Produce  json
// @Success 200 {object} []string
// @Failure 500 {object} response.ErrorResponse
// @Router /admin/login-options [post]
func (ct *Login) LoginOptions(c *gin.Context) {
	loginLimiter := global.LoginLimiter
	clientIp := c.ClientIP()
	banned, needCaptcha := loginLimiter.CheckSecurityStatus(clientIp)
	if banned {
		response.Fail(c, 101, response.TranslateMsg(c, "LoginBanned"))
		return
	}
	ops := service.AllService.OauthService.GetOauthProviders()
	response.Success(c, gin.H{
		"ops":          ops,
		"register":     global.Config.App.Register,
		"need_captcha": needCaptcha,
		"disable_pwd":  global.Config.App.DisablePwdLogin,
		"auto_oidc":    global.Config.App.DisablePwdLogin && len(ops) == 1,
		"invite_only":  global.Config.App.InviteOnly,
	})
}

// OidcAuth
// @Tags Oauth
// @Summary OidcAuth
// @Description OidcAuth
// @Accept  json
// @Produce  json
// @Router /admin/oidc/auth [post]
func (ct *Login) OidcAuth(c *gin.Context) {
	// o := &api.Oauth{}
	// o.OidcAuth(c)
	f := &apiReq.OidcAuthRequest{}
	err := c.ShouldBindJSON(f)
	if err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}

	err, state, verifier, nonce, url := service.AllService.OauthService.BeginAuth(f.Op)
	if err != nil {
		response.Error(c, response.TranslateMsg(c, err.Error()))
		return
	}

	service.AllService.OauthService.SetOauthCache(state, &service.OauthCacheItem{
		Action:     service.OauthActionTypeLogin,
		Op:         f.Op,
		Id:         f.Id,
		DeviceType: "webadmin",
		// DeviceOs: ct.Platform(c),
		DeviceOs: f.DeviceInfo.Os,
		Uuid:     f.Uuid,
		Verifier: verifier,
		Nonce:    nonce,
	}, 3600) // webauto process: Set TTL to 1 hour to avoid administrator login timeout

	response.Success(c, gin.H{
		"code": state,
		"url":  url,
	})
}

// OidcAuthQuery
// @Tags Oauth
// @Summary OidcAuthQuery
// @Description OidcAuthQuery
// @Accept  json
// @Produce  json
// @Success 200 {object} response.Response{data=admin.LoginPayload}
// @Failure 500 {object} response.Response
// @Router /admin/oidc/auth-query [get]
func (ct *Login) OidcAuthQuery(c *gin.Context) {
	o := &api.Oauth{}
	u, ut := o.OidcAuthQueryPre(c)
	if ut == nil {
		return
	}
	responseLoginSuccess(c, u, ut)
}

// setAuthCookie sets the session cookie: HttpOnly to prevent XSS reading, SameSite=Lax to mitigate CSRF.
// Secure is only enabled with HTTPS (direct TLS or reverse proxy X-Forwarded-Proto: https),
// Prevent browsers from rejecting cookies in local plaintext HTTP development environments.
func setAuthCookie(c *gin.Context, token string, maxAge int) {
	secure := false
	if c.Request.TLS != nil {
		secure = true
	} else if hp := c.GetHeader("X-Forwarded-Proto"); strings.EqualFold(hp, "https") {
		secure = true
	}
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("access_token", token, maxAge, "/", "", secure, true)
}

// clearAuthCookie clears session cookies (called when logout or login fails).
func clearAuthCookie(c *gin.Context) {
	secure := false
	if c.Request.TLS != nil {
		secure = true
	} else if hp := c.GetHeader("X-Forwarded-Proto"); strings.EqualFold(hp, "https") {
		secure = true
	}
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("access_token", "", -1, "/", "", secure, true)
}

func responseLoginSuccess(c *gin.Context, u *model.User, ut *model.UserToken) {
	// The token is only issued through HttpOnly Cookie and is no longer returned to the front-end JS, fundamentally eliminating the risk of XSS theft.
	setAuthCookie(c, ut.Token, int(ut.ExpiredAt-time.Now().Unix()))
	lp := &adResp.LoginPayload{}
	lp.FromUser(u)
	lp.Token = "" // No longer expose tokens to the front end
	lp.RouteNames = service.AllService.UserService.RouteNames(u)
	response.Success(c, lp)
}
