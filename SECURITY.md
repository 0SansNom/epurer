# Security

Épurer is a **user-space** macOS cleaner. It never installs a privileged helper,
never prompts for an admin password, and only deletes what the current user
can already unlink.

## Threat model

| Surface | Policy |
|---|---|
| Privilege | Empty entitlements; Hardened Runtime. Full Disk Access is a user TCC grant, not an entitlement. |
| App Sandbox | **Not enabled.** A cache cleaner that must reach DerivedData, Docker, and similar paths cannot live inside the App Sandbox. |
| Deletion | `CanRemove` / `SafeRemove` refuse protected roots (`/`, `/System`, `$HOME` exact, …), relative paths, pseudo targets as files, and sticky-directory names not owned by this UID. |
| Ignore list | `~/Library/Application Support/epurer/ignore.json`. Corrupt or unreadable config **fails closed** (clean/scan abort). Paths on the list are refused at clean time for every cleaner, including host-supplied `engine clean` requests. |
| Engine IPC | The app launches the bundled `epurer-engine` via `Process` + argv / JSON stdin — no shell. `EPURER_ENGINE` is honored only in **Debug** builds. |
| CLI `smart` | Requires confirmation unless `--yes` or `--dry-run`. |
| Install | `install.sh` verifies the release SHA-256 checksums file before extracting. |

## Trust boundary

Anyone who can run processes as your macOS user can already delete the same
files. The engine assumes a same-user host (the Swift app or a local CLI).
Do not expose `epurer engine` over the network.

## Reporting

Please open a private security advisory or mail the maintainers via GitHub
if you find a vulnerability that bypasses `SafeRemove`, the ignore list, or
confirmation gates.
