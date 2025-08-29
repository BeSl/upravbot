# CupBot Makefile
# Works on Windows with mingw32-make or WSL

.PHONY: help build test test-coverage test-race test-bench clean lint fmt vet deps install-deps

# Default target
help:
	@echo "CupBot Build Commands:"
	@echo ""
	@echo "  build          - Build the application"
	@echo "  test           - Run all tests"
	@echo "  test-coverage  - Run tests with coverage report"
	@echo "  test-race      - Run tests with race detection"
	@echo "  test-bench     - Run benchmarks"
	@echo "  lint           - Run linter"
	@echo "  fmt            - Format code"
	@echo "  vet            - Run go vet"
	@echo "  deps           - Download dependencies"
	@echo "  install-deps   - Install development dependencies"
	@echo "  clean          - Clean build artifacts"
	@echo ""

# Build the application
build:
	go build -ldflags="-w -s" -o cupbot.exe .

# Run all tests
test:
	go test -v ./internal/...

# Run tests with coverage
test-coverage:
	go test -coverprofile=coverage.out ./internal/...
	go tool cover -html=coverage.out -o coverage.html
	go tool cover -func=coverage.out

# Run tests with race detection
test-race:
	go test -race ./internal/...

# Run benchmarks
test-bench:
	go test -bench=. -benchmem ./internal/...

# Run linter (requires golangci-lint)
lint:
	golangci-lint run

# Format code
fmt:
	go fmt ./...
	goimports -w .

# Run go vet
vet:
	go vet ./...

# Download dependencies
deps:
	go mod download
	go mod tidy

# Install development dependencies
install-deps:
	go install golang.org/x/tools/cmd/goimports@latest
	# golangci-lint installation (Windows)
	# Download from: https://github.com/golangci/golangci-lint/releases

# Clean build artifacts
clean:
	del cupbot.exe 2>nul || echo "No executable to clean"
	del coverage.out 2>nul || echo "No coverage file to clean"
	del coverage.html 2>nul || echo "No coverage HTML to clean"
	rmdir /s /q test-results 2>nul || echo "No test results to clean"

# Full CI pipeline simulation
ci: deps vet lint test-race test-coverage

# Quick development checks
dev: fmt vet test