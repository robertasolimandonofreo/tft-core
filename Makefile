.PHONY: test test-verbose test-coverage test-race test-bench clean deps lint

GO_FILES := $(shell find . -name '*.go' -not -path './vendor/*')
TEST_TIMEOUT := 30s
COVERAGE_OUT := coverage/coverage.out
COVERAGE_HTML := coverage/coverage.html
SONAR_REPORT := sonar-report.json
UNIT_REPORT := unit-report.xml
COVERAGE_DIR := coverage

test:
	@echo "Running tests..."
	go test -timeout $(TEST_TIMEOUT) ./internal/...

test-coverage:
	@rm -f $(COVERAGE_OUT)
	@go test -v -json ./... -covermode=count -coverprofile=$(COVERAGE_DIR)/coverage.out \
	  -coverpkg=$(shell go list ./... | grep -v '/tests/' | paste -sd "," -) | tee $(COVERAGE_DIR)/test-output.txt | tee $(COVERAGE_DIR)/$(SONAR_REPORT) > $(COVERAGE_DIR)/$(UNIT_REPORT)
	@go tool cover -html=$(COVERAGE_OUT) -o $(COVERAGE_HTML)
	@echo "Coverage report generated: $(COVERAGE_HTML)"

##	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.54.2

lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed, running go vet instead"; \
		go vet ./...; \
	fi

fmt:
	@echo "Formatting code..."
	go fmt ./...

clean:
	@echo "Cleaning up..."
	rm -f $(COVERAGE_OUT) $(COVERAGE_HTML)
	go clean -testcache

run:
	@echo "Running application..."
	export $(grep -v '^#' .env | xargs)
	go mod tidy
	go mod download
	go run ./cmd/main.go

test-performance:
	@echo "Running performance tests..."
	go test -timeout 5m -bench=. -benchtime=10s ./internal/...

test-memory:
	@echo "Running memory tests..."
	rm -rf internal.test mem.prof memory_profile.txt memory_profile.png
	go test -timeout $(TEST_TIMEOUT) -memprofile=mem.prof ./internal/...
	go tool pprof -top -cum mem.prof > memory_profile.txt
	go tool pprof -png mem.prof > memory_profile.png

test-cpu:
	@echo "Running CPU profiling tests..."
	rm -rf internal.test cpu.prof cpu_profile.txt cpu_profile.png
	go test -timeout $(TEST_TIMEOUT) -cpuprofile=cpu.prof ./internal/...
	go tool pprof -top -cum cpu.prof > cpu_profile.txt
	go tool pprof -png cpu.prof > cpu_profile.png