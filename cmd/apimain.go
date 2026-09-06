package main

import (
	"context"
	"fmt"
	"io"
	"net"
	nethttp "net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/ymg2006/rustdesk-api/v2/config"
	"github.com/ymg2006/rustdesk-api/v2/global"
	"github.com/ymg2006/rustdesk-api/v2/http"
	"github.com/ymg2006/rustdesk-api/v2/lib/cache"
	"github.com/ymg2006/rustdesk-api/v2/lib/jwt"
	"github.com/ymg2006/rustdesk-api/v2/lib/lock"
	"github.com/ymg2006/rustdesk-api/v2/lib/logger"
	"github.com/ymg2006/rustdesk-api/v2/lib/orm"
	"github.com/ymg2006/rustdesk-api/v2/lib/upload"
	"github.com/ymg2006/rustdesk-api/v2/model"
	"github.com/ymg2006/rustdesk-api/v2/service"
	"github.com/ymg2006/rustdesk-api/v2/utils"
	"gorm.io/gorm"
)

const DatabaseVersion = 265

// @title Admin System API
// @version 1.0
// @description API endpoints
// @basePath /api
// @securityDefinitions.apikey token
// @in header
// @name api-token
// @securitydefinitions.apikey BearerAuth
// @in header
// @name Authorization

var rootCmd = &cobra.Command{
	Use:   "apimain",
	Short: "RUSTDESK API SERVER",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return InitGlobal()
	},
	Run: func(cmd *cobra.Command, args []string) {
		global.Logger.Info("API SERVER START")
		http.ApiInit()
		// Periodically clean up orphaned connection audit records in the background.
		go service.AllService.AuditService.StartStaleConnCloseSweep()
		// Fast connection heartbeat checks based on conns heartbeats (60s timeout) to detect abnormal disconnects.
		go service.AllService.AuditService.StartConnHeartbeatSweep()
	},
}

var resetPwdCmd = &cobra.Command{
	Use:     "reset-admin-pwd [pwd]",
	Example: "reset-admin-pwd 123456",
	Short:   "Reset Admin Password",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		pwd := args[0]
		admin := service.AllService.UserService.InfoById(1)
		if admin.Id == 0 {
			global.Logger.Warn("user not found! ")
			return
		}
		err := service.AllService.UserService.UpdatePassword(admin, pwd)
		if err != nil {
			global.Logger.Error("reset password fail! ", err)
			return
		}
		global.Logger.Info("reset password success! ")
	},
}
var resetUserPwdCmd = &cobra.Command{
	Use:     "reset-pwd [userId] [pwd]",
	Example: "reset-pwd 2 123456",
	Short:   "Reset User Password",
	Args:    cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		userId := args[0]
		pwd := args[1]
		uid, err := strconv.Atoi(userId)
		if err != nil {
			global.Logger.Warn("userId must be int!")
			return
		}
		if uid <= 0 {
			global.Logger.Warn("userId must be greater than 0! ")
			return
		}
		u := service.AllService.UserService.InfoById(uint(uid))
		if u.Id == 0 {
			global.Logger.Warn("user not found! ")
			return
		}
		err = service.AllService.UserService.UpdatePassword(u, pwd)
		if err != nil {
			global.Logger.Warn("reset password fail! ", err)
			return
		}
		global.Logger.Info("reset password success!")
	},
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&global.ConfigPath, "config", "c", "./conf/config.yaml", "choose config file")
	rootCmd.AddCommand(resetPwdCmd, resetUserPwdCmd)
}
func main() {
	err := rootCmd.Execute()
	CloseGlobal()
	if err != nil {
		if global.Logger != nil {
			global.Logger.Error(err)
		} else {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}
}

func InitGlobal() error {
	// Parse configuration.
	var err error
	global.Viper, err = config.Load(&global.Config, global.ConfigPath)
	if err != nil {
		return err
	}

	// Initialize logger.
	global.Logger = logger.New(&logger.Config{
		Path:         global.Config.Logger.Path,
		Level:        global.Config.Logger.Level,
		ReportCaller: global.Config.Logger.ReportCaller,
	})

	// Print a clock snapshot early during startup (configuration and logging are ready, HTTP is not started yet).
	// This proactively detects server clock drift, which was the root environment cause of intermittent MFA failures.
	logClockSnapshot()

	global.InitI18n()

	global.DB, err = initializeDatabase(&global.Config)
	if err != nil {
		return err
	}

	//validator
	global.ApiInitValidator()

	//oss
	global.Oss = &upload.Oss{
		AccessKeyId:     global.Config.Oss.AccessKeyId,
		AccessKeySecret: global.Config.Oss.AccessKeySecret,
		Host:            global.Config.Oss.Host,
		CallbackUrl:     global.Config.Oss.CallbackUrl,
		ExpireTime:      global.Config.Oss.ExpireTime,
		MaxByte:         global.Config.Oss.MaxByte,
	}

	//jwt
	//fmt.Println(global.Config.Jwt.PrivateKey)
	global.Jwt = jwt.NewJwt(global.Config.Jwt.Key, global.Config.Jwt.ExpireDuration)
	// SECURITY: When the JWT signing key is empty, MFA token generation silently returns an empty string.
	// That breaks the frontend MFA flow because mfa_token="" triggers the required-field validation error.
	// Fail fast during startup instead of exposing the problem at runtime.
	if len(global.Jwt.Key) == 0 {
		global.Logger.Fatalf("[SECURITY] jwt.key is empty. Please configure jwt.key in conf/config.yaml (openssl rand -hex is recommended)." +
			"An empty jwt.key makes the MFA flow completely unavailable.")
	}
	//locker
	global.Lock = lock.NewLocal()

	//service
	service.New(&global.Config, global.DB, global.Logger, global.Jwt, global.Lock)
	service.AllService.ProcessMonitorService = &service.ProcessMonitorService{}
	service.AllService.ServerStatusService = &service.ServerStatusService{}
	if err := DatabaseAutoUpdate(); err != nil {
		return err
	}
	if err := initializeCache(&global.Config); err != nil {
		return err
	}
	service.AllService.AlertService.StartChecker()

	global.LoginLimiter = utils.NewLoginLimiter(utils.SecurityPolicy{
		CaptchaThreshold: global.Config.App.CaptchaThreshold,
		BanThreshold:     global.Config.App.BanThreshold,
		AttemptsWindow:   banWindowDuration(),
		BanDuration:      banDurationDuration(),
	})
	global.LoginLimiter.RegisterProvider(utils.B64StringCaptchaProvider{})
	syncQRImages()
	return nil
}

func poolConfig(cfg *config.Config) orm.PoolConfig {
	return orm.PoolConfig{
		MaxIdleConns:    cfg.Gorm.MaxIdleConns,
		MaxOpenConns:    cfg.Gorm.MaxOpenConns,
		ConnMaxLifetime: cfg.Gorm.ConnMaxLifetime,
		ConnMaxIdleTime: cfg.Gorm.ConnMaxIdleTime,
		ConnectTimeout:  cfg.Gorm.ConnectTimeout,
	}
}

func initializeDatabase(cfg *config.Config) (*gorm.DB, error) {
	switch cfg.Gorm.Type {
	case config.TypeSqlite:
		pool := poolConfig(cfg)
		pool.ConnMaxLifetime = 0
		pool.ConnMaxIdleTime = 0
		db, err := orm.NewSqlite(&orm.SqliteConfig{Path: "./data/rustdeskapi.db", PoolConfig: pool}, global.Logger)
		if err != nil {
			return nil, err
		}
		global.Logger.Info("SQLite database enabled: path=./data/rustdeskapi.db")
		return db, nil
	case config.TypeMysql:
		dsn := buildMySQLDSN(cfg.Mysql, cfg.Gorm.ConnectTimeout, cfg.Mysql.Dbname)
		db, err := orm.NewMysql(&orm.MysqlConfig{Dsn: dsn, PoolConfig: poolConfig(cfg)}, global.Logger)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to MySQL addr=%s database=%s user=%s: %s", cfg.Mysql.Addr, cfg.Mysql.Dbname, cfg.Mysql.Username, redactSecret(err, cfg.Mysql.Password))
		}
		global.Logger.Infof("MySQL enabled: addr=%s database=%s user=%s", cfg.Mysql.Addr, cfg.Mysql.Dbname, cfg.Mysql.Username)
		return db, nil
	case config.TypePostgresql:
		dsn := buildPostgresqlDSN(cfg.Postgresql, cfg.Gorm.ConnectTimeout)
		db, err := orm.NewPostgresql(&orm.PostgresqlConfig{Dsn: dsn, PoolConfig: poolConfig(cfg)}, global.Logger)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to PostgreSQL host=%s port=%s database=%s user=%s sslmode=%s: %s", cfg.Postgresql.Host, cfg.Postgresql.Port, cfg.Postgresql.Dbname, cfg.Postgresql.User, cfg.Postgresql.Sslmode, redactSecret(err, cfg.Postgresql.Password))
		}
		global.Logger.Infof("PostgreSQL enabled: host=%s port=%s database=%s user=%s sslmode=%s", cfg.Postgresql.Host, cfg.Postgresql.Port, cfg.Postgresql.Dbname, cfg.Postgresql.User, cfg.Postgresql.Sslmode)
		return db, nil
	default:
		return nil, fmt.Errorf("unsupported database type: %s", cfg.Gorm.Type)
	}
}

func buildMySQLDSN(cfg config.Mysql, timeout time.Duration, database string) string {
	dsn := mysqldriver.Config{
		User:      cfg.Username,
		Passwd:    cfg.Password,
		Net:       "tcp",
		Addr:      cfg.Addr,
		DBName:    database,
		ParseTime: true,
		Loc:       time.Local,
		TLSConfig: cfg.Tls,
		Timeout:   timeout,
		Params:    map[string]string{"charset": "utf8mb4"},
	}
	return dsn.FormatDSN()
}

func buildPostgresqlDSN(cfg config.Postgresql, timeout time.Duration) string {
	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(cfg.User, cfg.Password),
		Host:   net.JoinHostPort(cfg.Host, cfg.Port),
		Path:   "/" + cfg.Dbname,
	}
	if cfg.Password == "" {
		u.User = url.User(cfg.User)
	}
	query := u.Query()
	query.Set("sslmode", cfg.Sslmode)
	query.Set("connect_timeout", strconv.FormatInt(max(1, int64(timeout/time.Second)), 10))
	if cfg.TimeZone != "" {
		query.Set("TimeZone", cfg.TimeZone)
	}
	u.RawQuery = query.Encode()
	return u.String()
}

func initializeCache(cfg *config.Config) error {
	switch cfg.Cache.Type {
	case cache.TypeFile:
		if err := os.MkdirAll(cfg.Cache.FileDir, 0755); err != nil {
			return fmt.Errorf("initialize file cache directory %s: %w", cfg.Cache.FileDir, err)
		}
		fc := cache.NewFileCache()
		fc.SetDir(cfg.Cache.FileDir)
		global.Cache = fc
		global.Logger.Infof("File cache enabled: dir=%s", cfg.Cache.FileDir)
		return nil
	case cache.TypeMem:
		global.Cache = cache.NewMemoryCache(0)
		global.Logger.Info("Memory cache enabled")
		return nil
	case cache.TypeRedis:
		client := redis.NewClient(&redis.Options{
			Addr:         cfg.Cache.RedisAddr,
			Password:     cfg.Cache.RedisPwd,
			DB:           cfg.Cache.RedisDb,
			DialTimeout:  cfg.Cache.ConnectTimeout,
			ReadTimeout:  cfg.Cache.ConnectTimeout,
			WriteTimeout: cfg.Cache.ConnectTimeout,
		})
		ctx, cancel := context.WithTimeout(context.Background(), cfg.Cache.ConnectTimeout)
		defer cancel()
		if err := client.Ping(ctx).Err(); err != nil {
			_ = client.Close()
			return fmt.Errorf("failed to connect to Redis cache addr=%s db=%d: %s", cfg.Cache.RedisAddr, cfg.Cache.RedisDb, redactSecret(err, cfg.Cache.RedisPwd))
		}
		global.Redis = client
		global.Cache = cache.NewRedisWithClient(client)
		global.Logger.Infof("Redis cache enabled: addr=%s db=%d", cfg.Cache.RedisAddr, cfg.Cache.RedisDb)
		return nil
	default:
		return fmt.Errorf("unsupported cache type: %s", cfg.Cache.Type)
	}
}

func redactSecret(err error, secret string) string {
	message := err.Error()
	if secret != "" {
		userInfoEncoded := strings.TrimPrefix(url.UserPassword("", secret).String(), ":")
		for _, encoded := range []string{secret, userInfoEncoded, url.QueryEscape(secret), url.PathEscape(secret)} {
			message = strings.ReplaceAll(message, encoded, "[REDACTED]")
		}
	}
	return message
}

// CloseGlobal owns shutdown of the shared SQL and Redis connection pools.
func CloseGlobal() {
	if closer, ok := global.Cache.(interface{ Close() error }); ok {
		_ = closer.Close()
	}
	global.Cache = nil
	if global.Redis != nil {
		_ = global.Redis.Close()
		global.Redis = nil
	}
	if global.DB != nil {
		if sqlDB, err := global.DB.DB(); err == nil {
			_ = sqlDB.Close()
		}
		global.DB = nil
	}
}

// syncQRImages copies configured payment QR images from absolute paths into resources/static/qr/.
// This allows URLs built by getQRURL, such as /static/qr/{filename}, to access the files correctly.
func syncQRImages() {
	if global.Config.Payment.Cashier.SiteName == "" {
		return
	}
	qrDir := global.Config.Gin.ResourcesPath + "/static/qr"
	if err := os.MkdirAll(qrDir, 0755); err != nil {
		global.Logger.Errorf("failed to create payment QR directory: %v", err)
		return
	}

	cashier := global.Config.Payment.Cashier
	paths := map[string]string{
		"Alipay QR code": cashier.AlipayQR,
		"WeChat QR code": cashier.WechatQR,
	}
	for name, src := range paths {
		if src == "" || strings.HasPrefix(src, "http") {
			continue
		}
		if !filepath.IsAbs(src) {
			continue
		}
		dst := qrDir + "/" + filepath.Base(src)
		if err := copyFile(src, dst); err != nil {
			global.Logger.Errorf("failed to copy %s image %s -> %s: %v", name, src, dst, err)
		} else {
			global.Logger.Infof("payment QR image copied: %s -> %s", name, dst)
		}
	}
}

// copyFile copies a file.
func copyFile(src, dst string) error {
	s, err := os.Open(src)
	if err != nil {
		return err
	}
	defer s.Close()

	d, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer d.Close()

	if _, err := io.Copy(d, s); err != nil {
		return err
	}
	return d.Sync()
}

// defaultClockCheckURL is the reference time source used by the startup clock snapshot.
// This site returns a standard RFC1123 Date header that is used as the reference time.
// If the time cannot be fetched in offline environments (local/CI), the check is skipped gracefully without affecting startup.
const defaultClockCheckURL = "https://www.tencent.com"

// logClockSnapshot prints one INFO-level startup clock snapshot: local UTC time vs reference time (HTTP Date header) and offset.
// This proactively detects server clock drift, which was the likely root environment cause of intermittent MFA failures.
//
// Robustness requirements:
//   - hard timeout of 3s per network request to avoid slow startup when there is no internet access;
//   - network errors, timeouts, missing Date headers, and parse failures are warned and skipped only;
//   - if global.Logger is not ready yet, fall back to a standalone logrus instance so logging itself never panics.
func logClockSnapshot() {
	lg := global.Logger
	if lg == nil {
		// Fallback: InitGlobal should already have initialized Logger; this only handles extreme timing cases.
		lg = logrus.New()
	}

	client := &nethttp.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(defaultClockCheckURL)
	if err != nil {
		lg.Warnf("[CLOCK] skipping clock snapshot: failed to connect to reference time source %s: %v", defaultClockCheckURL, err)
		return
	}
	defer resp.Body.Close()

	dateStr := resp.Header.Get("Date")
	if dateStr == "" {
		lg.Warnf("[CLOCK] skipping clock snapshot: reference time source %s did not return a Date header", defaultClockCheckURL)
		return
	}
	refTime, err := time.Parse(time.RFC1123, dateStr)
	if err != nil {
		lg.Warnf("[CLOCK] skipping clock snapshot: failed to parse Date header %q: %v", dateStr, err)
		return
	}

	// Convert both values to UTC for comparison to eliminate local timezone differences.
	// offsetMs is signed: positive means the local clock is ahead of the reference time, negative means it is behind.
	local := time.Now().UTC()
	offsetMs := local.Sub(refTime).Milliseconds()
	lg.Infof("[CLOCK] local=%s ref=%s offsetMs=%d",
		local.Format(time.RFC3339), refTime.Format(time.RFC3339), offsetMs)
}

// isValidDatabaseName validates database names using MySQL identifier constraints.
// Only letters, digits, and underscores are allowed; the name must start with a letter or underscore and be at most 64 bytes.
// This is defense-in-depth before CREATE DATABASE to avoid SQL injection if the database name is polluted.
func isValidDatabaseName(name string) bool {
	if name == "" || len(name) > 64 {
		return false
	}
	matched, err := regexp.MatchString(`^[A-Za-z_][A-Za-z0-9_]*$`, name)
	if err != nil {
		return false
	}
	return matched
}

// banWindowDuration returns the sliding window for failed login counting. Missing or invalid values (<=0) fall back to 15 minutes.
func banWindowDuration() time.Duration {
	m := global.Config.App.BanWindowMinutes
	if m <= 0 {
		m = 15
	}
	return time.Duration(m) * time.Minute
}

// banDurationDuration returns the ban duration after the threshold is reached. Missing or invalid values (<=0) fall back to 30 minutes.
func banDurationDuration() time.Duration {
	m := global.Config.App.BanDurationMinutes
	if m <= 0 {
		m = 30
	}
	return time.Duration(m) * time.Minute
}

func DatabaseAutoUpdate() error {
	version := DatabaseVersion

	db := global.DB

	if !db.Migrator().HasTable(&model.Version{}) {
		if err := Migrate(uint(version)); err != nil {
			return err
		}
	} else {
		// Find the latest version record.
		var v model.Version
		if err := db.Last(&v).Error; err != nil {
			return fmt.Errorf("read database schema version: %w", err)
		}
		if v.Version < uint(version) {
			if err := Migrate(uint(version)); err != nil {
				return err
			}
		}

		// Migration for version 245.
		if v.Version < 245 {
			// Set oauths.oauth_type to the same value as op.
			if err := db.Exec("update oauths set oauth_type = op").Error; err != nil {
				return fmt.Errorf("migrate OAuth type: %w", err)
			}
			if err := db.Exec("update oauths set issuer = 'https://accounts.google.com' where op = 'google'").Error; err != nil {
				return fmt.Errorf("migrate Google OAuth issuer: %w", err)
			}
			if err := db.Exec("update user_thirds set oauth_type = third_type, op = third_type").Error; err != nil {
				return fmt.Errorf("migrate third-party OAuth type: %w", err)
			}
			// Migrate old Google authorization data through email.
			uts := make([]model.UserThird, 0)
			if err := db.Where("oauth_type = ?", "google").Find(&uts).Error; err != nil {
				return fmt.Errorf("read Google OAuth users during migration: %w", err)
			}
			for _, ut := range uts {
				if ut.UserId > 0 {
					if err := db.Model(&model.User{}).Where("id = ?", ut.UserId).Update("email", ut.OpenId).Error; err != nil {
						return fmt.Errorf("migrate Google OAuth user %d: %w", ut.UserId, err)
					}
				}
			}
		}
		if v.Version < 246 {
			if err := db.Exec("update oauths set issuer = 'https://accounts.google.com' where op = 'google' and issuer is null").Error; err != nil {
				return fmt.Errorf("migrate missing Google OAuth issuer: %w", err)
			}
		}
	}

	// Fallback migration: ensure all new tables exist. AutoMigrate is idempotent and does not modify existing tables/columns.
	// This fixes upgrades from old versions where Migrate could be skipped because the version record was already current,
	// leaving newly added tables such as app_releases or station_messages missing.
	// AutoMigrate each model separately so one failure does not block the others.
	fallbackModels := []interface{}{
		&model.Version{},
		&model.AppRelease{},
		&model.User{},
		&model.UserToken{},
		&model.Tag{},
		&model.AddressBook{},
		&model.Peer{},
		&model.Group{},
		&model.UserThird{},
		&model.Oauth{},
		&model.LoginLog{},
		&model.ShareRecord{},
		&model.AuditConn{},
		&model.AuditFile{},
		&model.AddressBookCollection{},
		&model.AddressBookCollectionRule{},
		&model.ServerCmd{},
		&model.DeviceGroup{},
		&model.AlertChannel{},
		&model.AlertConfig{},
		&model.AlertTarget{},
		&model.StationMessage{},
		&model.ClientDownload{},
		&model.Strategy{},
		&model.ProcessMonitorRule{},
		&model.ProcessMonitorRulePeer{},
		&model.ProcessMonitorStatus{},
		&model.ServerStatusMonitor{},
		&model.PayOrder{},
		&model.InviteCode{},
		&model.Announcement{},
	}
	for _, m := range fallbackModels {
		if err := db.AutoMigrate(m); err != nil {
			return fmt.Errorf("fallback migrate %T: %w", m, err)
		}
	}

	// The legacy raw SQL is SQLite-specific. PostgreSQL and MySQL use GORM's
	// dialect-aware migration above and must never receive AUTOINCREMENT syntax.
	if global.Config.Gorm.Type == config.TypeSqlite {
		if err := ensureTable(db, &model.AppRelease{}, "app_releases", `CREATE TABLE IF NOT EXISTS app_releases (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		version varchar(32) NOT NULL DEFAULT '',
		platform varchar(16) NOT NULL DEFAULT '',
		url varchar(512) NOT NULL DEFAULT '',
		note text,
		status tinyint DEFAULT 1,
		created_at datetime,
		updated_at datetime
	)`); err != nil {
			return err
		}
		if err := ensureTable(db, &model.StationMessage{}, "station_messages", `CREATE TABLE IF NOT EXISTS station_messages (
		row_id INTEGER PRIMARY KEY AUTOINCREMENT,
		type varchar(32) NOT NULL DEFAULT '',
		title varchar(200) NOT NULL DEFAULT '',
		content text,
		peer_id varchar(128) NOT NULL DEFAULT '',
		sender_id integer NOT NULL DEFAULT 0,
		sender_name varchar(100) NOT NULL DEFAULT '',
		receiver_id integer NOT NULL DEFAULT 0,
		is_read integer NOT NULL DEFAULT 0,
		created_at integer
	)`); err != nil {
			return err
		}
		if err := ensureTable(db, &model.ClientDownload{}, "client_downloads", `CREATE TABLE IF NOT EXISTS client_downloads (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		version varchar(32) NOT NULL DEFAULT '',
		platform varchar(16) NOT NULL DEFAULT '',
		url varchar(512) NOT NULL DEFAULT '',
		note text,
		status tinyint DEFAULT 1,
		created_at datetime,
		updated_at datetime
	)`); err != nil {
			return err
		}
	}
	return nil
}

func ensureTable(db *gorm.DB, m interface{}, tableName, createSQL string) error {
	if !db.Migrator().HasTable(m) {
		global.Logger.Warnf("table %s does not exist; trying to create it with raw SQL...", tableName)
		if err := db.Exec(createSQL).Error; err != nil {
			return fmt.Errorf("create table %s with SQLite fallback: %w", tableName, err)
		} else {
			global.Logger.Infof("created table %s with raw SQL successfully", tableName)
		}
	}
	return nil
}
func Migrate(version uint) error {
	global.Logger.Info("Migrating....", version)
	err := global.DB.AutoMigrate(
		&model.Version{},
		&model.AppRelease{},
		&model.User{},
		&model.UserToken{},
		&model.Tag{},
		&model.AddressBook{},
		&model.Peer{},
		&model.Group{},
		&model.UserThird{},
		&model.Oauth{},
		&model.LoginLog{},
		&model.ShareRecord{},
		&model.AuditConn{},
		&model.AuditFile{},
		&model.AddressBookCollection{},
		&model.AddressBookCollectionRule{},
		&model.ServerCmd{},
		&model.DeviceGroup{},
		&model.AlertChannel{},
		&model.AlertConfig{},
		&model.AlertTarget{},
		&model.StationMessage{},
		&model.ClientDownload{},
		&model.Strategy{},
		&model.ProcessMonitorRule{},
		&model.ProcessMonitorRulePeer{},
		&model.ProcessMonitorStatus{},
		&model.ServerStatusMonitor{},
		&model.PayOrder{},
		&model.InviteCode{},
	)
	if err != nil {
		return fmt.Errorf("migrate database schema: %w", err)
	}
	if err := global.DB.Create(&model.Version{Version: version}).Error; err != nil {
		return fmt.Errorf("record database schema version: %w", err)
	}
	// Create a default user on first initialization.
	var vc int64
	if err := global.DB.Model(&model.Version{}).Count(&vc).Error; err != nil {
		return fmt.Errorf("count database schema versions: %w", err)
	}
	if vc == 1 {
		localizer := global.Localizer("")
		defaultGroup, _ := localizer.LocalizeMessage(&i18n.Message{
			ID: "DefaultGroup",
		})
		group := &model.Group{
			Name: defaultGroup,
			Type: model.GroupTypeDefault,
		}
		service.AllService.GroupService.Create(group)

		shareGroup, _ := localizer.LocalizeMessage(&i18n.Message{
			ID: "ShareGroup",
		})
		groupShare := &model.Group{
			Name: shareGroup,
			Type: model.GroupTypeShare,
		}
		service.AllService.GroupService.Create(groupShare)
		// Set admin flag.
		is_admin := true
		admin := &model.User{
			Username: "admin",
			Nickname: "Admin",
			Status:   model.COMMON_STATUS_ENABLE,
			IsAdmin:  &is_admin,
			GroupId:  1,
		}

		// Generate a random password.
		pwd := utils.RandomString(8)
		// SECURITY: The initial admin password is a highly sensitive credential and must never be written to file logs (./runtime/log.txt).
		// Otherwise, anyone who can read the log file could obtain the admin password. Print it once to stderr (console) only,
		// without persisting it to log files, and prompt the operator to change it after the first login.
		fmt.Fprintf(os.Stderr, "\n[INIT] Generated initial admin password: %s\n[INIT] Please change it after first login!\n\n", pwd)
		var err error
		admin.Password, err = utils.EncryptPassword(pwd)
		if err != nil {
			global.Logger.Fatalf("failed to generate admin password: %v", err)
		}
		if err := global.DB.Create(admin).Error; err != nil {
			return fmt.Errorf("create initial administrator: %w", err)
		}
	}
	return nil
}
