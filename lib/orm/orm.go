package orm

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type PoolConfig struct {
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
	ConnectTimeout  time.Duration
}

func configureAndPing(sqlDB *sql.DB, conf PoolConfig) error {
	sqlDB.SetMaxIdleConns(conf.MaxIdleConns)
	sqlDB.SetMaxOpenConns(conf.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(conf.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(conf.ConnMaxIdleTime)

	ctx, cancel := context.WithTimeout(context.Background(), conf.ConnectTimeout)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}
	return nil
}
