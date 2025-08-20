package keys

import "fmt"

// 用戶隊列鍵
func UserQueueKey(tenantID string, activityID int64, sessionID string) string {
	return fmt.Sprintf("user:queue:%s:%d:%s", tenantID, activityID, sessionID)
}

// 隊列序號鍵
func QueueSeqKey(tenantID string, activityID int64) string {
	return fmt.Sprintf("queue:seq:%s:%d", tenantID, activityID)
}

// 釋放序號鍵
func ReleaseSeqKey(tenantID string, activityID int64) string {
	return fmt.Sprintf("release:seq:%s:%d", tenantID, activityID)
}

// 活躍用戶鍵
func ActiveUsersKey(tenantID string, activityID int64) string {
	return fmt.Sprintf("active:users:%s:%d", tenantID, activityID)
}

// IP 節流鍵
func IPThrottleKey(tenantID string, activityID int64, ipHash string) string {
	return fmt.Sprintf("throttle:ip:%s:%d:%s", tenantID, activityID, ipHash)
}

// 用戶去重鍵
func UserDedupeKey(tenantID string, activityID int64) string {
	return fmt.Sprintf("dedupe:user:%s:%d", tenantID, activityID)
}

// 指標鍵
func MetricsKey(tenantID string, activityID int64, metric string) string {
	return fmt.Sprintf("metrics:%s:%d:%s", tenantID, activityID, metric)
}
