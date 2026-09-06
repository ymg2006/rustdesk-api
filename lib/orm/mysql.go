package orm

import (
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type MysqlConfig struct {
	Dsn string
	PoolConfig
}

func NewMysql(mysqlConf *MysqlConfig, logwriter logger.Writer) (*gorm.DB, error) {
	db, err := gorm.Open(mysql.New(mysql.Config{
		DSN:               mysqlConf.Dsn, // DSN data source name
		DefaultStringSize: 256,           // The default length of string type fields
		//DisableDatetimePrecision: true, // Disable datetime precision, which is not supported by databases before MySQL 5.6
		//DontSupportRenameIndex: true, // When renaming the index, delete and create a new one. Databases before MySQL 5.7 and MariaDB do not support renaming indexes.
		//DontSupportRenameColumn: true, // Use`change`to rename columns. Databases before MySQL 8 and MariaDB do not support renaming columns.
		//SkipInitializeWithVersion: false, // Automatically configure according to the current MySQL version
	}), &gorm.Config{
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
		return nil, fmt.Errorf("open MySQL database: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get MySQL connection pool: %w", err)
	}
	if err := configureAndPing(sqlDB, mysqlConf.PoolConfig); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("validate MySQL database: %w", err)
	}
	return db, nil
}
