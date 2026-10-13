.PHONY: build install test clean run-detect run-report help app-engine app-build app-bundle app-open dmg

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null | sed 's/^v//')
LDFLAGS := -X github.com/0SansNom/epurer/internal/version.Version=$(VERSION)
APP_ENGINE_DIR := app/Epurer/.engine
APP_BUNDLE := dist/Épurer.app
APP_DMG := dist/Epurer.dmg
XCODE_PROJECT := app/EpurerApp/Epurer.xcodeproj
XCODE_SCHEME := Epurer
XCODE_DERIVED := build/DerivedData
DEVELOPER_DIR ?= /Applications/Xcode.app/Contents/Developer
export DEVELOPER_DIR

# Build binary
build:
	@echo "🔨 Building epurer $(VERSION)..."
	@mkdir -p bin
	@go build -ldflags "$(LDFLAGS)" -o bin/epurer ./cmd/epurer
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
	@rm -rf bin/ build/DerivedData dist/Épurer.app $(APP_DMG)
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
# CLI archives + Homebrew formula + the DMG, attached to the GitHub release
release: dmg
	@echo "🚀 Building release binaries..."
	@goreleaser release --clean

# Universal (Apple silicon + Intel) Go engine next to the Swift package
app-engine:
	@mkdir -p $(APP_ENGINE_DIR)
	@GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS) -s -w" -o $(APP_ENGINE_DIR)/epurer-arm64 ./cmd/epurer
	@GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS) -s -w" -o $(APP_ENGINE_DIR)/epurer-amd64 ./cmd/epurer
	@lipo -create -output $(APP_ENGINE_DIR)/epurer $(APP_ENGINE_DIR)/epurer-arm64 $(APP_ENGINE_DIR)/epurer-amd64
	@rm $(APP_ENGINE_DIR)/epurer-arm64 $(APP_ENGINE_DIR)/epurer-amd64
	@echo "✅ Engine at $(APP_ENGINE_DIR)/epurer (universal)"

# Build the SwiftUI host via Xcode (needs full Xcode)
app-build: app-engine
	@xcodebuild \
		-project "$(XCODE_PROJECT)" \
		-scheme "$(XCODE_SCHEME)" \
		-configuration Release \
		-derivedDataPath "$(XCODE_DERIVED)" \
		ARCHS="arm64 x86_64" ONLY_ACTIVE_ARCH=NO \
		CODE_SIGN_IDENTITY="-" \
		CODE_SIGNING_ALLOWED=YES \
		build
	@echo "✅ Xcode app built"

# Assemble dist/Épurer.app from xcodebuild output and re-sign (hardened runtime).
app-bundle: app-build
	@rm -rf "$(APP_BUNDLE)"
	@mkdir -p dist
	@BUILT="$$(find "$(XCODE_DERIVED)/Build/Products/Release" -maxdepth 1 -name 'Epurer.app' -print -quit)"; \
		if [ -z "$$BUILT" ]; then echo "Epurer.app not found in DerivedData"; exit 1; fi; \
		cp -R "$$BUILT" "$(APP_BUNDLE)"
	@# Ensure engine is present under the APFS-safe resource name
	@test -x "$(APP_BUNDLE)/Contents/Resources/epurer-engine" || \
		(cp "$(APP_ENGINE_DIR)/epurer" "$(APP_BUNDLE)/Contents/Resources/epurer-engine" && chmod +x "$(APP_BUNDLE)/Contents/Resources/epurer-engine")
	@plutil -replace CFBundleShortVersionString -string "$(VERSION)" "$(APP_BUNDLE)/Contents/Info.plist"
	@plutil -replace CFBundleVersion -string "$(VERSION)" "$(APP_BUNDLE)/Contents/Info.plist"
	@codesign --force --options runtime --timestamp=none --sign - \
		--entitlements packaging/macos/epurer.entitlements \
		"$(APP_BUNDLE)/Contents/Resources/epurer-engine"
	@codesign --force --options runtime --timestamp=none --sign - \
		--entitlements packaging/macos/epurer.entitlements \
		"$(APP_BUNDLE)/Contents/MacOS/Epurer"
	@codesign --force --deep --options runtime --timestamp=none --sign - \
		--entitlements packaging/macos/epurer.entitlements \
		"$(APP_BUNDLE)"
	@echo "✅ App bundle: $(APP_BUNDLE) (hardened runtime, ad-hoc)"

app-open: app-bundle
	@open "$(APP_BUNDLE)"

# Drag-to-Applications disk image for the GitHub release
dmg: app-bundle
	@rm -rf build/dmg "$(APP_DMG)"
	@mkdir -p build/dmg
	@cp -R "$(APP_BUNDLE)" build/dmg/
	@ln -s /Applications build/dmg/Applications
	@hdiutil create -quiet -volname "Épurer" -srcfolder build/dmg -fs HFS+ -format UDZO "$(APP_DMG)"
	@rm -rf build/dmg
	@echo "✅ Disk image: $(APP_DMG)"

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
	@echo "  make app-engine  - Build Go engine into app/Epurer/.engine/"
	@echo "  make app-build   - Build SwiftUI app via xcodebuild"
	@echo "  make app-bundle  - Assemble dist/Épurer.app with embedded engine"
	@echo "  make app-open    - Build, assemble, and open the app"
	@echo "  make dmg         - Build dist/Epurer.dmg for a release"
	@echo "  make fmt         - Format Go code"
	@echo "  make lint        - Lint Go code"
	@echo "  make release     - Build the DMG, then publish with goreleaser"
	@echo "  make help        - Show this help message"
