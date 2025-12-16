#!/bin/bash

# Script to generate Goa code from design files
# This script follows best practices for error handling and logging

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

# Error handler
error_exit() {
    log_error "$1"
    exit 1
}

# Check if we're in the right directory
if [ ! -f "go.mod" ]; then
    error_exit "go.mod not found. Please run this script from the project root."
fi

# Check if design file exists
DESIGN_FILE="api/design/api.go"
if [ ! -f "$DESIGN_FILE" ]; then
    error_exit "Design file not found: $DESIGN_FILE"
fi

log_info "Starting Goa code generation..."

# Check if Go is installed
if ! command -v go &> /dev/null; then
    error_exit "Go is not installed or not in PATH"
fi

# Install Goa CLI
log_info "Installing Goa CLI..."
if ! go install goa.design/goa/v3/cmd/goa@latest; then
    error_exit "Failed to install Goa CLI"
fi

# Verify Goa installation
if ! command -v goa &> /dev/null && [ ! -f "$(go env GOPATH)/bin/goa" ]; then
    log_warn "Goa binary not found in PATH. Trying to use GOPATH/bin/goa..."
    export PATH="$PATH:$(go env GOPATH)/bin"
fi

# Generate Goa code
log_info "Generating Goa code from design files..."
if ! goa gen springstreet/api/design; then
    error_exit "Failed to generate Goa code"
fi

# Generate HTTP transport code
log_info "Generating HTTP transport code..."
if ! goa example springstreet/api/design; then
    error_exit "Failed to generate HTTP transport code"
fi

# Tidy go.mod
log_info "Tidying go.mod..."
if ! go mod tidy; then
    log_warn "go mod tidy completed with warnings"
fi

# Verify generated code
if [ ! -d "gen" ]; then
    error_exit "Generated code directory 'gen' not found"
fi

log_info "Code generation complete!"
log_info "You can now build and run the application with: go run cmd/api/main.go"

# Check for common issues
if [ -d "gen" ] && [ -z "$(ls -A gen 2>/dev/null)" ]; then
    log_warn "Generated code directory is empty. Check your design file."
fi

exit 0
