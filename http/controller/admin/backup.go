package admin

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ymg2006/rustdesk-api/v2/config"
	"github.com/ymg2006/rustdesk-api/v2/global"
	"github.com/ymg2006/rustdesk-api/v2/http/response"
)

type BackupCtl struct{}

// Config export configuration file
func (b *BackupCtl) Config(ctx *gin.Context) {
	cfgPath := global.ConfigPath
	if cfgPath == "" {
		cfgPath = config.DefaultConfig
	}
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		response.Fail(ctx, 500, response.TranslateMsg(ctx, "ReadConfigFailed")+err.Error())
		return
	}

	filename := fmt.Sprintf("rustdesk-api-config-%s.yaml", time.Now().Format("20060102_150405"))
	ctx.Header("Content-Type", "application/octet-stream")
	ctx.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	ctx.Data(200, "application/octet-stream", data)
}

// Database export database backup
func (b *BackupCtl) Database(ctx *gin.Context) {
	dbType := global.Config.Gorm.Type

	var filename string
	var data []byte
	var err error

	switch dbType {
	case config.TypeSqlite:
		filename = fmt.Sprintf("rustdesk-api-db-%s.db", time.Now().Format("20060102_150405"))
		data, err = b.exportSqlite()
	case config.TypeMysql:
		filename = fmt.Sprintf("rustdesk-api-db-%s.sql", time.Now().Format("20060102_150405"))
		data, err = b.exportMysql()
	case config.TypePostgresql:
		filename = fmt.Sprintf("rustdesk-api-db-%s.sql", time.Now().Format("20060102_150405"))
		data, err = b.exportPostgresql()
	default:
		response.Fail(ctx, 400, response.TranslateMsg(ctx, "UnsupportedDbType")+dbType)
		return
	}

	if err != nil {
		response.Fail(ctx, 500, response.TranslateMsg(ctx, "ExportDbFailed")+err.Error())
		return
	}

	ctx.Header("Content-Type", "application/octet-stream")
	ctx.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	ctx.Data(200, "application/octet-stream", data)
}

func (b *BackupCtl) exportSqlite() ([]byte, error) {
	dbPath := "./data/rustdeskapi.db"
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		// Try looking from the working directory or other common path
		absPath, _ := filepath.Abs(dbPath)
		if _, err := os.Stat(absPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("Database file does not exist:%s", dbPath)
		}
		dbPath = absPath
	}
	return os.ReadFile(dbPath)
}

func (b *BackupCtl) exportMysql() ([]byte, error) {
	mysql := global.Config.Mysql
	addr := mysql.Addr
	if addr == "" {
		addr = "127.0.0.1:3306"
	}
	host := addr
	port := "3306"
	if parts := strings.SplitN(addr, ":", 2); len(parts) == 2 {
		host = parts[0]
		port = parts[1]
	}

	args := []string{
		"-h" + host,
		"-P" + port,
		"-u" + mysql.Username,
		"-p" + mysql.Password,
		mysql.Dbname,
	}

	cmd := exec.Command("mysqldump", args...)
	return cmd.Output()
}

func (b *BackupCtl) exportPostgresql() ([]byte, error) {
	pg := global.Config.Postgresql
	port := pg.Port
	if port == "" {
		port = "5432"
	}

	// Pass the password using the PGPASSWORD environment variable
	cmd := exec.Command("pg_dump",
		"-h"+pg.Host,
		"-p"+port,
		"-U"+pg.User,
		"-d"+pg.Dbname,
	)
	cmd.Env = append(os.Environ(), "PGPASSWORD="+pg.Password)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("Failed to create stderr pipe:%w", err)
	}

	out, err := cmd.Output()
	if err != nil {
		errMsg, _ := io.ReadAll(stderr)
		if len(errMsg) > 0 {
			return nil, fmt.Errorf("pg_dump failed:%s", string(errMsg))
		}
		return nil, fmt.Errorf("pg_dump execution failed:%w", err)
	}
	return out, nil
}
