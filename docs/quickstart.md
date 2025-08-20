# 快速開始指南

本指南將幫助您在 5 分鐘內快速啟動隊列系統並體驗其功能。

## 🎯 目標

完成本指南後，您將能夠：
- ✅ 啟動隊列系統
- ✅ 創建一個測試活動
- ✅ 模擬用戶進入隊列
- ✅ 查看隊列狀態

## 📋 前置需求

確保您的系統已安裝：
- [Docker](https://docs.docker.com/get-docker/) 和 [Docker Compose](https://docs.docker.com/compose/install/)
- [Go 1.25.0+](https://golang.org/dl/)

## 🚀 步驟 1: 啟動依賴服務

```bash
# 啟動 PostgreSQL 和 Redis
docker-compose up -d postgres redis

# 檢查服務狀態
docker-compose ps
```

您應該看到類似以下的輸出：
```
NAME                   IMAGE            COMMAND                   SERVICE   CREATED          STATUS
queue-system-postgres-1   postgres:15      "docker-entrypoint.s…"   postgres   2 minutes ago    Up 2 minutes (healthy)
queue-system-redis-1      redis:7-alpine   "docker-entrypoint.s…"   redis      2 minutes ago    Up 2 minutes (healthy)
```

## 🚀 步驟 2: 啟動隊列系統

```bash
# 啟動 API 服務
go run cmd/api/main.go
```

您應該看到類似以下的輸出：
```
[GIN-debug] [WARNING] Creating an Engine instance with the Logger and Recovery middleware already attached.
[GIN-debug] [WARNING] Running in "debug" mode. Switch to "release" mode in production.
[GIN-debug] GET    /health                   --> queue-system/internal/routes.SetupRoutes.func1
[GIN-debug] POST   /api/v1/queue/enter       --> queue-system/internal/handlers.(*QueueHandler).EnterQueue-fm
[GIN-debug] GET    /api/v1/queue/status      --> queue-system/internal/handlers.(*QueueHandler).GetQueueStatus-fm
[GIN-debug] POST   /api/v1/admin/activities  --> queue-system/internal/handlers.(*AdminHandler).CreateActivity-fm
2025/08/19 14:46:13 Server starting on port 8080
```

## 🚀 步驟 3: 測試健康檢查

在新的終端視窗中執行：

```bash
# 測試健康檢查
curl http://localhost:8080/health
```

預期回應：
```json
{"status":"ok"}
```

## 🚀 步驟 4: 創建測試活動

```bash
# 創建一個搶購活動
curl -X POST http://localhost:8080/api/v1/admin/activities \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_id": "demo_shop",
    "name": "iPhone 15 限時搶購",
    "sku": "IPHONE15-128GB",
    "initial_stock": 100,
    "start_at": "2024-01-01T10:00:00Z",
    "end_at": "2024-01-01T18:00:00Z"
  }'
```

預期回應：
```json
{
  "success": true,
  "data": {
    "id": 1,
    "created_at": "2024-01-01T10:00:00Z"
  }
}
```

## 🚀 步驟 5: 模擬用戶進入隊列

```bash
# 用戶 A 進入隊列
curl -X POST http://localhost:8080/api/v1/queue/enter \
  -H "Content-Type: application/json" \
  -d '{
    "activity_id": 1,
    "user_hash": "user_a_123",
    "fingerprint": "fp_a_456"
  }'
```

預期回應：
```json
{
  "success": true,
  "data": {
    "request_id": "uuid-123",
    "seq": 1,
    "estimated_wait": 0,
    "polling_interval": 2000,
    "session_id": "session_abc123",
    "queue_length": 1
  }
}
```

```bash
# 用戶 B 進入隊列
curl -X POST http://localhost:8080/api/v1/queue/enter \
  -H "Content-Type: application/json" \
  -d '{
    "activity_id": 1,
    "user_hash": "user_b_456",
    "fingerprint": "fp_b_789"
  }'
```

預期回應：
```json
{
  "success": true,
  "data": {
    "request_id": "uuid-456",
    "seq": 2,
    "estimated_wait": 2,
    "polling_interval": 2000,
    "session_id": "session_def456",
    "queue_length": 2
  }
}
```

## 🚀 步驟 6: 查詢隊列狀態

```bash
# 查詢用戶 A 的隊列狀態
curl "http://localhost:8080/api/v1/queue/status?activity_id=1&session_id=session_abc123"
```

預期回應：
```json
{
  "success": true,
  "data": {
    "activity_id": 1,
    "session_id": "session_abc123",
    "seq": 1,
    "position": 1,
    "queue_length": 2,
    "estimated_wait": 0,
    "status": "waiting"
  }
}
```

## 🚀 步驟 7: 查看活動狀態

```bash
# 查看活動整體狀態
curl http://localhost:8080/api/v1/admin/activities/1/status
```

預期回應：
```json
{
  "success": true,
  "data": {
    "activity": {
      "id": 1,
      "name": "iPhone 15 限時搶購",
      "status": "active"
    },
    "queue_metrics": {
      "queue_seq": 2,
      "release_seq": 0,
      "queue_length": 2,
      "active_users": 2
    },
    "realtime_stats": {
      "enter_total": 2,
      "enter_rate": 0.1,
      "release_rate": 0,
      "last_updated": "2024-01-01T10:05:00Z"
    }
  }
}
```

## 🎉 恭喜！

您已經成功完成了隊列系統的快速體驗！現在您已經：

- ✅ 啟動了完整的隊列系統
- ✅ 創建了一個搶購活動
- ✅ 模擬了多個用戶進入隊列
- ✅ 查看了隊列和活動狀態

## 🔄 下一步

- 📖 閱讀 [API 參考](./api-reference.md) 了解完整的 API 功能
- 🛠️ 查看 [使用範例](./examples.md) 學習更多實際應用場景
- ⚙️ 參考 [配置說明](./configuration.md) 自定義系統配置
- 🚀 學習 [最佳實踐](./best-practices.md) 優化您的應用

## 🐛 遇到問題？

如果遇到任何問題，請參考：
- [故障排除](./troubleshooting.md) - 常見問題解決方案
- [安裝指南](./installation.md) - 詳細的安裝說明

## 📞 需要幫助？

如果您需要更多支援，請：
1. 檢查 [故障排除](./troubleshooting.md) 文檔
2. 查看系統日誌：`docker-compose logs`
3. 確認服務狀態：`docker-compose ps`
