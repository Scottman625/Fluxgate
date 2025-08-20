# 使用範例

本文檔提供了隊列系統的實際使用範例，涵蓋了常見的應用場景。

## 🎯 應用場景

### 1. 電商搶購活動

#### 場景描述
電商平台舉辦 iPhone 15 限時搶購活動，限量 100 台，活動時間 2 小時。

#### 實施步驟

**步驟 1: 創建搶購活動**
```bash
curl -X POST http://localhost:8080/api/v1/admin/activities \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_id": "apple_store",
    "name": "iPhone 15 Pro 限時搶購",
    "sku": "IPHONE15-PRO-256GB",
    "initial_stock": 100,
    "start_at": "2024-01-15T10:00:00Z",
    "end_at": "2024-01-15T12:00:00Z",
    "config": {
      "release_rate": 10,
      "poll_interval": 2000,
      "max_queue_size": 10000
    }
  }'
```

**步驟 2: 用戶進入隊列**
```bash
# 模擬多個用戶同時進入隊列
for i in {1..5}; do
  curl -X POST http://localhost:8080/api/v1/queue/enter \
    -H "Content-Type: application/json" \
    -d "{
      \"activity_id\": 1,
      \"user_hash\": \"user_${i}_hash\",
      \"fingerprint\": \"fp_${i}_123\"
    }" &
done
wait
```

**步驟 3: 監控隊列狀態**
```bash
# 查看活動狀態
curl http://localhost:8080/api/v1/admin/activities/1/status

# 查看特定用戶狀態
curl "http://localhost:8080/api/v1/queue/status?activity_id=1&session_id=session_abc123"
```

### 2. 演唱會票務預約

#### 場景描述
演唱會門票預約系統，支援多個票種和座位選擇。

#### 實施步驟

**步驟 1: 創建票務活動**
```bash
curl -X POST http://localhost:8080/api/v1/admin/activities \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_id": "concert_hall",
    "name": "周杰倫演唱會 2024",
    "sku": "JJ2024-TAIWAN",
    "initial_stock": 5000,
    "start_at": "2024-02-01T09:00:00Z",
    "end_at": "2024-02-01T18:00:00Z",
    "config": {
      "release_rate": 50,
      "poll_interval": 3000,
      "max_queue_size": 20000
    }
  }'
```

**步驟 2: 用戶預約流程**
```bash
# 用戶進入預約隊列
curl -X POST http://localhost:8080/api/v1/queue/enter \
  -H "Content-Type: application/json" \
  -d '{
    "activity_id": 1,
    "user_hash": "user_john_doe",
    "fingerprint": "fp_john_123"
  }'
```

**步驟 3: 輪詢隊列狀態**
```javascript
// 前端 JavaScript 輪詢範例
function pollQueueStatus(activityId, sessionId) {
  const pollInterval = 2000; // 2秒輪詢一次
  
  const poll = () => {
    fetch(`/api/v1/queue/status?activity_id=${activityId}&session_id=${sessionId}`)
      .then(response => response.json())
      .then(data => {
        if (data.success) {
          updateQueueUI(data.data);
          
          if (data.data.status === 'ready') {
            // 用戶可以購買了
            showPurchaseButton();
          } else {
            // 繼續輪詢
            setTimeout(poll, pollInterval);
          }
        }
      })
      .catch(error => {
        console.error('輪詢失敗:', error);
        setTimeout(poll, pollInterval);
      });
  };
  
  poll();
}
```

### 3. 限時優惠券發放

#### 場景描述
電商平台發放限時優惠券，每人限領一張。

#### 實施步驟

**步驟 1: 創建優惠券活動**
```bash
curl -X POST http://localhost:8080/api/v1/admin/activities \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_id": "ecommerce_platform",
    "name": "雙11優惠券發放",
    "sku": "COUPON-50OFF",
    "initial_stock": 10000,
    "start_at": "2024-11-11T00:00:00Z",
    "end_at": "2024-11-11T23:59:59Z",
    "config": {
      "release_rate": 100,
      "poll_interval": 1000,
      "max_queue_size": 50000
    }
  }'
```

**步驟 2: 用戶領取優惠券**
```bash
# 用戶進入隊列
curl -X POST http://localhost:8080/api/v1/queue/enter \
  -H "Content-Type: application/json" \
  -d '{
    "activity_id": 1,
    "user_hash": "user_12345",
    "fingerprint": "fp_user_12345"
  }'
```

### 4. 遊戲內測資格發放

#### 場景描述
遊戲公司發放內測資格，限制人數和時間。

#### 實施步驟

**步驟 1: 創建內測活動**
```bash
curl -X POST http://localhost:8080/api/v1/admin/activities \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_id": "game_studio",
    "name": "新遊戲內測資格",
    "sku": "BETA-ACCESS-2024",
    "initial_stock": 1000,
    "start_at": "2024-03-01T10:00:00Z",
    "end_at": "2024-03-01T22:00:00Z",
    "config": {
      "release_rate": 20,
      "poll_interval": 5000,
      "max_queue_size": 5000
    }
  }'
```

## 🔧 進階範例

### 1. 批量操作腳本

**批量創建活動**
```bash
#!/bin/bash
# create_activities.sh

activities=(
  '{"tenant_id":"shop1","name":"商品A搶購","sku":"ITEM-A","initial_stock":50}'
  '{"tenant_id":"shop2","name":"商品B搶購","sku":"ITEM-B","initial_stock":30}'
  '{"tenant_id":"shop3","name":"商品C搶購","sku":"ITEM-C","initial_stock":20}'
)

for activity in "${activities[@]}"; do
  curl -X POST http://localhost:8080/api/v1/admin/activities \
    -H "Content-Type: application/json" \
    -d "$activity"
  echo "創建活動: $activity"
done
```

**批量用戶進入隊列**
```bash
#!/bin/bash
# simulate_users.sh

activity_id=1
user_count=100

for i in $(seq 1 $user_count); do
  curl -X POST http://localhost:8080/api/v1/queue/enter \
    -H "Content-Type: application/json" \
    -d "{
      \"activity_id\": $activity_id,
      \"user_hash\": \"user_${i}_hash\",
      \"fingerprint\": \"fp_${i}_123\"
    }" &
  
  # 控制併發數量
  if [ $((i % 10)) -eq 0 ]; then
    wait
  fi
done

wait
echo "完成 $user_count 個用戶進入隊列"
```

### 2. 監控腳本

**實時監控隊列狀態**
```bash
#!/bin/bash
# monitor_queue.sh

activity_id=$1
interval=5

while true; do
  echo "=== $(date) ==="
  
  # 獲取活動狀態
  status=$(curl -s http://localhost:8080/api/v1/admin/activities/$activity_id/status)
  
  # 解析 JSON (需要 jq)
  queue_length=$(echo $status | jq -r '.data.queue_metrics.queue_length')
  active_users=$(echo $status | jq -r '.data.queue_metrics.active_users')
  
  echo "隊列長度: $queue_length"
  echo "活躍用戶: $active_users"
  echo "------------------------"
  
  sleep $interval
done
```

### 3. 前端整合範例

**React 組件範例**
```jsx
import React, { useState, useEffect } from 'react';

const QueueComponent = ({ activityId }) => {
  const [queueStatus, setQueueStatus] = useState(null);
  const [sessionId, setSessionId] = useState(null);
  const [isInQueue, setIsInQueue] = useState(false);

  // 進入隊列
  const enterQueue = async () => {
    try {
      const response = await fetch('/api/v1/queue/enter', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          activity_id: activityId,
          user_hash: 'user_hash_123',
          fingerprint: 'fp_123'
        })
      });

      const data = await response.json();
      if (data.success) {
        setSessionId(data.data.session_id);
        setIsInQueue(true);
        startPolling(data.data.session_id);
      }
    } catch (error) {
      console.error('進入隊列失敗:', error);
    }
  };

  // 輪詢隊列狀態
  const startPolling = (sessionId) => {
    const poll = async () => {
      try {
        const response = await fetch(
          `/api/v1/queue/status?activity_id=${activityId}&session_id=${sessionId}`
        );
        const data = await response.json();
        
        if (data.success) {
          setQueueStatus(data.data);
          
          if (data.data.status === 'ready') {
            setIsInQueue(false);
            alert('輪到您了！可以進行購買。');
          } else {
            setTimeout(poll, data.data.polling_interval || 2000);
          }
        }
      } catch (error) {
        console.error('輪詢失敗:', error);
        setTimeout(poll, 5000);
      }
    };
    
    poll();
  };

  return (
    <div className="queue-component">
      {!isInQueue ? (
        <button onClick={enterQueue}>進入隊列</button>
      ) : (
        <div className="queue-status">
          <h3>排隊中...</h3>
          {queueStatus && (
            <div>
              <p>當前位置: {queueStatus.position}</p>
              <p>隊列總長度: {queueStatus.queue_length}</p>
              <p>預計等待時間: {queueStatus.estimated_wait} 秒</p>
            </div>
          )}
        </div>
      )}
    </div>
  );
};

export default QueueComponent;
```

## 📊 效能測試範例

### 壓力測試腳本

```bash
#!/bin/bash
# stress_test.sh

activity_id=1
concurrent_users=1000
duration=60

echo "開始壓力測試..."
echo "併發用戶數: $concurrent_users"
echo "測試時長: $duration 秒"

# 使用 Apache Bench 進行壓力測試
ab -n $concurrent_users -c 100 \
   -p queue_enter.json \
   -T application/json \
   http://localhost:8080/api/v1/queue/enter

echo "壓力測試完成"
```

**queue_enter.json**
```json
{
  "activity_id": 1,
  "user_hash": "test_user",
  "fingerprint": "test_fp"
}
```

## 🔄 最佳實踐

### 1. 錯誤處理
```javascript
// 前端錯誤處理範例
async function enterQueue(activityId, userHash) {
  try {
    const response = await fetch('/api/v1/queue/enter', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ activity_id: activityId, user_hash: userHash })
    });

    const data = await response.json();
    
    if (!response.ok) {
      throw new Error(data.message || '進入隊列失敗');
    }
    
    return data;
  } catch (error) {
    console.error('進入隊列錯誤:', error);
    throw error;
  }
}
```

### 2. 重試機制
```javascript
// 重試機制範例
async function retryOperation(operation, maxRetries = 3) {
  for (let i = 0; i < maxRetries; i++) {
    try {
      return await operation();
    } catch (error) {
      if (i === maxRetries - 1) throw error;
      
      // 指數退避
      await new Promise(resolve => setTimeout(resolve, Math.pow(2, i) * 1000));
    }
  }
}
```

## 📝 注意事項

1. **用戶標識**: 確保 `user_hash` 的唯一性和安全性
2. **指紋識別**: 使用 `fingerprint` 防止同一用戶多個分頁
3. **輪詢間隔**: 根據活動配置調整輪詢頻率
4. **錯誤處理**: 實現適當的錯誤處理和重試機制
5. **監控告警**: 設置隊列長度和響應時間的監控告警

## 🔗 相關文檔

- [API 參考](./api-reference.md) - 完整的 API 文檔
- [最佳實踐](./best-practices.md) - 開發和部署建議
- [故障排除](./troubleshooting.md) - 常見問題解決
