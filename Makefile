.PHONY: build install test clean run-detect run-report help

# Build binary
build:
	@echo "🔨 Building epurer..."
	@mkdir -p bin
	@go build -o bin/epurer ./cmd/epurer
	@echo "✅ Build complete: bin/epurer"

# Install to /usr/local/bin
install: build
	@echo "📦 Installing to /usr/local/bin..."
	@sudo cp bin/epurer /usr/local/bin/
	@echo "✅ Installation complete"

# Run tests
test:
	@echo "🧪 Running tests..."
	@go test -v ./...

# Clean build artifacts
clean:
	@echo "🧹 Cleaning build artifacts..."
	@rm -rf bin/
	@echo "✅ Clean complete"

# Run detect command
run-detect: build
	@./bin/epurer detect

# Run report command
run-report: build
	@./bin/epurer report

# Run smart dry-run
run-smart: build
	@./bin/epurer smart --dry-run

# Format code
fmt:
	@echo "📝 Formatting code..."
	@go fmt ./...
	@echo "✅ Format complete"

# Lint code
lint:
	@echo "🔍 Linting code..."
	@golangci-lint run || true

# Build for release
release:
	@echo "🚀 Building release binaries..."
	@goreleaser release --clean

# Show help
help:
	@echo "Épurer - Makefile targets:"
	@echo ""
	@echo "  make build       - Build the binary"
	@echo "  make install     - Install to /usr/local/bin (requires sudo)"
	@echo "  make test        - Run tests"
	@echo "  make clean       - Remove build artifacts"
	@echo "  make run-detect  - Build and run detect command"
	@echo "  make run-report  - Build and run report command"
	@echo "  make run-smart   - Build and run smart dry-run"
	@echo "  make fmt         - Format Go code"
	@echo "  make lint        - Lint Go code"
	@echo "  make release     - Build release binaries with goreleaser"
	@echo "  make help        - Show this help message"
