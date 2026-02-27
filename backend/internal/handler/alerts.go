package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kickwatch/backend/internal/db"
	"github.com/kickwatch/backend/internal/model"
)

type createAlertRequest struct {
	DeviceID       string  `json:"device_id" binding:"required"`
	AlertType      string  `json:"alert_type"`
	Keyword        string  `json:"keyword"`
	CategoryID     string  `json:"category_id"`
	MinPercent     float64 `json:"min_percent"`
	VelocityThresh float64 `json:"velocity_thresh"`
}

func CreateAlert(c *gin.Context) {
	var req createAlertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	alertType := req.AlertType
	if alertType == "" {
		alertType = "keyword"
	}
	if alertType == "keyword" && req.Keyword == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "keyword is required for keyword alerts"})
		return
	}
	if alertType == "momentum" && req.VelocityThresh <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "velocity_thresh must be > 0 for momentum alerts"})
		return
	}

	deviceID, err := uuid.Parse(req.DeviceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid device_id"})
		return
	}

	alert := model.Alert{
		DeviceID:       deviceID,
		AlertType:      alertType,
		Keyword:        req.Keyword,
		CategoryID:     req.CategoryID,
		MinPercent:     req.MinPercent,
		VelocityThresh: req.VelocityThresh,
		IsEnabled:      true,
	}
	if err := db.DB.Create(&alert).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, alert)
}

func ListAlerts(c *gin.Context) {
	deviceIDStr := c.Query("device_id")
	if deviceIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device_id required"})
		return
	}
	deviceID, err := uuid.Parse(deviceIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid device_id"})
		return
	}

	var alerts []model.Alert
	if err := db.DB.Where("device_id = ?", deviceID).Find(&alerts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, alerts)
}

type updateAlertRequest struct {
	IsEnabled  *bool    `json:"is_enabled"`
	Keyword    *string  `json:"keyword"`
	CategoryID *string  `json:"category_id"`
	MinPercent *float64 `json:"min_percent"`
}

func UpdateAlert(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid alert id"})
		return
	}

	var alert model.Alert
	if err := db.DB.First(&alert, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "alert not found"})
		return
	}

	var req updateAlertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{}
	if req.IsEnabled != nil {
		updates["is_enabled"] = *req.IsEnabled
	}
	if req.Keyword != nil {
		updates["keyword"] = *req.Keyword
	}
	if req.CategoryID != nil {
		updates["category_id"] = *req.CategoryID
	}
	if req.MinPercent != nil {
		updates["min_percent"] = *req.MinPercent
	}

	if err := db.DB.Model(&alert).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, alert)
}

func DeleteAlert(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid alert id"})
		return
	}
	if err := db.DB.Delete(&model.Alert{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func GetAlertMatches(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid alert id"})
		return
	}

	since := time.Now().Add(-24 * time.Hour)
	if sinceStr := c.Query("since"); sinceStr != "" {
		if t, err := time.Parse(time.RFC3339, sinceStr); err == nil {
			since = t
		}
	}

	var matches []model.AlertMatch
	if err := db.DB.Where("alert_id = ? AND matched_at > ?", id, since).Find(&matches).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	pids := make([]string, 0, len(matches))
	for _, m := range matches {
		pids = append(pids, m.CampaignPID)
	}

	var campaigns []model.Campaign
	if len(pids) > 0 {
		db.DB.Where("pid IN ?", pids).Find(&campaigns)
	}
	c.JSON(http.StatusOK, campaigns)
}
