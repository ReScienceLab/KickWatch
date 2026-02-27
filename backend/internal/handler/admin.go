package handler

import (
	"net/http"
	"os"
	"sync/atomic"

	"github.com/gin-gonic/gin"
)

type backfillRunner interface {
	RunBackfill() error
}

var backfillRunning atomic.Bool

// TriggerBackfill starts a deep historical crawl in the background.
// POST /api/admin/backfill
// Requires X-Admin-Secret header matching ADMIN_SECRET env var.
// Only one backfill may run at a time; concurrent requests get 409.
func TriggerBackfill(svc backfillRunner) gin.HandlerFunc {
	return func(c *gin.Context) {
		secret := os.Getenv("ADMIN_SECRET")
		if secret == "" || c.GetHeader("X-Admin-Secret") != secret {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		if !backfillRunning.CompareAndSwap(false, true) {
			c.JSON(http.StatusConflict, gin.H{"error": "backfill already running"})
			return
		}
		go func() {
			defer backfillRunning.Store(false)
			if err := svc.RunBackfill(); err != nil {
				// logged inside RunBackfill
				_ = err
			}
		}()
		c.JSON(http.StatusAccepted, gin.H{"message": "backfill started in background"})
	}
}
