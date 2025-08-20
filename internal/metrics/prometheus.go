package metrics

import (
    "context"
    "database/sql"
    "fmt"
    "log"
    "net/http"
    "strconv"
    "strings"
    "time"

    "github.com/go-redis/redis/v8"
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
    "queue-system/pkg/keys"
)

// Prometheus 指標定義
var (
    // 隊列相關指標
    QueueLength = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "queue_length",
            Help: "Current queue length by tenant and activity",
        },
        []string{"tenant_id", "activity_id", "activity_name"},
    )

    QueueEnterTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "queue_enter_total",
            Help: "Total number of queue entries",
        },
        []string{"tenant_id", "activity_id", "status"},
    )

    QueueReleaseTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "queue_release_total",
            Help: "Total number of queue releases",
        },
        []string{"tenant_id", "activity_id", "method"},
    )

    QueueWaitTime = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "queue_wait_time_seconds",
            Help:    "Queue wait time distribution",
            Buckets: []float64{1, 5, 10, 30, 60, 120, 300, 600, 1200, 3600},
        },
        []string{"tenant_id", "activity_id"},
    )

    // 釋放調度器指標
    SchedulerActive = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "scheduler_active",
            Help: "Number of active release schedulers",
        },
        []string{"tenant_id", "activity_id"},
    )

    SchedulerReleaseRate = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "scheduler_release_rate",
            Help: "Current release rate per second",
        },
        []string{"tenant_id", "activity_id"},
    )

    SchedulerTotalReleased = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "scheduler_total_released",
            Help: "Total number of positions released by scheduler",
        },
        []string{"tenant_id", "activity_id"},
    )

    // API 相關指標
    APIRequestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "api_request_duration_seconds",
            Help:    "API request duration",
            Buckets: prometheus.DefBuckets,
        },
        []string{"method", "endpoint", "status"},
    )

    APIRequestTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "api_request_total",
            Help: "Total API requests",
        },
        []string{"method", "endpoint", "status"},
    )

    // 系統指標
    RedisConnections = prometheus.NewGauge(
        prometheus.GaugeOpts{
            Name: "redis_connections",
            Help: "Number of Redis connections",
        },
    )

    DatabaseConnections = prometheus.NewGauge(
        prometheus.GaugeOpts{
            Name: "database_connections",
            Help: "Number of database connections",
        },
    )

    ActiveUsers = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "active_users",
            Help: "Number of active users in queue",
        },
        []string{"tenant_id", "activity_id"},
    )
)

type MetricsCollector struct {
    db    *sql.DB
    redis *redis.Client
}

func NewMetricsCollector(db *sql.DB, redis *redis.Client) *MetricsCollector {
    return &MetricsCollector{
        db:    db,
        redis: redis,
    }
}

func (mc *MetricsCollector) RegisterMetrics() {
    prometheus.MustRegister(
        QueueLength,
        QueueEnterTotal,
        QueueReleaseTotal,
        QueueWaitTime,
        SchedulerActive,
        SchedulerReleaseRate,
        SchedulerTotalReleased,
        APIRequestDuration,
        APIRequestTotal,
        RedisConnections,
        DatabaseConnections,
        ActiveUsers,
    )
}

func (mc *MetricsCollector) StartCollection(ctx context.Context) {
    ticker := time.NewTicker(15 * time.Second) // 每 15 秒收集一次
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            mc.collectMetrics(ctx)
        }
    }
}

func (mc *MetricsCollector) collectMetrics(ctx context.Context) {
    // 收集隊列指標
    mc.collectQueueMetrics(ctx)
    
    // 收集調度器指標
    mc.collectSchedulerMetrics(ctx)
    
    // 收集系統指標
    mc.collectSystemMetrics(ctx)
}

func (mc *MetricsCollector) collectQueueMetrics(ctx context.Context) {
    // 查詢所有活躍活動
    query := `
        SELECT id, tenant_id, name, config_json
        FROM activities 
        WHERE status = 'active'`

    rows, err := mc.db.QueryContext(ctx, query)
    if err != nil {
        log.Printf("Failed to query activities for metrics: %v", err)
        return
    }
    defer rows.Close()

    // 全域統計變數
    totalQueueLength := int64(0)
    totalActiveUsers := int64(0)
    activeSchedulers := int64(0)

    for rows.Next() {
        var activityID int64
        var tenantID, name string
        var config string

        if err := rows.Scan(&activityID, &tenantID, &name, &config); err != nil {
            continue
        }

        activityIDStr := strconv.FormatInt(activityID, 10)
        labels := prometheus.Labels{
            "tenant_id":     tenantID,
            "activity_id":   activityIDStr,
            "activity_name": name,
        }

        // 收集隊列長度
        queueSeq := mc.getRedisInt(ctx, keys.QueueSeqKey(tenantID, activityID))
        releaseSeq := mc.getRedisInt(ctx, keys.ReleaseSeqKey(tenantID, activityID))
        queueLength := queueSeq - releaseSeq
        
        QueueLength.With(labels).Set(float64(queueLength))
        totalQueueLength += queueLength

        // 收集活躍用戶數
        activeUsersKey := keys.ActiveUsersKey(tenantID, activityID)
        activeCount := mc.redis.PFCount(ctx, activeUsersKey).Val()
        ActiveUsers.With(prometheus.Labels{
            "tenant_id":   tenantID,
            "activity_id": activityIDStr,
        }).Set(float64(activeCount))
        totalActiveUsers += activeCount

        // 收集進入隊列總數
        enterTotalKey := keys.MetricsKey(tenantID, activityID, "enter_total")
        enterTotal := mc.getRedisInt(ctx, enterTotalKey)
        QueueEnterTotal.With(prometheus.Labels{
            "tenant_id":   tenantID,
            "activity_id": activityIDStr,
            "status":      "success",
        }).Add(float64(enterTotal))

        // 收集釋放總數
        releaseTotalKey := keys.MetricsKey(tenantID, activityID, "release_total")
        releaseTotal := mc.getRedisInt(ctx, releaseTotalKey)
        QueueReleaseTotal.With(prometheus.Labels{
            "tenant_id":   tenantID,
            "activity_id": activityIDStr,
            "method":      "scheduler",
        }).Add(float64(releaseTotal))

        // 檢查調度器狀態
        schedulerStatusKey := keys.MetricsKey(tenantID, activityID, "scheduler_status")
        if status := mc.redis.Get(ctx, schedulerStatusKey).Val(); status == "active" {
            activeSchedulers++
        }
    }

    // 設置全域指標
    mc.redis.Set(ctx, "global:metrics:total_queue_length", totalQueueLength, 0)
    mc.redis.Set(ctx, "global:metrics:total_active_users", totalActiveUsers, 0)
    mc.redis.Set(ctx, "global:metrics:active_schedulers", activeSchedulers, 0)
}

func (mc *MetricsCollector) collectSchedulerMetrics(ctx context.Context) {
    // 收集調度器狀態
    pattern := "t:*:a:*:metrics:scheduler_status"
    keys, err := mc.redis.Keys(ctx, pattern).Result()
    if err != nil {
        log.Printf("Failed to get scheduler status keys: %v", err)
        return
    }

    for _, key := range keys {
        // 解析 key 獲取 tenant_id 和 activity_id
        // 格式: t:{tenant_id}:a:{activity_id}:metrics:scheduler_status
        parts := strings.Split(key, ":")
        if len(parts) >= 4 {
            tenantID := parts[1]
            activityIDStr := parts[3]
            
            status := mc.redis.Get(ctx, key).Val()
            if status == "running" {
                SchedulerActive.With(prometheus.Labels{
                    "tenant_id":   tenantID,
                    "activity_id": activityIDStr,
                }).Set(1)

                // 收集釋放速率
                rateKey := fmt.Sprintf("t:%s:a:%s:metrics:current_release_rate", tenantID, activityIDStr)
                rate := mc.getRedisInt(ctx, rateKey)
                SchedulerReleaseRate.With(prometheus.Labels{
                    "tenant_id":   tenantID,
                    "activity_id": activityIDStr,
                }).Set(float64(rate))

                // 收集總釋放數
                totalKey := fmt.Sprintf("t:%s:a:%s:metrics:total_released", tenantID, activityIDStr)
                total := mc.getRedisInt(ctx, totalKey)
                SchedulerTotalReleased.With(prometheus.Labels{
                    "tenant_id":   tenantID,
                    "activity_id": activityIDStr,
                }).Set(float64(total))
            } else {
                SchedulerActive.With(prometheus.Labels{
                    "tenant_id":   tenantID,
                    "activity_id": activityIDStr,
                }).Set(0)
            }
        }
    }
}

func (mc *MetricsCollector) collectSystemMetrics(ctx context.Context) {
    // Redis 連接數
    info := mc.redis.Info(ctx, "clients").Val()
    if strings.Contains(info, "connected_clients:") {
        lines := strings.Split(info, "\n")
        for _, line := range lines {
            if strings.HasPrefix(line, "connected_clients:") {
                parts := strings.Split(line, ":")
                if len(parts) == 2 {
                    if count, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil {
                        RedisConnections.Set(float64(count))
                    }
                }
                break
            }
        }
    }

    // 資料庫連接數
    var dbConnections int
    err := mc.db.QueryRowContext(ctx, "SELECT numbackends FROM pg_stat_database WHERE datname = current_database()").Scan(&dbConnections)
    if err == nil {
        DatabaseConnections.Set(float64(dbConnections))
    }
}

func (mc *MetricsCollector) getRedisInt(ctx context.Context, key string) int64 {
    val := mc.redis.Get(ctx, key).Val()
    if val == "" {
        return 0
    }
    
    intVal, err := strconv.ParseInt(val, 10, 64)
    if err != nil {
        return 0
    }
    
    return intVal
}

// 記錄 API 請求指標
func RecordAPIRequest(method, endpoint, status string, duration time.Duration) {
    APIRequestTotal.With(prometheus.Labels{
        "method":   method,
        "endpoint": endpoint,
        "status":   status,
    }).Inc()

    APIRequestDuration.With(prometheus.Labels{
        "method":   method,
        "endpoint": endpoint,
        "status":   status,
    }).Observe(duration.Seconds())
}

// 記錄隊列進入
func RecordQueueEnter(tenantID string, activityID int64, success bool) {
    status := "success"
    if !success {
        status = "failed"
    }

    QueueEnterTotal.With(prometheus.Labels{
        "tenant_id":   tenantID,
        "activity_id": strconv.FormatInt(activityID, 10),
        "status":      status,
    }).Inc()
}

// 記錄隊列等待時間
func RecordQueueWaitTime(tenantID string, activityID int64, waitSeconds float64) {
    QueueWaitTime.With(prometheus.Labels{
        "tenant_id":   tenantID,
        "activity_id": strconv.FormatInt(activityID, 10),
    }).Observe(waitSeconds)
}

// 記錄隊列釋放
func RecordQueueRelease(tenantID string, activityID int64, count int64, method string) {
    QueueReleaseTotal.With(prometheus.Labels{
        "tenant_id":   tenantID,
        "activity_id": strconv.FormatInt(activityID, 10),
        "method":      method,
    }).Add(float64(count))
}

// HTTP 中間件：記錄 API 指標
func MetricsMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        
        // 包裝 ResponseWriter 以捕獲狀態碼
        wrapped := &responseWriter{ResponseWriter: w, statusCode: 200}
        
        next.ServeHTTP(wrapped, r)
        
        duration := time.Since(start)
        status := strconv.Itoa(wrapped.statusCode)
        
        RecordAPIRequest(r.Method, r.URL.Path, status, duration)
    })
}

type responseWriter struct {
    http.ResponseWriter
    statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
    rw.statusCode = code
    rw.ResponseWriter.WriteHeader(code)
}

// 啟動 Prometheus HTTP 服務器
func StartMetricsServer(addr string) error {
    http.Handle("/metrics", promhttp.Handler())
    
    log.Printf("Starting Prometheus metrics server on %s", addr)
    return http.ListenAndServe(addr, nil)
}

