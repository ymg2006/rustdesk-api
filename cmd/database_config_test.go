package main

import (
	"net/url"
	"strings"
	"testing"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/ymg2006/rustdesk-api/v2/config"
)

func TestBuildPostgresqlDSN(t *testing.T) {
	passwords := []string{"secret", "p@ss", "a:b", "a/b", "a#b", "with space", "%&=", "'\"", ""}
	for _, password := range passwords {
		t.Run(password, func(t *testing.T) {
			dsn := buildPostgresqlDSN(config.Postgresql{
				Host: "postgres", Port: "5432", User: "rustdesk", Password: password,
				Dbname: "rustdesk_prod", Sslmode: "verify-full", TimeZone: "Europe/Paris",
			}, 7*time.Second)
			u, err := url.Parse(dsn)
			if err != nil {
				t.Fatal(err)
			}
			gotPassword, hasPassword := u.User.Password()
			if u.User.Username() != "rustdesk" || gotPassword != password || hasPassword != (password != "") {
				t.Fatalf("credentials did not round trip: user=%q password=%q set=%v", u.User.Username(), gotPassword, hasPassword)
			}
			if u.Hostname() != "postgres" || u.Port() != "5432" || strings.TrimPrefix(u.Path, "/") != "rustdesk_prod" {
				t.Fatalf("connection target did not round trip: %s", u.Redacted())
			}
			if u.Query().Get("sslmode") != "verify-full" || u.Query().Get("TimeZone") != "Europe/Paris" || u.Query().Get("connect_timeout") != "7" {
				t.Fatalf("query = %v", u.Query())
			}
		})
	}
}

func TestBuildPostgresqlDSNIPv6(t *testing.T) {
	dsn := buildPostgresqlDSN(config.Postgresql{Host: "2001:db8::1", Port: "5432", User: "app", Dbname: "db", Sslmode: "disable"}, time.Second)
	u, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	if u.Hostname() != "2001:db8::1" || u.Port() != "5432" {
		t.Fatalf("IPv6 target = %q", u.Host)
	}
}

func TestBuildMySQLDSNSpecialPassword(t *testing.T) {
	password := "p@ss:/?#%&= '"
	dsn := buildMySQLDSN(config.Mysql{Addr: "mysql:3306", Username: "app", Password: password, Tls: "false"}, 5*time.Second, "rustdesk")
	parsed, err := mysqldriver.ParseDSN(dsn)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Passwd != password || parsed.Addr != "mysql:3306" || parsed.DBName != "rustdesk" {
		t.Fatalf("MySQL DSN did not round trip")
	}
}

func TestRedactSecret(t *testing.T) {
	secret := "p@ss:/?#%&= '"
	encoded := strings.TrimPrefix(url.UserPassword("", secret).String(), ":")
	for _, message := range []string{"connect password=" + secret + " failed", "postgres://app:" + encoded + "@host/db"} {
		if got := redactSecret(&testError{message}, secret); strings.Contains(got, secret) || strings.Contains(got, encoded) {
			t.Fatalf("redacted error leaked secret: %s", got)
		}
	}
}

type testError struct{ message string }

func (e *testError) Error() string { return e.message }
