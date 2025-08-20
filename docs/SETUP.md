# 隊列系統設置指南

## 🚀 快速開始

### 方式一：使用 Docker Compose（推薦）

1. **啟動所有服務**
```bash
docker-compose up -d
```

這會自動：
- 啟動 PostgreSQL 資料庫（端口 5432）
- 啟動 Redis 快取（端口 6379）
- 執行資料庫遷移腳本
- 啟動 API 服務（端口 8080）

2. **檢查服務狀態**
```bash
docker-compose ps
```

3. **查看日誌**
```bash
docker-compose logs -f queue-api
```

### 方式二：本地開發設置

#### 1. 安裝 PostgreSQL

**Windows:**
- 下載並安裝 [PostgreSQL](https://www.postgresql.org/download/windows/)
- 或使用 Chocolatey: `choco install postgresql`

**macOS:**
```bash
brew install postgresql
brew services start postgresql
```

**Linux (Ubuntu):**
```bash
sudo apt update
sudo apt install postgresql postgresql-contrib
sudo systemctl start postgresql
```

#### 2. 安裝 Redis

**Windows:**
- 下載 [Redis for Windows](https://github.com/microsoftarchive/redis/releases)
- 或使用 WSL2

**macOS:**
```bash
brew install redis
brew services start redis
```

**Linux (Ubuntu):**
```bash
sudo apt install redis-server
sudo systemctl start redis-server
```

#### 3. 建立資料庫

1. **連接到 PostgreSQL**
```bash
psql -U postgres
```

2. **建立資料庫**
```sql
CREATE DATABASE queue_system;
```

3. **執行遷移腳本**
```bash
psql -U postgres -d queue_system -f migrations/001_init.sql
```

#### 4. 配置應用

1. **複製配置檔案**
```bash
cp internal/config/config.yaml.example internal/config/config.yaml
```

2. **修改配置**（如果需要）
```yaml
database:
  host: "localhost"
  port: 5432
  user: "postgres"
  password: "your_password"
  dbname: "queue_system"

redis:
  host: "localhost"
  port: 6379
```

#### 5. 運行應用

```bash
go run cmd/api/main.go
```

## 📊 資料庫結構

### 主要表結構

#### 1. `activities` - 活動表
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

#### 2. `queue_entries` - 隊列記錄表
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

#### 3. `system_config` - 系統配置表
```sql
CREATE TABLE system_config (
    key VARCHAR(100) PRIMARY KEY,
    value JSONB NOT NULL,
    description TEXT,
    updated_at TIMESTAMP DEFAULT NOW()
);
```

## 🔧 驗證設置

### 1. 檢查資料庫連接

```bash
# 連接到資料庫
psql -U postgres -d queue_system

# 查看表
\dt

# 查看資料
SELECT * FROM activities;
SELECT * FROM system_config;
```

### 2. 檢查 Redis 連接

```bash
# 連接到 Redis
redis-cli

# 測試連接
ping

# 查看鍵
keys *
```

### 3. 測試 API

```bash
# 健康檢查
curl http://localhost:8080/health

# 創建測試活動
curl -X POST http://localhost:8080/api/v1/admin/activities \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_id": "test_tenant",
    "name": "測試活動",
    "sku": "TEST001",
    "initial_stock": 100,
    "start_at": "2024-01-01T10:00:00Z",
    "end_at": "2024-01-01T18:00:00Z"
  }'
```

## 🐛 故障排除

### 常見問題

1. **PostgreSQL 連接失敗**
   ```bash
   # 檢查服務狀態
   sudo systemctl status postgresql
   
   # 檢查端口
   netstat -an | grep 5432
   ```

2. **Redis 連接失敗**
   ```bash
   # 檢查服務狀態
   sudo systemctl status redis
   
   # 檢查端口
   netstat -an | grep 6379
   ```

3. **權限問題**
   ```bash
   # PostgreSQL 權限
   sudo -u postgres psql
   ALTER USER postgres PASSWORD 'password';
   ```

4. **Docker 問題**
   ```bash
   # 清理容器
   docker-compose down -v
   docker-compose up -d
   
   # 查看日誌
   docker-compose logs postgres
   ```

## 📝 開發建議

1. **使用 Docker Compose 進行開發**，避免本地環境配置問題
2. **定期備份資料庫**：`pg_dump -U postgres queue_system > backup.sql`
3. **監控服務狀態**：使用 `docker-compose ps` 和 `docker-compose logs`
4. **使用環境變數**：在生產環境中使用環境變數覆蓋配置

## 🔄 資料庫遷移

當需要修改資料庫結構時：

1. 創建新的遷移檔案：`migrations/002_add_new_table.sql`
2. 在 Docker Compose 中，遷移會自動執行
3. 本地開發時手動執行：`psql -U postgres -d queue_system -f migrations/002_add_new_table.sql`
