package scanner

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
)

// Scanner scans the filesystem concurrently for patterns
type Scanner struct {
	workers    int
	homePath   string
	searchDirs []string // Directories to search in (e.g., ~/Projects, ~/Code)
}

// ScanResult contains a found path and its size
type ScanResult struct {
	Path string
	Size int64
	Err  error
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

// FindByPattern searches for all files/directories matching the pattern
// Pattern can be:
// - A glob pattern like "node_modules" or "*.log"
// - A path pattern like "**/node_modules" for recursive search
func (s *Scanner) FindByPattern(ctx context.Context, pattern string) <-chan ScanResult {
	results := make(chan ScanResult, 100) // Buffered channel for better performance

	go func() {
		defer close(results)

		var wg sync.WaitGroup
		semaphore := make(chan struct{}, s.workers) // Limit concurrent workers

		for _, dir := range s.searchDirs {
			select {
			case <-ctx.Done():
				return
			default:
			}

			wg.Add(1)
			semaphore <- struct{}{} // Acquire

			go func(searchDir string) {
				defer wg.Done()
				defer func() { <-semaphore }() // Release

				s.walkAndMatch(ctx, searchDir, pattern, results)
			}(dir)
		}

		wg.Wait()
	}()

	return results
}

// FindByPatternInDir searches for pattern in a specific directory
func (s *Scanner) FindByPatternInDir(ctx context.Context, dir, pattern string) <-chan ScanResult {
	results := make(chan ScanResult, 100)

	go func() {
		defer close(results)
		s.walkAndMatch(ctx, dir, pattern, results)
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

// walkAndMatch walks a directory tree and sends matching paths to results
func (s *Scanner) walkAndMatch(ctx context.Context, searchDir, pattern string, results chan<- ScanResult) {
	filepath.WalkDir(searchDir, func(path string, d fs.DirEntry, err error) error {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return filepath.SkipAll
		default:
		}

		// Skip if we can't access this path
		if err != nil {
			// Don't send errors for permission issues, just skip
			return nil
		}

		// Check if the base name matches the pattern
		baseName := filepath.Base(path)
		matched, err := filepath.Match(pattern, baseName)
		if err != nil {
			return nil // Invalid pattern, skip
		}

		if matched {
			size := int64(0)

			// Calculate size
			if d.IsDir() {
				size, _ = s.calculateDirSize(path)
			} else {
				if info, err := d.Info(); err == nil {
					size = info.Size()
				}
			}

			// Send result
			select {
			case results <- ScanResult{Path: path, Size: size}:
			case <-ctx.Done():
				return filepath.SkipAll
			}

			// Don't descend into matched directories
			if d.IsDir() {
				return filepath.SkipDir
			}
		}

		return nil
	})
}

// calculateDirSize calculates the total size of a directory recursively
func (s *Scanner) calculateDirSize(path string) (int64, error) {
	var size int64
	var mu sync.Mutex

	// Use a semaphore to limit concurrent goroutines
	semaphore := make(chan struct{}, 10)
	var wg sync.WaitGroup

	err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			// Continue despite errors (permission issues, etc.)
			return nil
		}

		if !info.IsDir() {
			wg.Add(1)
			semaphore <- struct{}{}

			go func(fileSize int64) {
				defer wg.Done()
				defer func() { <-semaphore }()

				mu.Lock()
				size += fileSize
				mu.Unlock()
			}(info.Size())
		}

		return nil
	})

	wg.Wait()
	return size, err
}

// FindMultiplePatterns searches for multiple patterns concurrently
func (s *Scanner) FindMultiplePatterns(ctx context.Context, patterns []string) map[string]<-chan ScanResult {
	results := make(map[string]<-chan ScanResult)

	for _, pattern := range patterns {
		results[pattern] = s.FindByPattern(ctx, pattern)
	}

	return results
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
