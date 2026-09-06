package orm

import (
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type PostgresqlConfig struct {
	Dsn string
	PoolConfig
}

func NewPostgresql(conf *PostgresqlConfig, logwriter logger.Writer) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(conf.Dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		Logger: logger.New(
			logwriter, // io writer
			logger.Config{
				SlowThreshold: time.Second, // Slow SQL threshold
				LogLevel:      logger.Warn, // Log level
				//IgnoreRecordNotFoundError: true,        // Ignore ErrRecordNotFound error for logger
				ParameterizedQueries: true, // Don't include params in the SQL log
				Colorful:             true,
			},
		),
	})
	if err != nil {
		return nil, fmt.Errorf("open PostgreSQL database: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get PostgreSQL connection pool: %w", err)
	}
	if err := configureAndPing(sqlDB, conf.PoolConfig); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("validate PostgreSQL database: %w", err)
	}
	return db, nil
}
