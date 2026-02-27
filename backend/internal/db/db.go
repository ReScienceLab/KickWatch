package db

import (
	"fmt"
	"log"
	"time"

	"github.com/kickwatch/backend/internal/config"
	"github.com/kickwatch/backend/internal/model"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Init(cfg *config.Config) error {
	if cfg.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}

	var err error
	DB, err = gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return fmt.Errorf("connect db: %w", err)
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("get sql.DB: %w", err)
	}
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetMaxOpenConns(20)
	sqlDB.SetConnMaxLifetime(time.Hour)

	if err := DB.AutoMigrate(
		&model.Campaign{},
		&model.CampaignSnapshot{},
		&model.Category{},
		&model.Device{},
		&model.Alert{},
		&model.AlertMatch{},
	); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	// Ensure campaigns.pid has a primary key constraint (AutoMigrate does not
	// add PKs to pre-existing tables; this DO block is idempotent).
	if err := DB.Exec(`
		DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM information_schema.table_constraints
				WHERE table_name = 'campaigns' AND constraint_type = 'PRIMARY KEY'
			) THEN
				ALTER TABLE campaigns ADD PRIMARY KEY (pid);
			END IF;
		END
		$$;
	`).Error; err != nil {
		return fmt.Errorf("migrate campaigns pk: %w", err)
	}

	log.Println("Database connected and migrated")
	return nil
}

func IsEnabled() bool {
	return DB != nil
}
