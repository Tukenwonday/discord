// Package database opens the GORM connection to Postgres, waits for it to
// become reachable and keeps the schema up to date.
package database

import (
	"fmt"
	"time"

	"github.com/cordis/backend/internal/models"
	"github.com/rs/zerolog"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// AllModels is the ordered list of entities managed by AutoMigrate.
func AllModels() []any {
	return []any{
		&models.User{},
		&models.Server{},
		&models.Role{},
		&models.Member{},
		&models.MemberRole{},
		&models.Channel{},
		&models.Message{},
		&models.Attachment{},
		&models.Reaction{},
		&models.DirectMessage{},
		&models.DMRecipient{},
		&models.Friend{},
		&models.Invite{},
	}
}

// Open dials Postgres, retrying with exponential backoff so the process can
// start alongside its database container.
func Open(dsn string, log zerolog.Logger) (*gorm.DB, error) {
	gormCfg := &gorm.Config{
		Logger:                                   gormlogger.Default.LogMode(gormlogger.Silent),
		DisableForeignKeyConstraintWhenMigrating: false,
		NowFunc:                                  func() time.Time { return time.Now().UTC() },
	}
	var lastErr error
	backoff := time.Second
	for attempt := 1; attempt <= 10; attempt++ {
		db, err := gorm.Open(postgres.Open(dsn), gormCfg)
		if err == nil {
			sqlDB, dbErr := db.DB()
			if dbErr == nil {
				if pingErr := sqlDB.Ping(); pingErr == nil {
					sqlDB.SetMaxOpenConns(40)
					sqlDB.SetMaxIdleConns(10)
					sqlDB.SetConnMaxLifetime(time.Hour)
					return db, nil
				} else {
					lastErr = pingErr
				}
			} else {
				lastErr = dbErr
			}
			_ = db
		} else {
			lastErr = err
		}
		log.Warn().Err(lastErr).Int("attempt", attempt).Msg("waiting for postgres")
		if attempt == 10 {
			break
		}
		time.Sleep(backoff)
		if backoff < 15*time.Second {
			backoff *= 2
		}
	}
	return nil, fmt.Errorf("connect postgres: %w", lastErr)
}

// AutoMigrate creates or updates every table the backend owns.
func AutoMigrate(db *gorm.DB) error {
	if err := db.AutoMigrate(AllModels()...); err != nil {
		return fmt.Errorf("automigrate: %w", err)
	}
	return nil
}

// Ping verifies the database is reachable, used by the readiness probe.
func Ping(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("database handle is nil")
	}
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("database handle: %w", err)
	}
	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("database ping: %w", err)
	}
	return nil
}
