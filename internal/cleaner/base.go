package cleaner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/0SansNom/epurer/internal/config"
	"github.com/0SansNom/epurer/internal/ignorelist"
	"github.com/0SansNom/epurer/internal/scantask"
	"github.com/0SansNom/epurer/pkg/utils"
)

// CleanTarget represents a single item that can be cleaned
type CleanTarget struct {
	Path        string             // Absolute path to the item
	Description string             // Human-readable description
	SizeBytes   int64              // Size in bytes
	Safety      config.SafetyLevel // Safety level of this operation
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

// IgnoreAware is implemented by cleaners that can skip paths on a persistent
// ignore list.
type IgnoreAware interface {
	SetIgnoreList(l *ignorelist.List)
}

// ProgressAware is implemented by cleaners that report walk progress.
type ProgressAware interface {
	SetProgress(func(scantask.Progress))
}

// fsCleaner carries the ignore list and walk progress shared by filesystem cleaners.
type fsCleaner struct {
	ignore   *ignorelist.List
	progress func(scantask.Progress)
}

// SetIgnoreList makes walks and the final filter respect a persistent ignore list.
func (f *fsCleaner) SetIgnoreList(l *ignorelist.List) {
	f.ignore = l
}

// SetProgress receives walk progress while this cleaner scans.
// A nil function disables progress reporting.
func (f *fsCleaner) SetProgress(fn func(scantask.Progress)) {
	f.progress = fn
}

func (f *fsCleaner) collect(ctx context.Context, walks ...scantask.Walk) ([]CleanTarget, error) {
	return collectWalks(ctx, f.ignore, f.progress, walks...)
}

// filterIgnored removes any target whose path is covered by the ignore list.
// Walks already apply that list while scanning. This is for targets that
// were not produced by a walk.
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

// CleanTargets removes each target from disk (or, in dry-run mode, just
// reports what would be freed) and returns one CleanResult per target. It's
// the shared implementation for cleaners whose targets are plain
// files/directories with no special handling; a cleaner with special-cased
// targets (e.g. DevOpsCleaner's "docker:" pseudo-paths, or the DNS and
// Homebrew commands) implements Clean itself instead of calling this.
//
// Paths covered by ignore are refused (left in place), even if the host
// supplied them after scan-time filtering.
func CleanTargets(ctx context.Context, targets []CleanTarget, dryRun bool, progress func(scantask.Progress), ignore *ignorelist.List) ([]CleanResult, error) {
	results := make([]CleanResult, 0, len(targets))
	var freed int64

	for _, target := range targets {
		result := CleanResult{
			Target:  target,
			Success: true,
		}

		if ignore != nil && ignore.IsIgnored(target.Path) {
			result.Success = false
			result.Error = fmt.Errorf("%w: on the ignore list", utils.ErrNotRemovable)
		} else if !utils.CanRemove(target.Path) {
			result.Success = false
			result.Error = utils.ErrNotRemovable
		} else if !dryRun {
			// Count bytes only when something was actually there. A missing
			// path with a writable parent is removable (idempotent), but it
			// did not free the scanned size again.
			existed := utils.PathExists(target.Path)
			if err := utils.SafeRemove(target.Path, false); err != nil {
				result.Success = false
				result.Error = err
			} else if existed {
				result.BytesFreed = target.SizeBytes
			}
		} else {
			result.BytesFreed = target.SizeBytes
		}

		if result.Success {
			freed += result.BytesFreed
		}
		noteCleanProgress(progress, target.Path, len(results)+1, freed)
		results = append(results, result)

		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}
	}

	return results, nil
}

// siblingExists reports whether any name sits next to path.
// A directory named build or target is only a build output when the project file is there.
func siblingExists(path string, names ...string) bool {
	parent := filepath.Dir(path)
	for _, name := range names {
		if _, err := os.Stat(filepath.Join(parent, name)); err == nil {
			return true
		}
	}
	return false
}

func noteCleanProgress(progress func(scantask.Progress), path string, done int, freed int64) {
	if progress == nil {
		return
	}
	progress(scantask.Progress{Path: path, Items: done, Bytes: freed})
}

// collectWalks runs each walk as one compound task and maps the results
// onto clean targets. A missing directory contributes nothing. An empty
// predicate becomes the ignore list, and an empty progress hook is filled in.
func collectWalks(ctx context.Context, ignore *ignorelist.List, progress func(scantask.Progress), walks ...scantask.Walk) ([]CleanTarget, error) {
	tasks := make([]scantask.Task, 0, len(walks))
	for i := range walks {
		if walks[i].Predicate == nil {
			walks[i].Predicate = scantask.Ignored{List: ignore}
		}
		if walks[i].OnProgress == nil {
			walks[i].OnProgress = progress
		}
		tasks = append(tasks, walks[i])
	}

	found, err := (scantask.Compound{Tasks: tasks}).Scan(ctx)
	if err != nil {
		return nil, err
	}

	targets := make([]CleanTarget, 0, len(found))
	for _, item := range found {
		targets = append(targets, CleanTarget{
			Path:        item.Path,
			Description: item.Description,
			SizeBytes:   item.SizeBytes,
			Safety:      item.Safety,
		})
	}
	return targets, nil
}
