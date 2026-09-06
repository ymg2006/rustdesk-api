package orm

import (
	"fmt"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type SqliteConfig struct {
	Path string
	PoolConfig
}

func NewSqlite(sqliteConf *SqliteConfig, logwriter logger.Writer) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(sqliteConf.Path), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		Logger: logger.New(
			logwriter, // io writer
			logger.Config{
				SlowThreshold:             time.Second, // Slow SQL threshold
				LogLevel:                  logger.Warn, // Log level
				IgnoreRecordNotFoundError: true,        // Ignore ErrRecordNotFound error for logger
				ParameterizedQueries:      true,        // Don't include params in the SQL log
				Colorful:                  true,
			},
		),
	})
	if err != nil {
		return nil, fmt.Errorf("open SQLite database: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get SQLite connection pool: %w", err)
	}
	if err := configureAndPing(sqlDB, sqliteConf.PoolConfig); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("validate SQLite database: %w", err)
	}
	return db, nil
}
