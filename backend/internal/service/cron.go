package service

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/kickwatch/backend/internal/model"
	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// envInt reads an integer from an env var, returning defaultVal if unset or invalid.
func envInt(key string, defaultVal int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return defaultVal
}

type CronService struct {
	db              *gorm.DB
	scrapingService *KickstarterScrapingService
	apnsClient      *APNsClient
	scheduler       *cron.Cron
}

func NewCronService(db *gorm.DB, scrapingService *KickstarterScrapingService, apns *APNsClient) *CronService {
	return &CronService{
		db:              db,
		scrapingService: scrapingService,
		apnsClient:      apns,
		scheduler:       cron.New(cron.WithLocation(time.UTC)),
	}
}

func (s *CronService) Start() {
	s.scheduler.AddFunc("0 2 * * *", func() {
		log.Println("Cron: starting nightly crawl")
		if err := s.RunCrawlNow(); err != nil {
			log.Printf("Cron: crawl error: %v", err)
		}
	})
	s.scheduler.Start()
	log.Println("Cron scheduler started (02:00 UTC daily)")

	// Log credit balance at startup so we know headroom before the first crawl.
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		usage, err := s.scrapingService.client.FetchUsage(ctx)
		if err != nil {
			log.Printf("ScrapingBee usage check failed: %v", err)
			return
		}
		pct := 0.0
		if usage.MaxCredits > 0 {
			pct = float64(usage.UsedCredits) / float64(usage.MaxCredits) * 100
		}
		log.Printf("ScrapingBee usage: %d/%d credits (%.1f%%), renews %s",
			usage.UsedCredits, usage.MaxCredits, pct, usage.RenewalDate)
		if pct >= 80 {
			log.Printf("WARNING: ScrapingBee credits above 80%% (%d/%d) — consider upgrading plan",
				usage.UsedCredits, usage.MaxCredits)
		}
	}()
}

func (s *CronService) Stop() {
	s.scheduler.Stop()
}

// syncCategories upserts the canonical category list into the DB so that
// clients and alert filters always see the current IDs and subcategories.
func (s *CronService) syncCategories() {
	result := s.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"name", "parent_id"}),
	}).Create(&kickstarterCategories)
	if result.Error != nil {
		log.Printf("Cron: category sync error: %v", result.Error)
	} else {
		log.Printf("Cron: synced %d categories", len(kickstarterCategories))
	}
}

// crawlSorts defines the sort strategies used in each nightly crawl pass.
// Default page depths can be overridden at runtime via env vars:
//
//	CRAWL_DEPTH_NEWEST  (default 10)
//	CRAWL_DEPTH_MAGIC   (default 5)
//	CRAWL_DEPTH_ENDDATE (default 3)
func buildCrawlSorts() []struct {
	sort      string
	pageDepth int
} {
	return []struct {
		sort      string
		pageDepth int
	}{
		{"newest", envInt("CRAWL_DEPTH_NEWEST", 10)},
		{"magic", envInt("CRAWL_DEPTH_MAGIC", 5)},
		{"end_date", envInt("CRAWL_DEPTH_ENDDATE", 3)},
	}
}

func (s *CronService) RunCrawlNow() error {
	s.syncCategories()

	upserted := 0
	seenPIDs := make(map[string]struct{})
	var allCampaigns []model.Campaign

	for _, sortCfg := range buildCrawlSorts() {
		for _, cat := range crawlCategories {
			depth := sortCfg.pageDepth
			// Assign a sticky session_id so all pages for this (sort, category)
			// pass through the same proxy IP — less likely to trigger rate limits.
			sessionID := rand.Intn(10_000_000) + 1
			for page := 1; page <= depth; page++ {
				campaigns, err := s.scrapingService.DiscoverCampaigns(cat.ID, sortCfg.sort, page, sessionID)
				if err != nil {
					log.Printf("Cron: ScrapingBee error sort=%s cat=%s page=%d: %v", sortCfg.sort, cat.ID, page, err)
					break
				}
				if len(campaigns) == 0 {
					break
				}
				now := time.Now()
				for i := range campaigns {
					campaigns[i].LastUpdatedAt = now
				}
				result := s.db.Clauses(clause.OnConflict{
					Columns: []clause.Column{{Name: "pid"}},
					DoUpdates: clause.AssignmentColumns([]string{
						"name", "blurb", "photo_url", "goal_amount", "goal_currency",
						"pledged_amount", "deadline", "state", "category_id", "category_name",
						"project_url", "creator_name", "percent_funded", "backers_count",
						"slug", "last_updated_at",
					}),
				}).Create(&campaigns)
				if result.Error != nil {
					log.Printf("Cron: upsert error: %v", result.Error)
				} else {
					upserted += len(campaigns)
					for _, c := range campaigns {
						seenPIDs[c.PID] = struct{}{}
					}
					allCampaigns = append(allCampaigns, campaigns...)
				}
				time.Sleep(500 * time.Millisecond)
			}
		}
	}
	log.Printf("Cron: crawl done, upserted %d campaigns", upserted)

	// Sanity check: a full crawl across all categories should always yield
	// at least some campaigns. Zero almost certainly means a parse failure
	// (e.g. Kickstarter changed their HTML structure), not a genuinely empty site.
	const minExpectedCampaigns = 50
	if len(seenPIDs) < minExpectedCampaigns {
		log.Printf("ERROR: crawl sanity check FAILED — only %d distinct campaigns seen (expected >=%d). "+
			"Possible HTML structure change or ScrapingBee degradation. "+
			"Check kickstarter_parser.go [data-project] selector.", len(seenPIDs), minExpectedCampaigns)
	}

	if len(allCampaigns) > 0 {
		s.storeSnapshots(allCampaigns)
		s.computeVelocity(allCampaigns)
	}

	return s.matchAlerts()
}

// RunBackfill performs a deep one-time crawl to seed campaigns that pre-date
// the system launch. It uses all sort strategies at configurable depth
// (BACKFILL_DEPTH_NEWEST, BACKFILL_DEPTH_MAGIC, BACKFILL_DEPTH_ENDDATE; defaults 25/15/10).
func (s *CronService) RunBackfill() error {
	sorts := []struct {
		sort  string
		depth int
	}{
		{"newest", envInt("BACKFILL_DEPTH_NEWEST", 25)},
		{"magic", envInt("BACKFILL_DEPTH_MAGIC", 15)},
		{"end_date", envInt("BACKFILL_DEPTH_ENDDATE", 10)},
	}

	upserted := 0
	for _, sortCfg := range sorts {
		for _, cat := range crawlCategories {
			depth := sortCfg.depth
			sessionID := rand.Intn(10_000_000) + 1
			for page := 1; page <= depth; page++ {
				campaigns, err := s.scrapingService.DiscoverCampaigns(cat.ID, sortCfg.sort, page, sessionID)
				if err != nil {
					log.Printf("Backfill: error sort=%s cat=%s page=%d: %v", sortCfg.sort, cat.ID, page, err)
					break
				}
				if len(campaigns) == 0 {
					break
				}
				now := time.Now()
				for i := range campaigns {
					campaigns[i].LastUpdatedAt = now
				}
				result := s.db.Clauses(clause.OnConflict{
					Columns: []clause.Column{{Name: "pid"}},
					DoUpdates: clause.AssignmentColumns([]string{
						"name", "blurb", "photo_url", "goal_amount", "goal_currency",
						"pledged_amount", "deadline", "state", "category_id", "category_name",
						"project_url", "creator_name", "percent_funded", "backers_count",
						"slug", "last_updated_at",
					}),
				}).Create(&campaigns)
				if result.Error != nil {
					log.Printf("Backfill: upsert error: %v", result.Error)
				} else {
					upserted += len(campaigns)
				}
				time.Sleep(500 * time.Millisecond)
			}
		}
	}
	log.Printf("Backfill: done, upserted %d campaigns", upserted)
	return nil
}

func (s *CronService) storeSnapshots(campaigns []model.Campaign) {
	snapshots := make([]model.CampaignSnapshot, 0, len(campaigns))
	now := time.Now()
	for _, c := range campaigns {
		snapshots = append(snapshots, model.CampaignSnapshot{
			CampaignPID:   c.PID,
			PledgedAmount: c.PledgedAmount,
			PercentFunded: c.PercentFunded,
			SnapshotAt:    now,
		})
	}
	if err := s.db.Create(&snapshots).Error; err != nil {
		log.Printf("Cron: snapshot insert error: %v", err)
	}
}

func (s *CronService) computeVelocity(campaigns []model.Campaign) {
	cutoff := time.Now().Add(-25 * time.Hour)

	for _, c := range campaigns {
		var prev model.CampaignSnapshot
		err := s.db.Where("campaign_pid = ? AND snapshot_at < ?", c.PID, cutoff).
			Order("snapshot_at DESC").First(&prev).Error
		if err != nil {
			continue
		}
		if prev.PledgedAmount <= 0 {
			continue
		}
		delta := c.PledgedAmount - prev.PledgedAmount
		velocityPct := (delta / prev.PledgedAmount) * 100

		s.db.Model(&model.Campaign{}).Where("pid = ?", c.PID).Updates(map[string]interface{}{
			"velocity_24h":  velocityPct,
			"ple_delta_24h": delta,
		})
	}
}

func (s *CronService) matchAlerts() error {
	cutoff := time.Now().Add(-25 * time.Hour)

	var alerts []model.Alert
	if err := s.db.Where("is_enabled = true").Find(&alerts).Error; err != nil {
		return fmt.Errorf("fetch alerts: %w", err)
	}

	for _, alert := range alerts {
		var campaigns []model.Campaign

		switch alert.AlertType {
		case "momentum":
			if alert.VelocityThresh <= 0 {
				continue
			}
			if err := s.db.Where(
				"first_seen_at > ? AND velocity_24h >= ?",
				cutoff, alert.VelocityThresh,
			).Find(&campaigns).Error; err != nil {
				log.Printf("Cron: momentum match error for alert %s: %v", alert.ID, err)
				continue
			}
		default: // "keyword"
			query := s.db.Where(
				"first_seen_at > ? AND name ILIKE ? AND percent_funded >= ?",
				cutoff, "%"+alert.Keyword+"%", alert.MinPercent,
			)
			if alert.CategoryID != "" {
				query = query.Where("category_id = ?", alert.CategoryID)
			}
			if err := query.Find(&campaigns).Error; err != nil {
				log.Printf("Cron: keyword match error for alert %s: %v", alert.ID, err)
				continue
			}
		}

		if len(campaigns) == 0 {
			continue
		}

		matches := make([]model.AlertMatch, 0, len(campaigns))
		for _, c := range campaigns {
			matches = append(matches, model.AlertMatch{
				AlertID:     alert.ID,
				CampaignPID: c.PID,
				MatchedAt:   time.Now(),
			})
		}
		s.db.Create(&matches)

		now := time.Now()
		s.db.Model(&alert).Update("last_matched_at", &now)

		s.sendAlertPush(alert, len(campaigns))
	}
	return nil
}

func (s *CronService) sendAlertPush(alert model.Alert, matchCount int) {
	if s.apnsClient == nil {
		return
	}
	var device model.Device
	if err := s.db.First(&device, "id = ?", alert.DeviceID).Error; err != nil {
		return
	}

	var title string
	switch alert.AlertType {
	case "momentum":
		title = fmt.Sprintf("%d campaigns surged +%.0f%% today", matchCount, alert.VelocityThresh)
	default:
		title = fmt.Sprintf("%d new \"%s\" campaigns", matchCount, alert.Keyword)
	}

	payload := APNsPayload{}
	payload.APS.Alert.Title = title
	payload.APS.Alert.Body = "Tap to see today's matches in KickWatch"
	payload.APS.Badge = 1
	payload.APS.Sound = "default"
	payload.AlertID = alert.ID.String()
	payload.MatchCount = matchCount

	if err := s.apnsClient.Send(device.DeviceToken, payload); err != nil {
		log.Printf("Cron: APNs error for device %s: %v", device.ID, err)
		if err.Error() == "apns: device token invalid (410)" {
			s.db.Delete(&device)
		}
	}
}
