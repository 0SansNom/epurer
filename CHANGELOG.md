# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2025-12-25

### Added

- Initial release of Mac Developer Cleaner
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

[1.0.0]: https://github.com/0SansNom/mac-dev-clean/releases/tag/v1.0.0
