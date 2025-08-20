# SSH 部署設置指南

本指南詳細說明如何設置 GitHub Actions 的 SSH 部署功能。

## 概述

SSH 部署允許 GitHub Actions 通過 SSH 連接到您的伺服器並自動部署應用程式。

## 步驟 1: 生成 SSH 金鑰對

### 在本地機器上執行

```bash
# Linux/macOS
./scripts/setup-ssh-deploy.sh

# Windows PowerShell
.\scripts\setup-ssh-deploy.ps1
```

或者手動生成：

```bash
ssh-keygen -t rsa -b 4096 -C "github-actions-deploy" -f ~/.ssh/github_actions_deploy -N ""
```

## 步驟 2: 設置 GitHub Secrets

在您的 GitHub 儲存庫中，前往 **Settings > Secrets and variables > Actions**，添加以下 secrets：

### 必需的 Secrets

| Secret 名稱 | 描述 | 範例值 |
|------------|------|--------|
| `SERVER_HOST` | 伺服器 IP 位址或域名 | `192.168.1.100` 或 `server.example.com` |
| `SERVER_USER` | SSH 用戶名 | `deploy` |
| `SERVER_SSH_KEY` | SSH 私鑰內容 | 整個私鑰檔案內容 |
| `SERVER_PORT` | SSH 埠號 | `22` |
| `APP_URL` | 應用程式 URL | `https://your-app-domain.com` |

### 可選的 Secrets

| Secret 名稱 | 描述 | 範例值 |
|------------|------|--------|
| `DATABASE_URL` | 資料庫連線字串 | `postgres://user:pass@host:5432/db` |
| `REDIS_ADDR` | Redis 伺服器位址 | `redis:6379` |
| `REDIS_PASSWORD` | Redis 密碼 | `your_redis_password` |

## 步驟 3: 設置目標伺服器

### 在目標伺服器上執行

```bash
# 上傳並執行伺服器設置腳本
scp scripts/setup-server.sh user@server:/tmp/
ssh user@server "chmod +x /tmp/setup-server.sh && /tmp/setup-server.sh"
```

### 手動設置伺服器

1. **創建 SSH 目錄**：
   ```bash
   mkdir -p ~/.ssh
   chmod 700 ~/.ssh
   ```

2. **添加公鑰到 authorized_keys**：
   ```bash
   echo "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQCq2KiP0BzrugG8uI5U6bEW63dBjL1i5OvaJVzWw4ktTsRx2BQCXbve04VsGg+kM9hXRbd9XOsdvWTHuKBcfl0bio4YqG+Tt6/4g3rug5gqP/9El6YBZmnPmMEmrAEjMzr/GH5uQvPk972zFJXw8b4TOdzIOQ2egFFG06yOJCr7adWvdK+hVPLutIczoxceWz8+Wx6q2/DWhiqYx52/mZDqeSKIQs4cSGg+QMtyNtwqXhC3CkSCnfy7ULVNUJYMVdt87LEBN73uxYzZihtfMgz+vFXKEGOr0SvEUqlsDD3cxmVhMweTwl+6F/QOzdZ5dOKtFvTeltanloqH0vP3batqbB6VgwjRiKRPwarPC1F76BY1IrInte9K7OPFh3zWxXABtkwQpRuDlzZN3L/jhtTO9z6PBTyAUiUrLRYOaAM6kfNXJtg00moR/g0jpts8vElIb4RHbvy68nfv4ikYxu2b2RJxt9DA8jSk/2BIy+Et0grAoUVaORJYvTJ5goEEVLIrmm3wbWD079da1JgkIuyS1GHWuERv5GYDtMAEDelCwxSq4SO0SubfdizQoKUwjKBj/GymDe9JmMbzNUf8Q/2hezg1fWJ8b8vBAcB5NZfBwkllwasu5hVx4ktvFEEhggJfmWGdZI9pro+YQPywJrgRy6SkNIuZ/qQmt9Sq0P7Dmw== github-actions-deploy" >> ~/.ssh/authorized_keys
   chmod 600 ~/.ssh/authorized_keys
   ```

3. **創建部署目錄**：
   ```bash
   sudo mkdir -p /opt/queue-system
   sudo chown $USER:$USER /opt/queue-system
   ```

4. **安裝 Docker 和 Docker Compose**：
   ```bash
   # Ubuntu/Debian
   sudo apt update
   sudo apt install docker.io docker-compose
   sudo usermod -aG docker $USER
   
   # CentOS/RHEL
   sudo yum install docker docker-compose
   sudo systemctl enable docker
   sudo systemctl start docker
   sudo usermod -aG docker $USER
   ```

## 步驟 4: 測試 SSH 連接

### 在本地測試

```bash
# 測試 SSH 連接
ssh -i ~/.ssh/github_actions_deploy user@server "echo 'SSH connection successful'"
```

### 在 GitHub Actions 中測試

創建一個測試工作流程：

```yaml
name: Test SSH Connection

on:
  workflow_dispatch:

jobs:
  test-ssh:
    runs-on: ubuntu-latest
    steps:
    - name: Test SSH connection
      uses: appleboy/ssh-action@v1.0.0
      with:
        host: ${{ secrets.SERVER_HOST }}
        username: ${{ secrets.SERVER_USER }}
        key: ${{ secrets.SERVER_SSH_KEY }}
        port: ${{ secrets.SERVER_PORT }}
        script: |
          echo "SSH connection successful!"
          whoami
          pwd
```

## 步驟 5: 配置部署腳本

### 更新 docker-compose.yml

在伺服器上創建 `/opt/queue-system/docker-compose.yml`：

```yaml
version: '3.8'

services:
  queue-server:
    image: ghcr.io/Scottman625/Fluxgate:master
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

### 創建環境變數檔案

創建 `/opt/queue-system/.env`：

```bash
DATABASE_URL=postgres://user:pass@host:5432/queuedb
REDIS_ADDR=redis:6379
REDIS_PASSWORD=your_redis_password
```

## 步驟 6: 啟用 GitHub Actions 工作流程

確保 `.github/workflows/ci-cd-ssh.yml` 檔案存在並包含正確的配置。

## 故障排除

### 常見問題

1. **SSH 連接失敗**
   - 檢查伺服器 IP 和埠號
   - 確認 SSH 服務正在運行
   - 檢查防火牆設置

2. **權限被拒絕**
   - 確認公鑰已正確添加到 authorized_keys
   - 檢查檔案權限 (600 for authorized_keys)
   - 確認 SSH 配置允許公鑰認證

3. **Docker 權限問題**
   - 將用戶添加到 docker 群組
   - 重新登入或重啟系統

4. **部署失敗**
   - 檢查 Docker 映像檔是否存在
   - 確認環境變數設置正確
   - 查看 Docker Compose 日誌

### 調試命令

```bash
# 檢查 SSH 服務狀態
sudo systemctl status sshd

# 查看 SSH 日誌
sudo journalctl -u sshd

# 測試 SSH 連接（詳細模式）
ssh -v -i ~/.ssh/github_actions_deploy user@server

# 檢查 Docker 狀態
docker ps
docker-compose logs

# 檢查檔案權限
ls -la ~/.ssh/
```

## 安全注意事項

1. **金鑰管理**
   - 定期輪換 SSH 金鑰
   - 不要在程式碼中硬編碼金鑰
   - 使用 GitHub Secrets 存儲敏感資訊

2. **伺服器安全**
   - 禁用 root SSH 登入
   - 使用非標準 SSH 埠號
   - 配置防火牆規則
   - 定期更新系統

3. **部署安全**
   - 限制部署用戶的權限
   - 使用專用的部署用戶
   - 監控部署日誌

## 下一步

設置完成後，您可以：

1. 推送程式碼觸發自動部署
2. 監控部署日誌
3. 設置部署通知
4. 配置回滾機制
