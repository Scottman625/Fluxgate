package services

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"queue-system/internal/models"

	"github.com/go-redis/redis/v8"
)

type AdminService struct {
	db    *sql.DB
	redis *redis.Client
}

func NewAdminService(db *sql.DB, redis *redis.Client) *AdminService {
	return &AdminService{
		db:    db,
		redis: redis,
	}
}

type CreateActivityRequest struct {
	TenantID     string                `json:"tenant_id" binding:"required"`
	Name         string                `json:"name" binding:"required"`
	SKU          string                `json:"sku" binding:"required"`
	InitialStock int                   `json:"initial_stock" binding:"required,min=1"`
	StartAt      time.Time             `json:"start_at" binding:"required"`
	EndAt        time.Time             `json:"end_at" binding:"required"`
	Config       models.ActivityConfig `json:"config"`
}

type CreateActivityResponse struct {
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}

func (s *AdminService) CreateActivity(ctx context.Context, req *CreateActivityRequest) (*CreateActivityResponse, error) {
	// 驗證時間範圍
	if req.EndAt.Before(req.StartAt) {
		return nil, fmt.Errorf("end_at must be after start_at")
	}

	// 設定預設配置
	if req.Config.ReleaseRate == 0 {
		req.Config.ReleaseRate = 10 // 預設每秒釋放 10 個
	}
	if req.Config.PollInterval == 0 {
		req.Config.PollInterval = 2000 // 預設 2 秒輪詢間隔
	}

	query := `
        INSERT INTO activities (tenant_id, name, sku, initial_stock, start_at, end_at, config_json)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
        RETURNING id, created_at`

	var resp CreateActivityResponse
	err := s.db.QueryRowContext(ctx, query,
		req.TenantID, req.Name, req.SKU, req.InitialStock,
		req.StartAt, req.EndAt, req.Config).Scan(&resp.ID, &resp.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create activity: %w", err)
	}

	return &resp, nil
}

type ActivityStatusResponse struct {
	Activity      *models.Activity `json:"activity"`
	QueueMetrics  QueueMetrics     `json:"queue_metrics"`
	RealtimeStats RealtimeStats    `json:"realtime_stats"`
}

type QueueMetrics struct {
	QueueSeq    int64 `json:"queue_seq"`
	ReleaseSeq  int64 `json:"release_seq"`
	QueueLength int64 `json:"queue_length"`
	ActiveUsers int64 `json:"active_users"`
}

type RealtimeStats struct {
	EnterTotal  int64     `json:"enter_total"`
	EnterRate   float64   `json:"enter_rate"`   // 每秒進入數
	ReleaseRate float64   `json:"release_rate"` // 每秒釋放數
	LastUpdated time.Time `json:"last_updated"`
}

func (s *AdminService) GetActivityStatus(ctx context.Context, activityID int64) (*ActivityStatusResponse, error) {
	// 1. 獲取活動基本資訊
	activity, err := s.getActivity(ctx, activityID)
	if err != nil {
		return nil, fmt.Errorf("activity not found: %w", err)
	}

	// 2. 獲取隊列指標
	queueMetrics, err := s.getQueueMetrics(ctx, activity.TenantID, activityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get queue metrics: %w", err)
	}

	// 3. 獲取即時統計
	realtimeStats, err := s.getRealtimeStats(ctx, activity.TenantID, activityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get realtime stats: %w", err)
	}

	return &ActivityStatusResponse{
		Activity:      activity,
		QueueMetrics:  *queueMetrics,
		RealtimeStats: *realtimeStats,
	}, nil
}

type UpdateActivityRequest struct {
	Status      *models.ActivityStatus `json:"status,omitempty"`
	ReleaseRate *int                   `json:"release_rate,omitempty"`
}

func (s *AdminService) UpdateActivity(ctx context.Context, activityID int64, req *UpdateActivityRequest) error {
	// 建構動態 SQL
	setParts := []string{}
	args := []interface{}{}
	argIndex := 1

	if req.Status != nil {
		setParts = append(setParts, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, *req.Status)
		argIndex++
	}

	if req.ReleaseRate != nil {
		// 更新活動配置中的 release_rate
		setParts = append(setParts, fmt.Sprintf("config_json = jsonb_set(config_json, '{release_rate}', $%d)", argIndex))
		args = append(args, *req.ReleaseRate)
		argIndex++
	}

	if len(setParts) == 0 {
		return fmt.Errorf("no fields to update")
	}

	setParts = append(setParts, fmt.Sprintf("updated_at = $%d", argIndex))
	args = append(args, time.Now())
	argIndex++

	args = append(args, activityID)

	query := fmt.Sprintf(`
        UPDATE activities 
        SET %s
        WHERE id = $%d`,
		joinStrings(setParts, ", "), argIndex)

	result, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update activity: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("activity not found")
	}

	return nil
}

func (s *AdminService) ListActivities(ctx context.Context, tenantID string) ([]*models.Activity, error) {
	query := `
        SELECT id, tenant_id, name, sku, initial_stock, start_at, end_at, status, config_json, created_at, updated_at
        FROM activities 
        WHERE tenant_id = $1
        ORDER BY created_at DESC`

	rows, err := s.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query activities: %w", err)
	}
	defer rows.Close()

	var activities []*models.Activity
	for rows.Next() {
		var activity models.Activity
		err := rows.Scan(
			&activity.ID, &activity.TenantID, &activity.Name, &activity.SKU,
			&activity.InitialStock, &activity.StartAt, &activity.EndAt,
			&activity.Status, &activity.Config, &activity.CreatedAt, &activity.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan activity: %w", err)
		}
		activities = append(activities, &activity)
	}

	return activities, nil
}

// 輔助方法
func (s *AdminService) getActivity(ctx context.Context, activityID int64) (*models.Activity, error) {
	query := `
        SELECT id, tenant_id, name, sku, initial_stock, start_at, end_at, status, config_json, created_at, updated_at
        FROM activities 
        WHERE id = $1`

	var activity models.Activity
	err := s.db.QueryRowContext(ctx, query, activityID).Scan(
		&activity.ID, &activity.TenantID, &activity.Name, &activity.SKU,
		&activity.InitialStock, &activity.StartAt, &activity.EndAt,
		&activity.Status, &activity.Config, &activity.CreatedAt, &activity.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &activity, nil
}

func (s *AdminService) getQueueMetrics(ctx context.Context, tenantID string, activityID int64) (*QueueMetrics, error) {
	// 使用 pipeline 批量獲取 Redis 數據
	pipe := s.redis.Pipeline()

	queueSeqCmd := pipe.Get(ctx, fmt.Sprintf("t:%s:a:%d:queue_seq", tenantID, activityID))
	releaseSeqCmd := pipe.Get(ctx, fmt.Sprintf("t:%s:a:%d:release_seq", tenantID, activityID))
	activeUsersCmd := pipe.PFCount(ctx, fmt.Sprintf("t:%s:a:%d:active_users", tenantID, activityID))

	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return nil, err
	}

	queueSeq := parseInt64(queueSeqCmd.Val(), 0)
	releaseSeq := parseInt64(releaseSeqCmd.Val(), 0)
	activeUsers := activeUsersCmd.Val()

	return &QueueMetrics{
		QueueSeq:    queueSeq,
		ReleaseSeq:  releaseSeq,
		QueueLength: max(0, queueSeq-releaseSeq),
		ActiveUsers: activeUsers,
	}, nil
}

func (s *AdminService) getRealtimeStats(ctx context.Context, tenantID string, activityID int64) (*RealtimeStats, error) {
	// 獲取進入總數
	enterTotalKey := fmt.Sprintf("t:%s:a:%d:metrics:enter_total", tenantID, activityID)
	enterTotal := parseInt64(s.redis.Get(ctx, enterTotalKey).Val(), 0)

	// 這裡簡化處理，實際應該計算速率
	return &RealtimeStats{
		EnterTotal:  enterTotal,
		EnterRate:   0, // 需要基於時間窗口計算
		ReleaseRate: 0, // 需要基於時間窗口計算
		LastUpdated: time.Now(),
	}, nil
}

func parseInt64(s string, defaultVal int64) int64 {
	if s == "" {
		return defaultVal
	}
	if val, err := strconv.ParseInt(s, 10, 64); err == nil {
		return val
	}
	return defaultVal
}

func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	if len(strs) == 1 {
		return strs[0]
	}

	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
