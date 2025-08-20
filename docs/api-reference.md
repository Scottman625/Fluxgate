# API 參考文檔

本文檔詳細說明了隊列系統的所有 API 端點、參數和回應格式。

## 📋 概述

### 基礎 URL
```
http://localhost:8080
```

### 認證
目前 API 不需要認證，但建議在生產環境中實現適當的認證機制。

### 回應格式
所有 API 回應都使用 JSON 格式，標準回應結構如下：

**成功回應**
```json
{
  "success": true,
  "data": {
    // 具體資料
  }
}
```

**錯誤回應**
```json
{
  "success": false,
  "error": "ERROR_CODE",
  "message": "錯誤描述",
  "request_id": "uuid-123"
}
```

### HTTP 狀態碼
- `200` - 成功
- `201` - 創建成功
- `400` - 請求參數錯誤
- `404` - 資源不存在
- `409` - 衝突（如用戶已在隊列中）
- `429` - 請求過於頻繁
- `500` - 伺服器內部錯誤

## 🏥 健康檢查

### GET /health

檢查系統健康狀態。

**請求**
```http
GET /health
```

**回應**
```json
{
  "status": "ok"
}
```

## 🎯 隊列 API

### POST /api/v1/queue/enter

用戶進入隊列。

**請求**
```http
POST /api/v1/queue/enter
Content-Type: application/json

{
  "activity_id": 1,
  "user_hash": "user_123",
  "fingerprint": "fp_456"
}
```

**參數說明**
| 參數 | 類型 | 必填 | 說明 |
|------|------|------|------|
| `activity_id` | integer | ✅ | 活動 ID |
| `user_hash` | string | ✅ | 用戶唯一標識 |
| `fingerprint` | string | ❌ | 瀏覽器指紋，用於防重複 |

**成功回應**
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

**錯誤回應**
```json
{
  "success": false,
  "error": "ACTIVITY_NOT_FOUND",
  "message": "activity not found",
  "request_id": "uuid-123"
}
```

**可能的錯誤碼**
- `ACTIVITY_NOT_FOUND` - 活動不存在
- `ACTIVITY_NOT_ACTIVE` - 活動未開始或已結束
- `RATE_LIMIT_EXCEEDED` - 請求頻率過高
- `USER_ALREADY_IN_QUEUE` - 用戶已在隊列中

### GET /api/v1/queue/status

查詢用戶在隊列中的狀態。

**請求**
```http
GET /api/v1/queue/status?activity_id=1&session_id=session_abc123
```

**查詢參數**
| 參數 | 類型 | 必填 | 說明 |
|------|------|------|------|
| `activity_id` | integer | ✅ | 活動 ID |
| `session_id` | string | ✅ | 會話 ID |

**成功回應**
```json
{
  "success": true,
  "data": {
    "activity_id": 1,
    "session_id": "session_abc123",
    "seq": 1,
    "position": 1,
    "queue_length": 5,
    "estimated_wait": 10,
    "status": "waiting"
  }
}
```

**狀態說明**
- `waiting` - 等待中
- `ready` - 可以進行購買
- `expired` - 會話已過期

## 🛠️ 管理 API

### POST /api/v1/admin/activities

創建新活動。

**請求**
```http
POST /api/v1/admin/activities
Content-Type: application/json

{
  "tenant_id": "shop_123",
  "name": "iPhone 15 搶購",
  "sku": "IPHONE15-128GB",
  "initial_stock": 100,
  "start_at": "2024-01-01T10:00:00Z",
  "end_at": "2024-01-01T18:00:00Z",
  "config": {
    "release_rate": 10,
    "poll_interval": 2000,
    "max_queue_size": 10000
  }
}
```

**參數說明**
| 參數 | 類型 | 必填 | 說明 |
|------|------|------|------|
| `tenant_id` | string | ✅ | 租戶 ID |
| `name` | string | ✅ | 活動名稱 |
| `sku` | string | ✅ | 商品 SKU |
| `initial_stock` | integer | ✅ | 初始庫存 |
| `start_at` | string | ✅ | 開始時間 (ISO 8601) |
| `end_at` | string | ✅ | 結束時間 (ISO 8601) |
| `config` | object | ❌ | 活動配置 |

**配置參數**
| 參數 | 類型 | 預設值 | 說明 |
|------|------|--------|------|
| `release_rate` | integer | 10 | 每秒釋放數量 |
| `poll_interval` | integer | 2000 | 輪詢間隔 (毫秒) |
| `max_queue_size` | integer | 10000 | 最大隊列長度 |

**成功回應**
```json
{
  "success": true,
  "data": {
    "id": 1,
    "created_at": "2024-01-01T10:00:00Z"
  }
}
```

### GET /api/v1/admin/activities

獲取活動列表。

**請求**
```http
GET /api/v1/admin/activities?tenant_id=shop_123&status=active
```

**查詢參數**
| 參數 | 類型 | 必填 | 說明 |
|------|------|------|------|
| `tenant_id` | string | ❌ | 租戶 ID 篩選 |
| `status` | string | ❌ | 狀態篩選 (draft, active, paused, ended) |
| `page` | integer | ❌ | 頁碼 (預設 1) |
| `limit` | integer | ❌ | 每頁數量 (預設 20) |

**成功回應**
```json
{
  "success": true,
  "data": {
    "activities": [
      {
        "id": 1,
        "tenant_id": "shop_123",
        "name": "iPhone 15 搶購",
        "sku": "IPHONE15-128GB",
        "initial_stock": 100,
        "start_at": "2024-01-01T10:00:00Z",
        "end_at": "2024-01-01T18:00:00Z",
        "status": "active",
        "created_at": "2024-01-01T09:00:00Z"
      }
    ],
    "pagination": {
      "page": 1,
      "limit": 20,
      "total": 1,
      "pages": 1
    }
  }
}
```

### GET /api/v1/admin/activities/:id/status

獲取活動詳細狀態。

**請求**
```http
GET /api/v1/admin/activities/1/status
```

**成功回應**
```json
{
  "success": true,
  "data": {
    "activity": {
      "id": 1,
      "tenant_id": "shop_123",
      "name": "iPhone 15 搶購",
      "sku": "IPHONE15-128GB",
      "initial_stock": 100,
      "start_at": "2024-01-01T10:00:00Z",
      "end_at": "2024-01-01T18:00:00Z",
      "status": "active",
      "config": {
        "release_rate": 10,
        "poll_interval": 2000
      }
    },
    "queue_metrics": {
      "queue_seq": 50,
      "release_seq": 30,
      "queue_length": 20,
      "active_users": 25
    },
    "realtime_stats": {
      "enter_total": 100,
      "enter_rate": 2.5,
      "release_rate": 10.0,
      "last_updated": "2024-01-01T10:30:00Z"
    }
  }
}
```

### PUT /api/v1/admin/activities/:id

更新活動資訊。

**請求**
```http
PUT /api/v1/admin/activities/1
Content-Type: application/json

{
  "name": "iPhone 15 Pro 搶購",
  "status": "paused",
  "config": {
    "release_rate": 5
  }
}
```

**參數說明**
| 參數 | 類型 | 必填 | 說明 |
|------|------|------|------|
| `name` | string | ❌ | 活動名稱 |
| `status` | string | ❌ | 活動狀態 |
| `config` | object | ❌ | 活動配置 |

**成功回應**
```json
{
  "success": true,
  "data": {
    "id": 1,
    "updated_at": "2024-01-01T11:00:00Z"
  }
}
```

## 📊 統計 API

### GET /api/v1/admin/activities/:id/analytics

獲取活動統計資料。

**請求**
```http
GET /api/v1/admin/activities/1/analytics?period=1h
```

**查詢參數**
| 參數 | 類型 | 必填 | 說明 |
|------|------|------|------|
| `period` | string | ❌ | 統計週期 (1h, 24h, 7d, 30d) |

**成功回應**
```json
{
  "success": true,
  "data": {
    "period": "1h",
    "metrics": {
      "total_entries": 1000,
      "unique_users": 800,
      "avg_wait_time": 120,
      "max_wait_time": 300,
      "conversion_rate": 0.75
    },
    "timeline": [
      {
        "timestamp": "2024-01-01T10:00:00Z",
        "entries": 50,
        "releases": 10
      }
    ]
  }
}
```

## 🔧 系統 API

### GET /api/v1/system/config

獲取系統配置。

**請求**
```http
GET /api/v1/system/config
```

**成功回應**
```json
{
  "success": true,
  "data": {
    "default_queue_ttl": 3600,
    "max_release_rate": 1000,
    "default_poll_interval": 2000
  }
}
```

### PUT /api/v1/system/config

更新系統配置。

**請求**
```http
PUT /api/v1/system/config
Content-Type: application/json

{
  "default_queue_ttl": 7200,
  "max_release_rate": 500
}
```

**成功回應**
```json
{
  "success": true,
  "data": {
    "updated_at": "2024-01-01T12:00:00Z"
  }
}
```

## 🚨 錯誤處理

### 錯誤碼說明

| 錯誤碼 | HTTP 狀態碼 | 說明 |
|--------|-------------|------|
| `INVALID_REQUEST` | 400 | 請求參數錯誤 |
| `ACTIVITY_NOT_FOUND` | 404 | 活動不存在 |
| `ACTIVITY_NOT_ACTIVE` | 409 | 活動未開始或已結束 |
| `USER_ALREADY_IN_QUEUE` | 409 | 用戶已在隊列中 |
| `RATE_LIMIT_EXCEEDED` | 429 | 請求頻率過高 |
| `INVALID_SEQUENCE` | 400 | 無效的序號 |
| `INTERNAL_ERROR` | 500 | 伺服器內部錯誤 |

### 錯誤回應範例

```json
{
  "success": false,
  "error": "RATE_LIMIT_EXCEEDED",
  "message": "請求頻率過高，請稍後再試",
  "request_id": "uuid-123",
  "retry_after": 60
}
```

## 📝 請求範例

### cURL 範例

**進入隊列**
```bash
curl -X POST http://localhost:8080/api/v1/queue/enter \
  -H "Content-Type: application/json" \
  -d '{
    "activity_id": 1,
    "user_hash": "user_123",
    "fingerprint": "fp_456"
  }'
```

**查詢隊列狀態**
```bash
curl "http://localhost:8080/api/v1/queue/status?activity_id=1&session_id=session_abc123"
```

**創建活動**
```bash
curl -X POST http://localhost:8080/api/v1/admin/activities \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_id": "shop_123",
    "name": "測試活動",
    "sku": "TEST-001",
    "initial_stock": 100,
    "start_at": "2024-01-01T10:00:00Z",
    "end_at": "2024-01-01T18:00:00Z"
  }'
```

### JavaScript 範例

**進入隊列**
```javascript
async function enterQueue(activityId, userHash) {
  const response = await fetch('/api/v1/queue/enter', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      activity_id: activityId,
      user_hash: userHash,
      fingerprint: 'fp_123'
    })
  });

  return await response.json();
}
```

**輪詢隊列狀態**
```javascript
async function pollQueueStatus(activityId, sessionId) {
  const response = await fetch(
    `/api/v1/queue/status?activity_id=${activityId}&session_id=${sessionId}`
  );
  
  return await response.json();
}
```

## 🔗 相關文檔

- [使用範例](./examples.md) - 實際應用案例
- [最佳實踐](./best-practices.md) - API 使用建議
- [故障排除](./troubleshooting.md) - 常見問題解決
