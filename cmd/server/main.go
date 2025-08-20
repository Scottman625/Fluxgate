package main

import (
    "context"
    "database/sql"
    "log"
    "net/http"
    "os"
    "os/signal"
    "strconv"
    "syscall"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/go-redis/redis/v8"
    _ "github.com/lib/pq"
    
    "queue-system/internal/handlers"
    "queue-system/internal/metrics"
    "queue-system/internal/monitoring"
    "queue-system/internal/services"
)

func main() {
    // 載入配置
    config := loadConfig()

    // 初始化資料庫
    db, err := sql.Open("postgres", config.DatabaseURL)
    if err != nil {
        log.Fatal("Failed to connect to database:", err)
    }
    defer db.Close()

    // 初始化 Redis
    rdb := redis.NewClient(&redis.Options{
        Addr:     config.RedisAddr,
        Password: config.RedisPassword,
        DB:       config.RedisDB,
    })
    defer rdb.Close()

    // 測試連接
    ctx := context.Background()
    if err := rdb.Ping(ctx).Err(); err != nil {
        log.Fatal("Failed to connect to Redis:", err)
    }

    // 初始化服務
    queueService := services.NewQueueService(db, rdb)
    releaseScheduler := services.NewReleaseScheduler(db, rdb)
    
    // 初始化監控
    metricsCollector := metrics.NewMetricsCollector(db, rdb)
    metricsCollector.RegisterMetrics()
    
    dashboard := monitoring.NewDashboard(db, rdb)

    // 啟動 Release Scheduler
    go func() {
        if err := releaseScheduler.Start(ctx); err != nil {
            log.Printf("Failed to start release scheduler: %v", err)
        }
    }()

    // 啟動指標收集
    go func() {
        metricsCollector.StartCollection(ctx)
    }()

    // 啟動 Prometheus 指標服務器
    go func() {
        if err := metrics.StartMetricsServer(":9090"); err != nil {
            log.Printf("Failed to start metrics server: %v", err)
        }
    }()

    // 設置 HTTP 路由
    router := setupRouter(queueService, dashboard)

    // 啟動 HTTP 服務器
    server := &http.Server{
        Addr:    ":" + config.Port,
        Handler: router,
    }

    go func() {
        log.Printf("Starting server on port %s", config.Port)
        if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatal("Failed to start server:", err)
        }
    }()

    // 優雅關閉
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    log.Println("Shutting down server...")

    // 停止 Release Scheduler
    releaseScheduler.Stop()

    // 關閉 HTTP 服務器
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    if err := server.Shutdown(ctx); err != nil {
        log.Fatal("Server forced to shutdown:", err)
    }

    log.Println("Server exited")
}

func setupRouter(queueService *services.QueueService, dashboard *monitoring.Dashboard) *gin.Engine {
    router := gin.Default()

    // 添加指標中間件
    router.Use(ginMetricsMiddleware())

    // 創建 handlers
    queueHandler := handlers.NewQueueHandler(queueService)

    // API 路由
    api := router.Group("/api/v1")
    {
        // 隊列相關 API
        api.POST("/queue/enter", queueHandler.EnterQueue)
        api.GET("/queue/status", queueHandler.GetQueueStatus)
        
        // 儀表板 API
        api.GET("/dashboard", dashboardHandler(dashboard))
        api.GET("/dashboard/activities/:id/history", activityHistoryHandler(dashboard))
        api.GET("/dashboard/metrics/realtime", realTimeMetricsHandler(dashboard))
    }

    // 靜態檔案服務
    router.Static("/web", "./web")
    router.StaticFile("/", "./web/dashboard/index.html")

    return router
}

type Config struct {
    DatabaseURL   string
    RedisAddr     string
    RedisPassword string
    RedisDB       int
    Port          string
}

func loadConfig() *Config {
    return &Config{
        DatabaseURL:   getEnv("DATABASE_URL", "postgres://user:password@localhost/queuedb?sslmode=disable"),
        RedisAddr:     getEnv("REDIS_ADDR", "localhost:6379"),
        RedisPassword: getEnv("REDIS_PASSWORD", ""),
        RedisDB:       0,
        Port:          getEnv("PORT", "8080"),
    }
}

func getEnv(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}

// Gin 指標中間件
func ginMetricsMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        
        c.Next()
        
        duration := time.Since(start)
        status := strconv.Itoa(c.Writer.Status())
        
        metrics.RecordAPIRequest(c.Request.Method, c.Request.URL.Path, status, duration)
    }
}

// Dashboard handlers
func dashboardHandler(dashboard *monitoring.Dashboard) gin.HandlerFunc {
    return func(c *gin.Context) {
        data, err := dashboard.GetDashboardData(c.Request.Context())
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }
        c.JSON(http.StatusOK, data)
    }
}

func activityHistoryHandler(dashboard *monitoring.Dashboard) gin.HandlerFunc {
    return func(c *gin.Context) {
        activityID, err := strconv.ParseInt(c.Param("id"), 10, 64)
        if err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": "invalid activity id"})
            return
        }
        
        tenantID := c.Query("tenant_id")
        if tenantID == "" {
            c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id is required"})
            return
        }
        
        hours := 24 // 預設 24 小時
        if h := c.Query("hours"); h != "" {
            if parsed, err := strconv.Atoi(h); err == nil {
                hours = parsed
            }
        }
        
        history, err := dashboard.GetActivityHistory(c.Request.Context(), tenantID, activityID, hours)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }
        c.JSON(http.StatusOK, history)
    }
}

func realTimeMetricsHandler(dashboard *monitoring.Dashboard) gin.HandlerFunc {
    return func(c *gin.Context) {
        metrics, err := dashboard.GetRealTimeMetrics(c.Request.Context())
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }
        c.JSON(http.StatusOK, metrics)
    }
}
