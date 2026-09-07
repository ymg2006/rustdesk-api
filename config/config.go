package config

import (
	"fmt"
	"strconv"
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
	Lang       string     `mapstructure:"lang"`
	App        App        `mapstructure:"app"`
	Admin      Admin      `mapstructure:"admin"`
	Gorm       Gorm       `mapstructure:"gorm"`
	Mysql      Mysql      `mapstructure:"mysql"`
	Postgresql Postgresql `mapstructure:"postgresql"`
	Gin        Gin        `mapstructure:"gin"`
	Logger     Logger     `mapstructure:"logger"`
	Redis      Redis      `mapstructure:"redis"`
	Cache      Cache      `mapstructure:"cache"`
	Oss        Oss        `mapstructure:"oss"`
	Jwt        Jwt        `mapstructure:"jwt"`
	Rustdesk   Rustdesk   `mapstructure:"rustdesk"`
	Proxy      Proxy      `mapstructure:"proxy"`
	Ldap       Ldap       `mapstructure:"ldap"`
	Cors       Cors       `mapstructure:"cors"`
	Server     Server     `mapstructure:"server"`
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
	v, err := Load(rowVal, path)
	if err != nil {
		panic(err)
	}
	return v
}

// Load reads, normalizes, and validates the application configuration. Defaults
// and explicit bindings make environment-only keys visible to Viper.Unmarshal.
func Load(rowVal *Config, path string) (*viper.Viper, error) {
	if path == "" {
		path = DefaultConfig
	}
	v := viper.New()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	v.SetEnvPrefix("RUSTDESK_API")
	v.AutomaticEnv()
	setDefaults(v)
	if err := bindEnvironment(v); err != nil {
		return nil, fmt.Errorf("bind configuration environment: %w", err)
	}
	v.SetConfigFile(path)
	v.SetConfigType("yaml")
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("fatal error config file: %w", err)
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
		return nil, fmt.Errorf("fatal error config: %w", err)
	}
	if err := rowVal.NormalizeAndValidate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}
	rowVal.Rustdesk.LoadKeyFile()
	rowVal.Admin.Init()
	return v, nil
}

func setDefaults(v *viper.Viper) {
	defaults := map[string]interface{}{
		"cache.type":              "file",
		"cache.file-dir":          "./runtime/cache",
		"cache.redis-addr":        "127.0.0.1:6379",
		"cache.redis-pwd":         "",
		"cache.redis-db":          0,
		"cache.connect-timeout":   "5s",
		"gorm.type":               TypeSqlite,
		"gorm.max-idle-conns":     10,
		"gorm.max-open-conns":     100,
		"gorm.conn-max-lifetime":  "1h",
		"gorm.conn-max-idle-time": "10m",
		"gorm.connect-timeout":    "10s",
		"mysql.addr":              "",
		"mysql.username":          "",
		"mysql.password":          "",
		"mysql.dbname":            "",
		"mysql.tls":               "false",
		"postgresql.host":         "127.0.0.1",
		"postgresql.port":         "5432",
		"postgresql.user":         "",
		"postgresql.password":     "",
		"postgresql.dbname":       "postgres",
		"postgresql.sslmode":      "disable",
		"postgresql.time-zone":    "UTC",
	}
	for key, value := range defaults {
		v.SetDefault(key, value)
	}
}

func bindEnvironment(v *viper.Viper) error {
	keys := []string{
		"cache.type", "cache.file-dir", "cache.redis-addr", "cache.redis-pwd", "cache.redis-db", "cache.connect-timeout",
		"gorm.type", "gorm.max-idle-conns", "gorm.max-open-conns", "gorm.conn-max-lifetime",
		"gorm.conn-max-idle-time", "gorm.connect-timeout",
		"mysql.addr", "mysql.username", "mysql.password", "mysql.dbname", "mysql.tls",
		"postgresql.host", "postgresql.port", "postgresql.user", "postgresql.password",
		"postgresql.dbname", "postgresql.sslmode", "postgresql.time-zone",
	}
	for _, key := range keys {
		if err := v.BindEnv(key); err != nil {
			return err
		}
	}
	return nil
}

// NormalizeAndValidate canonicalizes backend aliases and rejects configurations
// that would otherwise fail later or silently select a different backend.
func (c *Config) NormalizeAndValidate() error {
	c.Cache.Type = strings.ToLower(strings.TrimSpace(c.Cache.Type))
	if c.Cache.Type == "disabled" {
		c.Cache.Type = "none"
	}
	switch c.Cache.Type {
	case "file":
		if strings.TrimSpace(c.Cache.FileDir) == "" {
			return fmt.Errorf("cache.file-dir must not be empty for file cache")
		}
	case "memory", "redis", "none":
	default:
		return fmt.Errorf("unsupported cache type: %s", c.Cache.Type)
	}
	if c.Cache.Type == "redis" {
		if strings.TrimSpace(c.Cache.RedisAddr) == "" {
			return fmt.Errorf("cache.redis-addr must not be empty for Redis cache")
		}
		if c.Cache.ConnectTimeout <= 0 {
			return fmt.Errorf("cache.connect-timeout must be greater than zero")
		}
		if c.Cache.RedisDb < 0 {
			return fmt.Errorf("cache.redis-db must not be negative")
		}
	}

	c.Gorm.Type = strings.ToLower(strings.TrimSpace(c.Gorm.Type))
	if c.Gorm.Type == "postgres" {
		c.Gorm.Type = TypePostgresql
	}
	switch c.Gorm.Type {
	case TypeSqlite:
		if c.Gorm.ConnectTimeout <= 0 {
			c.Gorm.ConnectTimeout = 10 * time.Second
		}
		return nil
	case TypeMysql:
		if c.Gorm.ConnectTimeout <= 0 {
			return fmt.Errorf("gorm.connect-timeout must be greater than zero")
		}
		if strings.TrimSpace(c.Mysql.Addr) == "" {
			return fmt.Errorf("mysql.addr must not be empty")
		}
		if strings.TrimSpace(c.Mysql.Dbname) == "" {
			return fmt.Errorf("mysql.dbname must not be empty")
		}
		return nil
	case TypePostgresql:
		if c.Gorm.ConnectTimeout <= 0 {
			return fmt.Errorf("gorm.connect-timeout must be greater than zero")
		}
		if strings.TrimSpace(c.Postgresql.Host) == "" {
			return fmt.Errorf("postgresql.host must not be empty")
		}
		port, err := strconv.Atoi(c.Postgresql.Port)
		if err != nil || port < 1 || port > 65535 {
			return fmt.Errorf("postgresql.port must be an integer between 1 and 65535")
		}
		if strings.TrimSpace(c.Postgresql.User) == "" {
			return fmt.Errorf("postgresql.user must not be empty")
		}
		if strings.TrimSpace(c.Postgresql.Dbname) == "" {
			return fmt.Errorf("postgresql.dbname must not be empty")
		}
		sslmode := strings.ToLower(strings.TrimSpace(c.Postgresql.Sslmode))
		switch sslmode {
		case "disable", "allow", "prefer", "require", "verify-ca", "verify-full":
			c.Postgresql.Sslmode = sslmode
		default:
			return fmt.Errorf("unsupported postgresql.sslmode: %s", c.Postgresql.Sslmode)
		}
		return nil
	default:
		return fmt.Errorf("unsupported database type: %s", c.Gorm.Type)
	}
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
