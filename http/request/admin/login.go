package admin

type Login struct {
	Username  string `json:"username" validate:"required" label:"username"`
	Password  string `json:"password,omitempty" validate:"required" label:"password"`
	Platform  string `json:"platform" label:"platform"`
	Captcha   string `json:"captcha,omitempty" label:"captcha"`
	CaptchaId string `json:"captcha_id,omitempty"`
}

// MfaLogin is the second-factor MFA login request.
type MfaLogin struct {
	MfaToken     string `json:"mfa_token" validate:"required" label:"MFA token"`
	Code         string `json:"code" label:"TOTP code"`
	RecoveryCode string `json:"recovery_code" label:"recovery code"`
	Platform     string `json:"platform" label:"platform"`
}

type LoginLogQuery struct {
	UserId int `form:"user_id"`
	IsMy   int `form:"is_my"`
	PageQuery
}
type LoginTokenQuery struct {
	UserId int `form:"user_id"`
	PageQuery
}

type LoginLogIds struct {
	Ids []uint `json:"ids" validate:"required"`
}
