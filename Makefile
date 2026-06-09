# Makefile — sinai
# Build, test, and installation shortcuts for the project.

BINARY_NAME  := sinai
BUILD_DIR    := bin
CMD_PATH     := ./cmd/sinai
MODULE       := github.com/stanley/sinai

# Version extracted via git tag (fallback to "dev").
VERSION      ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS      := -s -w -X main.version=$(VERSION)

.PHONY: build run install clean test lint tidy

## build: Compile binary to ./bin/sinai
build:
	@echo "🔨 Building binary..."
	@mkdir -p $(BUILD_DIR)
	go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_PATH)
	@echo "🚀 Binary generated: $(BUILD_DIR)/$(BINARY_NAME)"

## run: Compile and run the application directly
run:
	@echo "🏃 Running application..."
	go run $(CMD_PATH)

## install: Install binary to $GOPATH/bin (or $GOBIN)
install:
	@echo "📥 Installing binary..."
	go install -ldflags "$(LDFLAGS)" $(CMD_PATH)
	@echo "✨ Installed successfully to $$(go env GOPATH)/bin/$(BINARY_NAME)"

## test: Run all tests and generate coverage report
test:
	@echo "🧪 Running tests..."
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "📊 Coverage report generated: coverage.html"

## lint: Run go vet and golangci-lint (must be installed)
lint:
	@echo "🔍 Running static analysis (go vet)..."
	go vet ./...
	@echo "🛡️ Running golangci-lint..."
	golangci-lint run ./...
	@echo "✅ Lint check complete!"

## tidy: Clean and organize go.mod and go.sum
tidy:
	@echo "🧹 Tidying go modules..."
	go mod tidy
	@echo "✨ Go modules are tidy!"

## clean: Remove build artifacts and coverage files
clean:
	@echo "🗑️ Cleaning build artifacts..."
	rm -rf $(BUILD_DIR) coverage.out coverage.html
	@echo "🧹 Clean complete!"
