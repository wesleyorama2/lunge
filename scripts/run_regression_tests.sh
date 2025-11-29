#!/bin/bash

# Performance Regression Test Runner
# This script runs performance benchmarks and compares them with baseline

set -e

BASELINE_FILE="benchmark_baseline.json"
RESULTS_FILE="benchmark_results.txt"
COMPARISON_FILE="benchmark_comparison.txt"

echo "========================================="
echo "Performance Regression Test Suite"
echo "========================================="
echo ""

# Check if baseline exists
if [ ! -f "$BASELINE_FILE" ]; then
    echo "⚠️  No baseline found at $BASELINE_FILE"
    echo "   Run 'go test -run=TestCreateBaseline -v' to create one"
    echo ""
fi

# Run efficiency regression tests
echo "Running efficiency regression tests..."
go test -v -run=TestRegressionEfficiency ./internal/performance/

if [ $? -ne 0 ]; then
    echo "❌ Efficiency regression tests FAILED"
    exit 1
fi

echo "✓ Efficiency regression tests PASSED"
echo ""

# Run performance benchmarks
echo "Running performance benchmarks..."
go test -bench=BenchmarkRegression -benchmem -benchtime=5s ./internal/performance/ | tee "$RESULTS_FILE"

if [ $? -ne 0 ]; then
    echo "❌ Performance benchmarks FAILED"
    exit 1
fi

echo "✓ Performance benchmarks completed"
echo ""

# Parse benchmark results and check thresholds
echo "Analyzing benchmark results..."

# Check for efficiency drops in benchmark output
if grep -q "REGRESSION" "$RESULTS_FILE"; then
    echo "❌ REGRESSION DETECTED in benchmark results"
    grep "REGRESSION" "$RESULTS_FILE"
    exit 1
fi

# Check for efficiency warnings
if grep -q "WARNING" "$RESULTS_FILE"; then
    echo "⚠️  Performance warnings detected:"
    grep "WARNING" "$RESULTS_FILE"
fi

echo "✓ No regressions detected"
echo ""

# Summary
echo "========================================="
echo "Regression Test Summary"
echo "========================================="
echo "✓ All efficiency tests passed (95%+ threshold)"
echo "✓ All benchmarks completed successfully"
echo "✓ No performance regressions detected"
echo ""
echo "Results saved to: $RESULTS_FILE"

exit 0
