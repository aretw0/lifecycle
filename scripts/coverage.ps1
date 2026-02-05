# scripts/coverage.ps1
# Runs test coverage and displays results per package (PowerShell version).

$CoverageFile = "./coverage.out"

Write-Host "`n[Lifecycle] Running tests and generating coverage profile..." -ForegroundColor Cyan
go test -coverprofile="$CoverageFile" ./... | Out-Null

if (Test-Path $CoverageFile) {
    Write-Host "--------------------------------------------------" -ForegroundColor Gray
    Write-Host "OVERALL SUMMARY" -ForegroundColor Yellow
    go tool cover -func="$CoverageFile" | Select-String "total:" | ForEach-Object {
        Write-Host $_ -ForegroundColor White
    }
    
    Write-Host "--------------------------------------------------" -ForegroundColor Gray
    Write-Host "PACKAGE BREAKDOWN" -ForegroundColor Yellow
    
    # Get all functions, then group by the file/package path
    $Results = go tool cover -func="$CoverageFile"
    
    # Show summary for critical packages
    $CriticalPackages = @("signal", "runtime", "control", "supervisor", "metrics", "termio")
    foreach ($pkg in $CriticalPackages) {
        $pkgLines = $Results | Select-String "pkg/$pkg"
        if ($pkgLines) {
            $total = 0
            $count = 0
            foreach ($line in $pkgLines) {
                if ($line -match "(\d+\.\d+)%") {
                    $total += [float]$matches[1]
                    $count++
                }
            }
            $avg = if ($count -gt 0) { $total / $count } else { 0 }
            Write-Host "$($pkg.PadRight(20)) $(" {0:N1}%" -f $avg)"
        }
        else {
            Write-Host "$($pkg.PadRight(20)) N/A"
        }
    }
    Write-Host "... (test results truncated for brevity) ..."
    
    Write-Host "--------------------------------------------------" -ForegroundColor Gray
    Remove-Item $CoverageFile
}
else {
    Write-Error "Failed to generate coverage profile."
    exit 1
}
