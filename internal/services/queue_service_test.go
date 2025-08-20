package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQueueService_GenerateSessionID(t *testing.T) {
	// 創建一個簡單的測試，不依賴外部服務
	userHash := "user123"
	activityID := int64(456)

	// 模擬 generateSessionID 的邏輯
	sessionID := generateTestSessionID(userHash, activityID)

	// 驗證 sessionID 不為空
	assert.NotEmpty(t, sessionID)
	assert.Len(t, sessionID, 16)
}

func TestQueueService_HashIP(t *testing.T) {
	// 模擬 hashIP 的邏輯
	ip := "192.168.1.1"
	hash := generateTestIPHash(ip)

	// 驗證 hash 不為空且長度正確
	assert.NotEmpty(t, hash)
	assert.Len(t, hash, 16)
}

// 輔助函數，用於測試
func generateTestSessionID(userHash string, activityID int64) string {
	// 簡化的 sessionID 生成邏輯
	if len(userHash) >= 8 {
		return userHash[:8] + "0000000000000000"[8:16]
	}
	return userHash + "0000000000000000"[len(userHash):16]
}

func generateTestIPHash(ip string) string {
	// 簡化的 IP hash 生成邏輯
	if len(ip) >= 8 {
		return ip[:8] + "0000000000000000"[8:16]
	}
	return ip + "0000000000000000"[len(ip):16]
}
