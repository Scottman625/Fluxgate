# 簡化測試用戶進入隊列功能

$baseUrl = "http://localhost:8080"

Write-Host "🧪 測試用戶進入隊列功能..." -ForegroundColor Green

# 測試數據
$testData = @{
    activity_id = 1
    user_hash   = "user-$(Get-Random)"
    fingerprint = "test-fingerprint-$(Get-Random)"
}

Write-Host "📋 測試數據: $($testData | ConvertTo-Json)"

# 測試進入隊列
Write-Host "1. 測試進入隊列..."
try {
    $body = @{
        activity_id = $testData.activity_id
        user_hash   = $testData.user_hash
        fingerprint = $testData.fingerprint
    } | ConvertTo-Json

    $response = Invoke-WebRequest -Uri "$baseUrl/api/v1/queue/enter" -Method POST -Body $body -ContentType "application/json" -UseBasicParsing
    
    Write-Host "   ✅ 進入隊列成功 (Status: $($response.StatusCode))" -ForegroundColor Green
    Write-Host "   📄 回應內容: $($response.Content)"
}
catch {
    Write-Host "   ❌ 進入隊列失敗: $($_.Exception.Message)" -ForegroundColor Red
}

# 測試 Dashboard 數據
Write-Host "2. 測試 Dashboard 數據..."
try {
    $response = Invoke-WebRequest -Uri "$baseUrl/api/v1/dashboard" -Method GET -UseBasicParsing
    
    Write-Host "   ✅ Dashboard 數據成功 (Status: $($response.StatusCode))" -ForegroundColor Green
    $dashboardData = $response.Content | ConvertFrom-Json
    
    Write-Host "   📊 總隊列長度: $($dashboardData.overview.total_users_in_queue)"
    Write-Host "   📊 活躍活動: $($dashboardData.overview.active_activities)"
}
catch {
    Write-Host "   ❌ Dashboard 數據失敗: $($_.Exception.Message)" -ForegroundColor Red
}

Write-Host "🔗 Dashboard: $baseUrl"
