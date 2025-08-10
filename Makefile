.PHONY: test test-verbose test-coverage test-race test-bench clean deps lint

# Variables
GO_FILES := $(shell find . -name '*.go' -not -path './vendor/*')
TEST_TIMEOUT := 30s
COVERAGE_OUT := coverage.out
COVERAGE_HTML := coverage.html

# Default target
all: deps lint test

# Install dependencies
deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

# Run tests
test:
	@echo "Running tests..."
	go test -timeout $(TEST_TIMEOUT) ./internal/...

# Run tests with verbose output
test-verbose:
	@echo "Running tests with verbose output..."
	go test -v -timeout $(TEST_TIMEOUT) ./internal/...

# Run tests with race detection
test-race:
	@echo "Running tests with race detection..."
	go test -race -timeout $(TEST_TIMEOUT) ./internal/...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -timeout $(TEST_TIMEOUT) -coverprofile=$(COVERAGE_OUT) ./internal/...
	go tool cover -html=$(COVERAGE_OUT) -o $(COVERAGE_HTML)
	@echo "Coverage report generated: $(COVERAGE_HTML)"

# Run benchmarks
test-bench:
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./internal/...

# Run specific test
test-specific:
	@if [ -z "$(TEST)" ]; then \
		echo "Usage: make test-specific TEST=TestName"; \
		exit 1; \
	fi
	@echo "Running test: $(TEST)"
	go test -v -run "$(TEST)" ./internal/...

# Run tests for specific package
test-package:
	@if [ -z "$(PKG)" ]; then \
		echo "Usage: make test-package PKG=package_name"; \
		exit 1; \
	fi
	@echo "Running tests for package: $(PKG)"
	go test -v ./internal/$(PKG)/...

# Generate test mocks (if using mockgen)
mocks:
	@echo "Generating mocks..."
	@# Add mockgen commands here if needed
	@echo "No mocks configured"

# Lint code
lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed, running go vet instead"; \
		go vet ./...; \
	fi

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Clean up generated files
clean:
	@echo "Cleaning up..."
	rm -f $(COVERAGE_OUT) $(COVERAGE_HTML)
	go clean -testcache

# Build the application
build:
	@echo "Building application..."
	go build -o bin/tft-core ./cmd/main.go

# Run the application
run:
	@echo "Running application..."
	go run ./cmd/main.go

# Install golangci-lint
install-lint:
	@echo "Installing golangci-lint..."
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.54.2

# Docker build for testing
docker-test:
	@echo "Running tests in Docker..."
	docker run --rm -v $(PWD):/app -w /app golang:1.24.0-alpine go test ./internal/...

# Performance tests
test-performance:
	@echo "Running performance tests..."
	go test -timeout 5m -bench=. -benchtime=10s ./internal/...

# Memory tests
test-memory:
	@echo "Running memory tests..."
	go test -timeout $(TEST_TIMEOUT) -memprofile=mem.prof ./internal/...
	go tool pprof mem.prof

# CPU profiling tests
test-cpu:
	@echo "Running CPU profiling tests..."
	go test -timeout $(TEST_TIMEOUT) -cpuprofile=cpu.prof ./internal/...
	go tool pprof cpu.prof

# Integration tests (separate from unit tests)
test-integration:
	@echo "Running integration tests..."
	@# Add integration test commands here
	@echo "No integration tests configured"

# Generate test report
test-report: test-coverage
	@echo "Generating test report..."
	@echo "=== Test Coverage Report ===" > test-report.txt
	@go tool cover -func=$(COVERAGE_OUT) >> test-report.txt
	@echo "" >> test-report.txt
	@echo "=== Test Results ===" >> test-report.txt
	@go test -json ./internal/... >> test-report.txt 2>&1 || true
	@echo "Test report generated: test-report.txt"

# Watch tests (requires entr or similar tool)
test-watch:
	@if command -v entr >/dev/null 2>&1; then \
		echo "Watching for file changes..."; \
		find . -name '*.go' | entr -c make test; \
	else \
		echo "entr not installed. Install with: brew install entr (macOS) or apt-get install entr (Ubuntu)"; \
	fi

# Help
help:
	@echo "Available targets:"
	@echo "  test           - Run all tests"
	@echo "  test-verbose   - Run tests with verbose output"
	@echo "  test-coverage  - Run tests with coverage report"
	@echo "  test-race      - Run tests with race detection"
	@echo "  test-bench     - Run benchmarks"
	@echo "  test-specific  - Run specific test (TEST=TestName)"
	@echo "  test-package   - Run tests for specific package (PKG=package_name)"
	@echo "  test-watch     - Watch files and run tests on changes"
	@echo "  lint           - Run linter"
	@echo "  fmt            - Format code"
	@echo "  build          - Build application"
	@echo "  run            - Run application"
	@echo "  clean          - Clean generated files"
	@echo "  deps           - Install dependencies"