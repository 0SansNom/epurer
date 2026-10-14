package scanner

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	"github.com/0SansNom/epurer/internal/ignorelist"
	"github.com/0SansNom/epurer/pkg/utils"
)

// Scanner scans the filesystem concurrently for patterns
type Scanner struct {
	workers    int
	homePath   string
	searchDirs []string // Directories to search in (e.g., ~/Projects, ~/Code)
	ignoreList *ignorelist.List
}

// SetIgnoreList sets the ignore list this scanner should respect. Paths
// covered by the list are skipped and not descended into.
func (s *Scanner) SetIgnoreList(l *ignorelist.List) {
	s.ignoreList = l
}

// ScanResult contains a found path and its size.
// Pattern is the glob that matched the base name, when the search used one.
type ScanResult struct {
	Path    string
	Pattern string
	Size    int64
	Err     error
}

// NewScanner creates a new Scanner with default configuration
func NewScanner() (*Scanner, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	// Default search directories - common project locations
	searchDirs := []string{
		filepath.Join(home, "Projects"),
		filepath.Join(home, "Code"),
		filepath.Join(home, "Development"),
		filepath.Join(home, "Developer"),
		filepath.Join(home, "GitHub"),
		filepath.Join(home, "Workspace"),
		filepath.Join(home, "dev"),
		filepath.Join(home, "Documents"),
		filepath.Join(home, "Desktop"),
	}

	// Filter to only existing directories
	var existingDirs []string
	for _, dir := range searchDirs {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			existingDirs = append(existingDirs, dir)
		}
	}

	return &Scanner{
		workers:    4, // Number of concurrent workers
		homePath:   home,
		searchDirs: existingDirs,
	}, nil
}

// NewScannerWithDirs creates a Scanner with custom search directories
func NewScannerWithDirs(dirs []string) (*Scanner, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	return &Scanner{
		workers:    4,
		homePath:   home,
		searchDirs: dirs,
	}, nil
}

// SetWorkers sets the number of concurrent workers
func (s *Scanner) SetWorkers(n int) {
	if n > 0 {
		s.workers = n
	}
}

// FindByPattern searches for all files and directories matching pattern.
// Pattern is a base-name glob such as "node_modules" or "*.log".
func (s *Scanner) FindByPattern(ctx context.Context, pattern string) <-chan ScanResult {
	return s.FindByPatterns(ctx, []string{pattern})
}

// FindByPatterns walks each search directory once and matches every pattern.
func (s *Scanner) FindByPatterns(ctx context.Context, patterns []string) <-chan ScanResult {
	results := make(chan ScanResult, 100)

	go func() {
		defer close(results)

		var wg sync.WaitGroup
		semaphore := make(chan struct{}, s.workers)

		for _, dir := range s.searchDirs {
			select {
			case <-ctx.Done():
				return
			default:
			}

			wg.Add(1)
			semaphore <- struct{}{}

			go func(searchDir string) {
				defer wg.Done()
				defer func() { <-semaphore }()

				s.walkAndMatch(ctx, searchDir, patterns, results)
			}(dir)
		}

		wg.Wait()
	}()

	return results
}

// FindExactPath checks a single specific path and returns its size
func (s *Scanner) FindExactPath(path string) (ScanResult, error) {
	info, err := os.Stat(path)
	if err != nil {
		return ScanResult{Path: path, Err: err}, err
	}

	var size int64
	if info.IsDir() {
		size, err = s.calculateDirSize(path)
	} else {
		size = info.Size()
	}

	return ScanResult{
		Path: path,
		Size: size,
		Err:  err,
	}, err
}

// walkAndMatch walks a directory tree and sends matching paths to results.
func (s *Scanner) walkAndMatch(ctx context.Context, searchDir string, patterns []string, results chan<- ScanResult) {
	filepath.WalkDir(searchDir, func(path string, d fs.DirEntry, err error) error {
		select {
		case <-ctx.Done():
			return filepath.SkipAll
		default:
		}

		if err != nil {
			return nil
		}

		if s.ignoreList != nil && s.ignoreList.IsIgnored(path) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		baseName := filepath.Base(path)
		var matched string
		for _, pattern := range patterns {
			ok, err := filepath.Match(pattern, baseName)
			if err != nil || !ok {
				continue
			}
			matched = pattern
			break
		}
		if matched == "" {
			return nil
		}

		size := int64(0)
		if d.IsDir() {
			size, _ = s.calculateDirSize(path)
		} else if info, err := d.Info(); err == nil {
			size = info.Size()
		}

		select {
		case results <- ScanResult{Path: path, Pattern: matched, Size: size}:
		case <-ctx.Done():
			return filepath.SkipAll
		}

		if d.IsDir() {
			return filepath.SkipDir
		}
		return nil
	})
}

// calculateDirSize calculates the total size of a directory recursively
func (s *Scanner) calculateDirSize(path string) (int64, error) {
	return utils.GetDirSize(path)
}

// GetSearchDirs returns the directories that will be searched
func (s *Scanner) GetSearchDirs() []string {
	return s.searchDirs
}

// AddSearchDir adds a directory to the search list
func (s *Scanner) AddSearchDir(dir string) error {
	info, err := os.Stat(dir)
	if err != nil {
		return err
	}

	if !info.IsDir() {
		return fs.ErrInvalid
	}

	s.searchDirs = append(s.searchDirs, dir)
	return nil
}
