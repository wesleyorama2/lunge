# Atomic Collector Stress Test Runner
# This script runs the stress test with monitoring for memory, goroutines, and panics

param(
    [switch]$WithPprof = $false,
    [string]$Duration = "5m",
    [int]$MonitorInterval = 10
)

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Atomic Collector Stress Test" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Check if test server is running
Write-Host "Checking if test server is running on localhost:80..." -ForegroundColor Yellow
try {
    $response = Invoke-WebRequest -Uri "http://localhost/health" -TimeoutSec 2 -ErrorAction Stop
    Write-Host "✓ Test server is running" -ForegroundColor Green
} catch {
    Write-Host "✗ Test server is not running!" -ForegroundColor Red
    Write-Host ""
    Write-Host "Please start the test server first:" -ForegroundColor Yellow
    Write-Host "  go run scripts/test-server.go" -ForegroundColor White
    Write-Host ""
    exit 1
}

Write-Host ""

# Build lunge if needed
if (-not (Test-Path "lunge.exe")) {
    Write-Host "Building lunge..." -ForegroundColor Yellow
    go build -o lunge.exe ./cmd/lunge
    if ($LASTEXITCODE -ne 0) {
        Write-Host "✗ Build failed!" -ForegroundColor Red
        exit 1
    }
    Write-Host "✓ Build successful" -ForegroundColor Green
} else {
    Write-Host "✓ Using existing lunge.exe" -ForegroundColor Green
}

Write-Host ""

# Set environment variable to use atomic collector
$env:LUNGE_USE_ATOMIC_COLLECTOR = "true"
Write-Host "✓ Atomic collector enabled (LUNGE_USE_ATOMIC_COLLECTOR=true)" -ForegroundColor Green

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Starting Stress Test" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Configuration:" -ForegroundColor Yellow
Write-Host "  - Target RPS: 10,000" -ForegroundColor White
Write-Host "  - Concurrency: 1,000 workers" -ForegroundColor White
Write-Host "  - Duration: $Duration" -ForegroundColor White
Write-Host "  - Warmup: 10s @ 1,000 RPS" -ForegroundColor White
Write-Host "  - Ramp-up: 30s" -ForegroundColor White
Write-Host "  - Ramp-down: 30s" -ForegroundColor White
Write-Host ""

# Run the stress test
$startTime = Get-Date
Write-Host "Test started at: $startTime" -ForegroundColor Cyan
Write-Host ""

if ($WithPprof) {
    Write-Host "Running with pprof profiling enabled..." -ForegroundColor Yellow
    Write-Host "Building profiling wrapper..." -ForegroundColor Yellow
    
    go build -o stress-test-profiler.exe scripts/stress-test-with-profiling.go
    if ($LASTEXITCODE -ne 0) {
        Write-Host "✗ Failed to build profiler!" -ForegroundColor Red
        exit 1
    }
    
    Write-Host "✓ Profiler built successfully" -ForegroundColor Green
    Write-Host ""
    Write-Host "Profiles will be saved to:" -ForegroundColor White
    Write-Host "  - CPU: stress-cpu.prof" -ForegroundColor White
    Write-Host "  - Memory: stress-mem.prof" -ForegroundColor White
    Write-Host "  - Goroutines: stress-goroutine.prof" -ForegroundColor White
    Write-Host ""
    
    .\stress-test-profiler.exe `
        -cpuprofile=stress-cpu.prof `
        -memprofile=stress-mem.prof `
        -goroutineprofile=stress-goroutine.prof `
        -monitor-interval="${MonitorInterval}s"
} else {
    Write-Host "Running without profiling (use -WithPprof to enable)..." -ForegroundColor Yellow
    Write-Host ""
    .\lunge.exe perf examples/atomic-collector-stress-test.json
}

}

$exitCode = $LASTEXITCODE
$endTime = Get-Date
$elapsed = $endTime - $startTime

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Test Completed" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "End time: $endTime" -ForegroundColor White
Write-Host "Total elapsed: $($elapsed.ToString())" -ForegroundColor White
Write-Host ""

if ($exitCode -eq 0) {
    Write-Host "✓ Test completed successfully!" -ForegroundColor Green
    Write-Host ""
    Write-Host "Results saved to: atomic-collector-stress-test.html" -ForegroundColor Cyan
    Write-Host ""
    
    if ($WithPprof) {
        Write-Host "Profile Analysis:" -ForegroundColor Yellow
        Write-Host "  To analyze CPU profile:" -ForegroundColor White
        Write-Host "    go tool pprof stress-cpu.prof" -ForegroundColor Gray
        Write-Host "  To analyze memory profile:" -ForegroundColor White
        Write-Host "    go tool pprof stress-mem.prof" -ForegroundColor Gray
        Write-Host "  To analyze goroutines:" -ForegroundColor White
        Write-Host "    go tool pprof stress-goroutine.prof" -ForegroundColor Gray
        Write-Host ""
    }
    
    Write-Host "Next steps:" -ForegroundColor Yellow
    Write-Host "  1. Open atomic-collector-stress-test.html to view results" -ForegroundColor White
    Write-Host "  2. Check for any panics or errors in the output above" -ForegroundColor White
    Write-Host "  3. Verify memory usage remained stable" -ForegroundColor White
    Write-Host "  4. Confirm no goroutine leaks occurred" -ForegroundColor White
} else {
    Write-Host "✗ Test failed with exit code: $exitCode" -ForegroundColor Red
    Write-Host ""
    Write-Host "Check the output above for errors or panics" -ForegroundColor Yellow
}

Write-Host ""
