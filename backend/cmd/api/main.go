package main

import (
	"log"

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

	if cfg.DatabaseURL != "" {
		if err := db.Init(cfg); err != nil {
			log.Fatalf("DB init: %v", err)
		}
	} else {
		log.Println("DATABASE_URL not set, running without database")
	}

	graphClient := service.NewKickstarterGraphClient()
	restClient := service.NewKickstarterRESTClient()

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
		cronSvc = service.NewCronService(db.DB, restClient, apnsClient)
		cronSvc.Start()
		defer cronSvc.Stop()
	}

	r := gin.Default()
	r.Use(middleware.CORS())
	r.Use(middleware.Logger())

	api := r.Group("/api")
	{
		api.GET("/health", handler.Health)

		api.GET("/campaigns", handler.ListCampaigns(graphClient))
		api.GET("/campaigns/search", handler.SearchCampaigns(graphClient))
		api.GET("/campaigns/:pid", handler.GetCampaign)
		api.GET("/categories", handler.ListCategories(graphClient))

		api.POST("/devices/register", handler.RegisterDevice)

		alerts := api.Group("/alerts")
		{
			alerts.POST("", handler.CreateAlert)
			alerts.GET("", handler.ListAlerts)
			alerts.PATCH("/:id", handler.UpdateAlert)
			alerts.DELETE("/:id", handler.DeleteAlert)
			alerts.GET("/:id/matches", handler.GetAlertMatches)
		}
	}

	log.Printf("KickWatch API starting on :%s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatal(err)
	}
}
