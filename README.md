<div align="center">

# Épurer

**Reclaim the disk space your dev tools quietly eat — safely, and without an admin password.**

[![CI](https://github.com/0SansNom/epurer/actions/workflows/ci.yml/badge.svg)](https://github.com/0SansNom/epurer/actions/workflows/ci.yml)
![macOS 14+](https://img.shields.io/badge/macOS-14%2B-5B8DE8?logo=apple&logoColor=white)
![Go 1.24](https://img.shields.io/badge/Go-1.24-8EDCCB?logo=go&logoColor=white)
[![License: MIT](https://img.shields.io/badge/license-MIT-CDE99A)](LICENSE)

<a href="https://github.com/0SansNom/epurer/releases/latest/download/Epurer.dmg"><img src="https://img.shields.io/badge/Download-Épurer.dmg-5B8DE8?style=for-the-badge&logo=apple&logoColor=white" alt="Download Épurer for macOS"></a>

<sub>macOS 14 or later · Apple silicon and Intel · Free and open source</sub>

<br><br>

<img src="docs/assets/epurer-hero.jpg" alt="Épurer home screen: disk usage ring, Scanner button and detected stacks" width="880">

</div>

<br>

Xcode DerivedData, `node_modules`, package-manager caches, simulators, Docker layers, model downloads… they pile up silently until your disk is full. Épurer finds them, tells you exactly what each one is and how safe it is to remove, and cleans only what you confirm.

## Highlights

- **One-click safe clean.** *Nettoyage sûr* (Safe Clean) removes only caches that rebuild themselves. Anything moderate or risky is left out.
- **Nothing deleted blindly.** Every clean goes through an estimate, a reviewable selection and a confirmation, and *Prévisualiser* (Preview) shows exactly what would be removed before it is.
- **Knows your stack.** Detects Frontend, Backend, Mobile, DevOps, Data/ML and AI tooling and only scans what you actually use.
- **Project-aware.** *Projets* finds build artifacts (`node_modules`, `target`, `.venv`, `Pods`, `.next`…) grouped by project. Your source code is never touched.
- **See where space goes.** *Explorer* measures any folder live, largest first.
- **Protect what matters.** Paths in *Chemins protégés* (Protected Paths) are never scanned or cleaned.
- **User space only.** No admin password, no privileged helper. Épurer only removes what your own user can already delete.
- **French and English.** The app follows your macOS language.

## Install

1. [Download **Épurer.dmg**](https://github.com/0SansNom/epurer/releases/latest/download/Epurer.dmg) from the latest release.
2. Open it and drag **Épurer** to **Applications**.
3. Open Épurer from Applications.

> **First launch.** Épurer is not notarized by Apple yet, so macOS blocks the first launch. Open **System Settings → Privacy & Security**, scroll down and click **Open Anyway** next to the Épurer message, then confirm. You only do this once.
>
> Prefer the terminal? `xattr -dr com.apple.quarantine /Applications/Épurer.app` does the same.

### Full Disk Access (optional)

Without it, Épurer still cleans everything your user owns, but some sizes are shown as minimums (`≥`). To measure everything, enable Épurer in **System Settings → Privacy & Security → Full Disk Access**. The app walks you through it and notices when access is granted.

## The app

| Screen | What it does |
|---|---|
| **Accueil** (Home) | Disk usage at a glance, the *Scanner* button, your last clean and the stacks detected on this Mac. |
| **Scan → Estimation → Sélection → Confirmation** | Scan at *Prudent*, *Standard* or *Agressif* level, see the space per domain, pick items with their safety level, confirm, clean. |
| **Nettoyage sûr** (Safe Clean) | Safe items only, cleaned in one gesture. |
| **Explorer** | Browse any folder sorted by size; reveal, protect or delete (always confirmed). |
| **Projets** (Projects) | Build artifacts grouped by project. Folders untouched for 7 days are preselected. |
| **Chemins protégés** (Protected Paths) | Folders and volumes Épurer must never touch. Drag and drop supported. |

## Supported technologies

| Domain | Examples |
|---|---|
| **Frontend** | npm, yarn, pnpm, Bun, Vite, Webpack, Next.js |
| **Backend** | Python, Java (Maven, Gradle), Go, Rust, PHP, Ruby |
| **Mobile** | Xcode, simulators, Android Studio, Flutter, CocoaPods |
| **DevOps** | Docker, Kubernetes, Terraform, Helm |
| **Data/ML** | Conda, Jupyter, TensorFlow, PyTorch, Hugging Face |
| **AI** | AI coding tool caches |
| **System** | User caches, logs, Homebrew, Trash, iOS backups |

## Safety first

Every item carries a safety level, and the scan level decides which ones you see.

| Level | Typical items | Shown at |
|---|---|---|
| 🟢 **Safe** | Caches and logs that rebuild themselves | all levels |
| 🟡 **Moderate** | Dependencies and build outputs (rebuilt on next build) | Standard, Aggressive |
| 🔴 **Risky** | Backups and data you may want to keep | Aggressive only, never preselected |

What Épurer will never do:

- ask for an administrator password or install a privileged helper;
- remove system-protected paths, your home folder itself, or anything in *Chemins protégés*;
- delete anything without a confirmation in the app, or without `--dry-run` being an option on the CLI.

Deletion is permanent: items are removed, not moved to the Trash. *Prévisualiser* and `--dry-run` exist so you can check first.

## Command line

The same engine is available as the `epurer` CLI, for scripts, CI and remote machines.

```bash
brew tap 0SansNom/tap
brew trust --tap 0sansnom/tap   # once: Homebrew only loads trusted third-party taps
brew install epurer
```

```bash
epurer detect                       # which dev tools are installed
epurer report                       # estimate what can be reclaimed
epurer smart --dry-run              # safe automatic cleanup, preview only
epurer clean --level standard       # clean, with confirmation
epurer purge ~/Projects --dry-run   # old build artifacts, grouped by project
epurer ignore add ~/Work/client     # protect a path for good
```

| Command | Description |
|---|---|
| `detect` | Detect installed development tools |
| `report` | Estimate reclaimable space |
| `clean` | Clean development caches and temporary files |
| `smart` | Safe automatic cleanup based on detected tools |
| `purge` | Find and remove old project build artifacts |
| `ignore` | Manage protected paths (`add`, `remove`, `list`), shared with the app |
| `engine` | JSON protocol used by the macOS app |

Common flags: `--dry-run` (show, delete nothing), `--level conservative|standard|aggressive`, `--domain frontend,backend,mobile,devops,dataml,ai,system`, `--verbose`.

## Build from source

Requires Xcode and Go 1.24+.

```bash
git clone https://github.com/0SansNom/epurer.git && cd epurer
make build        # CLI → bin/epurer
make test         # Go test suite
make app-bundle   # Xcode build → dist/Épurer.app (hardened runtime, ad-hoc signed)
make app-open     # build and launch the app
make dmg          # dist/Epurer.dmg (universal), as attached to releases
```

The app embeds the Go engine as `Contents/Resources/epurer-engine` and talks to it over a small JSON protocol (`epurer engine …`). Swift sources live in `app/Epurer/Sources`, the Xcode project in `app/EpurerApp`.

Notarized distribution requires an Apple Developer account. See `packaging/macos/sign-and-notarize.example.sh`.

## Releasing

Maintainers only. Breaking CLI removals (`ui`, `analyze`) ship as a major version.

```bash
# 1. CHANGELOG: replace Unreleased with today's date under ## [X.Y.Z]
# 2. Commit everything on master (leave local dumps like Epurer-Assets.zip untracked — they are gitignored)

brew install goreleaser          # once
./release.sh v2.0.0              # build, test, tag, make dist/Epurer.dmg
git push origin master --tags

# Prefer gh over a long-lived GITHUB_TOKEN in the shell:
gh auth login                    # or: gh auth refresh -h github.com
export GITHUB_TOKEN="$(gh auth token)"
make release                     # goreleaser release --clean
# → GitHub release + CLI archives + Homebrew tap + Epurer.dmg attached
```

`make release` runs GoReleaser (`.goreleaser.yml`). It needs a token that can write to `0SansNom/epurer` and `0SansNom/homebrew-tap`. Do not commit tokens.

## Contributing

Issues and pull requests are welcome. See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

[MIT](LICENSE). Built with [Cobra](https://github.com/spf13/cobra), [Lip Gloss](https://github.com/charmbracelet/lipgloss) and SwiftUI.
