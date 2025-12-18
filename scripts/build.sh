#!/bin/bash

# A robust build script for the KubeStack-AI application.
#
# This script handles:
# - Cross-platform compilation for various OS/architecture combinations.
# - Injecting version information (git tag, commit hash, build date) into the binary.
# - Running tests and linters as part of the build process.
# - Building a Docker container image.

# Exit immediately if a command exits with a non-zero status.
set -e

# --- Configuration ---
# The main package for the CLI application.
readonly PKG_PATH="github.com/kubestack-ai/kubestack-ai/cmd/ksa"
# The name of the output binary.
readonly BINARY_NAME="ksa"
# The directory to place the built binaries.
readonly OUTPUT_DIR="./bin"
# A space-separated list of platforms to build for (format: GOOS/GOARCH).
readonly PLATFORMS="linux/amd64 darwin/amd64 darwin/arm64 windows/amd64"

# --- Version Information ---
# Get the current version from the latest Git tag. Fallback to 'dev'.
readonly VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
# Get the current git commit hash.
readonly GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
# Get the current date in UTC.
readonly BUILD_DATE=$(date -u +'%Y-%m-%dT%H:%M:%SZ')
# The import path of the version package.
readonly VERSION_PKG="github.com/kubestack-ai/kubestack-ai/pkg/version"

# --- LDFLAGS ---
# These flags will embed the version information into the binary via the Go linker.
# -s: Omit the symbol table and debug information.
# -w: Omit the DWARF symbol table.
readonly LDFLAGS="-s -w \
    -X '${VERSION_PKG}.Version=${VERSION}' \
    -X '${VERSION_PKG}.GitCommit=${GIT_COMMIT}' \
    -X '${VERSION_PKG}.BuildDate=${BUILD_DATE}'"

# --- Functions ---

# Run code quality checks (linting).
run_lint() {
    echo "INFO: Running linter..."
    # Assumes golangci-lint is installed: https://golangci-lint.run/usage/install/
    if ! command -v golangci-lint &> /dev/null; then
        echo "WARN: golangci-lint not found, skipping lint check."
        return
    fi
    golangci-lint run ./...
}

# Run unit tests.
run_tests() {
    echo "INFO: Running unit tests..."
    go test -v -race -coverprofile=coverage.out ./...
}

# Build the binary for a specific platform.
build_platform() {
    local platform=$1
    # Split the platform string into OS and architecture.
    local os_arch=(${platform//\// })
    local os=${os_arch[0]}
    local arch=${os_arch[1]}

    local output_name="${OUTPUT_DIR}/${BINARY_NAME}-${os}-${arch}"
    if [ "$os" = "windows" ]; then
        output_name+=".exe"
    fi

    echo "INFO: Building for ${platform}..."
    # Set environment variables for cross-compilation and execute the build.
    CGO_ENABLED=0 GOOS=${os} GOARCH=${arch} go build -ldflags="${LDFLAGS}" -o "${output_name}" "${PKG_PATH}"
}

# Build the Docker image.
build_docker() {
    echo "INFO: Building Docker image..."
    docker build -t "kubestack-ai/ksa:${VERSION}" .
}

# --- Main Logic ---
main() {
    echo "INFO: Starting build process for KubeStack-AI version ${VERSION}"

    # Run optional quality checks first. Uncomment to enable.
    # run_lint
    # run_tests

    # Create the output directory if it doesn't exist.
    mkdir -p "${OUTPUT_DIR}"

    # Build for all specified platforms.
    for platform in ${PLATFORMS}; do
        build_platform "${platform}"
    done

    echo "INFO: All builds completed successfully."
    ls -l "${OUTPUT_DIR}"

    # Optional: Build the Docker image. Uncomment to enable.
    # if ! command -v docker &> /dev/null; then
    #     echo "WARN: docker not found, skipping docker image build."
    # else
    #     build_docker
    # fi
}

# Run the main function.
main

# Personal.AI order the ending
