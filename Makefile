.PHONY: test vet coverage serve-docs stress clean-zombies

# Run all tests
test:
	go test ./...

# Run vet tool in all files
vet:
	go vet ./...

# Run coverage tests (powershell on Windows needs double quotes for file paths)
coverage:
	go test -coverprofile="coverage.out" ./...
	go tool cover -func="coverage.out"

# Run local Go documentation server (pkgsite)
serve-docs:
	go tool godoc -http=:6060

# Run stress tests
stress:
	go test -v -tags=stress -count=1 ./pkg/worker/...

# Cleanup leaked processes during development (OS specific)
# Note: On Windows, we use powershell to find and kill processes with 'lifecycle' in the name.
clean-zombies:
ifeq ($(OS),Windows_NT)
	powershell -Command "Get-Process | Where-Object { $$_.ProcessName -match 'lifecycle' } | Stop-Process -Force"
else
	pkill -f lifecycle || true
endif
