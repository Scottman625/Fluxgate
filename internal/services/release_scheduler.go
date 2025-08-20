package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	    "queue-system/internal/models"
    "queue-system/pkg/keys"
)

type ReleaseScheduler struct {
	db       *sql.DB
	redis    *redis.Client
	running  map[int64]*SchedulerTask
	mu       sync.RWMutex
	stopChan chan struct{}
	wg       sync.WaitGroup
}

type SchedulerTask struct {
	ActivityID    int64
	TenantID      string
	ReleaseRate   int
	StopChan      chan struct{}
	LastRelease   time.Time
	TotalReleased int64
}

type ReleaseEvent struct {
	ActivityID   int64     `json:"activity_id"`
	TenantID     string    `json:"tenant_id"`
	PrevSeq      int64     `json:"prev_seq"`
	NewSeq       int64     `json:"new_seq"`
	ReleaseCount int64     `json:"release_count"`
	Timestamp    time.Time `json:"timestamp"`
	ReleaseRate  int       `json:"release_rate"`
}

func NewReleaseScheduler(db *sql.DB, redis *redis.Client) *ReleaseScheduler {
	return &ReleaseScheduler{
		db:       db,
		redis:    redis,
		running:  make(map[int64]*SchedulerTask),
		stopChan: make(chan struct{}),
	}
}

func (rs *ReleaseScheduler) Start(ctx context.Context) error {
	log.Println("Starting Release Scheduler...")

	// 載入所有活躍活動
	if err := rs.loadActiveActivities(ctx); err != nil {
		return fmt.Errorf("failed to load active activities: %w", err)
	}

	// 啟動監控 goroutine，定期檢查新活動
	rs.wg.Add(1)
	go rs.monitorActivities(ctx)

	// 啟動指標收集 goroutine
	rs.wg.Add(1)
	go rs.collectMetrics(ctx)

	log.Println("Release Scheduler started successfully")
	return nil
}

func (rs *ReleaseScheduler) Stop() {
	log.Println("Stopping Release Scheduler...")

	close(rs.stopChan)

	// 停止所有任務
	rs.mu.Lock()
	for activityID, task := range rs.running {
		close(task.StopChan)
		delete(rs.running, activityID)
	}
	rs.mu.Unlock()

	rs.wg.Wait()
	log.Println("Release Scheduler stopped")
}

func (rs *ReleaseScheduler) loadActiveActivities(ctx context.Context) error {
	query := `
        SELECT id, tenant_id, config_json
        FROM activities 
        WHERE status = 'active' 
        AND start_at <= NOW() 
        AND end_at > NOW()`

	rows, err := rs.db.QueryContext(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var activityID int64
		var tenantID string
		var config models.ActivityConfig

		if err := rows.Scan(&activityID, &tenantID, &config); err != nil {
			log.Printf("Failed to scan activity %d: %v", activityID, err)
			continue
		}

		if config.ReleaseRate > 0 {
			if err := rs.startActivityScheduler(ctx, activityID, tenantID, config.ReleaseRate); err != nil {
				log.Printf("Failed to start scheduler for activity %d: %v", activityID, err)
			}
		}
	}

	return nil
}

func (rs *ReleaseScheduler) startActivityScheduler(ctx context.Context, activityID int64, tenantID string, releaseRate int) error {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	// 檢查是否已經在運行
	if _, exists := rs.running[activityID]; exists {
		return nil
	}

	// 獲取當前 release_seq
	currentSeq, err := rs.getCurrentReleaseSeq(ctx, tenantID, activityID)
	if err != nil {
		log.Printf("Failed to get current release seq for activity %d: %v", activityID, err)
		currentSeq = 0
	}

	task := &SchedulerTask{
		ActivityID:    activityID,
		TenantID:      tenantID,
		ReleaseRate:   releaseRate,
		StopChan:      make(chan struct{}),
		LastRelease:   time.Now(),
		TotalReleased: currentSeq,
	}

	rs.running[activityID] = task

	// 啟動該活動的釋放任務
	rs.wg.Add(1)
	go rs.runActivityScheduler(ctx, task)

	log.Printf("Started release scheduler for activity %d (rate: %d/sec)", activityID, releaseRate)
	return nil
}

func (rs *ReleaseScheduler) runActivityScheduler(ctx context.Context, task *SchedulerTask) {
	defer rs.wg.Done()
	defer func() {
		rs.mu.Lock()
		delete(rs.running, task.ActivityID)
		rs.mu.Unlock()
		log.Printf("Stopped release scheduler for activity %d", task.ActivityID)
	}()

	// 計算釋放間隔（毫秒）
	releaseInterval := time.Duration(1000/task.ReleaseRate) * time.Millisecond
	if releaseInterval < 10*time.Millisecond {
		releaseInterval = 10 * time.Millisecond // 最小間隔 10ms
	}

	ticker := time.NewTicker(releaseInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-rs.stopChan:
			return
		case <-task.StopChan:
			return
		case <-ticker.C:
			if err := rs.performRelease(ctx, task); err != nil {
				log.Printf("Release failed for activity %d: %v", task.ActivityID, err)
			}
		}
	}
}

func (rs *ReleaseScheduler) performRelease(ctx context.Context, task *SchedulerTask) error {
	// 檢查活動是否仍然活躍
	if !rs.isActivityStillActive(ctx, task.ActivityID) {
		log.Printf("Activity %d is no longer active, stopping scheduler", task.ActivityID)
		close(task.StopChan)
		return nil
	}

	// 獲取當前隊列長度
	queueSeq, err := rs.getCurrentQueueSeq(ctx, task.TenantID, task.ActivityID)
	if err != nil {
		return fmt.Errorf("failed to get queue seq: %w", err)
	}

	releaseSeq, err := rs.getCurrentReleaseSeq(ctx, task.TenantID, task.ActivityID)
	if err != nil {
		return fmt.Errorf("failed to get release seq: %w", err)
	}

	// 計算需要釋放的數量
	queueLength := queueSeq - releaseSeq
	if queueLength <= 0 {
		return nil // 沒有人在排隊
	}

	// 計算本次釋放數量（基於時間間隔和速率）
	now := time.Now()
	timeSinceLastRelease := now.Sub(task.LastRelease)
	expectedReleases := int64(float64(task.ReleaseRate) * timeSinceLastRelease.Seconds())

	if expectedReleases <= 0 {
		expectedReleases = 1 // 至少釋放 1 個
	}

	// 不能超過隊列長度
	releaseCount := int(minInt64(expectedReleases, queueLength))
	if releaseCount <= 0 {
		return nil
	}

	// 執行釋放
	newReleaseSeq := releaseSeq + int64(releaseCount)
	if err := rs.updateReleaseSeq(ctx, task.TenantID, task.ActivityID, newReleaseSeq); err != nil {
		return fmt.Errorf("failed to update release seq: %w", err)
	}

	// 記錄釋放事件
	event := &ReleaseEvent{
		ActivityID:   task.ActivityID,
		TenantID:     task.TenantID,
		PrevSeq:      releaseSeq,
		NewSeq:       newReleaseSeq,
		ReleaseCount: int64(releaseCount),
		Timestamp:    now,
		ReleaseRate:  task.ReleaseRate,
	}

	// 異步記錄事件和更新指標
	go rs.recordReleaseEvent(context.Background(), event)
	go rs.updateReleaseMetrics(context.Background(), task.TenantID, task.ActivityID, int64(releaseCount))

	// 更新任務狀態
	task.LastRelease = now
	task.TotalReleased = newReleaseSeq

	log.Printf("Released %d positions for activity %d (new release_seq: %d)",
		releaseCount, task.ActivityID, newReleaseSeq)

	return nil
}

func (rs *ReleaseScheduler) monitorActivities(ctx context.Context) {
	defer rs.wg.Done()

	ticker := time.NewTicker(30 * time.Second) // 每 30 秒檢查一次
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-rs.stopChan:
			return
		case <-ticker.C:
			if err := rs.syncActiveActivities(ctx); err != nil {
				log.Printf("Failed to sync active activities: %v", err)
			}
		}
	}
}

func (rs *ReleaseScheduler) syncActiveActivities(ctx context.Context) error {
	// 獲取當前活躍活動
	query := `
        SELECT id, tenant_id, config_json
        FROM activities 
        WHERE status = 'active' 
        AND start_at <= NOW() 
        AND end_at > NOW()`

	rows, err := rs.db.QueryContext(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	activeActivities := make(map[int64]bool)

	for rows.Next() {
		var activityID int64
		var tenantID string
		var config models.ActivityConfig

		if err := rows.Scan(&activityID, &tenantID, &config); err != nil {
			continue
		}

		activeActivities[activityID] = true

		// 檢查是否需要啟動新的調度器
		rs.mu.RLock()
		_, exists := rs.running[activityID]
		rs.mu.RUnlock()

		if !exists && config.ReleaseRate > 0 {
			if err := rs.startActivityScheduler(ctx, activityID, tenantID, config.ReleaseRate); err != nil {
				log.Printf("Failed to start scheduler for new activity %d: %v", activityID, err)
			}
		} else if exists {
			// 檢查是否需要更新釋放速率
			rs.mu.RLock()
			task := rs.running[activityID]
			rs.mu.RUnlock()

			if task != nil && task.ReleaseRate != config.ReleaseRate {
				log.Printf("Updating release rate for activity %d: %d -> %d",
					activityID, task.ReleaseRate, config.ReleaseRate)
				task.ReleaseRate = config.ReleaseRate
			}
		}
	}

	// 停止不再活躍的活動調度器
	rs.mu.Lock()
	for activityID, task := range rs.running {
		if !activeActivities[activityID] {
			close(task.StopChan)
			delete(rs.running, activityID)
			log.Printf("Stopped scheduler for inactive activity %d", activityID)
		}
	}
	rs.mu.Unlock()

	return nil
}

func (rs *ReleaseScheduler) collectMetrics(ctx context.Context) {
	defer rs.wg.Done()

	ticker := time.NewTicker(10 * time.Second) // 每 10 秒收集一次指標
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-rs.stopChan:
			return
		case <-ticker.C:
			rs.updateSchedulerMetrics(ctx)
		}
	}
}

func (rs *ReleaseScheduler) updateSchedulerMetrics(ctx context.Context) {
	rs.mu.RLock()
	runningCount := len(rs.running)

	for activityID, task := range rs.running {
		// 更新調度器狀態指標
		key := keys.MetricsKey(task.TenantID, activityID, "scheduler_status")
		rs.redis.Set(ctx, key, "running", time.Hour)

		// 更新總釋放數
		key = keys.MetricsKey(task.TenantID, activityID, "total_released")
		rs.redis.Set(ctx, key, task.TotalReleased, time.Hour)

		// 更新當前釋放速率
		key = keys.MetricsKey(task.TenantID, activityID, "current_release_rate")
		rs.redis.Set(ctx, key, task.ReleaseRate, time.Hour)
	}
	rs.mu.RUnlock()

	// 更新全域指標
	rs.redis.Set(ctx, "global:metrics:active_schedulers", runningCount, time.Hour)
}

// 手動控制方法
func (rs *ReleaseScheduler) UpdateReleaseRate(ctx context.Context, activityID int64, newRate int) error {
	rs.mu.RLock()
	task, exists := rs.running[activityID]
	rs.mu.RUnlock()

	if !exists {
		return fmt.Errorf("scheduler not running for activity %d", activityID)
	}

	// 更新資料庫中的配置
	query := `
        UPDATE activities 
        SET config_json = jsonb_set(config_json, '{release_rate}', $1),
            updated_at = NOW()
        WHERE id = $2`

	_, err := rs.db.ExecContext(ctx, query, newRate, activityID)
	if err != nil {
		return fmt.Errorf("failed to update database: %w", err)
	}

	// 更新記憶體中的速率
	task.ReleaseRate = newRate

	log.Printf("Updated release rate for activity %d to %d/sec", activityID, newRate)
	return nil
}

func (rs *ReleaseScheduler) ManualRelease(ctx context.Context, activityID int64, count int64) error {
	rs.mu.RLock()
	task, exists := rs.running[activityID]
	rs.mu.RUnlock()

	if !exists {
		return fmt.Errorf("scheduler not running for activity %d", activityID)
	}

	releaseSeq, err := rs.getCurrentReleaseSeq(ctx, task.TenantID, activityID)
	if err != nil {
		return fmt.Errorf("failed to get current release seq: %w", err)
	}

	newReleaseSeq := releaseSeq + count
	if err := rs.updateReleaseSeq(ctx, task.TenantID, activityID, newReleaseSeq); err != nil {
		return fmt.Errorf("failed to update release seq: %w", err)
	}

	// 記錄手動釋放事件
	event := &ReleaseEvent{
		ActivityID:   activityID,
		TenantID:     task.TenantID,
		PrevSeq:      releaseSeq,
		NewSeq:       newReleaseSeq,
		ReleaseCount: count,
		Timestamp:    time.Now(),
		ReleaseRate:  -1, // 標記為手動釋放
	}

	go rs.recordReleaseEvent(context.Background(), event)

	log.Printf("Manual release: %d positions for activity %d", count, activityID)
	return nil
}

// 輔助方法
func (rs *ReleaseScheduler) getCurrentQueueSeq(ctx context.Context, tenantID string, activityID int64) (int64, error) {
	key := keys.QueueSeqKey(tenantID, activityID)
	result := rs.redis.Get(ctx, key)
	if result.Err() == redis.Nil {
		return 0, nil
	}
	if result.Err() != nil {
		return 0, result.Err()
	}
	return strconv.ParseInt(result.Val(), 10, 64)
}

func (rs *ReleaseScheduler) getCurrentReleaseSeq(ctx context.Context, tenantID string, activityID int64) (int64, error) {
	key := keys.ReleaseSeqKey(tenantID, activityID)
	result := rs.redis.Get(ctx, key)
	if result.Err() == redis.Nil {
		return 0, nil
	}
	if result.Err() != nil {
		return 0, result.Err()
	}
	return strconv.ParseInt(result.Val(), 10, 64)
}

func (rs *ReleaseScheduler) updateReleaseSeq(ctx context.Context, tenantID string, activityID int64, newSeq int64) error {
	key := keys.ReleaseSeqKey(tenantID, activityID)
	return rs.redis.Set(ctx, key, newSeq, 24*time.Hour).Err()
}

func (rs *ReleaseScheduler) isActivityStillActive(ctx context.Context, activityID int64) bool {
	query := `
        SELECT COUNT(*) 
        FROM activities 
        WHERE id = $1 
        AND status = 'active' 
        AND start_at <= NOW() 
        AND end_at > NOW()`

	var count int
	err := rs.db.QueryRowContext(ctx, query, activityID).Scan(&count)
	return err == nil && count > 0
}

func (rs *ReleaseScheduler) recordReleaseEvent(ctx context.Context, event *ReleaseEvent) {
	// 記錄到 Redis（用於即時監控）
	eventKey := fmt.Sprintf("t:%s:a:%d:events:release", event.TenantID, event.ActivityID)
	eventData, _ := json.Marshal(event)

	pipe := rs.redis.Pipeline()
	pipe.LPush(ctx, eventKey, eventData)
	pipe.LTrim(ctx, eventKey, 0, 99) // 只保留最近 100 個事件
	pipe.Expire(ctx, eventKey, time.Hour)
	pipe.Exec(ctx)
}

func (rs *ReleaseScheduler) updateReleaseMetrics(ctx context.Context, tenantID string, activityID int64, releaseCount int64) {
	// 更新總釋放數
	totalKey := keys.MetricsKey(tenantID, activityID, "release_total")
	rs.redis.IncrBy(ctx, totalKey, releaseCount)
	rs.redis.Expire(ctx, totalKey, 24*time.Hour)

	// 更新每小時釋放數（用於計算速率）
	hourKey := keys.MetricsKey(tenantID, activityID, fmt.Sprintf("release_hourly_%d", time.Now().Hour()))
	rs.redis.IncrBy(ctx, hourKey, releaseCount)
	rs.redis.Expire(ctx, hourKey, 2*time.Hour)
}

func minInt64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
