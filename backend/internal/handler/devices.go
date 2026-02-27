package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kickwatch/backend/internal/db"
	"github.com/kickwatch/backend/internal/model"
)

type registerDeviceRequest struct {
	DeviceToken string `json:"device_token" binding:"required"`
}

func RegisterDevice(c *gin.Context) {
	var req registerDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var device model.Device
	result := db.DB.Where("device_token = ?", req.DeviceToken).FirstOrCreate(&device, model.Device{
		DeviceToken: req.DeviceToken,
	})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"device_id": device.ID})
}
