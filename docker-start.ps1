# PowerShell 啟動腳本

Write-Host "🚀 啟動 Queue System 服務..." -ForegroundColor Green

# 停止並清理現有容器
Write-Host "清理現有容器..." -ForegroundColor Yellow
docker-compose down

# 建構並啟動所有服務
Write-Host "建構並啟動服務..." -ForegroundColor Yellow
docker-compose up --build -d

# 等待服務啟動
Write-Host "等待服務啟動..." -ForegroundColor Yellow
Start-Sleep -Seconds 20

# 檢查服務狀態
Write-Host "檢查服務狀態..." -ForegroundColor Yellow
docker-compose ps

Write-Host ""
Write-Host "✅ 服務已啟動！" -ForegroundColor Green
Write-Host ""
Write-Host "📊 可用的服務：" -ForegroundColor Cyan
Write-Host "  - Queue System API: http://localhost:8080" -ForegroundColor White
Write-Host "  - Queue Dashboard:   http://localhost:8080" -ForegroundColor White
Write-Host "  - Prometheus:        http://localhost:9091" -ForegroundColor White
Write-Host "  - Grafana:           http://localhost:3000 (admin/admin)" -ForegroundColor White
Write-Host ""
Write-Host "🔍 查看日誌:" -ForegroundColor Cyan
Write-Host "  docker-compose logs -f queue-server" -ForegroundColor White
Write-Host ""
Write-Host "🛑 停止服務:" -ForegroundColor Cyan
Write-Host "  docker-compose down" -ForegroundColor White
