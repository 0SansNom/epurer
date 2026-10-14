# Developing Épurer

Everything technical that does not belong in the user README: the command line, building from source and releasing.

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
epurer smart --yes                  # smart clean without the confirm prompt
epurer clean --level standard       # clean, with confirmation
epurer purge ~/Projects --dry-run   # old build artifacts, grouped by project
epurer ignore add ~/Work/client     # protect a path for good
```

| Command | Description |
|---|---|
| `detect` | Detect installed development tools |
| `report` | Estimate reclaimable space |
| `clean` | Clean development caches and temporary files |
| `smart` | Safe automatic cleanup (asks before delete; `--yes` to skip) |
| `purge` | Find and remove old project build artifacts |
| `ignore` | Manage protected paths (`add`, `remove`, `list`), shared with the app |
| `engine` | JSON protocol used by the macOS app |

Common flags: `--dry-run` (show, delete nothing), `--level conservative|standard|aggressive`, `--domain frontend,backend,mobile,devops,dataml,ai,system`, `--verbose`, `--yes` (skip confirm on `smart` / `purge`).

CLI install via [`install.sh`](../install.sh) downloads the latest GitHub release and verifies its SHA-256 checksums before extracting. See [SECURITY.md](../SECURITY.md) for the threat model.

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

# Prefer gh over a long-lived personal token in the shell:
gh auth login                    # or: gh auth refresh -h github.com
export GITHUB_TOKEN="$(gh auth token)"
make release                     # goreleaser release --clean
# → GitHub release + CLI archives + Homebrew formula + Epurer.dmg attached
```

`make release` runs GoReleaser (`.goreleaser.yml`, `version: 2`). It needs a token that can write to `0SansNom/epurer` and `0SansNom/homebrew-tap`. Do not commit tokens.

## Contributing

Issues and pull requests are welcome. See [CONTRIBUTING.md](../CONTRIBUTING.md).
