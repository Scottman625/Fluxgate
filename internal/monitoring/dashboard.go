package monitoring

import (
    "context"
    "database/sql"
    "encoding/json"
    "fmt"
    "strconv"
    "strings"
    "time"

    "github.com/go-redis/redis/v8"
    "queue-system/pkg/keys"
)

type Dashboard struct {
    db    *sql.DB
    redis *redis.Client
}

type DashboardData struct {
    Overview    *OverviewStats    `json:"overview"`
    Activities  []*ActivityStats  `json:"activities"`
    Schedulers  []*SchedulerStats `json:"schedulers"`
    SystemStats *SystemStats      `json:"system_stats"`
    Timestamp   time.Time         `json:"timestamp"`
}

type OverviewStats struct {
    TotalActivities   int     `json:"total_activities"`
    ActiveActivities  int     `json:"active_activities"`
    TotalUsersInQueue int64   `json:"total_users_in_queue"`
    TotalReleaseRate  int     `json:"total_release_rate"`
    AvgWaitTime       float64 `json:"avg_wait_time_seconds"`
}

type ActivityStats struct {
    ID               int64     `json:"id"`
    TenantID         string    `json:"tenant_id"`
    Name             string    `json:"name"`
    Status           string    `json:"status"`
    QueueLength      int64     `json:"queue_length"`
    ReleaseSeq       int64     `json:"release_seq"`
    QueueSeq         int64     `json:"queue_seq"`
    ActiveUsers      int64     `json:"active_users"`
    ReleaseRate      int       `json:"release_rate"`
    EstimatedWait    int       `json:"estimated_wait_seconds"`
    TotalEntered     int64     `json:"total_entered"`
    TotalReleased    int64     `json:"total_released"`
    CreatedAt        time.Time `json:"created_at"`
    StartAt          time.Time `json:"start_at"`
    EndAt            time.Time `json:"end_at"`
}

type SchedulerStats struct {
    ActivityID      int64     `json:"activity_id"`
    TenantID        string    `json:"tenant_id"`
    Status          string    `json:"status"`
    ReleaseRate     int       `json:"release_rate"`
    TotalReleased   int64     `json:"total_released"`
    LastRelease     time.Time `json:"last_release"`
    ReleasesPerHour int64     `json:"releases_per_hour"`
}

type SystemStats struct {
    RedisConnections    int     `json:"redis_connections"`
    DatabaseConnections int     `json:"database_connections"`
    MemoryUsage         int64   `json:"memory_usage_bytes"`
    CPUUsage            float64 `json:"cpu_usage_percent"`
    Uptime              int64   `json:"uptime_seconds"`
}

func NewDashboard(db *sql.DB, redis *redis.Client) *Dashboard {
    return &Dashboard{
        db:    db,
        redis: redis,
    }
}

func (d *Dashboard) GetDashboardData(ctx context.Context) (*DashboardData, error) {
    data := &DashboardData{
        Timestamp: time.Now(),
    }

    // 並行收集各種統計數據
    errCh := make(chan error, 4)
    
    go func() {
        overview, err := d.getOverviewStats(ctx)
        data.Overview = overview
        errCh <- err
    }()
    
    go func() {
        activities, err := d.getActivityStats(ctx)
        data.Activities = activities
        errCh <- err
    }()
    
    go func() {
        schedulers, err := d.getSchedulerStats(ctx)
        data.Schedulers = schedulers
        errCh <- err
    }()
    
    go func() {
        systemStats, err := d.getSystemStats(ctx)
        data.SystemStats = systemStats
        errCh <- err
    }()

    // 等待所有 goroutine 完成
    for i := 0; i < 4; i++ {
        if err := <-errCh; err != nil {
            return nil, fmt.Errorf("failed to collect dashboard data: %w", err)
        }
    }

    return data, nil
}

func (d *Dashboard) getOverviewStats(ctx context.Context) (*OverviewStats, error) {
    stats := &OverviewStats{}

    // 查詢活動統計
    query := `
        SELECT 
            COUNT(*) as total,
            COUNT(*) FILTER (WHERE status = 'active') as active
        FROM activities`

    err := d.db.QueryRowContext(ctx, query).Scan(&stats.TotalActivities, &stats.ActiveActivities)
    if err != nil {
        return nil, err
    }

    // 計算總排隊人數和釋放速率
    activities, err := d.getActivityStats(ctx)
    if err != nil {
        return nil, err
    }

    var totalQueue int64
    var totalRate int
    var totalWaitTime float64
    var activeCount int

    for _, activity := range activities {
        if activity.Status == "active" {
            totalQueue += activity.QueueLength
            totalRate += activity.ReleaseRate
            if activity.EstimatedWait > 0 {
                totalWaitTime += float64(activity.EstimatedWait)
                activeCount++
            }
        }
    }

    stats.TotalUsersInQueue = totalQueue
    stats.TotalReleaseRate = totalRate
    
    if activeCount > 0 {
        stats.AvgWaitTime = totalWaitTime / float64(activeCount)
    }

    return stats, nil
}

func (d *Dashboard) getActivityStats(ctx context.Context) ([]*ActivityStats, error) {
    query := `
        SELECT 
            id, tenant_id, name, status, config_json,
            created_at, start_at, end_at
        FROM activities 
        ORDER BY created_at DESC
        LIMIT 50`

    rows, err := d.db.QueryContext(ctx, query)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var activities []*ActivityStats

    for rows.Next() {
        activity := &ActivityStats{}
        var configJSON string

        err := rows.Scan(
            &activity.ID, &activity.TenantID, &activity.Name, &activity.Status,
            &configJSON, &activity.CreatedAt, &activity.StartAt, &activity.EndAt,
        )
        if err != nil {
            continue
        }

        // 解析配置
        var config struct {
            ReleaseRate int `json:"release_rate"`
        }
        json.Unmarshal([]byte(configJSON), &config)
        activity.ReleaseRate = config.ReleaseRate

        // 從 Redis 獲取實時數據
        d.enrichActivityWithRedisData(ctx, activity)

        activities = append(activities, activity)
    }

    return activities, nil
}

func (d *Dashboard) enrichActivityWithRedisData(ctx context.Context, activity *ActivityStats) {
    // 獲取隊列序號
    queueSeqKey := keys.QueueSeqKey(activity.TenantID, activity.ID)
    queueSeq, _ := d.redis.Get(ctx, queueSeqKey).Int64()
    activity.QueueSeq = queueSeq

    // 獲取釋放序號
    releaseSeqKey := keys.ReleaseSeqKey(activity.TenantID, activity.ID)
    releaseSeq, _ := d.redis.Get(ctx, releaseSeqKey).Int64()
    activity.ReleaseSeq = releaseSeq

    // 計算隊列長度
    activity.QueueLength = queueSeq - releaseSeq

    // 獲取活躍用戶數
    activeUsersKey := keys.ActiveUsersKey(activity.TenantID, activity.ID)
    activeUsers := d.redis.PFCount(ctx, activeUsersKey).Val()
    activity.ActiveUsers = activeUsers

    // 獲取總進入數
    enterTotalKey := keys.MetricsKey(activity.TenantID, activity.ID, "enter_total")
    enterTotal, _ := d.redis.Get(ctx, enterTotalKey).Int64()
    activity.TotalEntered = enterTotal

    // 獲取總釋放數
    releaseTotalKey := keys.MetricsKey(activity.TenantID, activity.ID, "release_total")
    releaseTotal, _ := d.redis.Get(ctx, releaseTotalKey).Int64()
    activity.TotalReleased = releaseTotal

    // 計算預估等待時間
    if activity.ReleaseRate > 0 && activity.QueueLength > 0 {
        activity.EstimatedWait = int(activity.QueueLength) / activity.ReleaseRate
    }
}

func (d *Dashboard) getSchedulerStats(ctx context.Context) ([]*SchedulerStats, error) {
    var schedulers []*SchedulerStats

    // 查找所有調度器狀態
    pattern := "t:*:a:*:metrics:scheduler_status"
    	redisKeys, err := d.redis.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, err
	}

	for _, key := range redisKeys {
        // 解析 key
        parts := strings.Split(key, ":")
        if len(parts) < 4 {
            continue
        }

        tenantID := parts[1]
        activityIDStr := parts[3]
        activityID, err := strconv.ParseInt(activityIDStr, 10, 64)
        if err != nil {
            continue
        }

        scheduler := &SchedulerStats{
            ActivityID: activityID,
            TenantID:   tenantID,
        }

        // 獲取狀態
        status := d.redis.Get(ctx, key).Val()
        scheduler.Status = status

        if status == "running" {
            // 獲取釋放速率
            rateKey := keys.MetricsKey(tenantID, activityID, "current_release_rate")
            rate, _ := d.redis.Get(ctx, rateKey).Int()
            scheduler.ReleaseRate = rate

            // 獲取總釋放數
            totalKey := keys.MetricsKey(tenantID, activityID, "total_released")
            total, _ := d.redis.Get(ctx, totalKey).Int64()
            scheduler.TotalReleased = total

            // 計算每小時釋放數
            currentHour := time.Now().Hour()
            hourlyKey := keys.MetricsKey(tenantID, activityID, fmt.Sprintf("release_hourly_%d", currentHour))
            hourlyReleases, _ := d.redis.Get(ctx, hourlyKey).Int64()
            scheduler.ReleasesPerHour = hourlyReleases
        }

        schedulers = append(schedulers, scheduler)
    }

    return schedulers, nil
}

func (d *Dashboard) getSystemStats(ctx context.Context) (*SystemStats, error) {
    stats := &SystemStats{}

    // Redis 連接數
    info := d.redis.Info(ctx, "clients").Val()
    if strings.Contains(info, "connected_clients:") {
        lines := strings.Split(info, "\n")
        for _, line := range lines {
            if strings.HasPrefix(line, "connected_clients:") {
                parts := strings.Split(line, ":")
                if len(parts) == 2 {
                    if count, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil {
                        stats.RedisConnections = count
                    }
                }
                break
            }
        }
    }

    // 資料庫連接數
    err := d.db.QueryRowContext(ctx, "SELECT numbackends FROM pg_stat_database WHERE datname = current_database()").Scan(&stats.DatabaseConnections)
    if err != nil {
        stats.DatabaseConnections = 0
    }

    // 記憶體使用量（從 Redis 獲取）
    memInfo := d.redis.Info(ctx, "memory").Val()
    if strings.Contains(memInfo, "used_memory:") {
        lines := strings.Split(memInfo, "\n")
        for _, line := range lines {
            if strings.HasPrefix(line, "used_memory:") {
                parts := strings.Split(line, ":")
                if len(parts) == 2 {
                    if memory, err := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64); err == nil {
                        stats.MemoryUsage = memory
                    }
                }
                break
            }
        }
    }

    return stats, nil
}

// 獲取活動的歷史數據
func (d *Dashboard) GetActivityHistory(ctx context.Context, tenantID string, activityID int64, hours int) ([]map[string]interface{}, error) {
    var history []map[string]interface{}

    // 從 Redis 獲取歷史釋放事件
    eventKey := fmt.Sprintf("t:%s:a:%d:events:release", tenantID, activityID)
    events, err := d.redis.LRange(ctx, eventKey, 0, -1).Result()
    if err != nil {
        return nil, err
    }

    cutoffTime := time.Now().Add(-time.Duration(hours) * time.Hour)

    for _, eventStr := range events {
        var event map[string]interface{}
        if err := json.Unmarshal([]byte(eventStr), &event); err != nil {
            continue
        }

        // 解析時間戳
        if timestampStr, ok := event["timestamp"].(string); ok {
            if timestamp, err := time.Parse(time.RFC3339, timestampStr); err == nil {
                if timestamp.After(cutoffTime) {
                    history = append(history, event)
                }
            }
        }
    }

    return history, nil
}

// 獲取實時指標
func (d *Dashboard) GetRealTimeMetrics(ctx context.Context) (map[string]interface{}, error) {
    metrics := make(map[string]interface{})

    // 獲取全域指標
    globalKeys := []string{
        "global:metrics:active_schedulers",
        "global:metrics:total_queue_length",
        "global:metrics:total_active_users",
    }

    for _, key := range globalKeys {
        val := d.redis.Get(ctx, key).Val()
        if val != "" {
            if intVal, err := strconv.ParseInt(val, 10, 64); err == nil {
                metrics[key] = intVal
            }
        }
    }

    // 添加時間戳
    metrics["timestamp"] = time.Now().Unix()

    return metrics, nil
}
