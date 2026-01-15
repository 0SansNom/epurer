# 🧹 Épurer

A powerful, intelligent CLI tool for cleaning development caches and temporary files on macOS.

Built with Go for maximum performance and zero runtime dependencies.

## ✨ Features

- **🔍 Smart Detection** - Automatically detects installed development tools
- **🚀 Concurrent Scanning** - Fast parallel filesystem scanning
- **🛡️ Safety Levels** - Conservative, Standard, and Aggressive cleaning modes
- **📊 Detailed Reports** - Beautiful tables showing what can be cleaned
- **🎯 Domain-Specific** - Targeted cleaning for different tech stacks
- **💾 Dry Run Mode** - Preview changes before executing
- **⚡ Fast & Efficient** - Written in Go with concurrent operations
- **🔒 Safe by Default** - Only removes easily rebuildable caches

## 🎯 Supported Technologies

### Frontend Development

- Node.js (`node_modules`, npm cache, yarn cache, pnpm store)
- Build outputs (dist, build, .next, out)
- Bundler caches (Vite, Webpack, Parcel, Turbo)
- Testing (coverage, .nyc_output)
- Linters (.eslintcache)
- Storybook

### Backend Development

- **Python**: `__pycache__`, pip cache, Poetry cache, pytest, mypy, tox
- **Java**: Maven repository, Gradle cache, target folders
- **Go**: Build cache, module cache
- **Rust**: Cargo cache, target folders
- **PHP**: Composer cache, vendor folders
- **Ruby**: Gem cache, Bundler cache

### Mobile Development

- **iOS/Xcode**: DerivedData, Archives, Device Support, Simulators
- **Android**: Build folders, Gradle cache, SDK cache, AVD
- **Flutter**: .dart_tool, build folders
- **CocoaPods**: Cache

### DevOps

- **Docker**: Dangling images, stopped containers, build cache, unused volumes
- **Kubernetes**: kubectl cache, Minikube
- **Terraform**: .terraform folders
- **Cloud CLIs**: AWS CLI cache, Helm cache
- **Vagrant**: Boxes

### Data Science / ML

- **Conda**: Package cache, environments
- **Jupyter**: Runtime files, checkpoints
- **TensorFlow/Keras**: Datasets and models cache
- **PyTorch**: Hub cache (pretrained models)
- **Hugging Face**: Transformers cache
- **Weights & Biases**: Cache and experiment logs
- **MLflow**: Experiment runs

### System

- Trash
- System caches
- System logs
- Temporary files
- DNS cache
- Homebrew cache
- Launchpad database
- iOS backups

## 🚀 Installation

### From Source

```bash
git clone https://github.com/0SansNom/epurer.git
cd epurer
make build
make install  # Requires sudo
```

### Using Go

```bash
go install github.com/0SansNom/epurer/cmd/epurer@latest
```

## 📖 Usage

### Quick Start

```bash
# Detect installed tools
epurer detect

# Generate a cleanup report
epurer report

# Smart automatic cleanup (conservative, dry-run)
epurer smart --dry-run

# Clean with default settings
epurer clean

# Clean specific domains
epurer clean --domain frontend,backend

# Aggressive clean (includes moderate and dangerous items)
epurer clean --level aggressive
```

### Commands

#### `detect` - Detect Development Tools

```bash
epurer detect
```

Scans your system and lists all detected development tools.

#### `report` - Generate Cleanup Report

```bash
epurer report [flags]

Flags:
  -l, --level string    Clean level: conservative|standard|aggressive (default "standard")
  -d, --domain strings  Domains to scan (comma-separated, empty = all)
  -v, --verbose         Show detailed breakdown
```

Generates a detailed report of what can be cleaned without actually deleting anything.

#### `clean` - Execute Cleanup

```bash
epurer clean [flags]

Flags:
      --dry-run         Show what would be cleaned without deleting
  -i, --interactive     Ask for confirmation (default true)
  -l, --level string    Clean level: conservative|standard|aggressive (default "standard")
  -d, --domain strings  Domains to clean (comma-separated, empty = all)
  -v, --verbose         Verbose output
```

Performs the actual cleanup operation.

#### `smart` - Smart Automatic Cleanup

```bash
epurer smart [flags]

Flags:
      --dry-run  Show what would be cleaned without deleting
```

Automatically detects tools and performs a safe, conservative cleanup.

### Safety Levels

- **🟢 Conservative** - Only cleans completely safe items (caches, logs)
- **🟡 Standard** - Includes items that need rebuild (node_modules, build outputs)
- **🔴 Aggressive** - Everything including potentially dangerous items (backups, data)

### Examples

```bash
# Dry-run to see what would be cleaned
epurer clean --dry-run

# Clean only frontend caches
epurer clean --domain frontend

# Aggressive clean (requires confirmation)
epurer clean --level aggressive

# Non-interactive clean
epurer clean --interactive=false

# Verbose report showing all files
epurer report --verbose

# Smart cleanup without confirmation
epurer smart
```

## 📊 Sample Output

```text
╔═══════════════════════════════════════════╗
║   🧹 Épurer v1.0                          ║
║   Intelligent cache cleanup for macOS     ║
╚═══════════════════════════════════════════╝

📊 Cleanup Estimation:

┌──────────┬────────┬─────────┬────────┬───────────┐
│  DOMAIN  │ ITEMS  │  SIZE   │ SAFETY │  IMPACT   │
├──────────┼────────┼─────────┼────────┼───────────┤
│ Frontend │  1,234 │ 12.5 GB │ 🟢 🟡  │ High      │
│ Mobile   │     15 │ 45.2 GB │ 🟢 🟡  │ Very High │
│ DevOps   │     42 │ 23.1 GB │ 🟢 🟡  │ High      │
│ Backend  │    567 │  3.4 GB │ 🟢     │ Medium    │
├──────────┼────────┼─────────┼────────┼───────────┤
│    Total │  1,858 │ 84.2 GB │        │           │
└──────────┴────────┴─────────┴────────┴───────────┘

🔐 Safety Levels:

  🟢 Safe - No risk, easily rebuilt (caches, logs)
  🟡 Moderate - Rebuild needed (dependencies, build outputs)
  🔴 Dangerous - Potential data loss (backups, databases)
```

## 🏗️ Architecture

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

## 🛠️ Development

```bash
# Build
make build

# Run tests
make test

# Format code
make fmt

# Lint
make lint

# Clean build artifacts
make clean
```

## 📝 License

MIT License - see LICENSE file for details

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## ⚠️ Disclaimer

This tool deletes files from your system. While it's designed to be safe and only target cache/temporary files, please:

- Use `--dry-run` first to preview changes
- Start with conservative mode
- Make sure you have backups of important data
- Review what will be deleted before confirming

## 🙏 Credits

Built with:

- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [tablewriter](https://github.com/olekukonko/tablewriter) - Beautiful tables
- [progressbar](https://github.com/schollz/progressbar) - Progress bars
- [go-humanize](https://github.com/dustin/go-humanize) - Human-readable formatting
- [color](https://github.com/fatih/color) - Colorful output
