.PHONY: build build-all test clean run

BINARY_NAME=hive-core
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

# Build for current platform
build:
	go build -ldflags="-X main.version=$(VERSION)" -o $(BINARY_NAME) ./cmd/orchestrator

HIVE_BINARY=hive

# Build Hive CLI (Single Binary)
build-hive:
	go build -ldflags="-X main.version=$(VERSION)" -o $(HIVE_BINARY) ./cmd/hive

install: build-hive
	@echo "Installing hive to /usr/local/bin (requires sudo)..."
	sudo mv $(HIVE_BINARY) /usr/local/bin/hive

# Build for all platforms
build-all:
	@mkdir -p dist
	GOOS=linux GOARCH=amd64 go build -ldflags="-X main.version=$(VERSION)" -o dist/$(BINARY_NAME)-linux-amd64 ./cmd/orchestrator
	GOOS=linux GOARCH=amd64 go build -ldflags="-X main.version=$(VERSION)" -o dist/$(HIVE_BINARY)-linux-amd64 ./cmd/hive
	GOOS=linux GOARCH=arm64 go build -ldflags="-X main.version=$(VERSION)" -o dist/$(BINARY_NAME)-linux-arm64 ./cmd/orchestrator
	GOOS=linux GOARCH=arm64 go build -ldflags="-X main.version=$(VERSION)" -o dist/$(HIVE_BINARY)-linux-arm64 ./cmd/hive
	GOOS=darwin GOARCH=amd64 go build -ldflags="-X main.version=$(VERSION)" -o dist/$(BINARY_NAME)-darwin-amd64 ./cmd/orchestrator
	GOOS=darwin GOARCH=amd64 go build -ldflags="-X main.version=$(VERSION)" -o dist/$(HIVE_BINARY)-darwin-amd64 ./cmd/hive
	GOOS=darwin GOARCH=arm64 go build -ldflags="-X main.version=$(VERSION)" -o dist/$(BINARY_NAME)-darwin-arm64 ./cmd/orchestrator
	GOOS=darwin GOARCH=arm64 go build -ldflags="-X main.version=$(VERSION)" -o dist/$(HIVE_BINARY)-darwin-arm64 ./cmd/hive
	GOOS=windows GOARCH=amd64 go build -ldflags="-X main.version=$(VERSION)" -o dist/$(BINARY_NAME)-windows-amd64.exe ./cmd/orchestrator
	GOOS=windows GOARCH=amd64 go build -ldflags="-X main.version=$(VERSION)" -o dist/$(HIVE_BINARY)-windows-amd64.exe ./cmd/hive

# Run tests
test:
	go test -v -cover ./...

# Run tests with race detection
test-race:
	go test -v -race ./...

# Run linter
lint:
	go vet ./...
	@command -v staticcheck > /dev/null && staticcheck ./... || echo "staticcheck not installed"

# Clean build artifacts
clean:
	rm -f $(BINARY_NAME)*
	rm -f $(HIVE_BINARY)*
	rm -rf dist/
	rm -rf logs/
	rm -rf hello_world/
	rm -f tasks.json

# Run the orchestrator
run: build
	./$(BINARY_NAME)

# Run with debug logging
run-debug: build
	./$(BINARY_NAME) --config config.json

# Format code
fmt:
	go fmt ./...

# Tidy dependencies
tidy:
	go mod tidy

# Show help
help:
	@echo "Available targets:"
	@echo "  build      - Build for current platform"
	@echo "  build-all  - Build for all platforms (Linux, macOS, Windows)"
	@echo "  test       - Run tests with coverage"
	@echo "  test-race  - Run tests with race detection"
	@echo "  lint       - Run linters"
	@echo "  clean      - Remove build artifacts"
	@echo "  run        - Build and run"
	@echo "  run-debug  - Build and run with debug logging"
	@echo "  fmt        - Format code"
	@echo "  tidy       - Tidy dependencies"
