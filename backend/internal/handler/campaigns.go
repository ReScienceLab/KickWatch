package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kickwatch/backend/internal/db"
	"github.com/kickwatch/backend/internal/model"
	"github.com/kickwatch/backend/internal/service"
)

var sortMap = map[string]string{
	"trending": "MAGIC",
	"newest":   "NEWEST",
	"ending":   "END_DATE",
}

func ListCampaigns(client *service.KickstarterScrapingService) gin.HandlerFunc {
	return func(c *gin.Context) {
		sort := c.DefaultQuery("sort", "trending")
		categoryID := c.Query("category_id")
		cursor := c.Query("cursor")
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
		if limit > 50 {
			limit = 50
		}

		// "hot" sort: served from DB by velocity_24h
		if sort == "hot" && db.IsEnabled() {
			var campaigns []model.Campaign
			q := db.DB.Where("state = 'live'").Order("velocity_24h DESC").Limit(limit)
			if categoryID != "" {
				q = q.Where("category_id = ?", categoryID)
			}
			if err := q.Find(&campaigns).Error; err == nil {
				c.JSON(http.StatusOK, gin.H{"campaigns": campaigns, "next_cursor": nil, "total": len(campaigns)})
				return
			}
		}

		gqlSort, ok := sortMap[sort]
		if !ok {
			gqlSort = "MAGIC"
		}

		result, err := client.Search("", categoryID, gqlSort, cursor, limit)
		if err != nil {
			// fallback to DB if GraphQL fails
			if db.IsEnabled() {
				var campaigns []model.Campaign
				q := db.DB.Where("state = 'live'").Order("last_updated_at DESC").Limit(limit)
				if categoryID != "" {
					q = q.Where("category_id = ?", categoryID)
				}
				if dbErr := q.Find(&campaigns).Error; dbErr == nil && len(campaigns) > 0 {
					c.JSON(http.StatusOK, gin.H{"campaigns": campaigns, "next_cursor": nil, "total": len(campaigns)})
					return
				}
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		var nextCursor interface{}
		if result.HasNextPage {
			nextCursor = result.NextCursor
		}
		c.JSON(http.StatusOK, gin.H{
			"campaigns":   result.Campaigns,
			"next_cursor": nextCursor,
			"total":       result.TotalCount,
		})
	}
}

func SearchCampaigns(client *service.KickstarterScrapingService) gin.HandlerFunc {
	return func(c *gin.Context) {
		q := c.Query("q")
		if q == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "q is required"})
			return
		}
		categoryID := c.Query("category_id")
		cursor := c.Query("cursor")

		result, err := client.Search(q, categoryID, "MAGIC", cursor, 20)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		var nextCursor interface{}
		if result.HasNextPage {
			nextCursor = result.NextCursor
		}
		c.JSON(http.StatusOK, gin.H{
			"campaigns":   result.Campaigns,
			"next_cursor": nextCursor,
		})
	}
}

func GetCampaign(c *gin.Context) {
	pid := c.Param("pid")
	if !db.IsEnabled() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
		return
	}
	var campaign model.Campaign
	if err := db.DB.First(&campaign, "pid = ?", pid).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "campaign not found"})
		return
	}
	c.JSON(http.StatusOK, campaign)
}

func ListCategories(client *service.KickstarterScrapingService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if db.IsEnabled() {
			var cats []model.Category
			if err := db.DB.Find(&cats).Error; err == nil && len(cats) > 0 {
				c.JSON(http.StatusOK, cats)
				return
			}
		}

		cats, err := client.FetchCategories()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if db.IsEnabled() && len(cats) > 0 {
			db.DB.Save(&cats)
		}
		c.JSON(http.StatusOK, cats)
	}
}

func GetCampaignHistory(c *gin.Context) {
	pid := c.Param("pid")
	days := c.DefaultQuery("days", "14")
	daysInt, _ := strconv.Atoi(days)
	if daysInt > 30 {
		daysInt = 30
	}

	if !db.IsEnabled() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
		return
	}

	cutoff := time.Now().Add(-time.Duration(daysInt) * 24 * time.Hour)

	var snapshots []model.CampaignSnapshot
	if err := db.DB.Where("campaign_pid = ? AND snapshot_at >= ?", pid, cutoff).
		Order("snapshot_at ASC").
		Find(&snapshots).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"history": snapshots})
}
