// Package version is the single source of truth for Épurer's own version
// string, used everywhere it's displayed (--version, the report header,
// the TUI header). It's set at build time via -ldflags (see Makefile and
// .goreleaser.yml); left unset it stays "dev" - an honest signal that the
// binary wasn't built through either of those, rather than a stale number.
package version

var Version = "dev"
