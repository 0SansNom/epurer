package cleaner

import (
	"context"

	"github.com/0SansNom/mac-dev-clean/internal/config"
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
