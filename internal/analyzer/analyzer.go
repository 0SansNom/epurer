// Package analyzer lists a directory's immediate children with their sizes,
// for use by an interactive disk-usage explorer.
package analyzer

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/0SansNom/epurer/pkg/utils"
)

// Entry is one immediate child of a scanned directory.
type Entry struct {
	Name  string
	Path  string
	Size  int64
	IsDir bool
	// Incomplete is true when Size is a lower bound, not the true total -
	// some part of this directory couldn't be read (most commonly macOS
	// blocking access to a TCC-protected folder, e.g. Photos Library,
	// without Full Disk Access granted to the terminal).
	Incomplete bool
}

// ListDir returns path's immediate children sorted by size, descending.
// Directory sizes are computed concurrently (bounded worker pool) since a
// naive sequential walk over many large subdirectories would be slow.
func ListDir(path string) ([]Entry, error) {
	return ListDirFunc(path, nil)
}

// ListDirFunc is ListDir that also calls onEntry (if set) as each child is
// measured, with how many are done out of the total. Calls never overlap.
func ListDirFunc(path string, onEntry func(e Entry, done, total int)) ([]Entry, error) {
	dirEntries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	entries := make([]Entry, len(dirEntries))

	var wg sync.WaitGroup
	var mu sync.Mutex
	done := 0
	sem := make(chan struct{}, 8)

	for i, de := range dirEntries {
		wg.Add(1)
		sem <- struct{}{}

		go func(i int, de os.DirEntry) {
			defer wg.Done()
			defer func() { <-sem }()

			full := filepath.Join(path, de.Name())
			isDir := de.IsDir()

			var size int64
			var incomplete bool
			if isDir {
				var sizeErr error
				size, sizeErr = utils.GetDirSize(full)
				incomplete = errors.Is(sizeErr, utils.ErrIncompleteSize)
			} else if info, err := de.Info(); err == nil {
				size = info.Size()
			}

			entries[i] = Entry{Name: de.Name(), Path: full, Size: size, IsDir: isDir, Incomplete: incomplete}
			if onEntry != nil {
				mu.Lock()
				done++
				onEntry(entries[i], done, len(dirEntries))
				mu.Unlock()
			}
		}(i, de)
	}

	wg.Wait()

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Size > entries[j].Size
	})

	return entries, nil
}
