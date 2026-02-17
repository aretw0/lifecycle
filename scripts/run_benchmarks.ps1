#!/usr/bin/env pwsh
# run_benchmarks.ps1
# Executes all lifecycle benchmarks and generates a summary report
# Usage: .\scripts\run_benchmarks.ps1

$ErrorActionPreference = "Stop"

Write-Host "==> Running lifecycle Benchmarks" -ForegroundColor Cyan
Write-Host ""

# Configuration
$BenchTime = "5s"
$OutputDir = "benchmark_results"

# Create output directory if it doesn't exist
if (-not (Test-Path $OutputDir)) {
    New-Item -ItemType Directory -Path $OutputDir | Out-Null
}

$Timestamp = Get-Date -Format "yyyyMMdd_HHmmss"

# Run runtime benchmarks
Write-Host "==> Benchmarking pkg/core/runtime..." -ForegroundColor Yellow
$RuntimeOutput = "$OutputDir/runtime_$Timestamp.txt"
go test -bench=. -benchmem -benchtime=$BenchTime ./pkg/core/runtime/ | Tee-Object -FilePath $RuntimeOutput
Write-Host ""

# Run router benchmarks
Write-Host "==> Benchmarking pkg/events (router)..." -ForegroundColor Yellow
$RouterOutput = "$OutputDir/router_$Timestamp.txt"
go test -bench=. -benchmem -benchtime=$BenchTime ./pkg/events/ | Tee-Object -FilePath $RouterOutput
Write-Host ""

# Summary
Write-Host "==> Benchmark Complete!" -ForegroundColor Green
Write-Host ""
Write-Host "Results saved to:" -ForegroundColor Cyan
Write-Host "  - $RuntimeOutput"
Write-Host "  - $RouterOutput"
Write-Host ""
Write-Host "To compare with previous runs:" -ForegroundColor Yellow
Write-Host "  benchstat $OutputDir/runtime_PREVIOUS.txt $OutputDir/runtime_$Timestamp.txt"
Write-Host ""
Write-Host "To view detailed profiling:" -ForegroundColor Yellow
Write-Host "  go test -bench=BenchmarkGoVsRawGoroutine -cpuprofile=cpu.prof ./pkg/core/runtime/"
Write-Host "  go tool pprof cpu.prof"
Write-Host ""
