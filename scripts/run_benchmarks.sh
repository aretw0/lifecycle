#!/usr/bin/env bash
# run_benchmarks.sh
# Executes all lifecycle benchmarks and generates a summary report
# Usage: ./scripts/run_benchmarks.sh

set -euo pipefail

echo "==> Running lifecycle Benchmarks"
echo ""

# Configuration
BENCH_TIME="5s"
OUTPUT_DIR="benchmark_results"

# Create output directory if it doesn't exist
mkdir -p "$OUTPUT_DIR"

TIMESTAMP=$(date +%Y%m%d_%H%M%S)

# Run runtime benchmarks
echo "==> Benchmarking pkg/core/runtime..."
RUNTIME_OUTPUT="$OUTPUT_DIR/runtime_$TIMESTAMP.txt"
go test -bench=. -benchmem -benchtime="$BENCH_TIME" ./pkg/core/runtime/ | tee "$RUNTIME_OUTPUT"
echo ""

# Run router benchmarks
echo "==> Benchmarking pkg/events (router)..."
ROUTER_OUTPUT="$OUTPUT_DIR/router_$TIMESTAMP.txt"
go test -bench=. -benchmem -benchtime="$BENCH_TIME" ./pkg/events/ | tee "$ROUTER_OUTPUT"
echo ""

# Summary
echo "==> Benchmark Complete!"
echo ""
echo "Results saved to:"
echo "  - $RUNTIME_OUTPUT"
echo "  - $ROUTER_OUTPUT"
echo ""
echo "To compare with previous runs:"
echo "  benchstat $OUTPUT_DIR/runtime_PREVIOUS.txt $OUTPUT_DIR/runtime_$TIMESTAMP.txt"
echo ""
echo "To view detailed profiling:"
echo "  go test -bench=BenchmarkGoVsRawGoroutine -cpuprofile=cpu.prof ./pkg/core/runtime/"
echo "  go tool pprof cpu.prof"
echo ""
