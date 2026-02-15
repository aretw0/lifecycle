.PHONY: tidy vet test coverage serve-docs stress clean-zombies work-on-procio work-on-introspection work-off-procio work-off-introspection work-off-all

# --- OS Detection & Command Abstraction ---
ifeq ($(OS),Windows_NT)
BINARY := trellis.exe
RM := del /F /Q
CURL := curl.exe
# Windows needs backslashes for 'go work edit -dropuse' to match go.work content
DROP_WORK = if exist go.work ( go work edit -dropuse $(subst /,\,$(1)) )
INIT_WORK = if not exist go.work ( echo "Initializing go.work..." & go work init . )
else
BINARY := trellis
RM := rm -f
CURL := curl
# Linux/macOS uses forward slashes
DROP_WORK = [ -f go.work ] && go work edit -dropuse $(1)
INIT_WORK = [ -f go.work ] || ( echo "Initializing go.work..." && go work init . )
endif


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

# Helper to get the correct path (uses WORK_PATH if provided, else default)
GET_PATH = $(if $(WORK_PATH),$(WORK_PATH),$(1))

# Enable local development mode for procio
# Usage: make work-on-procio [WORK_PATH=../procio]
work-on-procio:
	@echo "Enabling local procio..."
	@$(INIT_WORK)
	go work use $(call GET_PATH,../procio)

# Enable local development mode for introspection
# Usage: make work-on-introspection [WORK_PATH=../introspection]
work-on-introspection:
	@echo "Enabling local introspection..."
	@$(INIT_WORK)
	go work use $(call GET_PATH,../introspection)

# Disable local procio
# Usage: make work-off-procio [WORK_PATH=../procio]
work-off-procio:
	@echo "Disabling local procio..."
	@$(call DROP_WORK,$(call GET_PATH,../procio))

# Disable local introspection
# Usage: make work-off-introspection [WORK_PATH=../introspection]
work-off-introspection:
	@echo "Disabling local introspection..."
	@$(call DROP_WORK,$(call GET_PATH,../introspection))

# Disable local development mode by removing go.work (nuclear option)
work-off-all:
	@echo "Disabling local workspace mode..."
	@$(RM) go.work
