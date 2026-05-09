BINARY_NAME=cortex
BUILD_DIR=dist
MAIN_PATH=./cmd/cortex

.PHONY: build run test lint clean install fmt vet tidy

## build: compile the binary for current platform
build:
	@echo "→ Building $(BINARY_NAME)..."
	@go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "✓ Built: $(BUILD_DIR)/$(BINARY_NAME)"

## run: build and run with default args
run: build
	@./$(BUILD_DIR)/$(BINARY_NAME)

## install: install binary to GOPATH/bin
install:
	@echo "→ Installing $(BINARY_NAME)..."
	@go install $(MAIN_PATH)
	@echo "✓ Installed to $(shell go env GOPATH)/bin/$(BINARY_NAME)"

## test: run all tests with race detector
test:
	@echo "→ Running tests..."
	@go test -race -v ./...

## test-cover: run tests and show coverage
test-cover:
	@go test -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "✓ Coverage report: coverage.html"

## lint: run golangci-lint
lint:
	@echo "→ Linting..."
	@golangci-lint run ./...

## fmt: format all Go files
fmt:
	@go fmt ./...
	@echo "✓ Formatted"

## vet: run go vet
vet:
	@go vet ./...
	@echo "✓ Vet passed"

## tidy: tidy and verify go modules
tidy:
	@go mod tidy
	@go mod verify
	@echo "✓ Modules tidy"

## clean: remove build artifacts
clean:
	@rm -rf $(BUILD_DIR) coverage.out coverage.html
	@echo "✓ Cleaned"

## help: list all available targets
help:
	@grep -E '^##' Makefile | sed 's/## //'