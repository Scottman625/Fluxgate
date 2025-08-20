package routes

import (
	"queue-system/internal/handlers"
	"queue-system/internal/middleware"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(queueHandler *handlers.QueueHandler, adminHandler *handlers.AdminHandler) *gin.Engine {
	r := gin.Default()

	// 全域中間件
	r.Use(middleware.RequestID())
	r.Use(middleware.CORS())
	r.Use(gin.Recovery())

	// 健康檢查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// API v1 路由群組
	v1 := r.Group("/api/v1")
	{
		// 隊列相關 API
		queue := v1.Group("/queue")
		{
			queue.POST("/enter", queueHandler.EnterQueue)
			queue.GET("/status", queueHandler.GetQueueStatus)
		}

		// Admin API
		admin := v1.Group("/admin")
		{
			admin.POST("/activities", adminHandler.CreateActivity)
			admin.GET("/activities", adminHandler.ListActivities)
			admin.GET("/activities/:id/status", adminHandler.GetActivityStatus)
			admin.PUT("/activities/:id", adminHandler.UpdateActivity)
		}
	}

	return r
}
