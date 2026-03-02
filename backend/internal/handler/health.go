package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kickwatch/backend/internal/service"
)

func Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "kickwatch-api"})
}

func CronStatus(cronSvc *service.CronService) gin.HandlerFunc {
	return func(c *gin.Context) {
		status := gin.H{
			"last_crawl_at":    nil,
			"last_crawl_count": cronSvc.LastCrawlCount,
			"last_crawl_error": cronSvc.LastCrawlError,
		}
		if !cronSvc.LastCrawlAt.IsZero() {
			status["last_crawl_at"] = cronSvc.LastCrawlAt
		}
		c.JSON(http.StatusOK, status)
	}
}
