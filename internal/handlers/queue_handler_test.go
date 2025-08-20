package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"queue-system/internal/models"
	"queue-system/internal/services"
)

// Mock QueueService
type MockQueueService struct {
	mock.Mock
}

func (m *MockQueueService) EnterQueue(ctx context.Context, req *models.EnterQueueRequest) (*models.EnterQueueResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.EnterQueueResponse), args.Error(1)
}

func (m *MockQueueService) GetQueueStatus(ctx context.Context, activityID int64, sessionID string) (*models.QueueStatusResponse, error) {
	args := m.Called(ctx, activityID, sessionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.QueueStatusResponse), args.Error(1)
}

func TestEnterQueue(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		mockResponse   *models.EnterQueueResponse
		mockError      error
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name: "成功進入隊列",
			requestBody: map[string]interface{}{
				"activity_id": 1,
				"user_hash":   "test-user",
				"fingerprint": "test-fingerprint",
			},
			mockResponse: &models.EnterQueueResponse{
				RequestID:        "test-request-id",
				Seq:              1,
				EstimatedWait:    0,
				PollingInterval:  2000,
				SessionID:        "test-session-id",
				QueueLength:      1,
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"success": true,
				"data": map[string]interface{}{
					"request_id":        "test-request-id",
					"seq":               float64(1),
					"estimated_wait":    float64(0),
					"polling_interval":  float64(2000),
					"session_id":        "test-session-id",
					"queue_length":      float64(1),
				},
			},
		},
		{
			name: "缺少必要參數",
			requestBody: map[string]interface{}{
				"activity_id": 1,
				// 缺少 user_hash
			},
			mockResponse:   nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"success": false,
				"error":   "INVALID_REQUEST",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 設置 mock
			mockService := new(MockQueueService)
			if tt.mockResponse != nil {
				mockService.On("EnterQueue", mock.Anything, mock.Anything).Return(tt.mockResponse, tt.mockError)
			}

			// 創建 handler
			handler := NewQueueHandler(mockService)

			// 創建測試請求
			body, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("POST", "/api/v1/queue/enter", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			// 創建 response recorder
			w := httptest.NewRecorder()

			// 設置 Gin context
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			// 執行 handler
			handler.EnterQueue(c)

			// 驗證結果
			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			// 驗證回應結構
			if tt.expectedBody["success"].(bool) {
				assert.True(t, response["success"].(bool))
				assert.NotNil(t, response["data"])
			} else {
				assert.False(t, response["success"].(bool))
				assert.NotNil(t, response["error"])
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestGetQueueStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		activityID     string
		sessionID      string
		mockResponse   *models.QueueStatusResponse
		mockError      error
		expectedStatus int
	}{
		{
			name:       "成功獲取隊列狀態",
			activityID: "1",
			sessionID:  "test-session-id",
			mockResponse: &models.QueueStatusResponse{
				ActivityID:     1,
				SessionID:      "test-session-id",
				Seq:            1,
				Position:       1,
				QueueLength:    5,
				EstimatedWait:  10,
				Status:         "waiting",
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "無效的 activity_id",
			activityID:     "invalid",
			sessionID:      "test-session-id",
			mockResponse:   nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 設置 mock
			mockService := new(MockQueueService)
			if tt.mockResponse != nil {
				activityID := int64(1)
				mockService.On("GetQueueStatus", mock.Anything, activityID, tt.sessionID).Return(tt.mockResponse, tt.mockError)
			}

			// 創建 handler
			handler := NewQueueHandler(mockService)

			// 創建測試請求
			url := "/api/v1/queue/status?activity_id=" + tt.activityID + "&session_id=" + tt.sessionID
			req, _ := http.NewRequest("GET", url, nil)

			// 創建 response recorder
			w := httptest.NewRecorder()

			// 設置 Gin context
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			// 執行 handler
			handler.GetQueueStatus(c)

			// 驗證結果
			assert.Equal(t, tt.expectedStatus, w.Code)

			mockService.AssertExpectations(t)
		})
	}
}
