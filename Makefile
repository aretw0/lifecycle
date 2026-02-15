.PHONY: tidy vet test coverage serve-docs stress clean-zombies work-on-procio work-on-introspection work-off-procio work-off-introspection work-off-all

# Ensure dependencies are clean
tidy:
	go mod tidy

# Run vet tool in all files
vet:
	go vet ./...

# Run all tests
# Note: -race is mandatory for verifying behavioral logic and concurrency safety.
test:
	go test -race -timeout 90s ./...

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

# --- Dependency Management (Dev vs Prod) ---

# Switch specific dependencies to local version

# Enable local development mode for procio by creating/updating go.work
# Usage: make work-on-procio [WORK_PATH=../procio]
work-on-procio:
	@echo "Enabling local procio..."
	@if not exist go.work ( echo "Initializing go.work..." & go work init . )
	@if "$(WORK_PATH)"=="" ( go work use ../procio ) else ( go work use $(WORK_PATH) )

# Enable local development mode for introspection by creating/updating go.work
# Usage: make work-on-introspection [WORK_PATH=../introspection]
work-on-introspection:
	@echo "Enabling local introspection..."
	@if not exist go.work ( echo "Initializing go.work..." & go work init . )
	@if "$(WORK_PATH)"=="" ( go work use ../introspection ) else ( go work use $(WORK_PATH) )

# Disable local procio (remove from go.work)
# Usage: make work-off-procio [WORK_PATH=../procio]
work-off-procio:
	@echo "Disabling local procio..."
	@if exist go.work ( \
		if "$(WORK_PATH)"=="" ( go work edit -dropuse ../procio ) else ( go work edit -dropuse $(WORK_PATH) ) \
	)

# Disable local introspection (remove from go.work)
# Usage: make work-off-introspection [WORK_PATH=../introspection]
work-off-introspection:
	@echo "Disabling local introspection..."
	@if exist go.work ( \
		if "$(WORK_PATH)"=="" ( go work edit -dropuse ../introspection ) else ( go work edit -dropuse $(WORK_PATH) ) \
	)

# Disable local development mode by removing go.work (nuclear option)
work-off-all:
	@echo "Disabling local workspace mode..."
	@if exist go.work ( del go.work )
