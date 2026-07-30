package model

import "time"

type User struct {
	IdModel
	Username string `json:"username" gorm:"default:'';not null;uniqueIndex"`
	Email    string `json:"email" gorm:"default:'';not null;index"`
	// Email	string     	`json:"email" `
	Password  string     `json:"-" gorm:"default:'';not null;"`
	Nickname  string     `json:"nickname" gorm:"default:'';not null;"`
	Avatar    string     `json:"avatar" gorm:"default:'';not null;"`
	GroupId   uint       `json:"group_id" gorm:"default:0;not null;index"`
	Role      string     `json:"role" gorm:"size:32;default:'user';not null;index"`
	IsAdmin   *bool      `json:"is_admin" gorm:"default:0;not null;"`
	Status    StatusCode `json:"status" gorm:"default:1;not null;"`
	Remark    string     `json:"remark" gorm:"default:'';not null;"`
	ExpiredAt int64      `json:"expired_at" gorm:"default:0;not null;"` // Account expiration timestamp, 0=never expires
	// MFA(TOTP) related fields:mfa_secret/mfa_recoverynot exposed to the outside world
	MfaEnabled  bool   `json:"mfa_enabled" gorm:"default:0;not null;"`
	MfaSecret   string `json:"-" gorm:"default:'';not null;"`
	MfaRecovery string `json:"-" gorm:"default:'';not null;"`
	// SubscriptionPlan Subscription package ID, empty string means not subscribed
	SubscriptionPlan string `json:"subscription_plan" gorm:"size:32;default:''"`
	// SubscriptionExpireAt Subscription expiration time, empty means never subscribed
	SubscriptionExpireAt *time.Time `json:"subscription_expire_at" gorm:"default:null"`
	TimeModel
}

// SubscriptionStatus returns subscription status: active / expired / none / permanent
func (u *User) SubscriptionStatus() string {
	if u.SubscriptionExpireAt == nil {
		return "none"
	}
	if u.SubscriptionExpireAt.Year() >= 9999 {
		return "permanent"
	}
	if u.SubscriptionExpireAt.Before(time.Now()) {
		return "expired"
	}
	return "active"
}

// IsSubscriptionActive Whether the subscription is valid (permanent = true, expired = false, never subscribed = false)
func (u *User) IsSubscriptionActive() bool {
	status := u.SubscriptionStatus()
	return status == "active" || status == "permanent"
}

// SubscriptionDaysLeft returns the remaining days of the subscription; permanently returns -1
func (u *User) SubscriptionDaysLeft() int {
	if u.SubscriptionExpireAt == nil {
		return 0
	}
	if u.SubscriptionExpireAt.Year() >= 9999 {
		return -1
	}
	now := time.Now()
	if u.SubscriptionExpireAt.Before(now) {
		return 0
	}
	days := int(u.SubscriptionExpireAt.Sub(now).Hours() / 24)
	if days < 0 {
		return 0
	}
	return days
}

// The BeforeSave hook is used to ensure that the email field has a reasonable default value
//func (u *User) BeforeSave(tx *gorm.DB) (err error) {
//	// If email is empty, set it to the default value
//	if u.Email == "" {
//		u.Email = fmt.Sprintf("%s@example.com", u.Username)
//	}
//	return nil
//}

type UserList struct {
	Users []*User `json:"list,omitempty"`
	Pagination
}

var UserRouteNames = []string{
	"MyTagList", "MyAddressBookList", "MyInfo", "MyAddressBookCollection", "MyPeer", "MyShareRecordList", "MyLoginLog",
	"StationMessages", "HomePage", "MySubscription",
}
var AdminRouteNames = []string{"*"}
