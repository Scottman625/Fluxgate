package handlers

import (
	"net/http"
	"strconv"

	"queue-system/internal/services"

	"github.com/gin-gonic/gin"
)

type AdminHandler struct {
	adminService *services.AdminService
}

func NewAdminHandler(adminService *services.AdminService) *AdminHandler {
	return &AdminHandler{
		adminService: adminService,
	}
}

// POST /admin/activities
func (h *AdminHandler) CreateActivity(c *gin.Context) {
	var req services.CreateActivityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":      "INVALID_REQUEST",
			"message":    err.Error(),
			"request_id": c.GetString("request_id"),
		})
		return
	}

	resp, err := h.adminService.CreateActivity(c.Request.Context(), &req)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorCode := "INTERNAL_ERROR"

		if contains(err.Error(), "end_at must be after start_at") {
			statusCode = http.StatusBadRequest
			errorCode = "INVALID_TIME_RANGE"
		}

		c.JSON(statusCode, gin.H{
			"error":      errorCode,
			"message":    err.Error(),
			"request_id": c.GetString("request_id"),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    resp,
	})
}

// GET /admin/activities/:id/status
func (h *AdminHandler) GetActivityStatus(c *gin.Context) {
	activityID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":      "INVALID_ACTIVITY_ID",
			"message":    "Activity ID must be a valid integer",
			"request_id": c.GetString("request_id"),
		})
		return
	}

	resp, err := h.adminService.GetActivityStatus(c.Request.Context(), activityID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorCode := "INTERNAL_ERROR"

		if contains(err.Error(), "activity not found") {
			statusCode = http.StatusNotFound
			errorCode = "ACTIVITY_NOT_FOUND"
		}

		c.JSON(statusCode, gin.H{
			"error":      errorCode,
			"message":    err.Error(),
			"request_id": c.GetString("request_id"),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    resp,
	})
}

// PUT /admin/activities/:id
func (h *AdminHandler) UpdateActivity(c *gin.Context) {
	activityID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":      "INVALID_ACTIVITY_ID",
			"message":    "Activity ID must be a valid integer",
			"request_id": c.GetString("request_id"),
		})
		return
	}

	var req services.UpdateActivityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":      "INVALID_REQUEST",
			"message":    err.Error(),
			"request_id": c.GetString("request_id"),
		})
		return
	}

	err = h.adminService.UpdateActivity(c.Request.Context(), activityID, &req)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorCode := "INTERNAL_ERROR"

		switch {
		case contains(err.Error(), "activity not found"):
			statusCode = http.StatusNotFound
			errorCode = "ACTIVITY_NOT_FOUND"
		case contains(err.Error(), "no fields to update"):
			statusCode = http.StatusBadRequest
			errorCode = "NO_FIELDS_TO_UPDATE"
		}

		c.JSON(statusCode, gin.H{
			"error":      errorCode,
			"message":    err.Error(),
			"request_id": c.GetString("request_id"),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Activity updated successfully",
	})
}

// GET /admin/activities
func (h *AdminHandler) ListActivities(c *gin.Context) {
	tenantID := c.Query("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":      "MISSING_TENANT_ID",
			"message":    "tenant_id is required",
			"request_id": c.GetString("request_id"),
		})
		return
	}

	activities, err := h.adminService.ListActivities(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":      "INTERNAL_ERROR",
			"message":    err.Error(),
			"request_id": c.GetString("request_id"),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    activities,
	})
}
