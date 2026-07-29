package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

const (
	DebugMode     = "debug"
	ReleaseMode   = "release"
	DefaultConfig = "conf/config.yaml"
)

type App struct {
	WebClient          int           `mapstructure:"web-client"`
	Register           bool          `mapstructure:"register"`
	RegisterStatus     int           `mapstructure:"register-status"`
	ShowSwagger        int           `mapstructure:"show-swagger"`
	TokenExpire        time.Duration `mapstructure:"token-expire"`
	WebSso             bool          `mapstructure:"web-sso"`
	DisablePwdLogin    bool          `mapstructure:"disable-pwd-login"`
	CaptchaThreshold   int           `mapstructure:"captcha-threshold"`
	BanThreshold       int           `mapstructure:"ban-threshold"`
	BanWindowMinutes   int           `mapstructure:"ban-window-minutes"`
	BanDurationMinutes int           `mapstructure:"ban-duration-minutes"`
	InviteOnly         bool          `mapstructure:"invite-only"`
}
type Admin struct {
	Title           string `mapstructure:"title"`
	Hello           string `mapstructure:"hello"`
	HelloFile       string `mapstructure:"hello-file"`
	IdServerPort    int    `mapstructure:"id-server-port"`
	RelayServerPort int    `mapstructure:"relay-server-port"`
	// RelayStatsHost is the loopback address used to query hbbr load and connection counts.
	// hbbr only accepts command connections from the local machine, so api-server must be deployed on the same host as hbbr.
	// Use 127.0.0.1 here, or another loopback address of that host.
	RelayStatsHost string `mapstructure:"relay-stats-host"`
}
type Cors struct {
	// AllowOrigins is the CORS origin allowlist as a YAML list, for example ["https://admin.example.com"].
	// An empty list disables CORS by omitting Access-Control-Allow-Origin. See http/middleware/cors.go.
	AllowOrigins []string `mapstructure:"allow-origins"`
}

// Server controls HTTP server timeouts and optional TLS security hardening.
type Server struct {
	// ReadTimeout is the read timeout in seconds for the whole request, including body. It mitigates slow attacks. Default: 15.
	ReadTimeout int `mapstructure:"read-timeout"`
	// WriteTimeout is the response write timeout in seconds. Default: 30.
	WriteTimeout int `mapstructure:"write-timeout"`
	// IdleTimeout is the maximum lifetime in seconds for idle keep-alive connections. Default: 120.
	IdleTimeout int `mapstructure:"idle-timeout"`
	// TLS is optional. It is disabled by default for plain HTTP behind a TLS-terminating reverse proxy.
	TLS ServerTLS `mapstructure:"tls"`
}

// ServerTLS configures optional TLS. When enabled, api-server listens with HTTPS directly.
type ServerTLS struct {
	// Enabled controls whether TLS is enabled. When true, api-server listens with HTTPS directly.
	Enabled bool `mapstructure:"enabled"`
	// CertFile is the certificate file path in PEM format, for example /path/to/fullchain.pem.
	CertFile string `mapstructure:"cert-file"`
	// KeyFile is the private key file path in PEM format, for example /path/to/privkey.pem.
	KeyFile string `mapstructure:"key-file"`
}
type Config struct {
	Lang       string `mapstructure:"lang"`
	App        App
	Admin      Admin
	Gorm       Gorm
	Mysql      Mysql
	Postgresql Postgresql
	Gin        Gin
	Logger     Logger
	Redis      Redis
	Cache      Cache
	Oss        Oss
	Jwt        Jwt
	Rustdesk   Rustdesk
	Proxy      Proxy
	Ldap       Ldap
	Cors       Cors
	Server     Server
	// Payment platform configuration.
	Payment PaymentConfig `mapstructure:"payment"`
	// Subscription plan configuration.
	Subscription SubscriptionConfig `mapstructure:"subscription"`
	// MfaTotpSkew is the number of TOTP time-step periods tolerated for clock drift; each period is 30 seconds.
	// The default totp.Validate skew is 1 (only ±30s). Server/client clock drift or user input delay can occasionally hit the boundary,
	// so the default is relaxed to 3 (±90s). Operators can adjust mfa_totp_skew in config.yaml without recompiling.
	// Values below 1 fall back to the service default of 3.
	MfaTotpSkew int `mapstructure:"mfa_totp_skew"`
}

func (a *Admin) Init() {
	if a.IdServerPort == 0 {
		a.IdServerPort = DefaultIdServerPort
	}
	if a.RelayServerPort == 0 {
		a.RelayServerPort = DefaultRelayServerPort
	}
	if a.RelayStatsHost == "" {
		a.RelayStatsHost = "127.0.0.1"
	}
}

// Init initializes configuration.
func Init(rowVal *Config, path string) *viper.Viper {
	if path == "" {
		path = DefaultConfig
	}
	v := viper.New()
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	v.SetEnvPrefix("RUSTDESK_API")
	v.SetConfigFile(path)
	v.SetConfigType("yaml")
	err := v.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}
	/*
		v.WatchConfig()


			// Watching configuration changes is not necessary for now.
			v.OnConfigChange(func(e fsnotify.Event) {
				// Configuration file change watcher.
				fmt.Println("config file changed:", e.Name)
				if err2 := v.Unmarshal(rowVal); err2 != nil {
					fmt.Println(err2)
				}
				rowVal.Rustdesk.LoadKeyFile()
				rowVal.Rustdesk.ParsePort()
			})
	*/
	if err := v.Unmarshal(rowVal); err != nil {
		panic(fmt.Errorf("Fatal error config: %s \n", err))
	}
	rowVal.Rustdesk.LoadKeyFile()
	rowVal.Admin.Init()
	return v
}

// ReadEnv reads environment variables.
func ReadEnv(rowVal interface{}) *viper.Viper {
	v := viper.New()
	v.AutomaticEnv()
	if err := v.Unmarshal(rowVal); err != nil {
		fmt.Println(err)
	}
	return v
}
