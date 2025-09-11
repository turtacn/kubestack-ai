# Makefile for KubeStack-AI

# --- Variables ---
BINARY_NAME=ksa
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

# Use 'go list' to find the main package path dynamically.
MAIN_PKG = $(shell go list -f '{{.Dir}}' ./cmd/ksa)

# --- Targets ---
# Phony targets are not actual files.
.PHONY: all build run test lint clean help

# The default target executed when you just run `make`.
all: build

# Build the application using the dedicated build script.
build:
	@echo "INFO: Building binaries for all platforms..."
	@./scripts/build.sh

# Run the application locally with a default config for quick testing.
run:
	@echo "INFO: Building and running the application..."
	@go run ./cmd/ksa --config ./configs/config.yaml

# Run all unit tests with race detection and code coverage.
test:
	@echo "INFO: Running tests..."
	@go test -v -race -cover ./...

# Run the linter to check for code style and quality issues.
# Assumes golangci-lint is installed: https://golangci-lint.run/usage/install/
lint:
	@echo "INFO: Running linter..."
	@if ! command -v golangci-lint &> /dev/null; then \
		echo "WARN: golangci-lint not found. Please install it to run the linter."; \
	else \
		golangci-lint run ./...; \
	fi

# Clean up build artifacts.
clean:
	@echo "INFO: Cleaning up build artifacts..."
	@rm -rf ./bin
	@rm -f coverage.out

# Display this help message.
help:
	@echo "Usage: make <target>"
	@echo ""
	@echo "Targets:"
	@echo "  all      Build binaries for all platforms (default)."
	@echo "  build    Alias for 'all'."
	@echo "  run      Build and run the application locally for quick testing."
	@echo "  test     Run all unit tests."
	@echo "  lint     Run the Go linter."
	@echo "  clean    Remove all build artifacts and coverage reports."
	@echo "  help     Show this help message."

# Personal.AI order the ending
