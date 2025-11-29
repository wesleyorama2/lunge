# Monitor the stress test progress
# This script checks if the test is running and shows progress

Write-Host "Stress Test Monitor" -ForegroundColor Cyan
Write-Host "==================" -ForegroundColor Cyan
Write-Host ""

# Check if lunge process is running
$lungeProcess = Get-Process -Name "lunge" -ErrorAction SilentlyContinue

if ($lungeProcess) {
    Write-Host "Stress test is running" -ForegroundColor Green
    Write-Host ""
    Write-Host "Process Details:" -ForegroundColor Yellow
    Write-Host "  PID: $($lungeProcess.Id)" -ForegroundColor White
    Write-Host "  CPU: $([math]::Round($lungeProcess.CPU, 2))s" -ForegroundColor White
    Write-Host "  Memory: $([math]::Round($lungeProcess.WorkingSet64 / 1MB, 2)) MB" -ForegroundColor White
    Write-Host "  Threads: $($lungeProcess.Threads.Count)" -ForegroundColor White
    Write-Host ""
    
    # Check if output file exists
    if (Test-Path "atomic-collector-stress-test.html") {
        $fileInfo = Get-Item "atomic-collector-stress-test.html"
        Write-Host "Report file exists (last modified: $($fileInfo.LastWriteTime))" -ForegroundColor Green
    } else {
        Write-Host "Report file not yet generated" -ForegroundColor Yellow
    }
    
    Write-Host ""
    Write-Host "Test Configuration:" -ForegroundColor Yellow
    Write-Host "  Target RPS: 1,000" -ForegroundColor White
    Write-Host "  Concurrency: 1,000 workers" -ForegroundColor White
    Write-Host "  Duration: 5 minutes" -ForegroundColor White
    Write-Host "  Total time: ~6 minutes (with warmup and ramp)" -ForegroundColor White
    Write-Host ""
    Write-Host "The test is running in the background." -ForegroundColor Cyan
    Write-Host "Check back in a few minutes for results." -ForegroundColor Cyan
    
} else {
    Write-Host "No stress test is currently running" -ForegroundColor Red
    Write-Host ""
    
    # Check if report exists
    if (Test-Path "atomic-collector-stress-test.html") {
        Write-Host "Report file found - test may have completed" -ForegroundColor Green
        Write-Host ""
        Write-Host "Open atomic-collector-stress-test.html to view results" -ForegroundColor Cyan
    } else {
        Write-Host "No report file found - test has not been run yet" -ForegroundColor Yellow
    }
}

Write-Host ""
