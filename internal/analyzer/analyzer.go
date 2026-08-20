// Package analyzer lists a directory's immediate children with their sizes,
// for use by an interactive disk-usage explorer.
package analyzer

import (
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
}

// ListDir returns path's immediate children sorted by size, descending.
// Directory sizes are computed concurrently (bounded worker pool) since a
// naive sequential walk over many large subdirectories would be slow.
func ListDir(path string) ([]Entry, error) {
	dirEntries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	entries := make([]Entry, len(dirEntries))

	var wg sync.WaitGroup
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
			if isDir {
				size, _ = utils.GetDirSize(full)
			} else if info, err := de.Info(); err == nil {
				size = info.Size()
			}

			entries[i] = Entry{Name: de.Name(), Path: full, Size: size, IsDir: isDir}
		}(i, de)
	}

	wg.Wait()

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Size > entries[j].Size
	})

	return entries, nil
}
