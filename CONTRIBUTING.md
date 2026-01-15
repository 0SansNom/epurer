# Contributing to Épurer

Thank you for your interest in contributing to Épurer! This document provides guidelines for contributing to the project.

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://github.com/YOUR_USERNAME/epurer.git`
3. Create a new branch: `git checkout -b feature/your-feature-name`
4. Make your changes
5. Test your changes: `make test`
6. Commit your changes: `git commit -am 'Add some feature'`
7. Push to the branch: `git push origin feature/your-feature-name`
8. Create a Pull Request

## Development Setup

### Prerequisites

- Go 1.21 or later
- macOS (for testing)
- Make

### Building

```bash
# Build the binary
make build

# Run tests
make test

# Format code
make fmt

# Lint code (requires golangci-lint)
make lint
```

## Project Structure

```text
epurer/
├── cmd/epurer/    # CLI entry point
├── internal/
│   ├── cleaner/          # Domain-specific cleaners
│   ├── config/           # Configuration and types
│   ├── detector/         # Tool detection
│   ├── reporter/         # Output formatting
│   └── scanner/          # Concurrent file scanning
└── pkg/utils/            # Utility functions
```

## Adding a New Cleaner

To add support for a new technology or domain:

1. Create a new cleaner file in `internal/cleaner/`
2. Implement the `Cleaner` interface:
   - `Name() string` - Return a human-readable name
   - `Domain() config.Domain` - Return the domain category
   - `Detect(ctx context.Context) (bool, error)` - Detect if applicable
   - `Scan(ctx context.Context, cfg *config.Config) ([]CleanTarget, error)` - Find cleanable items
   - `Clean(ctx context.Context, targets []CleanTarget, dryRun bool) ([]CleanResult, error)` - Execute cleanup

3. Add factory function (e.g., `NewYourCleaner()`)
4. Register in `cmd/epurer/main.go` in `initAllCleaners()`
5. Add tests
6. Update documentation

### Example Cleaner

```go
package cleaner

import (
    "context"
    "github.com/0SansNom/epurer/internal/config"
    "github.com/0SansNom/epurer/internal/scanner"
)

type YourCleaner struct {
    scanner *scanner.Scanner
}

func NewYourCleaner() (Cleaner, error) {
    s, err := scanner.NewScanner()
    if err != nil {
        return nil, err
    }
    return &YourCleaner{scanner: s}, nil
}

func (y *YourCleaner) Name() string {
    return "Your Domain"
}

func (y *YourCleaner) Domain() config.Domain {
    return config.DomainFrontend // Use appropriate domain
}

func (y *YourCleaner) Detect(ctx context.Context) (bool, error) {
    // Check if your tool is installed
    return utils.CommandExists("your-tool"), nil
}

func (y *YourCleaner) Scan(ctx context.Context, cfg *config.Config) ([]CleanTarget, error) {
    // Find cleanable items
    targets := []CleanTarget{}
    // ... scanning logic ...
    return targets, nil
}

func (y *YourCleaner) Clean(ctx context.Context, targets []CleanTarget, dryRun bool) ([]CleanResult, error) {
    // Clean the targets
    results := []CleanResult{}
    // ... cleaning logic ...
    return results, nil
}
```

## Testing

- Write tests for all new features
- Ensure existing tests pass: `make test`
- Test manually with different scenarios:
  - `make run-detect`
  - `make run-report`
  - `make run-smart`

## Code Style

- Follow standard Go formatting: `go fmt`
- Use meaningful variable and function names
- Add comments for exported functions
- Keep functions focused and small
- Handle errors appropriately

## Safety Guidelines

This tool deletes files, so safety is paramount:

1. **Always use safety levels correctly**:
   - `Safe` - Only for easily rebuildable caches
   - `Moderate` - For items requiring rebuild (node_modules, etc.)
   - `Dangerous` - For items with potential data loss

2. **Respect dry-run mode** - Never delete if `dryRun` is true

3. **Provide accurate size estimates** - Users rely on these

4. **Clear descriptions** - Help users understand what will be deleted

5. **Test thoroughly** - Test on your own system first

## Commit Messages

Use clear, descriptive commit messages:

- `feat: Add support for Rust cargo cache`
- `fix: Correct size calculation for nested directories`
- `docs: Update README with new examples`
- `test: Add tests for mobile cleaner`
- `refactor: Improve scanner performance`

## Pull Request Process

1. Update the README.md with details of changes if applicable
2. Update CHANGELOG.md with your changes
3. Ensure all tests pass
4. Update documentation
5. Wait for review from maintainers

## Questions?

- Open an issue for bug reports
- Use discussions for questions
- Check existing issues before creating new ones

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
