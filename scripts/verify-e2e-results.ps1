# Verification script for E2E test results
# Parses HTML reports and verifies that atomic collector achieves 200+ RPS

param(
    [string]$OldCollectorReport = "old-collector-test.html",
    [string]$NewCollectorReport = "atomic-collector-test.html"
)

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "E2E Test Results Verification" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

function Extract-RPS {
    param([string]$FilePath)
    
    if (-not (Test-Path $FilePath)) {
        return $null
    }
    
    $content = Get-Content $FilePath -Raw
    
    # Try to extract Average RPS from HTML
    if ($content -match 'Average RPS[:\s]+([0-9.]+)') {
        return [double]$matches[1]
    }
    
    # Try alternative patterns
    if ($content -match 'averageRPS["\s:]+([0-9.]+)') {
        return [double]$matches[1]
    }
    
    if ($content -match 'Throughput[:\s]+([0-9.]+)\s+req/s') {
        return [double]$matches[1]
    }
    
    return $null
}

function Extract-Efficiency {
    param([string]$FilePath)
    
    if (-not (Test-Path $FilePath)) {
        return $null
    }
    
    $content = Get-Content $FilePath -Raw
    
    # Try to extract Efficiency percentage
    if ($content -match 'Efficiency[:\s]+([0-9.]+)%') {
        return [double]$matches[1]
    }
    
    if ($content -match 'efficiency["\s:]+([0-9.]+)') {
        return [double]$matches[1] * 100
    }
    
    return $null
}

# Verify old collector report
if (Test-Path $OldCollectorReport) {
    Write-Host "Old Collector Results:" -ForegroundColor Yellow
    $oldRPS = Extract-RPS $OldCollectorReport
    
    if ($oldRPS) {
        Write-Host "  Average RPS: $oldRPS" -ForegroundColor White
        
        if ($oldRPS -lt 150) {
            Write-Host "  ✓ Confirmed bottleneck (< 150 RPS)" -ForegroundColor Green
        } else {
            Write-Host "  ⚠ Expected lower RPS due to bottleneck" -ForegroundColor Yellow
        }
    } else {
        Write-Host "  ⚠ Could not extract RPS from report" -ForegroundColor Yellow
    }
    Write-Host ""
} else {
    Write-Host "Old collector report not found: $OldCollectorReport" -ForegroundColor Yellow
    Write-Host ""
}

# Verify new collector report
if (Test-Path $NewCollectorReport) {
    Write-Host "Atomic Collector Results:" -ForegroundColor Yellow
    $newRPS = Extract-RPS $NewCollectorReport
    $efficiency = Extract-Efficiency $NewCollectorReport
    
    $passed = $true
    
    if ($newRPS) {
        Write-Host "  Average RPS: $newRPS" -ForegroundColor White
        
        if ($newRPS -ge 200) {
            Write-Host "  ✓ Achieved target 200+ RPS" -ForegroundColor Green
        } elseif ($newRPS -ge 190) {
            Write-Host "  ⚠ Close to target (190-200 RPS)" -ForegroundColor Yellow
            $passed = $false
        } else {
            Write-Host "  ✗ Did not achieve target 200 RPS" -ForegroundColor Red
            $passed = $false
        }
    } else {
        Write-Host "  ⚠ Could not extract RPS from report" -ForegroundColor Yellow
        $passed = $false
    }
    
    if ($efficiency) {
        Write-Host "  Efficiency: $efficiency%" -ForegroundColor White
        
        if ($efficiency -ge 95) {
            Write-Host "  ✓ High efficiency (>= 95%)" -ForegroundColor Green
        } elseif ($efficiency -ge 90) {
            Write-Host "  ⚠ Good efficiency (90-95%)" -ForegroundColor Yellow
        } else {
            Write-Host "  ✗ Low efficiency (< 90%)" -ForegroundColor Red
            $passed = $false
        }
    }
    
    Write-Host ""
    
    # Compare with old collector
    if ($oldRPS -and $newRPS) {
        $improvement = (($newRPS - $oldRPS) / $oldRPS) * 100
        Write-Host "Performance Improvement:" -ForegroundColor Yellow
        Write-Host "  Old: $oldRPS RPS" -ForegroundColor White
        Write-Host "  New: $newRPS RPS" -ForegroundColor White
        Write-Host "  Improvement: $([math]::Round($improvement, 1))%" -ForegroundColor $(if ($improvement -gt 30) { "Green" } else { "Yellow" })
        Write-Host ""
    }
    
    if ($passed) {
        Write-Host "========================================" -ForegroundColor Green
        Write-Host "✓ E2E Test PASSED" -ForegroundColor Green
        Write-Host "========================================" -ForegroundColor Green
        exit 0
    } else {
        Write-Host "========================================" -ForegroundColor Red
        Write-Host "✗ E2E Test FAILED" -ForegroundColor Red
        Write-Host "========================================" -ForegroundColor Red
        exit 1
    }
} else {
    Write-Host "Atomic collector report not found: $NewCollectorReport" -ForegroundColor Red
    Write-Host ""
    exit 1
}
