# PowerShell 測試資料設定腳本

$headers = @{"Content-Type"="application/json"}
$baseUrl = "http://localhost:8080/api/v1/shifts/requirements"

Write-Host "=====================================" -ForegroundColor Cyan
Write-Host "   排班系統 - 測試資料設定工具" -ForegroundColor Cyan
Write-Host "=====================================" -ForegroundColor Cyan
Write-Host ""

# 設定今日 (2025-11-20) 班表
Write-Host "正在設定 2025-11-20 的班表需求..." -ForegroundColor Yellow

try {
    # 早班
    $response1 = Invoke-RestMethod -Uri $baseUrl -Method Post -Headers $headers `
      -Body '{"date":"2025-11-20","shift_type":"morning","required_count":3}'
    Write-Host "✅ 早班: 需要 3 人" -ForegroundColor Green

    # 中班
    $response2 = Invoke-RestMethod -Uri $baseUrl -Method Post -Headers $headers `
      -Body '{"date":"2025-11-20","shift_type":"afternoon","required_count":2}'
    Write-Host "✅ 中班: 需要 2 人" -ForegroundColor Green

    # 晚班
    $response3 = Invoke-RestMethod -Uri $baseUrl -Method Post -Headers $headers `
      -Body '{"date":"2025-11-20","shift_type":"evening","required_count":2}'
    Write-Host "✅ 晚班: 需要 2 人" -ForegroundColor Green

    Write-Host ""
    Write-Host "正在設定 2025-11-21 的班表需求..." -ForegroundColor Yellow

    # 明日早班
    $response4 = Invoke-RestMethod -Uri $baseUrl -Method Post -Headers $headers `
      -Body '{"date":"2025-11-21","shift_type":"morning","required_count":4}'
    Write-Host "✅ 早班: 需要 4 人" -ForegroundColor Green

    # 明日中班
    $response5 = Invoke-RestMethod -Uri $baseUrl -Method Post -Headers $headers `
      -Body '{"date":"2025-11-21","shift_type":"afternoon","required_count":3}'
    Write-Host "✅ 中班: 需要 3 人" -ForegroundColor Green

    # 明日晚班
    $response6 = Invoke-RestMethod -Uri $baseUrl -Method Post -Headers $headers `
      -Body '{"date":"2025-11-21","shift_type":"evening","required_count":2}'
    Write-Host "✅ 晚班: 需要 2 人" -ForegroundColor Green

    Write-Host ""
    Write-Host "=====================================" -ForegroundColor Cyan
    Write-Host "✅ 測試資料設定完成！" -ForegroundColor Green
    Write-Host "=====================================" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "請重新整理前端頁面查看班表資料：" -ForegroundColor Yellow
    Write-Host "http://localhost:3000/schedule" -ForegroundColor Cyan
    Write-Host ""
}
catch {
    Write-Host ""
    Write-Host "❌ 錯誤：無法設定測試資料" -ForegroundColor Red
    Write-Host "請確認後端服務是否運行在 http://localhost:8080" -ForegroundColor Yellow
    Write-Host ""
    Write-Host "錯誤訊息：" -ForegroundColor Red
    Write-Host $_.Exception.Message -ForegroundColor Red
    Write-Host ""
}
