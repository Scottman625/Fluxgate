# 測試 Docker 部署的 API

$baseUrl = "http://localhost:8080"

Write-Host "🧪 測試 Queue System API..." -ForegroundColor Green
Write-Host ""

# 測試健康檢查
Write-Host "1. 測試 Dashboard 端點..." -ForegroundColor Yellow
try {
    $response = Invoke-WebRequest -Uri "$baseUrl/api/v1/dashboard" -Method GET -UseBasicParsing
    Write-Host "   ✅ Dashboard 端點正常 (Status: $($response.StatusCode))" -ForegroundColor Green
}
catch {
    Write-Host "   ❌ Dashboard 端點失敗: $($_.Exception.Message)" -ForegroundColor Red
}

# 測試實時指標
Write-Host "2. 測試實時指標端點..." -ForegroundColor Yellow
try {
    $response = Invoke-WebRequest -Uri "$baseUrl/api/v1/dashboard/metrics/realtime" -Method GET -UseBasicParsing
    Write-Host "   ✅ 實時指標端點正常 (Status: $($response.StatusCode))" -ForegroundColor Green
}
catch {
    Write-Host "   ❌ 實時指標端點失敗: $($_.Exception.Message)" -ForegroundColor Red
}

# 測試 Prometheus 指標
Write-Host "3. 測試 Prometheus 指標..." -ForegroundColor Yellow
try {
    $response = Invoke-WebRequest -Uri "$baseUrl:9090/metrics" -Method GET -UseBasicParsing
    Write-Host "   ✅ Prometheus 指標正常 (Status: $($response.StatusCode))" -ForegroundColor Green
}
catch {
    Write-Host "   ❌ Prometheus 指標失敗: $($_.Exception.Message)" -ForegroundColor Red
}

# 測試主頁
Write-Host "4. 測試主頁..." -ForegroundColor Yellow
try {
    $response = Invoke-WebRequest -Uri "$baseUrl/" -Method GET -UseBasicParsing
    Write-Host "   ✅ 主頁正常 (Status: $($response.StatusCode))" -ForegroundColor Green
}
catch {
    Write-Host "   ❌ 主頁失敗: $($_.Exception.Message)" -ForegroundColor Red
}

Write-Host ""
Write-Host "🔗 可用的服務連結：" -ForegroundColor Cyan
Write-Host "  - Queue System:  $baseUrl" -ForegroundColor White
Write-Host "  - Prometheus:    http://localhost:9091" -ForegroundColor White
Write-Host "  - Grafana:       http://localhost:3000" -ForegroundColor White

Write-Host ""
Write-Host "📋 查看容器狀態：" -ForegroundColor Cyan
docker-compose ps
