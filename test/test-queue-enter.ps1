# 測試用戶進入隊列功能

$baseUrl = "http://localhost:8080"

Write-Host "🧪 測試用戶進入隊列功能..." -ForegroundColor Green
Write-Host ""

# 測試數據
$testData = @{
    tenant_id   = "test-tenant"
    activity_id = 1
    fingerprint = "test-fingerprint-$(Get-Random)"
    user_id     = "user-$(Get-Random)"
}

Write-Host "📋 測試數據:" -ForegroundColor Yellow
$testData | Format-Table

Write-Host ""

# 測試進入隊列
Write-Host "1. 測試進入隊列..." -ForegroundColor Yellow
try {
    $body = @{
        tenant_id   = $testData.tenant_id
        activity_id = $testData.activity_id
        fingerprint = $testData.fingerprint
        user_id     = $testData.user_id
    } | ConvertTo-Json

    $response = Invoke-WebRequest -Uri "$baseUrl/api/v1/queue/enter" -Method POST -Body $body -ContentType "application/json" -UseBasicParsing
    
    Write-Host "   ✅ 進入隊列成功 (Status: $($response.StatusCode))" -ForegroundColor Green
    Write-Host "   📄 回應內容:" -ForegroundColor Cyan
    $response.Content | ConvertFrom-Json | ConvertTo-Json -Depth 3
}
catch {
    Write-Host "   ❌ 進入隊列失敗: $($_.Exception.Message)" -ForegroundColor Red
    if ($_.Exception.Response) {
        $errorResponse = $_.Exception.Response.GetResponseStream()
        $reader = New-Object System.IO.StreamReader($errorResponse)
        $errorContent = $reader.ReadToEnd()
        Write-Host "   📄 錯誤詳情: $errorContent" -ForegroundColor Red
    }
}

Write-Host ""

# 測試查詢隊列狀態
Write-Host "2. 測試查詢隊列狀態..." -ForegroundColor Yellow
try {
    $queryParams = "tenant_id=$($testData.tenant_id)&activity_id=$($testData.activity_id)&fingerprint=$($testData.fingerprint)"
    $response = Invoke-WebRequest -Uri "$baseUrl/api/v1/queue/status?$queryParams" -Method GET -UseBasicParsing
    
    Write-Host "   ✅ 查詢隊列狀態成功 (Status: $($response.StatusCode))" -ForegroundColor Green
    Write-Host "   📄 回應內容:" -ForegroundColor Cyan
    $response.Content | ConvertFrom-Json | ConvertTo-Json -Depth 3
}
catch {
    Write-Host "   ❌ 查詢隊列狀態失敗: $($_.Exception.Message)" -ForegroundColor Red
}

Write-Host ""

# 測試 Dashboard 數據
Write-Host "3. 測試 Dashboard 數據..." -ForegroundColor Yellow
try {
    $response = Invoke-WebRequest -Uri "$baseUrl/api/v1/dashboard" -Method GET -UseBasicParsing
    
    Write-Host "   ✅ Dashboard 數據成功 (Status: $($response.StatusCode))" -ForegroundColor Green
    $dashboardData = $response.Content | ConvertFrom-Json
    
    Write-Host "   📊 總隊列長度: $($dashboardData.overview.total_users_in_queue)" -ForegroundColor Cyan
    Write-Host "   📊 活躍活動: $($dashboardData.overview.active_activities)" -ForegroundColor Cyan
    Write-Host "   📊 總釋放速率: $($dashboardData.overview.total_release_rate)" -ForegroundColor Cyan
}
catch {
    Write-Host "   ❌ Dashboard 數據失敗: $($_.Exception.Message)" -ForegroundColor Red
}

Write-Host ""
Write-Host "🔗 可用的服務連結：" -ForegroundColor Cyan
Write-Host "  - Dashboard:     $baseUrl" -ForegroundColor White
Write-Host "  - 測試頁面:      $baseUrl/web/examples/basic-usage.html" -ForegroundColor White
Write-Host "  - Prometheus:    http://localhost:9091" -ForegroundColor White
Write-Host "  - Grafana:       http://localhost:3000" -ForegroundColor White
