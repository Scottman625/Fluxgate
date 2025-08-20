package services

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"strconv"
	"time"

	"queue-system/internal/models"
	"queue-system/pkg/keys"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

type QueueService struct {
	db    *sql.DB
	redis *redis.Client
}

func NewQueueService(db *sql.DB, redis *redis.Client) *QueueService {
	return &QueueService{
		db:    db,
		redis: redis,
	}
}

type EnterQueueRequest struct {
	ActivityID  int64  `json:"activity_id" binding:"required"`
	UserHash    string `json:"user_hash" binding:"required"`
	Fingerprint string `json:"fingerprint"`
	IPAddress   string `json:"-"` // 從 header 取得，不從 body
}

type EnterQueueResponse struct {
	RequestID       string `json:"request_id"`
	Seq             int64  `json:"seq"`
	EstimatedWait   int    `json:"estimated_wait"`
	PollingInterval int    `json:"polling_interval"`
	SessionID       string `json:"session_id"`
	QueueLength     int64  `json:"queue_length"`
}

func (s *QueueService) EnterQueue(ctx context.Context, req *EnterQueueRequest) (*EnterQueueResponse, error) {
	requestID := uuid.New().String()

	// 1. 驗證活動存在且狀態正確
	activity, err := s.getActivity(ctx, req.ActivityID)
	if err != nil {
		return nil, fmt.Errorf("activity not found: %w", err)
	}

	if !s.isActivityActive(activity) {
		return nil, fmt.Errorf("activity is not active")
	}

	// 2. 檢查用戶是否已在隊列中
	sessionID := s.generateSessionID(req.UserHash, req.ActivityID)
	existingSeq, err := s.getExistingSeq(ctx, activity.TenantID, req.ActivityID, sessionID)
	if err == nil && existingSeq > 0 {
		// 用戶已在隊列中，返回現有序號
		queueLength, _ := s.getQueueLength(ctx, activity.TenantID, req.ActivityID)
		return &EnterQueueResponse{
			RequestID:       requestID,
			Seq:             existingSeq,
			EstimatedWait:   s.calculateETA(existingSeq, activity),
			PollingInterval: activity.Config.PollInterval,
			SessionID:       sessionID,
			QueueLength:     queueLength,
		}, nil
	}

	// 3. IP 節流檢查
	if err := s.checkIPThrottle(ctx, activity.TenantID, req.ActivityID, req.IPAddress); err != nil {
		return nil, err
	}

	// 4. 用戶去重檢查（防止同一用戶多個分頁）
	if err := s.checkUserDedupe(ctx, activity.TenantID, req.ActivityID, req.UserHash); err != nil {
		return nil, err
	}

	// 5. 分配序號並記錄
	seq, err := s.assignSequenceNumber(ctx, activity.TenantID, req.ActivityID, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to assign sequence number: %w", err)
	}

	// 6. 記錄到資料庫（非同步）
	go s.recordQueueEntry(context.Background(), &models.QueueEntry{
		ActivityID:  req.ActivityID,
		UserHash:    req.UserHash,
		SessionID:   sessionID,
		SeqNumber:   seq,
		Fingerprint: req.Fingerprint,
		IPHash:      s.hashIP(req.IPAddress),
		CreatedAt:   time.Now(),
	})

	// 7. 更新統計
	s.updateMetrics(ctx, activity.TenantID, req.ActivityID, "enter")

	queueLength, _ := s.getQueueLength(ctx, activity.TenantID, req.ActivityID)

	return &EnterQueueResponse{
		RequestID:       requestID,
		Seq:             seq,
		EstimatedWait:   s.calculateETA(seq, activity),
		PollingInterval: activity.Config.PollInterval,
		SessionID:       sessionID,
		QueueLength:     queueLength,
	}, nil
}

type QueueStatusRequest struct {
	ActivityID int64  `form:"activity_id" binding:"required"`
	Seq        int64  `form:"seq" binding:"required"`
	SessionID  string `form:"session_id" binding:"required"`
}

type QueueStatusResponse struct {
	RequestID   string      `json:"request_id"`
	ReleaseSeq  int64       `json:"release_seq"`
	QueueSeq    int64       `json:"queue_seq"`
	Position    int64       `json:"position"`
	ETA         int         `json:"eta"`
	ETADetails  *ETAResult  `json:"eta_details,omitempty"`
	State       QueueState  `json:"state"`
	QueueLength int64       `json:"queue_length"`
	NextPollMs  int         `json:"next_poll_ms"`
}

type QueueState string

const (
	StateWaiting  QueueState = "waiting"
	StateEligible QueueState = "eligible"
	StateExpired  QueueState = "expired"
)

func (s *QueueService) GetQueueStatus(ctx context.Context, req *QueueStatusRequest) (*QueueStatusResponse, error) {
	requestID := uuid.New().String()

	// 1. 驗證活動
	activity, err := s.getActivity(ctx, req.ActivityID)
	if err != nil {
		return nil, fmt.Errorf("activity not found: %w", err)
	}

	// 2. 驗證用戶序號
	userSeq, err := s.getExistingSeq(ctx, activity.TenantID, req.ActivityID, req.SessionID)
	if err != nil || userSeq != req.Seq {
		return nil, fmt.Errorf("invalid sequence number")
	}

	// 3. 獲取當前釋放序號
	releaseSeq, err := s.getReleaseSeq(ctx, activity.TenantID, req.ActivityID)
	if err != nil {
		releaseSeq = 0 // 預設值
	}

	// 4. 獲取隊列總長度
	queueSeq, err := s.getCurrentQueueSeq(ctx, activity.TenantID, req.ActivityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get queue length: %w", err)
	}

	// 5. 計算位置和狀態
	position := req.Seq - releaseSeq
	var state QueueState
	var nextPollMs int

	if position <= 0 {
		state = StateEligible
		nextPollMs = 0 // 立即可以請求 reservation
	} else {
		state = StateWaiting
		nextPollMs = activity.Config.PollInterval
	}

	// 6. 檢查是否過期（活動結束）
	if time.Now().After(activity.EndAt) {
		state = StateExpired
	}

	return &QueueStatusResponse{
		RequestID:   requestID,
		ReleaseSeq:  releaseSeq,
		QueueSeq:    queueSeq,
		Position:    max(0, position),
		ETA:         s.calculateETA(req.Seq, activity),
		State:       state,
		QueueLength: queueSeq - releaseSeq,
		NextPollMs:  nextPollMs,
	}, nil
}

// 輔助方法
func (s *QueueService) getActivity(ctx context.Context, activityID int64) (*models.Activity, error) {
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

func (s *QueueService) isActivityActive(activity *models.Activity) bool {
	now := time.Now()
	return activity.Status == models.StatusActive &&
		now.After(activity.StartAt) &&
		now.Before(activity.EndAt)
}

func (s *QueueService) generateSessionID(userHash string, activityID int64) string {
	data := fmt.Sprintf("%s:%d:%d", userHash, activityID, time.Now().Unix()/3600) // 每小時更新
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])[:16] // 取前16個字符
}

func (s *QueueService) getExistingSeq(ctx context.Context, tenantID string, activityID int64, sessionID string) (int64, error) {
	key := keys.UserQueueKey(tenantID, activityID, sessionID)
	result := s.redis.Get(ctx, key)
	if result.Err() != nil {
		return 0, result.Err()
	}

	return strconv.ParseInt(result.Val(), 10, 64)
}

func (s *QueueService) assignSequenceNumber(ctx context.Context, tenantID string, activityID int64, sessionID string) (int64, error) {
	// 使用 Redis 事務確保原子性
	pipe := s.redis.TxPipeline()

	// 增加隊列序號
	queueSeqKey := keys.QueueSeqKey(tenantID, activityID)
	incrCmd := pipe.Incr(ctx, queueSeqKey)

	// 設定用戶序號，TTL 4小時
	userQueueKey := keys.UserQueueKey(tenantID, activityID, sessionID)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, err
	}

	seq := incrCmd.Val()

	// 設定用戶序號和 TTL
	s.redis.Set(ctx, userQueueKey, seq, 4*time.Hour)

	// 更新活躍用戶統計
	activeUsersKey := keys.ActiveUsersKey(tenantID, activityID)
	s.redis.PFAdd(ctx, activeUsersKey, sessionID)

	return seq, nil
}

func (s *QueueService) checkIPThrottle(ctx context.Context, tenantID string, activityID int64, ipAddress string) error {
	if ipAddress == "" {
		return nil // 跳過檢查
	}

	ipHash := s.hashIP(ipAddress)
	key := keys.IPThrottleKey(tenantID, activityID, ipHash)

	// 使用滑動窗口限制：60秒內最多10次請求
	count, err := s.redis.Incr(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("throttle check failed: %w", err)
	}

	if count == 1 {
		s.redis.Expire(ctx, key, 60*time.Second)
	}

	if count > 10 {
		return fmt.Errorf("rate limit exceeded")
	}

	return nil
}

func (s *QueueService) checkUserDedupe(ctx context.Context, tenantID string, activityID int64, userHash string) error {
	key := keys.UserDedupeKey(tenantID, activityID)

	// 檢查用戶是否已存在
	exists, err := s.redis.SIsMember(ctx, key, userHash).Result()
	if err != nil {
		return fmt.Errorf("dedupe check failed: %w", err)
	}

	if exists {
		return fmt.Errorf("user already in queue")
	}

	// 添加用戶到去重集合，TTL 4小時
	s.redis.SAdd(ctx, key, userHash)
	s.redis.Expire(ctx, key, 4*time.Hour)

	return nil
}

func (s *QueueService) getReleaseSeq(ctx context.Context, tenantID string, activityID int64) (int64, error) {
	key := keys.ReleaseSeqKey(tenantID, activityID)
	result := s.redis.Get(ctx, key)
	if result.Err() == redis.Nil {
		return 0, nil // 預設值
	}
	if result.Err() != nil {
		return 0, result.Err()
	}

	return strconv.ParseInt(result.Val(), 10, 64)
}

func (s *QueueService) getCurrentQueueSeq(ctx context.Context, tenantID string, activityID int64) (int64, error) {
	key := keys.QueueSeqKey(tenantID, activityID)
	result := s.redis.Get(ctx, key)
	if result.Err() == redis.Nil {
		return 0, nil
	}
	if result.Err() != nil {
		return 0, result.Err()
	}

	return strconv.ParseInt(result.Val(), 10, 64)
}

func (s *QueueService) getQueueLength(ctx context.Context, tenantID string, activityID int64) (int64, error) {
	queueSeq, err := s.getCurrentQueueSeq(ctx, tenantID, activityID)
	if err != nil {
		return 0, err
	}

	releaseSeq, err := s.getReleaseSeq(ctx, tenantID, activityID)
	if err != nil {
		return 0, err
	}

	return max(0, queueSeq-releaseSeq), nil
}

func (s *QueueService) calculateETA(userSeq int64, activity *models.Activity) int {
	if activity.Config.ReleaseRate <= 0 {
		return -1 // 未知
	}

	// 簡化計算：假設當前 release_seq = 0
	return int(userSeq / int64(activity.Config.ReleaseRate))
}

func (s *QueueService) hashIP(ip string) string {
	hash := sha256.Sum256([]byte(ip + "salt")) // 實際應用中使用配置的 salt
	return hex.EncodeToString(hash[:])[:16]
}

func (s *QueueService) recordQueueEntry(ctx context.Context, entry *models.QueueEntry) {
	query := `
        INSERT INTO queue_entries (activity_id, user_hash, session_id, seq_number, fingerprint, ip_hash, created_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
        ON CONFLICT (activity_id, session_id) DO NOTHING`

	s.db.ExecContext(ctx, query,
		entry.ActivityID, entry.UserHash, entry.SessionID,
		entry.SeqNumber, entry.Fingerprint, entry.IPHash, entry.CreatedAt)
}

func (s *QueueService) updateMetrics(ctx context.Context, tenantID string, activityID int64, action string) {
	key := keys.MetricsKey(tenantID, activityID, action+"_total")
	s.redis.Incr(ctx, key)
	s.redis.Expire(ctx, key, 24*time.Hour)
}

func max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

// 在 services/queue_service.go 中添加
func (s *QueueService) GetQueueStatusWithETA(ctx context.Context, req *QueueStatusRequest) (*QueueStatusResponse, error) {
	// 原有的狀態查詢邏輯...
	resp, err := s.GetQueueStatus(ctx, req)
	if err != nil {
		return nil, err
	}

	// 獲取活動資訊
	activity, err := s.getActivity(ctx, req.ActivityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get activity: %w", err)
	}

	// 計算 ETA
	etaCalc := NewETACalculator(s.redis)
	eta, err := etaCalc.CalculateETA(ctx, activity, req.Seq)
	if err != nil {
		log.Printf("Failed to calculate ETA for activity %d, seq %d: %v", req.ActivityID, req.Seq, err)
		// 使用基本 ETA 作為回退
		eta = &ETAResult{
			EstimatedWaitSeconds: int(resp.Position * 2), // 保守估計
			EstimatedWaitTime:    time.Now().Add(time.Duration(resp.Position*2) * time.Second),
			Confidence:           0.3,
			NextPollInterval:     activity.Config.PollInterval,
			Method:               "fallback",
		}
	}

	// 添加 ETA 資訊到回應
	resp.ETADetails = eta

	return resp, nil
}
