package scantask

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"

	"github.com/0SansNom/epurer/internal/config"
	"github.com/0SansNom/epurer/pkg/utils"
)

// Target is one cleanable item found by a scan task.
type Target struct {
	Path        string
	Description string
	SizeBytes   int64
	Safety      config.SafetyLevel
}

// Progress is the state of a walk after measuring one entry.
type Progress struct {
	Path  string
	Items int
	Bytes int64
}

// Task scans without deleting.
type Task interface {
	Scan(ctx context.Context) ([]Target, error)
}

// Walk measures each top-level entry under Root.
// A nil Predicate keeps every non-empty entry.
// Zero-size entries are visited and then skipped.
type Walk struct {
	Root        string
	Description string
	Safety      config.SafetyLevel
	Predicate   Predicate
	OnProgress  func(Progress)
}

// Scan reads Root's immediate children. A missing root yields no targets.
func (w Walk) Scan(ctx context.Context) ([]Target, error) {
	entries, err := os.ReadDir(w.Root)
	if err != nil {
		if os.IsNotExist(err) || os.IsPermission(err) {
			return nil, nil
		}
		return nil, err
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	var targets []Target
	var items int
	var seen int64

	for _, ent := range entries {
		if err := ctx.Err(); err != nil {
			return targets, err
		}

		path := filepath.Join(w.Root, ent.Name())
		size, err := entrySize(path, ent)
		if err != nil {
			continue
		}

		items++
		seen += size
		if w.OnProgress != nil {
			w.OnProgress(Progress{Path: path, Items: items, Bytes: seen})
		}

		if size == 0 {
			continue
		}

		entry := Entry{Path: path, Name: ent.Name(), Size: size}
		if w.Predicate != nil && w.Predicate.Exclude(entry) {
			continue
		}

		desc := ent.Name()
		if w.Description != "" {
			desc = w.Description + ": " + ent.Name()
		}
		targets = append(targets, Target{
			Path:        path,
			Description: desc,
			SizeBytes:   size,
			Safety:      w.Safety,
		})
	}

	return targets, nil
}

func entrySize(path string, ent os.DirEntry) (int64, error) {
	if ent.IsDir() {
		size, err := utils.GetDirSize(path)
		if err != nil && !errors.Is(err, utils.ErrIncompleteSize) {
			return 0, err
		}
		return size, nil
	}

	info, err := ent.Info()
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// Compound runs tasks in order and concatenates their targets.
// It stops when the context is cancelled or a task returns an error,
// keeping whatever earlier tasks already found.
type Compound struct {
	Tasks []Task
}

// Scan runs each task until one fails or the context is done.
func (c Compound) Scan(ctx context.Context) ([]Target, error) {
	var all []Target
	for _, task := range c.Tasks {
		if err := ctx.Err(); err != nil {
			return all, err
		}

		targets, err := task.Scan(ctx)
		all = append(all, targets...)
		if err != nil {
			return all, err
		}
	}
	return all, nil
}
