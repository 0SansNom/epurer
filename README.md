# 🧹 Épurer

Intelligent cache cleaner for macOS developers. Reclaim disk space by cleaning development caches safely.

## Installation

```bash
# From source
git clone https://github.com/0SansNom/epurer.git
cd epurer && make build && make install

# Or with Go (after pushing to GitHub)
go install github.com/0SansNom/epurer/cmd/epurer@latest
```

## Quick Start

```bash
epurer detect          # See what's installed
epurer report          # Preview what can be cleaned
epurer smart --dry-run # Safe automatic cleanup (preview)
epurer ui              # Interactive mode
```

## Commands

| Command | Description |
|---------|-------------|
| `detect` | Detect installed development tools |
| `report` | Generate cleanup report |
| `clean` | Execute cleanup |
| `smart` | Automatic safe cleanup |
| `ui` | Interactive TUI mode |

### Options

```bash
--dry-run              # Preview without deleting
--level <level>        # conservative, standard, aggressive
--domain <domains>     # frontend, backend, mobile, devops, dataml, system
--verbose              # Detailed output
```

## Supported Technologies

| Domain | Tools |
|--------|-------|
| **Frontend** | Node.js, npm, yarn, pnpm, Vite, Webpack, Next.js |
| **Backend** | Python, Java, Go, Rust, PHP, Ruby, Maven, Gradle |
| **Mobile** | Xcode, Android Studio, Flutter, CocoaPods |
| **DevOps** | Docker, Kubernetes, Terraform, Helm |
| **Data/ML** | Conda, Jupyter, TensorFlow, PyTorch, Hugging Face |
| **System** | Caches, logs, Homebrew, Trash, iOS backups |

## Safety Levels

| Level | Description |
|-------|-------------|
| `Safe` | Caches, logs - no risk |
| `Mod` | Dependencies, builds - rebuild needed |
| `Risk` | Backups, data - potential loss |

## Example Output

```
╔═══════════════════════════════════════╗
║             🧹 Épurer v1.0            ║
║  Intelligent cache cleanup for macOS  ║
╚═══════════════════════════════════════╝

 DOMAIN       ITEMS      SIZE  SAFETY    IMPACT
──────────────────────────────────────────────────
 Frontend     1,234   12.5 GB  Safe Mod  High
 Mobile          15   45.2 GB  Safe Mod  Very High
 Backend        567    3.4 GB  Safe      Medium
──────────────────────────────────────────────────
 Total        1,816   61.1 GB
```

## Interactive Mode

```bash
epurer ui
```

Controls: `↑↓` navigate · `Space` toggle · `a` all · `n` none · `Enter` confirm · `q` quit

## License

MIT

## Credits

Built with [Cobra](https://github.com/spf13/cobra), [Bubble Tea](https://github.com/charmbracelet/bubbletea), [Lip Gloss](https://github.com/charmbracelet/lipgloss)
