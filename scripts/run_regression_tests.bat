@echo off
REM Performance Regression Test Runner for Windows
REM This script runs performance benchmarks and compares them with baseline

setlocal enabledelayedexpansion

set BASELINE_FILE=benchmark_baseline.json
set RESULTS_FILE=benchmark_results.txt
set COMPARISON_FILE=benchmark_comparison.txt

echo =========================================
echo Performance Regression Test Suite
echo =========================================
echo.

REM Check if baseline exists
if not exist "%BASELINE_FILE%" (
    echo WARNING: No baseline found at %BASELINE_FILE%
    echo    Run 'go test -run=TestCreateBaseline -v' to create one
    echo.
)

REM Run efficiency regression tests
echo Running efficiency regression tests...
go test -v -run=TestRegressionEfficiency ./internal/performance/

if errorlevel 1 (
    echo FAILED: Efficiency regression tests FAILED
    exit /b 1
)

echo PASSED: Efficiency regression tests PASSED
echo.

REM Run performance benchmarks
echo Running performance benchmarks...
go test -bench=BenchmarkRegression -benchmem -benchtime=5s ./internal/performance/ > "%RESULTS_FILE%" 2>&1

if errorlevel 1 (
    echo FAILED: Performance benchmarks FAILED
    type "%RESULTS_FILE%"
    exit /b 1
)

type "%RESULTS_FILE%"
echo PASSED: Performance benchmarks completed
echo.

REM Parse benchmark results and check thresholds
echo Analyzing benchmark results...

REM Check for efficiency drops in benchmark output
findstr /C:"REGRESSION" "%RESULTS_FILE%" >nul
if not errorlevel 1 (
    echo FAILED: REGRESSION DETECTED in benchmark results
    findstr /C:"REGRESSION" "%RESULTS_FILE%"
    exit /b 1
)

REM Check for efficiency warnings
findstr /C:"WARNING" "%RESULTS_FILE%" >nul
if not errorlevel 1 (
    echo WARNING: Performance warnings detected:
    findstr /C:"WARNING" "%RESULTS_FILE%"
)

echo PASSED: No regressions detected
echo.

REM Summary
echo =========================================
echo Regression Test Summary
echo =========================================
echo PASSED: All efficiency tests passed (95%% threshold)
echo PASSED: All benchmarks completed successfully
echo PASSED: No performance regressions detected
echo.
echo Results saved to: %RESULTS_FILE%

exit /b 0
