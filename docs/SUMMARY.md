# 隊列系統專案總結

## 🎯 專案概述

這是一個**高併發隊列系統**，專為處理限時搶購、票務預約等場景設計。系統採用微服務架構，使用 Go 語言開發，具備高可用性和可擴展性。

## ✅ 已完成的工作

### 1. **Import 路徑修正**
- ✅ 修正了所有 import 路徑錯誤
- ✅ 將 `github.com/your-org/queue-system` 修正為 `queue-system`
- ✅ 統一了內部包的組織結構
- ✅ 將 `routes` 和 `middleware` 目錄移動到 `internal` 目錄下

### 2. **依賴管理**
- ✅ 添加了所有必要的 Go 模組依賴
- ✅ 創建了 `pkg/keys` 包提供 Redis 鍵生成功能
- ✅ 修正了 `go.mod` 和 `go.sum` 檔案

### 3. **資料庫設置**
- ✅ 創建了完整的資料庫遷移腳本 (`migrations/001_initial.sql`)
- ✅ 建立了三個核心表：
  - `activities` - 活動管理表
  - `queue_entries` - 隊列記錄表
  - `system_config` - 系統配置表
- ✅ 設置了適當的索引和約束

### 4. **Docker 環境**
- ✅ 配置了 `docker-compose.yml`
- ✅ PostgreSQL 和 Redis 服務正常運行
- ✅ 自動執行資料庫遷移

### 5. **測試框架**
- ✅ 創建了 `pkg/keys` 的單元測試
- ✅ 創建了 `internal/services` 的測試
- ✅ 所有測試通過

### 6. **API 服務**
- ✅ 應用成功啟動並運行在端口 8080
- ✅ 健康檢查 API 正常回應
- ✅ 所有路由正確註冊

## 🏗️ 系統架構

### **技術棧**
- **後端**: Go 1.25.0 + Gin 框架
- **資料庫**: PostgreSQL 15
- **快取**: Redis 7
- **配置管理**: Viper
- **容器化**: Docker + Docker Compose

### **核心功能模組**
1. **隊列管理服務** - 處理用戶進入隊列、序號分配
2. **管理服務** - 活動創建、狀態管理
3. **中間件** - CORS、請求 ID 追蹤
4. **資料存儲** - PostgreSQL 持久化 + Redis 快取

### **API 端點**
- `GET /health` - 健康檢查
- `POST /api/v1/queue/enter` - 進入隊列
- `GET /api/v1/queue/status` - 查詢隊列狀態
- `POST /api/v1/admin/activities` - 創建活動
- `GET /api/v1/admin/activities/:id/status` - 活動狀態

## 🚀 如何運行

### **快速開始（推薦）**
```bash
# 1. 啟動依賴服務
docker-compose up -d postgres redis

# 2. 運行應用
go run cmd/api/main.go

# 3. 測試 API
.\test_api.ps1
```

### **測試**
```bash
# 運行所有測試
go test ./...

# 運行特定測試
go test ./pkg/keys
go test ./internal/services
```

## 📊 資料庫結構

### **activities 表**
```sql
CREATE TABLE activities (
    id BIGSERIAL PRIMARY KEY,
    tenant_id VARCHAR(50) NOT NULL,
    name VARCHAR(200) NOT NULL,
    sku VARCHAR(100) NOT NULL,
    initial_stock INTEGER NOT NULL,
    start_at TIMESTAMP NOT NULL,
    end_at TIMESTAMP NOT NULL,
    status VARCHAR(20) DEFAULT 'draft',
    config_json JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
```

### **queue_entries 表**
```sql
CREATE TABLE queue_entries (
    id BIGSERIAL PRIMARY KEY,
    activity_id BIGINT REFERENCES activities(id),
    user_hash VARCHAR(64) NOT NULL,
    session_id VARCHAR(100) NOT NULL,
    seq_number BIGINT NOT NULL,
    fingerprint JSONB,
    ip_hash VARCHAR(64),
    created_at TIMESTAMP DEFAULT NOW()
);
```

## 🔧 配置說明

### **主要配置檔案**
- `internal/config/config.yaml` - 應用配置
- `docker-compose.yml` - Docker 服務配置
- `go.mod` - Go 模組依賴

### **環境變數支援**
- 支援 `QUEUE_` 前綴的環境變數
- 可覆蓋配置檔案中的設定

## 🛡️ 安全特性

1. **IP 節流控制** - 防止惡意請求
2. **用戶去重機制** - 防止同一用戶多個分頁
3. **請求 ID 追蹤** - 便於問題排查
4. **CORS 支援** - 跨域請求處理

## 📈 效能優化

1. **Redis 快取** - 高併發隊列操作
2. **連接池** - 資料庫和 Redis 連接復用
3. **索引優化** - 資料庫查詢效能
4. **非同步處理** - 資料庫寫入不阻塞

## 📝 開發指南

### **新增功能流程**
1. 在 `internal/services` 中新增業務邏輯
2. 在 `internal/handlers` 中新增 HTTP 處理器
3. 在 `internal/routes` 中新增路由
4. 編寫對應的測試

### **程式碼風格**
- 使用 Go 官方程式碼風格
- 函數和變數使用 camelCase
- 包名使用小寫

## 🐛 故障排除

### **常見問題**
1. **配置檔案路徑** - 確保 `internal/config/config.yaml` 存在
2. **資料庫連接** - 檢查 PostgreSQL 是否運行在端口 5432
3. **Redis 連接** - 檢查 Redis 是否運行在端口 6379
4. **端口衝突** - 確保端口 8080 未被佔用

### **日誌查看**
```bash
# 查看應用日誌
docker-compose logs queue-api

# 查看資料庫日誌
docker-compose logs postgres

# 查看 Redis 日誌
docker-compose logs redis
```

## 🎉 總結

這個隊列系統專案已經成功建立並運行，具備了：

- ✅ **完整的開發環境**
- ✅ **資料庫和快取設置**
- ✅ **API 服務運行**
- ✅ **測試框架**
- ✅ **Docker 支援**
- ✅ **文檔和指南**

系統現在可以處理高併發的隊列需求，適合用於搶購、預約等場景。所有核心功能都已實現並經過測試驗證。
