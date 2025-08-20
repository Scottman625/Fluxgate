# Fluxgate Queue System

一個高效能的排隊系統，使用 Go 語言開發，支援 Docker 容器化部署和完整的 CI/CD 流程。

## 🚀 功能特色

- **高效能排隊**: 基於 Redis 的高效能隊列管理
- **實時監控**: 完整的 Dashboard 和 Prometheus 指標
- **容器化部署**: Docker 和 Docker Compose 支援
- **CI/CD 流程**: GitHub Actions 自動化測試和部署
- **RESTful API**: 完整的 REST API 介面
- **前端 SDK**: JavaScript SDK 和 UI 組件

## 📋 系統架構

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Frontend      │    │   API Server    │    │   Database      │
│   (Dashboard)   │◄──►│   (Go/Gin)      │◄──►│   (PostgreSQL)  │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                │
                                ▼
                       ┌─────────────────┐
                       │   Cache         │
                       │   (Redis)       │
                       └─────────────────┘
```

## 🛠️ 技術棧

- **後端**: Go 1.25, Gin Framework
- **資料庫**: PostgreSQL 15
- **快取**: Redis 7
- **容器化**: Docker, Docker Compose
- **監控**: Prometheus, Grafana
- **CI/CD**: GitHub Actions
- **前端**: HTML5, CSS3, JavaScript, Chart.js

## 🚀 快速開始

### 使用 Docker Compose (推薦)

1. **克隆專案**
   ```bash
   git clone https://github.com/your-username/queue-system.git
   cd queue-system
   ```

2. **啟動服務**
   ```bash
   docker-compose up -d
   ```

3. **訪問應用**
   - Dashboard: http://localhost:8085
   - API 文檔: http://localhost:8085/api/v1
   - Prometheus: http://localhost:9093
   - Grafana: http://localhost:3001 (admin/admin)

### 本地開發

1. **安裝依賴**
   ```bash
   go mod download
   ```

2. **設置環境變數**
   ```bash
   export DATABASE_URL="postgres://postgres:password@localhost:5432/queuedb?sslmode=disable"
   export REDIS_ADDR="localhost:6379"
   export PORT="8080"
   ```

3. **運行應用**
   ```bash
   go run cmd/server/main.go
   ```

## 🧪 測試

### 本地測試
```bash
# 執行所有測試
go test -v ./...

# 執行測試並檢查覆蓋率
go test -cover ./...

# 使用測試腳本
./scripts/test-local.sh          # Linux/macOS
.\scripts\test-local.ps1         # Windows
```

### CI/CD 測試
GitHub Actions 會自動執行以下測試：
- 單元測試
- 程式碼檢查 (linting)
- Docker 建置測試
- 安全掃描

## 📊 監控和指標

### Dashboard
- 實時隊列狀態
- 釋放速率圖表
- 系統資源使用情況
- 活動歷史記錄

### Prometheus 指標
- HTTP 請求數和響應時間
- 隊列長度和活躍用戶數
- 調度器狀態和釋放速率
- 系統資源使用率

### Grafana 儀表板
- 預設的監控儀表板
- 可自訂的圖表和告警

## 🔧 API 使用

### 進入隊列
```bash
curl -X POST http://localhost:8080/api/v1/queue/enter \
  -H "Content-Type: application/json" \
  -d '{
    "activity_id": 1,
    "user_hash": "user_123",
    "fingerprint": "fp_456"
  }'
```

### 查詢隊列狀態
```bash
curl "http://localhost:8080/api/v1/queue/status?activity_id=1&session_id=session_abc123"
```

### 獲取實時指標
```bash
curl http://localhost:8080/api/v1/dashboard/metrics/realtime
```

## 🎯 前端整合

### JavaScript SDK
```javascript
import { QueueSDK } from './sdk/queue-sdk.js';

const sdk = new QueueSDK({
    baseUrl: 'http://localhost:8080',
    activityId: 1
});

// 進入隊列
const result = await sdk.enterQueue('user_123', 'fp_456');
```

### UI 組件
```javascript
import { QueueWidget } from './components/queue-widget.js';

const widget = new QueueWidget('#queue-container', {
    activityId: 1,
    theme: 'light'
});
```

## 🔄 CI/CD 流程

### GitHub Actions 工作流程

1. **測試階段**
   - 單元測試
   - 程式碼檢查
   - 依賴驗證

2. **建置階段**
   - Docker 映像檔建置
   - 推送到 GitHub Container Registry
   - 自動標籤管理

3. **部署階段**
   - 自動部署到測試環境
   - 整合測試
   - 安全掃描

4. **安全掃描**
   - Trivy 漏洞掃描
   - 程式碼安全分析

### 部署策略
- **測試環境**: 自動部署
- **生產環境**: 手動觸發，需要審核

## 📁 專案結構

```
queue-system/
├── .github/workflows/     # GitHub Actions 工作流程
├── cmd/server/           # 主程式入口
├── internal/             # 內部套件
│   ├── handlers/         # HTTP 處理器
│   ├── models/          # 資料模型
│   ├── services/        # 業務邏輯
│   ├── monitoring/      # 監控相關
│   └── metrics/         # 指標收集
├── web/                 # 前端檔案
│   ├── dashboard/       # Dashboard 頁面
│   ├── sdk/            # JavaScript SDK
│   ├── components/     # UI 組件
│   └── examples/       # 使用範例
├── migrations/          # 資料庫遷移
├── scripts/            # 腳本檔案
├── docs/               # 文檔
├── docker-compose.yml  # Docker Compose 配置
├── Dockerfile          # Docker 建置檔案
└── README.md           # 專案說明
```

## 🔐 安全性

- 使用 HTTPS 加密通訊
- 輸入驗證和清理
- SQL 注入防護
- XSS 防護
- 定期安全掃描

## 🤝 貢獻

1. Fork 專案
2. 建立功能分支 (`git checkout -b feature/AmazingFeature`)
3. 提交變更 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 開啟 Pull Request

## 📄 授權

本專案採用 MIT 授權條款 - 詳見 [LICENSE](LICENSE) 檔案

## 📞 支援

- 問題回報: [GitHub Issues](https://github.com/your-username/queue-system/issues)
- 文檔: [docs/](docs/)
- 郵件: support@fluxgate.com

## 🔗 相關連結

- [API 文檔](docs/api-reference.md)
- [部署指南](docs/deployment.md)
- [GitHub Actions 說明](docs/github-actions.md)
- [開發指南](docs/development.md)

---

**Fluxgate Queue System** - 高效能排隊解決方案 🚀
