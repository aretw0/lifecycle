.PHONY: test vet coverage serve-docs stress clean-zombies

# Run all tests
# Note: -race is mandatory for verifying behavioral logic and concurrency safety.
test:
	go test -race -timeout 90s ./...

# Run vet tool in all files
vet:
	go vet ./...

# Run coverage tests (powershell on Windows needs double quotes for file paths)
coverage:
	go test -race -timeout 90s -coverprofile="coverage.out" ./...
	go tool cover -func="coverage.out"

# Run local Go documentation server (pkgsite)
serve-docs:
	go tool godoc -http=:6060

# Run stress tests
stress:
	go test -race -v -tags=stress -count=1 -timeout 2m ./pkg/worker/...

# Cleanup leaked processes during development (OS specific)
# Note: On Windows, we use powershell to find and kill processes with 'lifecycle' in the name.
clean-zombies:
ifeq ($(OS),Windows_NT)
	powershell -Command "Get-Process | Where-Object { $$_.ProcessName -match 'lifecycle' } | Stop-Process -Force"
else
	pkill -f lifecycle || true
endif

# Enable local development mode by creating a go.work file
# Usage: make work-on [WORK_PATH=../procio]
work-on:
	@echo "Enabling local workspace mode..."
	@if exist go.work ( echo "go.work already exists." ) else ( \
		echo "Initializing go.work..." & \
		go work init . & \
		if "$(WORK_PATH)"=="" ( go work use ../procio ) else ( go work use $(WORK_PATH) ) \
	)

# Disable local development mode by removing go.work
work-off:
	@echo "Disabling local workspace mode..."
	@if exist go.work ( del go.work )

# Ensure dependencies are clean
tidy:
	go mod tidy
