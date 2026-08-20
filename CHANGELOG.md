# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.2.0] - 2026-08-20

### Added

- `epurer purge [path]` - scans project directories for removable build
  artifacts (node_modules, target, .venv, Pods, vendor, build, .next,
  .turbo, cmake-build-*), grouped by project. Detects project type from
  marker files and verifies ambiguous artifact names against a sibling
  marker to cut false positives. Age-preselected (`--min-age`, default
  7 days); interactive review by default, `--force`/`--yes` for
  non-interactive use.
- `epurer analyze [path]` - interactive, size-sorted disk usage explorer
  with drill-down navigation, Finder reveal, and confirmed deletion.
- `epurer ignore add/remove/list` - persistent path exclusion list,
  respected by every cleaner.
- AI coding tool cache cleaner (Claude, Cursor, GitHub Copilot,
  Continue.dev, ChatGPT desktop).

### Fixed

- `SafeRemove` now refuses to touch protected paths (empty path, `$HOME`,
  system roots) as a backstop against upstream path bugs.
- `epurer ui` previously never deleted anything - its cleaning step was
  hard-coded to report a fake result instead of calling `SafeRemove`.
- Backend/Mobile/DevOps/DataML cleaners were misreporting their domain
  category (all returned `DomainFrontend`).

### Changed

- Deduplicated the identical `Clean()` implementation shared by six
  cleaners into a single helper.
- CI now matches the Go version declared in `go.mod` and runs `go vet`
  and the test suite with the race detector.

## [1.0.0] - 2025-12-25

### Added

- Initial release of Épurer
- Smart detection of development tools and frameworks
- Concurrent filesystem scanning for high performance
- Support for multiple technology stacks:
  - Frontend (Node.js, npm, yarn, pnpm, bundlers)
  - Backend (Python, Java, Go, Rust, PHP, Ruby)
  - Mobile (Xcode, Android, Flutter)
  - DevOps (Docker, Kubernetes, Terraform)
  - Data Science/ML (Conda, Jupyter, TensorFlow, PyTorch)
  - System (caches, logs, Homebrew)
- Three safety levels: Conservative, Standard, and Aggressive
- Beautiful CLI interface with colored output and tables
- Dry-run mode for safe preview
- Interactive confirmation for destructive operations
- Domain-specific filtering for targeted cleanup
- Detailed reporting with size estimations
- Smart automatic cleanup mode
- Progress tracking during cleanup
- Comprehensive documentation and examples

### Features

- 🔍 Automatic tool detection
- 🚀 Concurrent scanning with configurable workers
- 🛡️ Safety-first approach with three cleanup levels
- 📊 Beautiful tables and progress bars
- 💾 Dry-run mode for preview
- 🎯 Domain-specific cleaning
- ⚡ Written in Go for maximum performance
- 🔒 Safe by default - only targets rebuildable caches

[1.2.0]: https://github.com/0SansNom/epurer/releases/tag/v1.2.0
[1.0.0]: https://github.com/0SansNom/epurer/releases/tag/v1.0.0
