# 隊列系統 API 測試腳本

Write-Host "🚀 開始測試隊列系統 API..." -ForegroundColor Green

# 等待服務啟動
Start-Sleep -Seconds 2

# 1. 健康檢查
Write-Host "`n📋 1. 測試健康檢查..." -ForegroundColor Yellow
try {
    $response = Invoke-WebRequest -Uri "http://localhost:8080/health" -Method GET
    Write-Host "✅ 健康檢查成功: $($response.StatusCode)" -ForegroundColor Green
    Write-Host "回應內容: $($response.Content)" -ForegroundColor Gray
} catch {
    Write-Host "❌ 健康檢查失敗: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

# 2. 創建活動
Write-Host "`n📋 2. 測試創建活動..." -ForegroundColor Yellow
$activityData = @{
    tenant_id = "test_tenant"
    name = "測試活動"
    sku = "TEST001"
    initial_stock = 100
    start_at = "2024-01-01T10:00:00Z"
    end_at = "2024-01-01T18:00:00Z"
} | ConvertTo-Json

try {
    $response = Invoke-WebRequest -Uri "http://localhost:8080/api/v1/admin/activities" -Method POST -Body $activityData -ContentType "application/json"
    Write-Host "✅ 創建活動成功: $($response.StatusCode)" -ForegroundColor Green
    Write-Host "回應內容: $($response.Content)" -ForegroundColor Gray
    
    # 解析回應獲取活動 ID
    $result = $response.Content | ConvertFrom-Json
    $activityId = $result.data.id
    Write-Host "活動 ID: $activityId" -ForegroundColor Cyan
} catch {
    Write-Host "❌ 創建活動失敗: $($_.Exception.Message)" -ForegroundColor Red
    if ($_.Exception.Response) {
        $errorContent = $_.Exception.Response.GetResponseStream()
        $reader = New-Object System.IO.StreamReader($errorContent)
        $errorBody = $reader.ReadToEnd()
        Write-Host "錯誤詳情: $errorBody" -ForegroundColor Red
    }
}

# 3. 進入隊列
Write-Host "`n📋 3. 測試進入隊列..." -ForegroundColor Yellow
$queueData = @{
    activity_id = 1
    user_hash = "user123"
    fingerprint = "fp123"
} | ConvertTo-Json

try {
    $response = Invoke-WebRequest -Uri "http://localhost:8080/api/v1/queue/enter" -Method POST -Body $queueData -ContentType "application/json"
    Write-Host "✅ 進入隊列成功: $($response.StatusCode)" -ForegroundColor Green
    Write-Host "回應內容: $($response.Content)" -ForegroundColor Gray
    
    # 解析回應獲取 session_id
    $result = $response.Content | ConvertFrom-Json
    $sessionId = $result.data.session_id
    Write-Host "Session ID: $sessionId" -ForegroundColor Cyan
} catch {
    Write-Host "❌ 進入隊列失敗: $($_.Exception.Message)" -ForegroundColor Red
    if ($_.Exception.Response) {
        $errorContent = $_.Exception.Response.GetResponseStream()
        $reader = New-Object System.IO.StreamReader($errorContent)
        $errorBody = $reader.ReadToEnd()
        Write-Host "錯誤詳情: $errorBody" -ForegroundColor Red
    }
}

# 4. 查詢隊列狀態
Write-Host "`n📋 4. 測試查詢隊列狀態..." -ForegroundColor Yellow
try {
    $response = Invoke-WebRequest -Uri "http://localhost:8080/api/v1/queue/status?activity_id=1&session_id=session123" -Method GET
    Write-Host "✅ 查詢隊列狀態成功: $($response.StatusCode)" -ForegroundColor Green
    Write-Host "回應內容: $($response.Content)" -ForegroundColor Gray
} catch {
    Write-Host "❌ 查詢隊列狀態失敗: $($_.Exception.Message)" -ForegroundColor Red
}

# 5. 查詢活動狀態
Write-Host "`n📋 5. 測試查詢活動狀態..." -ForegroundColor Yellow
try {
    $response = Invoke-WebRequest -Uri "http://localhost:8080/api/v1/admin/activities/1/status" -Method GET
    Write-Host "✅ 查詢活動狀態成功: $($response.StatusCode)" -ForegroundColor Green
    Write-Host "回應內容: $($response.Content)" -ForegroundColor Gray
} catch {
    Write-Host "❌ 查詢活動狀態失敗: $($_.Exception.Message)" -ForegroundColor Red
}

Write-Host "`n🎉 API 測試完成！" -ForegroundColor Green
