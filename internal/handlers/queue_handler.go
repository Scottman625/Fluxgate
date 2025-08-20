package handlers

import (
	"net/http"

	"queue-system/internal/services"

	"github.com/gin-gonic/gin"
)

type QueueHandler struct {
	queueService *services.QueueService
}

func NewQueueHandler(queueService *services.QueueService) *QueueHandler {
	return &QueueHandler{
		queueService: queueService,
	}
}

// POST /queue/enter
func (h *QueueHandler) EnterQueue(c *gin.Context) {
	var req services.EnterQueueRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":      "INVALID_REQUEST",
			"message":    err.Error(),
			"request_id": c.GetString("request_id"),
		})
		return
	}

	// 從 header 獲取 IP 地址
	req.IPAddress = getClientIP(c)

	resp, err := h.queueService.EnterQueue(c.Request.Context(), &req)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorCode := "INTERNAL_ERROR"

		// 根據錯誤類型返回不同狀態碼
		switch {
		case contains(err.Error(), "activity not found"):
			statusCode = http.StatusNotFound
			errorCode = "ACTIVITY_NOT_FOUND"
		case contains(err.Error(), "activity is not active"):
			statusCode = http.StatusConflict
			errorCode = "ACTIVITY_NOT_ACTIVE"
		case contains(err.Error(), "rate limit exceeded"):
			statusCode = http.StatusTooManyRequests
			errorCode = "RATE_LIMIT_EXCEEDED"
		case contains(err.Error(), "user already in queue"):
			statusCode = http.StatusConflict
			errorCode = "USER_ALREADY_IN_QUEUE"
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

// GET /queue/status
func (h *QueueHandler) GetQueueStatus(c *gin.Context) {
	var req services.QueueStatusRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":      "INVALID_REQUEST",
			"message":    err.Error(),
			"request_id": c.GetString("request_id"),
		})
		return
	}

	resp, err := h.queueService.GetQueueStatus(c.Request.Context(), &req)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorCode := "INTERNAL_ERROR"

		switch {
		case contains(err.Error(), "activity not found"):
			statusCode = http.StatusNotFound
			errorCode = "ACTIVITY_NOT_FOUND"
		case contains(err.Error(), "invalid sequence number"):
			statusCode = http.StatusBadRequest
			errorCode = "INVALID_SEQUENCE"
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

// 輔助函數
func getClientIP(c *gin.Context) string {
	// 優先從 X-Forwarded-For 取得
	if ip := c.GetHeader("X-Forwarded-For"); ip != "" {
		return ip
	}
	// 其次從 X-Real-IP 取得
	if ip := c.GetHeader("X-Real-IP"); ip != "" {
		return ip
	}
	// 最後從 RemoteAddr 取得
	return c.ClientIP()
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) &&
			(s[:len(substr)] == substr ||
				s[len(s)-len(substr):] == substr ||
				containsSubstring(s, substr))))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
