#!/bin/bash

# SSH 部署設置腳本
# 用於設置 GitHub Actions 的 SSH 部署

set -e

echo "🔧 設置 SSH 部署環境..."

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

# 檢查是否已存在 SSH 金鑰
if [ -f ~/.ssh/github_actions_deploy ]; then
    print_warning "SSH 金鑰已存在，跳過生成步驟"
else
    print_status "生成 SSH 金鑰對..."
    ssh-keygen -t rsa -b 4096 -C "github-actions-deploy" -f ~/.ssh/github_actions_deploy -N ""
fi

# 顯示私鑰（用於 GitHub Secrets）
echo ""
print_status "私鑰內容 (複製到 GitHub Secrets > SERVER_SSH_KEY):"
echo "=========================================="
cat ~/.ssh/github_actions_deploy
echo "=========================================="

# 顯示公鑰（需要添加到伺服器的 authorized_keys）
echo ""
print_status "公鑰內容 (添加到伺服器的 ~/.ssh/authorized_keys):"
echo "=========================================="
cat ~/.ssh/github_actions_deploy.pub
echo "=========================================="

# 設置權限
chmod 600 ~/.ssh/github_actions_deploy
chmod 644 ~/.ssh/github_actions_deploy.pub

print_status "SSH 金鑰設置完成！"
echo ""
print_warning "下一步操作："
echo "1. 複製私鑰內容到 GitHub Secrets > SERVER_SSH_KEY"
echo "2. 將公鑰添加到目標伺服器的 ~/.ssh/authorized_keys"
echo "3. 設置其他必要的 GitHub Secrets"
