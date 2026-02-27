package handler

import (
	"net/http"
	"strconv"

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
		gqlSort, ok := sortMap[sort]
		if !ok {
			gqlSort = "MAGIC"
		}
		categoryID := c.Query("category_id")
		cursor := c.Query("cursor")
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
		if limit > 50 {
			limit = 50
		}

		result, err := client.Search("", categoryID, gqlSort, cursor, limit)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		nextCursor := ""
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

		nextCursor := ""
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
