package handler

import (
	"encoding/base64"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kickwatch/backend/internal/db"
	"github.com/kickwatch/backend/internal/model"
	"github.com/kickwatch/backend/internal/service"
)

var sortMap = map[string]string{
	"trending": "MAGIC",
	"hot":      "MAGIC", // Fallback uses MAGIC for hot sort (close approximation)
	"newest":   "NEWEST",
	"ending":   "END_DATE",
}

func ListCampaigns(client *service.KickstarterScrapingService) gin.HandlerFunc {
	return func(c *gin.Context) {
		sort := c.DefaultQuery("sort", "trending")
		categoryID := c.Query("category_id")
		cursor := c.Query("cursor")
		state := c.DefaultQuery("state", "live")
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
		if limit > 50 {
			limit = 50
		}

		// Serve ALL sorts from DB for client requests (zero ScrapingBee credits)
		// Trade-offs for cost optimization:
		// - trending: velocity_24h approximates Kickstarter's MAGIC (not exact but close)
		// - newest: first_seen_at = crawl time, not launch time (daily crawl minimizes drift)
		// - hot: velocity_24h (our metric)
		// - ending: deadline (exact from Kickstarter)
		if db.IsEnabled() {
			// Detect cursor source: ScrapingBee uses "page:N", DB uses base64 offsets
			// If cursor is from ScrapingBee, fall through to ScrapingBee to maintain format
			if cursor != "" && strings.HasPrefix(cursor, "page:") {
				// ScrapingBee cursor format detected - fall through to ScrapingBee path
				// to avoid format mismatch (cannot mix base64 offsets with page numbers)
				goto useScrapingBee
			}

			offset := 0
			if cursor != "" {
				if decoded, err := base64.StdEncoding.DecodeString(cursor); err == nil {
					offset, _ = strconv.Atoi(string(decoded))
				} else {
					// Invalid cursor format - treat as offset 0 but log warning
					// This shouldn't happen if client respects cursor format
					log.Printf("ListCampaigns: invalid cursor format %q, treating as offset 0", cursor)
				}
			}

			var campaigns []model.Campaign
			q := db.DB.Where("state = ?", state).Offset(offset).Limit(limit + 1)

			// Only filter by deadline for live campaigns
			if state == "live" {
				q = q.Where("deadline >= ?", time.Now())
			}

			// Map sort to DB columns
			switch sort {
			case "trending", "hot":
				q = q.Order("velocity_24h DESC, percent_funded DESC")
			case "newest":
				q = q.Order("first_seen_at DESC")
			case "ending":
				q = q.Order("deadline ASC")
			default:
				q = q.Order("velocity_24h DESC, percent_funded DESC")
			}

			if categoryID != "" {
				q = q.Where("category_id = ?", categoryID)
			}

			// Return DB results if we have data
			// Note: Once using DB cursors, always use DB to maintain cursor format consistency
			if err := q.Find(&campaigns).Error; err == nil {
				// Return DB results if we have data, OR if we're paginating with a DB cursor
				// (cursor != "" and not a ScrapingBee cursor means it's a DB cursor)
				if len(campaigns) > 0 || cursor != "" {
					hasMore := len(campaigns) > limit
					if hasMore {
						campaigns = campaigns[:limit]
					}

					var nextCursor interface{}
					if hasMore {
						nextOffset := offset + limit
						nextCursor = base64.StdEncoding.EncodeToString([]byte(strconv.Itoa(nextOffset)))
					}

					// Don't include total for DB queries (we don't track global count)
					c.JSON(http.StatusOK, gin.H{"campaigns": campaigns, "next_cursor": nextCursor})
					return
				}
			}
			// Only fall through to ScrapingBee on first load (cursor == "") and DB empty/failed
		}

	useScrapingBee:
		// ScrapingBee fallback for:
		// - First load (cursor == "") when DB is unavailable or empty (cold start)
		// - ScrapingBee pagination (cursor starts with "page:")
		// - SearchCampaigns endpoint (user search with query text)
		gqlSort, ok := sortMap[sort]
		if !ok {
			gqlSort = "MAGIC"
		}

		result, err := client.Search("", categoryID, gqlSort, cursor, limit)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database and API unavailable"})
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
	// No deadline filter - allow viewing ended campaigns (e.g., from bookmarks/history)
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
	daysInt, err := strconv.Atoi(days)
	if err != nil || daysInt < 1 {
		daysInt = 14
	}
	if daysInt > 30 {
		daysInt = 30
	}

	if !db.IsEnabled() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database not available"})
		return
	}

	cutoff := time.Now().Add(-time.Duration(daysInt) * 24 * time.Hour)

	var snapshots []model.CampaignSnapshot
	// Group by snapshot_date and return the latest snapshot per day
	// Using DISTINCT ON to get one row per (campaign_pid, snapshot_date)
	if err := db.DB.
		Where("campaign_pid = ? AND snapshot_date >= DATE(?)", pid, cutoff).
		Order("snapshot_date ASC, snapshot_at DESC").
		Find(&snapshots).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Since GORM doesn't support DISTINCT ON directly, we deduplicate in Go
	seen := make(map[string]bool)
	dedupedSnapshots := make([]model.CampaignSnapshot, 0, len(snapshots))
	for _, s := range snapshots {
		dateKey := s.SnapshotDate.Format("2006-01-02")
		if !seen[dateKey] {
			seen[dateKey] = true
			dedupedSnapshots = append(dedupedSnapshots, s)
		}
	}

	c.JSON(http.StatusOK, gin.H{"history": dedupedSnapshots})
}
