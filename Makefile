.PHONY: tidy vet lint test coverage \ 
	serve-docs stress clean-zombies \
	work-on-procio work-on-introspection work-off-procio work-off-introspection work-off-all

# Go parameters
GOCMD=go
GOVET=$(GOCMD) vet
GORUN=$(GOCMD) run
GOMOD=$(GOCMD) mod
GOWORK=$(GOCMD) work
GOTEST=$(GOCMD) test
GOTOOL=$(GOCMD) tool

# --- OS Detection & Command Abstraction ---
ifeq ($(OS),Windows_NT)
RM := del /F /Q
# Windows needs backslashes for 'go work edit -dropuse' to match go.work content
DROP_WORK = if exist go.work ( $(GOWORK) edit -dropuse $(subst /,\,$(1)) )
INIT_WORK = if not exist go.work ( echo "Initializing go.work..." & $(GOWORK) init . )
else
RM := rm -f
# Linux/macOS uses forward slashes
DROP_WORK = [ -f go.work ] && $(GOWORK) edit -dropuse $(1)
INIT_WORK = [ -f go.work ] || ( echo "Initializing go.work..." && $(GOWORK) init . )
endif


# Ensure dependencies are clean
tidy:
	@echo "Tidying Go modules..."
	$(GOMOD) tidy

# Run vet tool in all files
vet:
	@echo "Running go vet..."
	$(GOVET) ./...

# Run ineffassign to detect ineffectual assignments
lint:
	$(GORUN) github.com/gordonklaus/ineffassign@latest ./...

# Run all tests
# Note: -race is mandatory for verifying behavioral logic and concurrency safety.
test:
	$(GOTEST) -race -timeout 90s ./...

# Run coverage tests (powershell on Windows needs double quotes for file paths)
coverage:
	$(GOTEST) -race -timeout 90s -coverprofile="coverage.out" ./...
	$(GOTOOL) cover -func="coverage.out"

# Run benchmarks (performance measurements)
# Note: Some benchmarks intentionally trigger panics (stack capture tests) and generate error logs.
# This is expected behavior and does not affect benchmark accuracy.
benchmark:
	@echo "==> Running lifecycle benchmarks..."
	$(GOTEST) -bench=. -benchmem -benchtime=5s ./pkg/core/runtime/
	$(GOTEST) -bench=. -benchmem -benchtime=5s ./pkg/events/

# Run local Go documentation server (pkgsite)
serve-docs:
	$(GOTOOL) godoc -http=:6060

# Run stress tests
stress:
	$(GOTEST) -race -v -tags=stress -count=1 -timeout 2m ./pkg/worker/...

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
	$(GOWORK) use $(call GET_PATH,../procio)

# Enable local development mode for introspection
# Usage: make work-on-introspection [WORK_PATH=../introspection]
work-on-introspection:
	@echo "Enabling local introspection..."
	@$(INIT_WORK)
	$(GOWORK) use $(call GET_PATH,../introspection)

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
