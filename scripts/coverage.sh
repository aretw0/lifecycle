#!/bin/bash
# scripts/coverage.sh
# Runs test coverage and displays results per package.

COVERAGE_FILE="coverage.out"

echo "Running tests and generating coverage profile..."
go test -coverprofile="$COVERAGE_FILE" ./... > /dev/null 2>&1

if [ -f "$COVERAGE_FILE" ]; then
    echo "--------------------------------------------------"
    echo -e "\033[1;33mOVERALL SUMMARY\033[0m"
    go tool cover -func="$COVERAGE_FILE" | grep "total:"
    echo "--------------------------------------------------"
    echo -e "\033[1;33mCRITICAL PACKAGE COVERAGE\033[0m"
    # Mapping for new hierarchy
    declare -A pkgs=( ["signal"]="core/signal" ["runtime"]="core/runtime" ["worker"]="core/worker" ["supervisor"]="core/supervisor" ["events"]="events" ["metrics"]="core/metrics" ["termio"]="core/termio" )
    
    for name in "${!pkgs[@]}"; do
        path=${pkgs[$name]}
        VAL=$(go tool cover -func="$COVERAGE_FILE" | grep "$path" | awk '{sum+=$3; count++} END {if (count>0) print sum/count "%"; else print "N/A"}')
        printf "%-20s %s\n" "$name" "$VAL"
    done
    echo "--------------------------------------------------"
    rm "$COVERAGE_FILE"
else
    echo "Failed to generate coverage profile."
    exit 1
fi
