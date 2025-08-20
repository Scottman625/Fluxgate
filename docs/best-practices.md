# 最佳實踐指南

本文檔提供了隊列系統的最佳實踐建議，幫助您構建高效、穩定和可擴展的隊列應用。

## 🏗️ 架構設計

### 1. 系統架構原則

#### 微服務架構
```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   前端應用      │    │   隊列系統      │    │   業務系統      │
│                 │    │                 │    │                 │
│ - 用戶界面      │◄──►│ - 隊列管理      │◄──►│ - 訂單處理      │
│ - 狀態輪詢      │    │ - 序號分配      │    │ - 庫存管理      │
│ - 錯誤處理      │    │ - 節流控制      │    │ - 支付處理      │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

#### 高可用設計
- **多實例部署**: 使用負載均衡器分發請求
- **資料庫主從**: PostgreSQL 主從複製
- **Redis 集群**: Redis Cluster 或 Sentinel
- **健康檢查**: 定期檢查服務狀態

### 2. 資料庫設計

#### 索引優化
```sql
-- 活動表索引
CREATE INDEX idx_activities_tenant_status ON activities (tenant_id, status);
CREATE INDEX idx_activities_time_range ON activities (start_at, end_at);

-- 隊列表索引
CREATE INDEX idx_queue_entries_activity_seq ON queue_entries (activity_id, seq_number);
CREATE INDEX idx_queue_entries_user_hash ON queue_entries (activity_id, user_hash);
CREATE INDEX idx_queue_entries_created_at ON queue_entries (created_at);
```

#### 分區策略
```sql
-- 按時間分區
CREATE TABLE queue_entries_2024_01 PARTITION OF queue_entries
FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');

CREATE TABLE queue_entries_2024_02 PARTITION OF queue_entries
FOR VALUES FROM ('2024-02-01') TO ('2024-03-01');
```

## 🚀 效能優化

### 1. Redis 優化

#### 記憶體優化
```go
// 使用 Pipeline 批量操作
func (s *QueueService) batchUpdateQueue(activityID int64, updates []QueueUpdate) error {
    pipe := s.redis.Pipeline()
    
    for _, update := range updates {
        pipe.HSet(ctx, keys.QueueSeqKey(tenantID, activityID), update.Field, update.Value)
    }
    
    _, err := pipe.Exec(ctx)
    return err
}
```

#### 鍵設計原則
```go
// 好的鍵設計
user:queue:tenant1:123:session456    // 用戶隊列鍵
queue:seq:tenant1:123               // 隊列序號鍵
throttle:ip:tenant1:123:iphash789   // IP 節流鍵

// 避免的鍵設計
user_queue_tenant1_123_session456   // 過長且不易讀
q:1:2:3:4:5                        // 過於簡短，意義不明
```

### 2. 連接池配置

#### 資料庫連接池
```yaml
database:
  max_open_conns: 25        # 最大連接數
  max_idle_conns: 5         # 最大空閒連接
  conn_max_lifetime: 300    # 連接最大生命週期（秒）
  conn_max_idle_time: 60    # 空閒連接最大生命週期（秒）
```

#### Redis 連接池
```yaml
redis:
  pool_size: 10             # 連接池大小
  min_idle_conns: 5         # 最小空閒連接
  max_retries: 3            # 最大重試次數
  dial_timeout: 5           # 連接超時（秒）
  read_timeout: 3           # 讀取超時（秒）
  write_timeout: 3          # 寫入超時（秒）
```

### 3. 快取策略

#### 多層快取
```go
// 記憶體快取 + Redis 快取
type CacheService struct {
    memoryCache *sync.Map
    redisClient *redis.Client
}

func (c *CacheService) GetActivity(id int64) (*Activity, error) {
    // 1. 檢查記憶體快取
    if cached, ok := c.memoryCache.Load(id); ok {
        return cached.(*Activity), nil
    }
    
    // 2. 檢查 Redis 快取
    if activity, err := c.getFromRedis(id); err == nil {
        c.memoryCache.Store(id, activity)
        return activity, nil
    }
    
    // 3. 從資料庫查詢
    activity, err := c.getFromDatabase(id)
    if err != nil {
        return nil, err
    }
    
    // 4. 更新快取
    c.setToRedis(id, activity)
    c.memoryCache.Store(id, activity)
    
    return activity, nil
}
```

## 🛡️ 安全最佳實踐

### 1. 輸入驗證

#### 參數驗證
```go
type EnterQueueRequest struct {
    ActivityID int64  `json:"activity_id" validate:"required,gt=0"`
    UserHash   string `json:"user_hash" validate:"required,min=1,max=64"`
    Fingerprint string `json:"fingerprint" validate:"omitempty,max=128"`
}

func (h *QueueHandler) EnterQueue(c *gin.Context) {
    var req EnterQueueRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": "invalid request"})
        return
    }
    
    // 使用 validator 進行驗證
    if err := validator.New().Struct(req); err != nil {
        c.JSON(400, gin.H{"error": "validation failed"})
        return
    }
}
```

#### SQL 注入防護
```go
// 使用參數化查詢
func (s *ActivityService) GetActivity(id int64) (*Activity, error) {
    query := `SELECT id, tenant_id, name, sku FROM activities WHERE id = $1`
    var activity Activity
    err := s.db.QueryRow(query, id).Scan(&activity.ID, &activity.TenantID, &activity.Name, &activity.SKU)
    return &activity, err
}
```

### 2. 節流控制

#### IP 節流
```go
func (s *QueueService) checkIPThrottle(tenantID string, activityID int64, ipHash string) error {
    key := keys.IPThrottleKey(tenantID, activityID, ipHash)
    
    // 使用 Redis 的 INCR 和 EXPIRE 實現節流
    count, err := s.redis.Incr(ctx, key).Result()
    if err != nil {
        return err
    }
    
    // 設置過期時間
    if count == 1 {
        s.redis.Expire(ctx, key, time.Minute)
    }
    
    // 檢查是否超過限制
    if count > 10 { // 每分鐘最多 10 次請求
        return errors.New("rate limit exceeded")
    }
    
    return nil
}
```

#### 用戶節流
```go
func (s *QueueService) checkUserThrottle(tenantID string, activityID int64, userHash string) error {
    key := keys.UserDedupeKey(tenantID, activityID)
    
    // 檢查用戶是否已在隊列中
    exists, err := s.redis.SIsMember(ctx, key, userHash).Result()
    if err != nil {
        return err
    }
    
    if exists {
        return errors.New("user already in queue")
    }
    
    return nil
}
```

### 3. 認證授權

#### JWT 認證
```go
func AuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        token := c.GetHeader("Authorization")
        if token == "" {
            c.JSON(401, gin.H{"error": "unauthorized"})
            c.Abort()
            return
        }
        
        // 驗證 JWT token
        claims, err := validateJWT(token)
        if err != nil {
            c.JSON(401, gin.H{"error": "invalid token"})
            c.Abort()
            return
        }
        
        // 將用戶資訊存入上下文
        c.Set("user_id", claims.UserID)
        c.Set("tenant_id", claims.TenantID)
        c.Next()
    }
}
```

## 📊 監控告警

### 1. 指標監控

#### 關鍵指標
```go
type Metrics struct {
    QueueLength     prometheus.Gauge
    EnterRate       prometheus.Counter
    ReleaseRate     prometheus.Counter
    ResponseTime    prometheus.Histogram
    ErrorRate       prometheus.Counter
}

func (s *QueueService) recordMetrics(activityID int64, operation string, duration time.Duration) {
    s.metrics.ResponseTime.WithLabelValues(operation).Observe(duration.Seconds())
    s.metrics.EnterRate.WithLabelValues(fmt.Sprintf("%d", activityID)).Inc()
}
```

#### 健康檢查
```go
func (s *QueueService) HealthCheck() map[string]interface{} {
    return map[string]interface{}{
        "status": "healthy",
        "timestamp": time.Now().Unix(),
        "version": "1.0.0",
        "dependencies": map[string]interface{}{
            "database": s.checkDatabaseHealth(),
            "redis": s.checkRedisHealth(),
        },
    }
}
```

### 2. 日誌記錄

#### 結構化日誌
```go
type Logger struct {
    logger *zap.Logger
}

func (l *Logger) LogQueueEvent(event string, fields map[string]interface{}) {
    l.logger.Info("queue event",
        zap.String("event", event),
        zap.Any("fields", fields),
        zap.Time("timestamp", time.Now()),
    )
}

// 使用範例
logger.LogQueueEvent("user_entered_queue", map[string]interface{}{
    "activity_id": 123,
    "user_hash": "user_123",
    "seq": 456,
    "queue_length": 1000,
})
```

#### 日誌級別
```go
// 開發環境
logger.SetLevel(zap.DebugLevel)

// 生產環境
logger.SetLevel(zap.InfoLevel)

// 錯誤日誌
logger.Error("failed to enter queue",
    zap.Error(err),
    zap.Int64("activity_id", activityID),
    zap.String("user_hash", userHash),
)
```

## 🔄 錯誤處理

### 1. 錯誤分類

#### 業務錯誤
```go
type BusinessError struct {
    Code    string `json:"code"`
    Message string `json:"message"`
    Details map[string]interface{} `json:"details,omitempty"`
}

var (
    ErrActivityNotFound = &BusinessError{
        Code:    "ACTIVITY_NOT_FOUND",
        Message: "活動不存在",
    }
    
    ErrUserAlreadyInQueue = &BusinessError{
        Code:    "USER_ALREADY_IN_QUEUE",
        Message: "用戶已在隊列中",
    }
)
```

#### 系統錯誤
```go
type SystemError struct {
    Code    string `json:"code"`
    Message string `json:"message"`
    RequestID string `json:"request_id"`
}

func (e *SystemError) Error() string {
    return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}
```

### 2. 重試機制

#### 指數退避
```go
func retryWithBackoff(operation func() error, maxRetries int) error {
    var err error
    
    for i := 0; i < maxRetries; i++ {
        if err = operation(); err == nil {
            return nil
        }
        
        if i == maxRetries-1 {
            break
        }
        
        // 指數退避
        backoff := time.Duration(math.Pow(2, float64(i))) * time.Second
        time.Sleep(backoff)
    }
    
    return fmt.Errorf("operation failed after %d retries: %w", maxRetries, err)
}
```

#### 電路斷路器
```go
type CircuitBreaker struct {
    failures  int64
    threshold int64
    timeout   time.Duration
    lastFailure time.Time
    state     int32 // 0: closed, 1: open, 2: half-open
}

func (cb *CircuitBreaker) Execute(operation func() error) error {
    if cb.isOpen() {
        return errors.New("circuit breaker is open")
    }
    
    err := operation()
    if err != nil {
        cb.recordFailure()
    } else {
        cb.recordSuccess()
    }
    
    return err
}
```

## 🚀 部署最佳實踐

### 1. 容器化部署

#### Dockerfile 優化
```dockerfile
# 多階段構建
FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main cmd/api/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/main .
COPY --from=builder /app/internal/config/config.yaml ./config/

EXPOSE 8080
CMD ["./main"]
```

#### Docker Compose 配置
```yaml
version: '3.8'

services:
  queue-api:
    build: .
    ports:
      - "8080:8080"
    environment:
      - GIN_MODE=release
      - QUEUE_DATABASE_HOST=postgres
      - QUEUE_REDIS_HOST=redis
    depends_on:
      - postgres
      - redis
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3

  postgres:
    image: postgres:15
    environment:
      POSTGRES_DB: queue_system
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: password
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./migrations:/docker-entrypoint-initdb.d
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 10s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    command: redis-server --appendonly yes
    volumes:
      - redis_data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5

volumes:
  postgres_data:
  redis_data:
```

### 2. 環境配置

#### 環境變數管理
```bash
# .env.production
QUEUE_SERVER_PORT=8080
QUEUE_DATABASE_HOST=postgres.prod
QUEUE_DATABASE_PORT=5432
QUEUE_DATABASE_USER=queue_user
QUEUE_DATABASE_PASSWORD=secure_password
QUEUE_REDIS_HOST=redis.prod
QUEUE_REDIS_PORT=6379
QUEUE_REDIS_PASSWORD=redis_password
```

#### 配置驗證
```go
func (c *Config) Validate() error {
    if c.Server.Port == "" {
        return errors.New("server port is required")
    }
    
    if c.Database.Host == "" {
        return errors.New("database host is required")
    }
    
    if c.Redis.Host == "" {
        return errors.New("redis host is required")
    }
    
    return nil
}
```

## 📈 效能測試

### 1. 負載測試

#### 使用 Apache Bench
```bash
#!/bin/bash
# load_test.sh

echo "開始負載測試..."

# 測試進入隊列 API
ab -n 1000 -c 100 \
   -p queue_enter.json \
   -T application/json \
   http://localhost:8080/api/v1/queue/enter

# 測試查詢狀態 API
ab -n 2000 -c 50 \
   http://localhost:8080/api/v1/queue/status?activity_id=1&session_id=test

echo "負載測試完成"
```

#### 使用 wrk
```bash
# 測試進入隊列
wrk -t12 -c400 -d30s \
    -s queue_enter.lua \
    http://localhost:8080/api/v1/queue/enter
```

### 2. 壓力測試

#### 併發用戶模擬
```go
func simulateConcurrentUsers(activityID int64, userCount int) {
    var wg sync.WaitGroup
    semaphore := make(chan struct{}, 100) // 限制併發數
    
    for i := 0; i < userCount; i++ {
        wg.Add(1)
        go func(userID int) {
            defer wg.Done()
            semaphore <- struct{}{}
            defer func() { <-semaphore }()
            
            // 模擬用戶進入隊列
            enterQueue(activityID, fmt.Sprintf("user_%d", userID))
        }(i)
    }
    
    wg.Wait()
}
```

## 🔗 相關文檔

- [API 參考](./api-reference.md) - 完整的 API 文檔
- [使用範例](./examples.md) - 實際應用案例
- [故障排除](./troubleshooting.md) - 常見問題解決
- [部署指南](./deployment.md) - 生產環境部署
