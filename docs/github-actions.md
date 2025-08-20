# GitHub Actions CI/CD 流程說明

## 概述

本專案使用 GitHub Actions 實現持續整合和持續部署 (CI/CD) 流程，確保程式碼品質和自動化部署。

## 工作流程架構

### 觸發條件
- **Push 到 main 或 develop 分支**
- **Pull Request 到 main 分支**

### 工作階段 (Jobs)

#### 1. 測試階段 (test)
- **運行環境**: Ubuntu Latest
- **服務**: PostgreSQL 15, Redis 7
- **步驟**:
  - 檢出程式碼
  - 設置 Go 1.25 環境
  - 安裝依賴
  - 執行單元測試
  - 執行程式碼檢查 (linting)

#### 2. 建置階段 (build)
- **依賴**: 測試階段成功
- **權限**: 讀取內容，寫入套件
- **步驟**:
  - 設置 Docker Buildx
  - 登入 GitHub Container Registry
  - 提取映像檔中繼資料
  - 建置並推送 Docker 映像檔

#### 3. 部署階段 (deploy)
- **依賴**: 建置階段成功
- **觸發**: 僅在 main 分支
- **步驟**:
  - 部署到測試環境
  - 執行整合測試

#### 4. 安全掃描 (security)
- **並行執行**: 與其他階段並行
- **步驟**:
  - 使用 Trivy 進行漏洞掃描
  - 上傳結果到 GitHub Security 標籤

## 檔案結構

```
.github/
└── workflows/
    └── ci-cd.yml          # 主要 CI/CD 工作流程
```

## 環境變數

### 自動設定
- `GITHUB_TOKEN`: GitHub 自動提供的權杖
- `REGISTRY`: GitHub Container Registry (ghcr.io)
- `IMAGE_NAME`: 儲存庫名稱

### 測試環境
- `DATABASE_URL`: PostgreSQL 測試資料庫連線字串
- `REDIS_ADDR`: Redis 測試伺服器位址
- `PORT`: 應用程式埠號

## Docker 映像檔標籤策略

### 自動標籤
- **分支標籤**: `main`, `develop`
- **PR 標籤**: `pr-{number}`
- **語意化版本**: `v1.0.0`, `v1.0`
- **SHA 標籤**: 提交的 SHA 值

### 範例標籤
```
ghcr.io/username/queue-system:main
ghcr.io/username/queue-system:pr-123
ghcr.io/username/queue-system:v1.0.0
ghcr.io/username/queue-system:sha-abc123
```

## 測試策略

### 單元測試
- 使用 `go test` 執行所有測試
- 涵蓋 handlers, services, models 等核心模組
- 使用 mock 物件隔離外部依賴

### 整合測試
- 在部署階段執行
- 測試完整的 API 端點
- 驗證資料庫和 Redis 整合

### 安全測試
- Trivy 漏洞掃描
- 檢查 Docker 映像檔安全性
- 掃描程式碼中的安全問題

## 部署策略

### 測試環境
- 自動部署到測試環境
- 執行整合測試
- 驗證功能完整性

### 生產環境
- 手動觸發部署
- 需要審核和批准
- 藍綠部署或滾動更新

## 監控和通知

### 成功通知
- 工作流程成功時發送通知
- 包含部署連結和狀態

### 失敗通知
- 工作流程失敗時立即通知
- 包含錯誤詳情和修復建議

## 最佳實踐

### 程式碼品質
1. **測試覆蓋率**: 維持高測試覆蓋率
2. **程式碼檢查**: 使用 golint 確保程式碼品質
3. **依賴管理**: 定期更新依賴套件

### 安全性
1. **漏洞掃描**: 每次建置都進行安全掃描
2. **權限最小化**: 只授予必要權限
3. **機密管理**: 使用 GitHub Secrets 管理敏感資訊

### 效能優化
1. **快取策略**: 使用 GitHub Actions 快取
2. **並行執行**: 最大化並行執行效率
3. **映像檔優化**: 使用多階段建置

## 故障排除

### 常見問題

#### 1. 測試失敗
```bash
# 本地執行測試
go test -v ./...

# 檢查測試覆蓋率
go test -cover ./...
```

#### 2. Docker 建置失敗
```bash
# 本地建置測試
docker build -t queue-system .

# 檢查 Dockerfile 語法
docker build --no-cache -t queue-system .
```

#### 3. 部署失敗
- 檢查環境變數設定
- 驗證網路連線
- 確認權限設定

### 日誌查看
- GitHub Actions 頁面查看詳細日誌
- 使用 `act` 工具本地測試工作流程

## 自訂和擴展

### 添加新階段
1. 在 `.github/workflows/ci-cd.yml` 中添加新 job
2. 定義觸發條件和依賴關係
3. 設定適當的權限和環境

### 自訂通知
1. 使用 GitHub 的 webhook 功能
2. 整合 Slack, Teams 等通知系統
3. 設定條件式通知

### 環境特定配置
1. 為不同環境建立不同的工作流程
2. 使用環境變數管理配置
3. 實作環境特定的部署策略

## 相關連結

- [GitHub Actions 官方文檔](https://docs.github.com/en/actions)
- [Docker 官方文檔](https://docs.docker.com/)
- [Go 測試文檔](https://golang.org/pkg/testing/)
- [Trivy 安全掃描](https://aquasecurity.github.io/trivy/)
