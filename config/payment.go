package config

// PaymentConfig configures QR-code based payments implemented locally without depending on an external platform.
type PaymentConfig struct {
	// Enable controls whether QR-code payment is enabled.
	Enable bool `mapstructure:"enable" yaml:"enable"`
	// SecretKey is the signing key used to verify callback notifications.
	SecretKey string `mapstructure:"secret_key" yaml:"secret_key"`
	// NotifyURL is the asynchronous callback URL used by the payment confirmation tool.
	NotifyURL string `mapstructure:"notify_url" yaml:"notify_url"`
	// OrderExpireSec is the unpaid-order auto-close timeout in seconds. Default: 600.
	OrderExpireSec int `mapstructure:"order_expire_sec" yaml:"order_expire_sec"`

	// Cashier configures checkout-page display settings.
	Cashier CashierConfig `mapstructure:"cashier" yaml:"cashier"`
}

// CashierConfig configures checkout-page display settings.
type CashierConfig struct {
	// SiteName is displayed at the top of the checkout page.
	SiteName string `mapstructure:"site_name" yaml:"site_name"`
	// AlipayQR is the Alipay payment QR image path, relative to resources/ or absolute.
	AlipayQR string `mapstructure:"alipay_qr" yaml:"alipay_qr"`
	// WechatQR is the WeChat payment QR image path.
	WechatQR string `mapstructure:"wechat_qr" yaml:"wechat_qr"`
	// MonitorTip is the message explaining how payment confirmation works.
	MonitorTip string `mapstructure:"monitor_tip" yaml:"monitor_tip"`
}

// PlanOption defines one subscription duration option.
type PlanOption struct {
	// Key is the identifier passed by the frontend.
	Key string `mapstructure:"key" yaml:"key"`
	// Name is the display name, for example "1 month" or "3 months".
	Name string `mapstructure:"name" yaml:"name"`
	// PriceCents is the price in cents.
	PriceCents int64 `mapstructure:"price_cents" yaml:"price_cents"`
	// PeriodDays is the subscription duration in days.
	PeriodDays int `mapstructure:"period_days" yaml:"period_days"`
}

// SubscriptionConfig configures subscription plans.
type SubscriptionConfig struct {
	// Plan is the plan identifier.
	Plan string `mapstructure:"plan" yaml:"plan"`
	// Plans is the list of available duration options.
	Plans []PlanOption `mapstructure:"plans" yaml:"plans"`
	// PriceCents is kept for backward compatibility and is used for single-price configurations.
	PriceCents int64 `mapstructure:"price_cents" yaml:"price_cents"`
	// PeriodDays is kept for backward compatibility and is used for single-duration configurations.
	PeriodDays int `mapstructure:"period_days" yaml:"period_days"`
	// RemindDays lists reminder days before expiration for frontend use.
	RemindDays []int `mapstructure:"remind_days" yaml:"remind_days"`
}

// LookupPlan finds a PlanOption by key and returns nil when none is found.
func (sc *SubscriptionConfig) LookupPlan(key string) *PlanOption {
	if key == "" && len(sc.Plans) > 0 {
		return &sc.Plans[0]
	}
	for _, p := range sc.Plans {
		if p.Key == key {
			return &p
		}
	}
	return nil
}
