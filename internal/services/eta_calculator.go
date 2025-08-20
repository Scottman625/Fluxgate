package services

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	    "queue-system/internal/models"
    "queue-system/pkg/keys"
)

type ETACalculator struct {
	redis *redis.Client
}

type ETAResult struct {
	EstimatedWaitSeconds int       `json:"estimated_wait_seconds"`
	EstimatedWaitTime    time.Time `json:"estimated_wait_time"`
	Confidence           float64   `json:"confidence"` // 0.0 - 1.0
	NextPollInterval     int       `json:"next_poll_interval_ms"`
	Method               string    `json:"method"` // "historical", "current_rate", "static"
}

func NewETACalculator(redis *redis.Client) *ETACalculator {
	return &ETACalculator{
		redis: redis,
	}
}

func (calc *ETACalculator) CalculateETA(ctx context.Context, activity *models.Activity, userSeq int64) (*ETAResult, error) {
	// 獲取當前狀態
	releaseSeq, err := calc.getCurrentReleaseSeq(ctx, activity.TenantID, activity.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get release seq: %w", err)
	}

	position := userSeq - releaseSeq
	if position <= 0 {
		// 用戶已經可以進入
		return &ETAResult{
			EstimatedWaitSeconds: 0,
			EstimatedWaitTime:    time.Now(),
			Confidence:           1.0,
			NextPollInterval:     0, // 立即輪詢
			Method:               "immediate",
		}, nil
	}

	// 嘗試不同的 ETA 計算方法
	methods := []func(context.Context, *models.Activity, int64) (*ETAResult, error){
		calc.calculateHistoricalETA,
		calc.calculateCurrentRateETA,
		calc.calculateStaticETA,
	}

	for _, method := range methods {
		if result, err := method(ctx, activity, position); err == nil {
			result.NextPollInterval = calc.calculatePollInterval(result.EstimatedWaitSeconds, activity.Config.PollInterval)
			return result, nil
		}
	}

	// 回退到基本計算
	return calc.calculateBasicETA(activity, position), nil
}

// 基於歷史數據的 ETA 計算
func (calc *ETACalculator) calculateHistoricalETA(ctx context.Context, activity *models.Activity, position int64) (*ETAResult, error) {
	// 獲取過去 1 小時的釋放數據
	releaseData, err := calc.getHistoricalReleaseData(ctx, activity.TenantID, activity.ID, time.Hour)
	if err != nil || len(releaseData) < 3 {
		return nil, fmt.Errorf("insufficient historical data")
	}

	// 計算平均釋放速率
	totalReleases := int64(0)
	totalTime := time.Duration(0)

	for i := 1; i < len(releaseData); i++ {
		releases := releaseData[i-1].ReleaseCount
		duration := releaseData[i-1].Timestamp.Sub(releaseData[i].Timestamp)

		totalReleases += releases
		totalTime += duration
	}

	if totalTime == 0 {
		return nil, fmt.Errorf("no time data available")
	}

	avgRatePerSecond := float64(totalReleases) / totalTime.Seconds()
	if avgRatePerSecond <= 0 {
		return nil, fmt.Errorf("invalid release rate")
	}

	estimatedSeconds := int(float64(position) / avgRatePerSecond)
	confidence := calc.calculateConfidence(releaseData, avgRatePerSecond)

	return &ETAResult{
		EstimatedWaitSeconds: estimatedSeconds,
		EstimatedWaitTime:    time.Now().Add(time.Duration(estimatedSeconds) * time.Second),
		Confidence:           confidence,
		Method:               "historical",
	}, nil
}

// 基於當前釋放速率的 ETA 計算
func (calc *ETACalculator) calculateCurrentRateETA(ctx context.Context, activity *models.Activity, position int64) (*ETAResult, error) {
	// 獲取最近 5 分鐘的釋放數據
	recentData, err := calc.getHistoricalReleaseData(ctx, activity.TenantID, activity.ID, 5*time.Minute)
	if err != nil || len(recentData) < 2 {
		return nil, fmt.Errorf("insufficient recent data")
	}

	// 計算最近的釋放速率
	latestEvent := recentData[0]
	prevEvent := recentData[1]

	duration := latestEvent.Timestamp.Sub(prevEvent.Timestamp)
	if duration <= 0 {
		return nil, fmt.Errorf("invalid time duration")
	}

	currentRate := float64(latestEvent.ReleaseCount) / duration.Seconds()
	if currentRate <= 0 {
		return nil, fmt.Errorf("invalid current rate")
	}

	estimatedSeconds := int(float64(position) / currentRate)

	// 當前速率的信心度較低，因為樣本較小
	confidence := 0.6

	return &ETAResult{
		EstimatedWaitSeconds: estimatedSeconds,
		EstimatedWaitTime:    time.Now().Add(time.Duration(estimatedSeconds) * time.Second),
		Confidence:           confidence,
		Method:               "current_rate",
	}, nil
}

// 基於配置釋放速率的 ETA 計算
func (calc *ETACalculator) calculateStaticETA(ctx context.Context, activity *models.Activity, position int64) (*ETAResult, error) {
	if activity.Config.ReleaseRate <= 0 {
		return nil, fmt.Errorf("invalid configured release rate")
	}

	estimatedSeconds := int(float64(position) / float64(activity.Config.ReleaseRate))

	// 靜態配置的信心度中等
	confidence := 0.5

	return &ETAResult{
		EstimatedWaitSeconds: estimatedSeconds,
		EstimatedWaitTime:    time.Now().Add(time.Duration(estimatedSeconds) * time.Second),
		Confidence:           confidence,
		Method:               "static",
	}, nil
}

// 基本 ETA 計算（回退方案）
func (calc *ETACalculator) calculateBasicETA(activity *models.Activity, position int64) *ETAResult {
	// 使用保守估計：每秒釋放 1 個
	estimatedSeconds := int(position)

	return &ETAResult{
		EstimatedWaitSeconds: estimatedSeconds,
		EstimatedWaitTime:    time.Now().Add(time.Duration(estimatedSeconds) * time.Second),
		Confidence:           0.3,
		Method:               "basic",
	}
}

// 計算動態輪詢間隔
func (calc *ETACalculator) calculatePollInterval(etaSeconds int, defaultInterval int) int {
	if etaSeconds <= 0 {
		return 0 // 立即輪詢
	}

	// 根據 ETA 動態調整輪詢間隔
	switch {
	case etaSeconds <= 30:
		return min(1000, defaultInterval) // 1 秒
	case etaSeconds <= 120:
		return min(2000, defaultInterval) // 2 秒
	case etaSeconds <= 300:
		return min(5000, defaultInterval) // 5 秒
	case etaSeconds <= 600:
		return min(10000, defaultInterval) // 10 秒
	default:
		return min(30000, defaultInterval) // 30 秒
	}
}

// 獲取歷史釋放數據
func (calc *ETACalculator) getHistoricalReleaseData(ctx context.Context, tenantID string, activityID int64, duration time.Duration) ([]*ReleaseEvent, error) {
	eventKey := fmt.Sprintf("t:%s:a:%d:events:release", tenantID, activityID)

	// 從 Redis 獲取最近的釋放事件
	events, err := calc.redis.LRange(ctx, eventKey, 0, 49).Result() // 最多 50 個事件
	if err != nil {
		return nil, err
	}

	var releaseEvents []*ReleaseEvent
	cutoffTime := time.Now().Add(-duration)

	for _, eventStr := range events {
		var event ReleaseEvent
		if err := json.Unmarshal([]byte(eventStr), &event); err != nil {
			continue
		}

		if event.Timestamp.After(cutoffTime) {
			releaseEvents = append(releaseEvents, &event)
		}
	}

	return releaseEvents, nil
}

// 計算信心度 (續)
func (calc *ETACalculator) calculateConfidence(data []*ReleaseEvent, avgRate float64) float64 {
	if len(data) < 2 {
		return 0.3
	}

	// 計算釋放速率的變異係數
	var rates []float64
	for i := 1; i < len(data); i++ {
		duration := data[i-1].Timestamp.Sub(data[i].Timestamp)
		if duration > 0 {
			rate := float64(data[i-1].ReleaseCount) / duration.Seconds()
			rates = append(rates, rate)
		}
	}

	if len(rates) < 2 {
		return 0.4
	}

	// 計算標準差
	var variance float64
	for _, rate := range rates {
		variance += math.Pow(rate-avgRate, 2)
	}
	variance /= float64(len(rates))
	stdDev := math.Sqrt(variance)

	// 變異係數 = 標準差 / 平均值
	cv := stdDev / avgRate

	// 信心度與變異係數成反比
	confidence := 1.0 / (1.0 + cv)

	// 限制在 0.3 - 0.9 之間
	if confidence < 0.3 {
		confidence = 0.3
	} else if confidence > 0.9 {
		confidence = 0.9
	}

	return confidence
}

func (calc *ETACalculator) getCurrentReleaseSeq(ctx context.Context, tenantID string, activityID int64) (int64, error) {
	key := keys.ReleaseSeqKey(tenantID, activityID)
	result := calc.redis.Get(ctx, key)
	if result.Err() == redis.Nil {
		return 0, nil
	}
	if result.Err() != nil {
		return 0, result.Err()
	}
	return strconv.ParseInt(result.Val(), 10, 64)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
