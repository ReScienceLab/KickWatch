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

	// CRITICAL: Run all column renames BEFORE AutoMigrate
	// Otherwise GORM creates empty columns with NOT NULL constraints that fail

	// Fix campaigns table: the original column was p_id (GORM snake_case of PID).
	// After adding gorm:"column:pid", AutoMigrate would create a separate pid column.
	// This migration handles all transition states idempotently.
	if err := DB.Exec(`
		DO $$
		BEGIN
			-- Case 1: p_id exists but pid does not — simple rename
			IF EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_name = 'campaigns' AND column_name = 'p_id'
			) AND NOT EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_name = 'campaigns' AND column_name = 'pid'
			) THEN
				ALTER TABLE campaigns RENAME COLUMN p_id TO pid;

			-- Case 2: both columns exist — copy data across, drop old column
			ELSIF EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_name = 'campaigns' AND column_name = 'p_id'
			) AND EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_name = 'campaigns' AND column_name = 'pid'
			) THEN
				UPDATE campaigns SET pid = p_id WHERE pid IS NULL OR pid = '';
				ALTER TABLE campaigns DROP CONSTRAINT IF EXISTS campaigns_pkey;
				ALTER TABLE campaigns DROP COLUMN IF EXISTS p_id;
			END IF;

			-- Ensure pid is the primary key (check column specifically, not just any PK)
			IF NOT EXISTS (
				SELECT 1 FROM information_schema.table_constraints tc
				JOIN information_schema.key_column_usage kcu
					ON tc.constraint_name = kcu.constraint_name
					AND tc.table_schema = kcu.table_schema
				WHERE tc.table_name = 'campaigns'
					AND tc.constraint_type = 'PRIMARY KEY'
					AND kcu.column_name = 'pid'
			) THEN
				ALTER TABLE campaigns DROP CONSTRAINT IF EXISTS campaigns_pkey;
				ALTER TABLE campaigns ADD PRIMARY KEY (pid);
			END IF;
		END
		$$;
	`).Error; err != nil {
		return fmt.Errorf("migrate campaigns pk: %w", err)
	}

	// Ensure velocity_24h and pledge_delta_24h columns exist (added in develop branch)
	if err := DB.Exec(`
		DO $$
		BEGIN
			-- Add velocity_24h column if it doesn't exist
			IF NOT EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_name = 'campaigns' AND column_name = 'velocity_24h'
			) THEN
				ALTER TABLE campaigns ADD COLUMN velocity_24h DOUBLE PRECISION DEFAULT 0;
			END IF;

			-- Add pledge_delta_24h column if it doesn't exist (named ple_delta_24h in DB due to GORM naming)
			IF NOT EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_name = 'campaigns' AND column_name = 'ple_delta_24h'
			) THEN
				ALTER TABLE campaigns ADD COLUMN ple_delta_24h DOUBLE PRECISION DEFAULT 0;
			END IF;
		END
		$$;
	`).Error; err != nil {
		return fmt.Errorf("migrate velocity columns: %w", err)
	}

	// Fix campaign_snapshots FK: rename campaign_p_id -> campaign_pid (same issue as campaigns.pid)
	if err := DB.Exec(`
		DO $$
		BEGIN
			-- Case 1: campaign_p_id exists but campaign_pid does not — simple rename
			IF EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_name = 'campaign_snapshots' AND column_name = 'campaign_p_id'
			) AND NOT EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_name = 'campaign_snapshots' AND column_name = 'campaign_pid'
			) THEN
				ALTER TABLE campaign_snapshots RENAME COLUMN campaign_p_id TO campaign_pid;

			-- Case 2: both columns exist — copy data from old to new, drop old column
			ELSIF EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_name = 'campaign_snapshots' AND column_name = 'campaign_p_id'
			) AND EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_name = 'campaign_snapshots' AND column_name = 'campaign_pid'
			) THEN
				UPDATE campaign_snapshots SET campaign_pid = campaign_p_id WHERE campaign_pid IS NULL OR campaign_pid = '';
				ALTER TABLE campaign_snapshots DROP COLUMN IF EXISTS campaign_p_id;
			END IF;
		END
		$$;
	`).Error; err != nil {
		return fmt.Errorf("migrate campaign_snapshots fk: %w", err)
	}

	// NOW run AutoMigrate after all column renames are complete
	if err := DB.AutoMigrate(
		&model.Campaign{},
		&model.CampaignSnapshot{},
		&model.Category{},
		&model.Device{},
		&model.Alert{},
		&model.AlertMatch{},
	); err != nil {
		return fmt.Errorf("automigrate: %w", err)
	}

	// Create composite indexes for optimal query performance
	// All queries filter WHERE state='live' AND deadline >= NOW(), so deadline comes first
	// to exclude expired rows early and avoid table scans as expired data grows
	if err := DB.Exec(`
		-- Trending: WHERE state='live' AND deadline >= NOW() ORDER BY velocity_24h DESC, percent_funded DESC
		CREATE INDEX IF NOT EXISTS idx_campaigns_trending 
		ON campaigns(state, deadline, velocity_24h DESC, percent_funded DESC);
		
		-- Newest: WHERE state='live' AND deadline >= NOW() ORDER BY first_seen_at DESC
		CREATE INDEX IF NOT EXISTS idx_campaigns_newest 
		ON campaigns(state, deadline, first_seen_at DESC);
		
		-- Ending: WHERE state='live' AND deadline >= NOW() ORDER BY deadline ASC
		CREATE INDEX IF NOT EXISTS idx_campaigns_ending 
		ON campaigns(state, deadline ASC);
		
		-- Category+Trending: WHERE state='live' AND deadline >= NOW() AND category_id=? ORDER BY velocity_24h DESC, percent_funded DESC
		CREATE INDEX IF NOT EXISTS idx_campaigns_category_trending 
		ON campaigns(state, deadline, category_id, velocity_24h DESC, percent_funded DESC);
		
		-- Category+Newest: WHERE state='live' AND deadline >= NOW() AND category_id=? ORDER BY first_seen_at DESC
		CREATE INDEX IF NOT EXISTS idx_campaigns_category_newest 
		ON campaigns(state, deadline, category_id, first_seen_at DESC);
		
		-- Category+Ending: WHERE state='live' AND deadline >= NOW() AND category_id=? ORDER BY deadline ASC
		CREATE INDEX IF NOT EXISTS idx_campaigns_category_ending 
		ON campaigns(state, deadline, category_id);
	`).Error; err != nil {
		return fmt.Errorf("create composite indexes: %w", err)
	}

	log.Println("Database connected and migrated")
	return nil
}

func IsEnabled() bool {
	return DB != nil
}
