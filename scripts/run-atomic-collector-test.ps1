# Quick test script for Atomic Metrics Collector
# Runs a single performance test with the atomic collector enabled

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Atomic Collector Performance Test" -ForegroundColor Cyan
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

# Enable atomic collector
$env:LUNGE_USE_ATOMIC_COLLECTOR = "true"

Write-Host "Running performance test with ATOMIC collector..." -ForegroundColor Yellow
Write-Host "Target: 200 RPS" -ForegroundColor Yellow
Write-Host "Expected: 200+ RPS (vs ~132 RPS with old collector)" -ForegroundColor Yellow
Write-Host ""

# Run the test
.\lunge.exe perf --config examples/atomic-collector-e2e-test.json --performance atomicCollectorTest --environment local --output atomic-collector-e2e-test.html

if ($LASTEXITCODE -ne 0) {
    Write-Host ""
    Write-Host "Test failed!" -ForegroundColor Red
    $env:LUNGE_USE_ATOMIC_COLLECTOR = ""
    exit 1
}

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Test Completed Successfully!" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "Report generated: atomic-collector-e2e-test.html" -ForegroundColor Yellow
Write-Host ""
Write-Host "Open the HTML report to verify:" -ForegroundColor Yellow
Write-Host "  ✓ Actual RPS achieves 200+ (vs 132 with old collector)" -ForegroundColor White
Write-Host "  ✓ Efficiency metrics show >95% efficiency" -ForegroundColor White
Write-Host "  ✓ Low CPU overhead (<5%)" -ForegroundColor White
Write-Host "  ✓ Accurate metrics (within 1%)" -ForegroundColor White
Write-Host ""

# Clean up environment variable
$env:LUNGE_USE_ATOMIC_COLLECTOR = ""
