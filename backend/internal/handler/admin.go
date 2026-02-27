package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type backfillRunner interface {
	RunBackfill() error
}

// TriggerBackfill starts a deep historical crawl in the background.
// POST /api/admin/backfill
func TriggerBackfill(svc backfillRunner) gin.HandlerFunc {
	return func(c *gin.Context) {
		go func() {
			if err := svc.RunBackfill(); err != nil {
				// logged inside RunBackfill
				_ = err
			}
		}()
		c.JSON(http.StatusAccepted, gin.H{"message": "backfill started in background"})
	}
}
