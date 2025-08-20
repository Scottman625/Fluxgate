# Queue System - Docker 部署指南

## 🚀 快速啟動

### 使用 PowerShell (Windows)
```powershell
.\docker-start.ps1
```

### 使用 Bash (Linux/Mac)
```bash
chmod +x docker-start.sh
./docker-start.sh
```

### 手動啟動
```bash
# 建構並啟動所有服務
docker-compose up --build -d

# 查看服務狀態
docker-compose ps

# 查看日誌
docker-compose logs -f queue-server
```

## 📊 服務端點

| 服務 | 端口 | 描述 | URL |
|------|------|------|-----|
| Queue System API | 8080 | 主要 API 服務 | http://localhost:8080 |
| Queue Dashboard | 8080 | Web 儀表板 | http://localhost:8080 |
| Prometheus 指標 | 9090 | 應用指標收集 | http://localhost:8080:9090/metrics |
| Prometheus UI | 9091 | Prometheus 查詢界面 | http://localhost:9091 |
| Grafana | 3000 | 監控儀表板 | http://localhost:3000 |
| PostgreSQL | 5432 | 資料庫 | localhost:5432 |
| Redis | 6379 | 快取服務 | localhost:6379 |

## 🔐 預設認證資訊

- **Grafana**: admin / admin
- **PostgreSQL**: user / password
- **Redis**: 無密碼

## 🧪 API 測試

運行 API 測試腳本：
```powershell
.\test-docker-api.ps1
```

### 手動測試 API

```bash
# 測試儀表板
curl http://localhost:8080/api/v1/dashboard

# 測試實時指標
curl http://localhost:8080/api/v1/dashboard/metrics/realtime

# 測試 Prometheus 指標
curl http://localhost:8080:9090/metrics
```

## 📁 服務架構

```
queue-system/
├── queue-server     # 主要 API 服務
├── postgres         # PostgreSQL 資料庫
├── redis           # Redis 快取
├── prometheus      # 指標收集
└── grafana         # 監控儀表板
```

## 🔧 配置說明

### 環境變數
主要配置都在 `docker-compose.yml` 中：

```yaml
environment:
  DATABASE_URL: "postgres://user:password@postgres/queuedb?sslmode=disable"
  REDIS_ADDR: "redis:6379"
  REDIS_PASSWORD: ""
  PORT: "8080"
  GIN_MODE: "release"
```

### 數據持久化
以下目錄會被持久化保存：
- `postgres_data`: PostgreSQL 數據
- `redis_data`: Redis 數據
- `prometheus_data`: Prometheus 指標數據
- `grafana_data`: Grafana 配置和儀表板

## 📊 監控設置

### Prometheus
- 自動收集應用指標
- 配置文件: `prometheus.yml`
- 存儲保留時間: 200 小時

### Grafana
- 預設儀表板配置: `grafana-dashboard.json`
- 自動連接 Prometheus 數據源
- 可視化隊列系統關鍵指標

## 🛠️ 開發和調試

### 查看日誌
```bash
# 查看所有服務日誌
docker-compose logs -f

# 查看特定服務日誌
docker-compose logs -f queue-server
docker-compose logs -f postgres
docker-compose logs -f redis
```

### 進入容器
```bash
# 進入應用容器
docker-compose exec queue-server sh

# 進入資料庫容器
docker-compose exec postgres psql -U user -d queuedb
```

### 重建服務
```bash
# 重建並重啟特定服務
docker-compose up --build -d queue-server

# 重建所有服務
docker-compose down
docker-compose up --build -d
```

## 🛑 停止服務

```bash
# 停止所有服務
docker-compose down

# 停止服務並刪除數據卷
docker-compose down -v

# 停止服務並刪除所有相關資源
docker-compose down -v --rmi all
```

## 📋 健康檢查

系統包含以下健康檢查：

- **PostgreSQL**: `pg_isready -U user -d queuedb`
- **Redis**: `redis-cli ping`
- **Queue Server**: HTTP GET `/api/v1/dashboard`

可以通過以下命令查看健康狀態：
```bash
docker-compose ps
```

## 🚨 故障排除

### 常見問題

1. **端口被占用**
   - 修改 `docker-compose.yml` 中的端口映射
   - 或停止占用端口的服務

2. **資料庫連接失敗**
   - 確保 PostgreSQL 容器正常啟動
   - 檢查 `DATABASE_URL` 環境變數

3. **Redis 連接失敗**
   - 確保 Redis 容器正常啟動
   - 檢查 `REDIS_ADDR` 環境變數

4. **應用無法啟動**
   - 檢查應用日誌: `docker-compose logs queue-server`
   - 確保所有依賴服務都已正常啟動

### 重置系統
```bash
# 完全重置（刪除所有數據）
docker-compose down -v
docker system prune -f
docker-compose up --build -d
```
