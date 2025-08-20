# 部署流程說明

本文件詳細說明 GitHub Actions 的完整部署流程，包括 Docker 映像檔建置、推送和伺服器部署。

## 部署流程概覽

```mermaid
graph TD
    A[推送程式碼到 master] --> B[觸發 GitHub Actions]
    B --> C[建置 Docker 映像檔]
    C --> D[推送到 GHCR]
    D --> E[SSH 連接到伺服器]
    E --> F[拉取最新映像檔]
    F --> G[更新 docker-compose.yml]
    G --> H[重啟服務]
    H --> I[健康檢查]
    I --> J[部署完成]
```

## 詳細步驟說明

### 1. 觸發條件

部署會在以下情況觸發：
- 推送程式碼到 `master` 分支
- 手動觸發 (`workflow_dispatch`)

### 2. 建置階段

#### 2.1 設置 Docker Buildx
```yaml
- name: Set up Docker Buildx
  uses: docker/setup-buildx-action@v3
```

#### 2.2 登入 GitHub Container Registry
```yaml
- name: Log in to GitHub Container Registry
  uses: docker/login-action@v3
  with:
    registry: ghcr.io
    username: ${{ github.actor }}
    password: ${{ secrets.GITHUB_TOKEN }}
```

#### 2.3 建置和推送映像檔
```yaml
- name: Build and push Docker image
  uses: docker/build-push-action@v5
  with:
    context: .
    push: true
    tags: |
      ghcr.io/Scottman625/Fluxgate:${{ github.sha }}
      ghcr.io/Scottman625/Fluxgate:master
```

**映像檔標籤策略**：
- `${{ github.sha }}`: 特定提交的映像檔（用於回滾）
- `master`: 最新版本標籤

### 3. 部署階段

#### 3.1 SSH 連接到伺服器
```yaml
- name: Deploy to server
  uses: appleboy/ssh-action@v1.0.0
  with:
    host: ${{ secrets.SERVER_HOST }}
    username: ${{ secrets.SERVER_USER }}
    key: ${{ secrets.SERVER_SSH_KEY }}
    port: ${{ secrets.SERVER_PORT }}
```

#### 3.2 伺服器端部署腳本

```bash
# 停止現有服務
docker-compose down

# 登入 GitHub Container Registry
echo ${{ secrets.GITHUB_TOKEN }} | docker login ghcr.io -u ${{ github.actor }} --password-stdin

# 拉取最新映像檔
docker pull ghcr.io/Scottman625/Fluxgate:${{ github.sha }}

# 更新 docker-compose.yml 中的映像檔標籤
sed -i "s|IMAGE_TAG|${{ github.sha }}|g" docker-compose.yml

# 啟動服務
docker-compose up -d

# 等待服務啟動
sleep 30

# 健康檢查
curl -f http://localhost:8085/health || exit 1
```

### 4. 健康檢查

#### 4.1 內部健康檢查
在伺服器上執行：
```bash
curl -f http://localhost:8085/health
```

#### 4.2 外部健康檢查
從 GitHub Actions 執行：
```bash
curl -f ${{ secrets.APP_URL }}:8085/health
```

## 必要的 GitHub Secrets

| Secret 名稱 | 描述 | 範例值 |
|------------|------|--------|
| `SERVER_HOST` | 伺服器 IP 或域名 | `192.168.1.100` |
| `SERVER_USER` | SSH 用戶名 | `deploy` |
| `SERVER_SSH_KEY` | SSH 私鑰 | 完整的私鑰內容 |
| `SERVER_PORT` | SSH 埠號 | `22` |
| `APP_URL` | 應用程式 URL | `https://your-domain.com` |

## 伺服器準備工作

### 1. 安裝必要軟體
```bash
# 安裝 Docker
sudo apt update
sudo apt install docker.io docker-compose

# 將用戶添加到 docker 群組
sudo usermod -aG docker $USER
```

### 2. 設置部署目錄
```bash
# 創建部署目錄
sudo mkdir -p /opt/queue-system
sudo chown $USER:$USER /opt/queue-system
```

### 3. 創建 docker-compose.yml
```yaml
version: '3.8'

services:
  queue-server:
    image: ghcr.io/Scottman625/Fluxgate:IMAGE_TAG
    ports:
      - "8085:8080"
      - "9092:9090"
    environment:
      - DATABASE_URL=${DATABASE_URL}
      - REDIS_ADDR=${REDIS_ADDR}
      - REDIS_PASSWORD=${REDIS_PASSWORD}
      - PORT=8080
      - GIN_MODE=release
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
```

### 4. 設置環境變數
創建 `/opt/queue-system/.env`：
```bash
DATABASE_URL=postgres://user:pass@host:5432/queuedb
REDIS_ADDR=redis:6379
REDIS_PASSWORD=your_redis_password
```

## 故障排除

### 常見問題

1. **映像檔建置失敗**
   ```bash
   # 檢查 Dockerfile 語法
   docker build --no-cache .
   
   # 檢查 GitHub Token 權限
   # 確保有 packages:write 權限
   ```

2. **推送映像檔失敗**
   ```bash
   # 檢查 GHCR 權限
   # 確保儲存庫有適當的權限設置
   ```

3. **SSH 連接失敗**
   ```bash
   # 測試 SSH 連接
   ssh -i ~/.ssh/github_actions_deploy user@server
   
   # 檢查防火牆設置
   sudo ufw status
   ```

4. **Docker 登入失敗**
   ```bash
   # 手動測試登入
   echo $GITHUB_TOKEN | docker login ghcr.io -u $USERNAME --password-stdin
   ```

5. **服務啟動失敗**
   ```bash
   # 檢查容器日誌
   docker-compose logs queue-server
   
   # 檢查映像檔是否存在
   docker images | grep Fluxgate
   ```

### 調試命令

```bash
# 檢查服務狀態
docker-compose ps

# 查看容器日誌
docker-compose logs -f queue-server

# 檢查映像檔
docker images ghcr.io/Scottman625/Fluxgate

# 測試健康檢查端點
curl -v http://localhost:8085/health

# 檢查網路連接
docker network ls
docker network inspect queue-system_default
```

## 回滾策略

### 自動回滾
如果健康檢查失敗，可以自動回滾到上一個版本：

```yaml
- name: Rollback on failure
  if: failure()
  run: |
    # 回滾到上一個版本
    docker pull ghcr.io/Scottman625/Fluxgate:previous
    sed -i "s|IMAGE_TAG|previous|g" docker-compose.yml
    docker-compose up -d
```

### 手動回滾
```bash
# 在伺服器上執行
cd /opt/queue-system
docker pull ghcr.io/Scottman625/Fluxgate:previous_commit_sha
sed -i "s|IMAGE_TAG|previous_commit_sha|g" docker-compose.yml
docker-compose up -d
```

## 監控和通知

### 部署通知
可以添加 Slack 或 Discord 通知：

```yaml
- name: Notify deployment success
  if: success()
  uses: 8398a7/action-slack@v3
  with:
    status: success
    webhook_url: ${{ secrets.SLACK_WEBHOOK }}
```

### 部署監控
- 監控部署時間
- 追蹤部署成功率
- 記錄部署歷史

## 安全考量

1. **映像檔安全**
   - 定期更新基礎映像檔
   - 掃描映像檔漏洞
   - 使用多階段建置減少攻擊面

2. **部署安全**
   - 使用專用的部署用戶
   - 限制 SSH 訪問
   - 加密敏感資訊

3. **網路安全**
   - 使用防火牆
   - 配置反向代理
   - 實施 SSL/TLS
