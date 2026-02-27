package service

import (
	"fmt"
	"log"
	"time"

	"github.com/kickwatch/backend/internal/model"
	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var rootCategories = []string{
	"1", "3", "4", "5", "6", "7", "9", "10", "11", "12", "13", "14", "15", "16", "17",
}

type CronService struct {
	db         *gorm.DB
	restClient *KickstarterRESTClient
	apnsClient *APNsClient
	scheduler  *cron.Cron
}

func NewCronService(db *gorm.DB, restClient *KickstarterRESTClient, apns *APNsClient) *CronService {
	return &CronService{
		db:         db,
		restClient: restClient,
		apnsClient: apns,
		scheduler:  cron.New(cron.WithLocation(time.UTC)),
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
}

func (s *CronService) Stop() {
	s.scheduler.Stop()
}

func (s *CronService) RunCrawlNow() error {
	upserted := 0
	var allCampaigns []model.Campaign

	for _, catID := range rootCategories {
		for page := 1; page <= 10; page++ {
			campaigns, err := s.restClient.DiscoverCampaigns(catID, "newest", page)
			if err != nil {
				log.Printf("Cron: REST error cat=%s page=%d: %v", catID, page, err)
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
					"project_url", "creator_name", "percent_funded", "slug", "last_updated_at",
				}),
			}).Create(&campaigns)
			if result.Error != nil {
				log.Printf("Cron: upsert error: %v", result.Error)
			} else {
				upserted += len(campaigns)
				allCampaigns = append(allCampaigns, campaigns...)
			}
			time.Sleep(500 * time.Millisecond)
		}
	}
	log.Printf("Cron: crawl done, upserted %d campaigns", upserted)

	if len(allCampaigns) > 0 {
		s.storeSnapshots(allCampaigns)
		s.computeVelocity(allCampaigns)
	}

	return s.matchAlerts()
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
