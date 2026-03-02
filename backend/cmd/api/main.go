package main

import (
	"context"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/kickwatch/backend/internal/config"
	"github.com/kickwatch/backend/internal/db"
	"github.com/kickwatch/backend/internal/handler"
	"github.com/kickwatch/backend/internal/middleware"
	"github.com/kickwatch/backend/internal/service"
)

func main() {
	_ = godotenv.Load()

	cfg := config.Load()

	// Validate required ScrapingBee API key
	if cfg.ScrapingBeeAPIKey == "" {
		log.Fatalf("SCRAPINGBEE_API_KEY is required but not set in environment")
	}

	if cfg.DatabaseURL != "" {
		if err := db.Init(cfg); err != nil {
			log.Fatalf("DB init: %v", err)
		}
	} else {
		log.Println("DATABASE_URL not set, running without database")
	}

	// Initialize ScrapingBee service
	scrapingService := service.NewKickstarterScrapingService(
		cfg.ScrapingBeeAPIKey,
		cfg.ScrapingBeeMaxConcurrent,
	)
	log.Printf("ScrapingBee service initialized (max concurrent: %d)", cfg.ScrapingBeeMaxConcurrent)

	var cronSvc *service.CronService
	if db.IsEnabled() {
		var apnsClient *service.APNsClient
		if cfg.APNSKeyPath != "" {
			var err error
			apnsClient, err = service.NewAPNsClient(cfg)
			if err != nil {
				log.Printf("APNs init failed (push disabled): %v", err)
			}
		}

		// Initialize Vertex AI translator
		var translator *service.TranslatorService
		if cfg.VertexAIProjectID != "" && cfg.VertexAILocation != "" {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			var err error
			translator, err = service.NewTranslatorService(ctx, cfg.VertexAIProjectID, cfg.VertexAILocation)
			if err != nil {
				log.Printf("Vertex AI translator init failed (translation disabled): %v", err)
			} else {
				log.Printf("Vertex AI translator initialized (project=%s, location=%s)", cfg.VertexAIProjectID, cfg.VertexAILocation)
			}
		}

		cronSvc = service.NewCronService(db.DB, scrapingService, apnsClient, translator)
		cronSvc.Start()
		defer cronSvc.Stop()

		go func() {
			time.Sleep(3 * time.Second)
			log.Println("Startup: triggering initial crawl")
			if err := cronSvc.RunCrawlNow(); err != nil {
				log.Printf("Startup crawl error: %v", err)
			}
		}()
	}

	r := gin.Default()
	r.Use(middleware.CORS())
	r.Use(middleware.Logger())

	api := r.Group("/api")
	{
		api.GET("/health", handler.Health)

		api.GET("/campaigns", handler.ListCampaigns(scrapingService))
		api.GET("/campaigns/search", handler.SearchCampaigns(scrapingService))
		api.GET("/campaigns/:pid", handler.GetCampaign)
		api.GET("/campaigns/:pid/history", handler.GetCampaignHistory)
		api.GET("/categories", handler.ListCategories(scrapingService))

		api.POST("/devices/register", handler.RegisterDevice)

		alerts := api.Group("/alerts")
		{
			alerts.POST("", handler.CreateAlert)
			alerts.GET("", handler.ListAlerts)
			alerts.PATCH("/:id", handler.UpdateAlert)
			alerts.DELETE("/:id", handler.DeleteAlert)
			alerts.GET("/:id/matches", handler.GetAlertMatches)
		}

		if cronSvc != nil {
			api.POST("/admin/backfill", handler.TriggerBackfill(cronSvc))
			api.GET("/admin/cron-status", handler.CronStatus(cronSvc))
		}
	}

	log.Printf("KickWatch API starting on :%s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatal(err)
	}
}
