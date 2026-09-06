package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

var databaseAndCacheEnv = []string{
	"RUSTDESK_API_CACHE_TYPE", "RUSTDESK_API_CACHE_FILE_DIR", "RUSTDESK_API_CACHE_REDIS_ADDR",
	"RUSTDESK_API_CACHE_REDIS_PWD", "RUSTDESK_API_CACHE_REDIS_DB", "RUSTDESK_API_GORM_TYPE",
	"RUSTDESK_API_CACHE_CONNECT_TIMEOUT",
	"RUSTDESK_API_GORM_MAX_IDLE_CONNS", "RUSTDESK_API_GORM_MAX_OPEN_CONNS",
	"RUSTDESK_API_GORM_CONN_MAX_LIFETIME", "RUSTDESK_API_GORM_CONN_MAX_IDLE_TIME",
	"RUSTDESK_API_GORM_CONNECT_TIMEOUT", "RUSTDESK_API_MYSQL_ADDR", "RUSTDESK_API_MYSQL_USERNAME",
	"RUSTDESK_API_MYSQL_PASSWORD", "RUSTDESK_API_MYSQL_DBNAME", "RUSTDESK_API_MYSQL_TLS",
	"RUSTDESK_API_POSTGRESQL_HOST", "RUSTDESK_API_POSTGRESQL_PORT", "RUSTDESK_API_POSTGRESQL_USER",
	"RUSTDESK_API_POSTGRESQL_PASSWORD", "RUSTDESK_API_POSTGRESQL_DBNAME",
	"RUSTDESK_API_POSTGRESQL_SSLMODE", "RUSTDESK_API_POSTGRESQL_TIME_ZONE",
}

func clearConfigEnvironment(t *testing.T) {
	t.Helper()
	for _, key := range databaseAndCacheEnv {
		value, exists := os.LookupEnv(key)
		if err := os.Unsetenv(key); err != nil {
			t.Fatal(err)
		}
		key, value, exists := key, value, exists
		t.Cleanup(func() {
			if exists {
				_ = os.Setenv(key, value)
			} else {
				_ = os.Unsetenv(key)
			}
		})
	}
}

func loadTestConfig(t *testing.T, yaml string) Config {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0600); err != nil {
		t.Fatal(err)
	}
	var cfg Config
	if _, err := Load(&cfg, path); err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	return cfg
}

func TestLoadCacheDefaultsWithoutYAMLSection(t *testing.T) {
	clearConfigEnvironment(t)
	cfg := loadTestConfig(t, "lang: en-US\n")
	if cfg.Cache.Type != "file" || cfg.Cache.FileDir != "./runtime/cache" {
		t.Fatalf("cache defaults = %#v", cfg.Cache)
	}
	if cfg.Gorm.Type != TypeSqlite {
		t.Fatalf("default database = %q, want sqlite", cfg.Gorm.Type)
	}
}

func TestCheckedInConfigurationLoads(t *testing.T) {
	clearConfigEnvironment(t)
	var cfg Config
	if _, err := Load(&cfg, filepath.Join("..", "conf", "config.yaml")); err != nil {
		t.Fatalf("checked-in config does not load: %v", err)
	}
}

func TestLoadYAMLCacheConfiguration(t *testing.T) {
	clearConfigEnvironment(t)
	cfg := loadTestConfig(t, "cache:\n  type: memory\n  file-dir: ./custom-cache\n")
	if cfg.Cache.Type != "memory" || cfg.Cache.FileDir != "./custom-cache" {
		t.Fatalf("cache config = %#v", cfg.Cache)
	}
}

func TestLoadEnvironmentOnlyRedisConfiguration(t *testing.T) {
	clearConfigEnvironment(t)
	t.Setenv("RUSTDESK_API_CACHE_TYPE", "redis")
	t.Setenv("RUSTDESK_API_CACHE_REDIS_ADDR", "dragonfly:6379")
	t.Setenv("RUSTDESK_API_CACHE_REDIS_PWD", "test-password")
	t.Setenv("RUSTDESK_API_CACHE_REDIS_DB", "1")
	cfg := loadTestConfig(t, "lang: en-US\n")

	if cfg.Cache.Type != "redis" || cfg.Cache.RedisAddr != "dragonfly:6379" ||
		cfg.Cache.RedisPwd != "test-password" || cfg.Cache.RedisDb != 1 {
		t.Fatalf("environment-only Redis config = %#v", cfg.Cache)
	}
}

func TestEnvironmentOverridesYAMLCache(t *testing.T) {
	clearConfigEnvironment(t)
	t.Setenv("RUSTDESK_API_CACHE_TYPE", "redis")
	t.Setenv("RUSTDESK_API_CACHE_REDIS_ADDR", "dragonfly:6379")
	t.Setenv("RUSTDESK_API_CACHE_REDIS_PWD", "secret")
	t.Setenv("RUSTDESK_API_CACHE_REDIS_DB", "1")
	cfg := loadTestConfig(t, "cache:\n  type: file\n  redis-addr: old:6379\n  redis-pwd: old\n  redis-db: 0\n")
	if cfg.Cache.RedisDb != 1 || cfg.Cache.RedisPwd != "secret" || cfg.Cache.RedisAddr != "dragonfly:6379" {
		t.Fatalf("overridden Redis config = %#v", cfg.Cache)
	}
}

func TestLoadEnvironmentOnlyPostgresqlConfiguration(t *testing.T) {
	clearConfigEnvironment(t)
	t.Setenv("RUSTDESK_API_GORM_TYPE", "postgres")
	t.Setenv("RUSTDESK_API_POSTGRESQL_HOST", "postgres")
	t.Setenv("RUSTDESK_API_POSTGRESQL_PORT", "5544")
	t.Setenv("RUSTDESK_API_POSTGRESQL_USER", "rustdesk")
	t.Setenv("RUSTDESK_API_POSTGRESQL_PASSWORD", "p@ss:/?#%&= '")
	t.Setenv("RUSTDESK_API_POSTGRESQL_DBNAME", "rustdesk_prod")
	t.Setenv("RUSTDESK_API_POSTGRESQL_SSLMODE", "require")
	t.Setenv("RUSTDESK_API_POSTGRESQL_TIME_ZONE", "Europe/Paris")
	cfg := loadTestConfig(t, "lang: en-US\n")

	if cfg.Gorm.Type != TypePostgresql || cfg.Postgresql.Host != "postgres" || cfg.Postgresql.Port != "5544" ||
		cfg.Postgresql.User != "rustdesk" || cfg.Postgresql.Password != "p@ss:/?#%&= '" ||
		cfg.Postgresql.Dbname != "rustdesk_prod" || cfg.Postgresql.Sslmode != "require" ||
		cfg.Postgresql.TimeZone != "Europe/Paris" {
		t.Fatalf("environment-only PostgreSQL config = %#v / type=%q", cfg.Postgresql, cfg.Gorm.Type)
	}
}

func TestPostgresqlEnvironmentOverridesYAML(t *testing.T) {
	clearConfigEnvironment(t)
	t.Setenv("RUSTDESK_API_GORM_TYPE", "postgresql")
	t.Setenv("RUSTDESK_API_POSTGRESQL_HOST", "env-postgres")
	t.Setenv("RUSTDESK_API_POSTGRESQL_PORT", "5433")
	t.Setenv("RUSTDESK_API_POSTGRESQL_USER", "env-user")
	t.Setenv("RUSTDESK_API_POSTGRESQL_PASSWORD", "env-password")
	t.Setenv("RUSTDESK_API_POSTGRESQL_DBNAME", "env-db")
	t.Setenv("RUSTDESK_API_POSTGRESQL_SSLMODE", "verify-full")
	cfg := loadTestConfig(t, "gorm:\n  type: sqlite\npostgresql:\n  host: yaml-host\n  port: '5432'\n  user: yaml-user\n  dbname: yaml-db\n  sslmode: disable\n")
	if cfg.Gorm.Type != TypePostgresql || cfg.Postgresql.Host != "env-postgres" || cfg.Postgresql.Port != "5433" ||
		cfg.Postgresql.User != "env-user" || cfg.Postgresql.Password != "env-password" ||
		cfg.Postgresql.Dbname != "env-db" || cfg.Postgresql.Sslmode != "verify-full" {
		t.Fatalf("overridden PostgreSQL config = %#v / type=%q", cfg.Postgresql, cfg.Gorm.Type)
	}
}

func TestLoadMySQLConfiguration(t *testing.T) {
	clearConfigEnvironment(t)
	cfg := loadTestConfig(t, "gorm:\n  type: mysql\nmysql:\n  addr: mysql:3306\n  username: app\n  password: secret\n  dbname: rustdesk\n  tls: 'true'\n")
	if cfg.Gorm.Type != TypeMysql || cfg.Mysql.Addr != "mysql:3306" || cfg.Mysql.Dbname != "rustdesk" {
		t.Fatalf("MySQL config = %#v / type=%q", cfg.Mysql, cfg.Gorm.Type)
	}
}

func TestPoolDurationEnvironment(t *testing.T) {
	clearConfigEnvironment(t)
	t.Setenv("RUSTDESK_API_GORM_CONN_MAX_LIFETIME", "30m")
	t.Setenv("RUSTDESK_API_GORM_CONN_MAX_IDLE_TIME", "2m")
	t.Setenv("RUSTDESK_API_GORM_CONNECT_TIMEOUT", "5s")
	cfg := loadTestConfig(t, "lang: en-US\n")
	if cfg.Gorm.ConnMaxLifetime != 30*time.Minute || cfg.Gorm.ConnMaxIdleTime != 2*time.Minute || cfg.Gorm.ConnectTimeout != 5*time.Second {
		t.Fatalf("pool durations = %#v", cfg.Gorm)
	}
}

func TestInvalidBackendConfiguration(t *testing.T) {
	clearConfigEnvironment(t)
	for _, tc := range []struct{ name, yaml string }{
		{"cache", "cache:\n  type: foobar\n"},
		{"database", "gorm:\n  type: oracle\n"},
		{"postgres-port", "gorm:\n  type: postgresql\npostgresql:\n  host: postgres\n  port: nope\n  user: app\n  dbname: rustdesk\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.yaml")
			if err := os.WriteFile(path, []byte(tc.yaml), 0600); err != nil {
				t.Fatal(err)
			}
			var cfg Config
			if _, err := Load(&cfg, path); err == nil {
				t.Fatal("Load() succeeded for invalid configuration")
			}
		})
	}
}
