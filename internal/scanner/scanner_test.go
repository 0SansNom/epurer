package scanner

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewScanner(t *testing.T) {
	scanner, err := NewScanner()
	if err != nil {
		t.Fatalf("Failed to create scanner: %v", err)
	}

	if scanner == nil {
		t.Fatal("Scanner is nil")
	}

	if scanner.workers != 4 {
		t.Errorf("Expected 4 workers, got %d", scanner.workers)
	}
}

func TestSetWorkers(t *testing.T) {
	scanner, _ := NewScanner()

	scanner.SetWorkers(8)
	if scanner.workers != 8 {
		t.Errorf("Expected 8 workers, got %d", scanner.workers)
	}

	// Test invalid value (should be ignored)
	scanner.SetWorkers(0)
	if scanner.workers != 8 {
		t.Errorf("Expected workers to remain 8, got %d", scanner.workers)
	}
}

func TestFindExactPath(t *testing.T) {
	scanner, _ := NewScanner()

	// Test with a known file (go.mod should exist in the project root)
	result, err := scanner.FindExactPath("../../go.mod")
	if err != nil {
		t.Fatalf("Failed to find go.mod: %v", err)
	}

	if result.Size == 0 {
		t.Error("Expected non-zero size for go.mod")
	}

	if result.Path == "" {
		t.Error("Expected non-empty path")
	}
}

func TestFindByPattern(t *testing.T) {
	scanner, _ := NewScanner()

	// Create a temporary directory with test files
	tmpDir, err := os.MkdirTemp("", "scanner-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files
	testFiles := []string{
		filepath.Join(tmpDir, "test1.txt"),
		filepath.Join(tmpDir, "test2.txt"),
		filepath.Join(tmpDir, "other.log"),
	}

	for _, file := range testFiles {
		if err := os.WriteFile(file, []byte("test content"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Create scanner with custom search dir
	scanner, _ = NewScannerWithDirs([]string{tmpDir})

	// Test pattern matching
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	results := scanner.FindByPattern(ctx, "*.txt")

	count := 0
	for result := range results {
		if result.Err != nil {
			t.Errorf("Error scanning: %v", result.Err)
			continue
		}
		count++
	}

	if count != 2 {
		t.Errorf("Expected 2 .txt files, found %d", count)
	}
}

func TestAddSearchDir(t *testing.T) {
	scanner, _ := NewScanner()

	initialCount := len(scanner.GetSearchDirs())

	tmpDir, err := os.MkdirTemp("", "scanner-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	err = scanner.AddSearchDir(tmpDir)
	if err != nil {
		t.Errorf("Failed to add search dir: %v", err)
	}

	if len(scanner.GetSearchDirs()) != initialCount+1 {
		t.Errorf("Expected %d search dirs, got %d", initialCount+1, len(scanner.GetSearchDirs()))
	}
}
