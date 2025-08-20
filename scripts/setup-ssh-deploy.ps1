# SSH 部署設置腳本 (PowerShell 版本)
# 用於設置 GitHub Actions 的 SSH 部署

Write-Host "🔧 設置 SSH 部署環境..." -ForegroundColor Green

# 函數定義
function Write-Status {
    param([string]$Message)
    Write-Host "✅ $Message" -ForegroundColor Green
}

function Write-Warning {
    param([string]$Message)
    Write-Host "⚠️  $Message" -ForegroundColor Yellow
}

function Write-Error {
    param([string]$Message)
    Write-Host "❌ $Message" -ForegroundColor Red
}

# 檢查是否已存在 SSH 金鑰
$privateKeyPath = "$env:USERPROFILE\.ssh\github_actions_deploy"
$publicKeyPath = "$env:USERPROFILE\.ssh\github_actions_deploy.pub"

if (Test-Path $privateKeyPath) {
    Write-Warning "SSH 金鑰已存在，跳過生成步驟"
}
else {
    Write-Status "生成 SSH 金鑰對..."
    ssh-keygen -t rsa -b 4096 -C "github-actions-deploy" -f $privateKeyPath -N '""'
}

# 顯示私鑰（用於 GitHub Secrets）
Write-Host ""
Write-Status "私鑰內容 (複製到 GitHub Secrets > SERVER_SSH_KEY):"
Write-Host "==========================================" -ForegroundColor Gray
Get-Content $privateKeyPath
Write-Host "==========================================" -ForegroundColor Gray

# 顯示公鑰（需要添加到伺服器的 authorized_keys）
Write-Host ""
Write-Status "公鑰內容 (添加到伺服器的 ~/.ssh/authorized_keys):"
Write-Host "==========================================" -ForegroundColor Gray
Get-Content $publicKeyPath
Write-Host "==========================================" -ForegroundColor Gray

Write-Status "SSH 金鑰設置完成！"
Write-Host ""
Write-Warning "下一步操作："
Write-Host "1. 複製私鑰內容到 GitHub Secrets > SERVER_SSH_KEY"
Write-Host "2. 將公鑰添加到目標伺服器的 ~/.ssh/authorized_keys"
Write-Host "3. 設置其他必要的 GitHub Secrets"
