package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"queue-system/internal/config"
	"queue-system/internal/db"
	"queue-system/internal/handlers"
	"queue-system/internal/redis"
	"queue-system/internal/routes"
	"queue-system/internal/services"
)

func main() {
	// 載入配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 初始化資料庫連接
	database, err := db.NewPostgresDB(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	// 初始化 Redis 連接
	redisClient, err := redis.NewRedisClient(&cfg.Redis)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()

	// 初始化服務
	queueService := services.NewQueueService(database, redisClient)
	adminService := services.NewAdminService(database, redisClient)

	// 初始化處理器
	queueHandler := handlers.NewQueueHandler(queueService)
	adminHandler := handlers.NewAdminHandler(adminService)

	// 設定路由
	router := routes.SetupRoutes(queueHandler, adminHandler)

	// 建立 HTTP 伺服器
	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
	}

	// 啟動伺服器（非阻塞）
	go func() {
		log.Printf("Server starting on port %s", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// 等待中斷信號
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// 優雅關閉
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
