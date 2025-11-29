# End-to-End Performance Test for Atomic Metrics Collector
# This script runs performance tests with both the old and new collectors
# and compares the results to verify the atomic collector achieves 200+ RPS

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Atomic Collector E2E Performance Test" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Check if local server is running
Write-Host "Checking if local server is running on http://localhost..." -ForegroundColor Yellow
try {
    $response = Invoke-WebRequest -Uri "http://localhost/health" -TimeoutSec 5 -ErrorAction Stop
    Write-Host "Local server is running!" -ForegroundColor Green
} catch {
    Write-Host "Local server is not running on http://localhost!" -ForegroundColor Red
    Write-Host "Please start your local test server first." -ForegroundColor Yellow
    exit 1
}
Write-Host ""

# Build the application
Write-Host "Building lunge..." -ForegroundColor Yellow
go build -o lunge.exe ./cmd/lunge
if ($LASTEXITCODE -ne 0) {
    Write-Host "Build failed!" -ForegroundColor Red
    exit 1
}
Write-Host "Build successful!" -ForegroundColor Green
Write-Host ""

# Test 1: Run with OLD collector (default)
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Test 1: OLD Collector (DefaultMetricsCollector)" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

$env:LUNGE_USE_ATOMIC_COLLECTOR = "false"
Write-Host "Running performance test with OLD collector..." -ForegroundColor Yellow
Write-Host "Target: 200 RPS, Expected: ~132 RPS (known bottleneck)" -ForegroundColor Yellow
Write-Host ""

$oldStart = Get-Date
.\lunge.exe perf --config examples/atomic-collector-e2e-test.json --performance atomicCollectorTest --environment local --output old-collector-test.html
$oldEnd = Get-Date
$oldDuration = ($oldEnd - $oldStart).TotalSeconds

if ($LASTEXITCODE -ne 0) {
    Write-Host "Old collector test failed!" -ForegroundColor Red
} else {
    Write-Host "Old collector test completed in $oldDuration seconds" -ForegroundColor Green
}
Write-Host ""

# Test 2: Run with NEW atomic collector
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Test 2: NEW Collector (AtomicMetricsCollector)" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

$env:LUNGE_USE_ATOMIC_COLLECTOR = "true"
Write-Host "Running performance test with ATOMIC collector..." -ForegroundColor Yellow
Write-Host "Target: 200 RPS, Expected: 200+ RPS (fixed bottleneck)" -ForegroundColor Yellow
Write-Host ""

$newStart = Get-Date
.\lunge.exe perf --config examples/atomic-collector-e2e-test.json --performance atomicCollectorTest --environment local --output atomic-collector-test.html
$newEnd = Get-Date
$newDuration = ($newEnd - $newStart).TotalSeconds

if ($LASTEXITCODE -ne 0) {
    Write-Host "Atomic collector test failed!" -ForegroundColor Red
    exit 1
} else {
    Write-Host "Atomic collector test completed in $newDuration seconds" -ForegroundColor Green
}
Write-Host ""

# Summary
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Test Summary" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "Old Collector Test Duration: $oldDuration seconds" -ForegroundColor White
Write-Host "New Collector Test Duration: $newDuration seconds" -ForegroundColor White
Write-Host ""
Write-Host "Reports generated:" -ForegroundColor Yellow
Write-Host "  - old-collector-test.html (OLD collector)" -ForegroundColor White
Write-Host "  - atomic-collector-test.html (NEW collector)" -ForegroundColor White
Write-Host ""
Write-Host "Next steps:" -ForegroundColor Yellow
Write-Host "  1. Open both HTML reports in a browser" -ForegroundColor White
Write-Host "  2. Compare actual RPS achieved" -ForegroundColor White
Write-Host "  3. Verify atomic collector achieves 200+ RPS" -ForegroundColor White
Write-Host "  4. Compare CPU usage and efficiency metrics" -ForegroundColor White
Write-Host ""

# Clean up environment variable
$env:LUNGE_USE_ATOMIC_COLLECTOR = ""

Write-Host "E2E test completed!" -ForegroundColor Green
Write-Host ""

# Run verification
Write-Host "Running verification..." -ForegroundColor Yellow
.\scripts\verify-e2e-results.ps1 -OldCollectorReport "old-collector-test.html" -NewCollectorReport "atomic-collector-test.html"
