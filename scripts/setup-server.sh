#!/bin/bash

# 伺服器端 SSH 設置腳本
# 在目標部署伺服器上執行

set -e

echo "🔧 設置伺服器 SSH 環境..."

# 顏色定義
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

print_status() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

# 檢查是否為 root 用戶
if [ "$EUID" -eq 0 ]; then
    print_error "請不要使用 root 用戶執行此腳本"
    exit 1
fi

# 創建 .ssh 目錄
if [ ! -d ~/.ssh ]; then
    print_status "創建 .ssh 目錄..."
    mkdir -p ~/.ssh
    chmod 700 ~/.ssh
fi

# 創建或備份 authorized_keys
if [ -f ~/.ssh/authorized_keys ]; then
    print_warning "備份現有的 authorized_keys..."
    cp ~/.ssh/authorized_keys ~/.ssh/authorized_keys.backup.$(date +%Y%m%d_%H%M%S)
fi

# 提示用戶添加公鑰
echo ""
print_warning "請將 GitHub Actions 的公鑰添加到 authorized_keys:"
echo "1. 複製公鑰內容"
echo "2. 執行: echo '公鑰內容' >> ~/.ssh/authorized_keys"
echo "3. 或者手動編輯 ~/.ssh/authorized_keys 文件"

# 設置權限
chmod 600 ~/.ssh/authorized_keys

# 檢查 SSH 配置
print_status "檢查 SSH 配置..."
if ! grep -q "PubkeyAuthentication yes" /etc/ssh/sshd_config; then
    print_warning "SSH 公鑰認證可能未啟用，請檢查 /etc/ssh/sshd_config"
fi

if ! grep -q "AuthorizedKeysFile" /etc/ssh/sshd_config; then
    print_warning "AuthorizedKeysFile 可能未配置，請檢查 /etc/ssh/sshd_config"
fi

# 創建部署目錄
DEPLOY_DIR="/opt/queue-system"
if [ ! -d "$DEPLOY_DIR" ]; then
    print_status "創建部署目錄..."
    sudo mkdir -p "$DEPLOY_DIR"
    sudo chown $USER:$USER "$DEPLOY_DIR"
fi

# 創建 docker-compose.yml 模板
if [ ! -f "$DEPLOY_DIR/docker-compose.yml" ]; then
    print_status "創建 docker-compose.yml 模板..."
    cat > "$DEPLOY_DIR/docker-compose.yml" << 'EOF'
version: '3.8'

services:
  queue-server:
    image: ghcr.io/scottman625/fluxgate:IMAGE_TAG
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
EOF
fi

print_status "伺服器設置完成！"
echo ""
print_warning "下一步操作："
echo "1. 將 GitHub Actions 公鑰添加到 ~/.ssh/authorized_keys"
echo "2. 測試 SSH 連接: ssh -i ~/.ssh/github_actions_deploy user@server"
echo "3. 配置環境變數在 docker-compose.yml 中"
