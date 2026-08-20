package cleaner

import (
	"context"

	"github.com/0SansNom/epurer/internal/config"
	"github.com/0SansNom/epurer/internal/ignorelist"
)

// CleanTarget represents a single item that can be cleaned
type CleanTarget struct {
	Path        string              // Absolute path to the item
	Description string              // Human-readable description
	SizeBytes   int64               // Size in bytes
	Safety      config.SafetyLevel  // Safety level of this operation
}

// CleanResult represents the outcome of a clean operation
type CleanResult struct {
	Target     CleanTarget // The target that was cleaned
	Success    bool        // Whether the operation succeeded
	BytesFreed int64       // Actual bytes freed (may differ from target size)
	Error      error       // Error if operation failed
}

// Cleaner is the interface that all domain cleaners must implement
type Cleaner interface {
	// Name returns a human-readable name for this cleaner
	Name() string

	// Domain returns the domain this cleaner belongs to
	Domain() config.Domain

	// Detect checks if this cleaner is applicable to the current system
	// For example, NodeModulesCleaner would check if node is installed
	Detect(ctx context.Context) (bool, error)

	// Scan finds all targets that could be cleaned without actually deleting them
	Scan(ctx context.Context, cfg *config.Config) ([]CleanTarget, error)

	// Clean executes the actual cleanup operation on the given targets
	Clean(ctx context.Context, targets []CleanTarget, dryRun bool) ([]CleanResult, error)
}

// IgnoreAware is implemented by cleaners that scan the filesystem and can
// therefore respect a persistent ignore list. Cleaners that only touch fixed,
// well-known paths (e.g. system caches) don't need to implement it.
type IgnoreAware interface {
	SetIgnoreList(l *ignorelist.List)
}

// filterIgnored removes any target whose path is covered by the ignore
// list. It's for cleaners whose targets are fixed, well-known paths rather
// than filesystem scan results - scanner-backed cleaners filter earlier via
// Scanner.SetIgnoreList (which also avoids descending into ignored trees),
// so they don't need this.
func filterIgnored(targets []CleanTarget, ignoreList *ignorelist.List) []CleanTarget {
	if ignoreList == nil {
		return targets
	}

	filtered := make([]CleanTarget, 0, len(targets))
	for _, t := range targets {
		if !ignoreList.IsIgnored(t.Path) {
			filtered = append(filtered, t)
		}
	}

	return filtered
}
