package utils

import (
	"os"
	"path/filepath"
	"testing"
)

// =============================================================================
// PathExists Tests
// =============================================================================

func TestPathExists(t *testing.T) {
	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "pathexists-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	// Test existing file
	if !PathExists(tmpPath) {
		t.Errorf("PathExists(%q) = false, want true", tmpPath)
	}

	// Test existing directory
	tmpDir, err := os.MkdirTemp("", "pathexists-dir-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	if !PathExists(tmpDir) {
		t.Errorf("PathExists(%q) = false, want true", tmpDir)
	}

	// Test non-existent path
	nonExistent := "/this/path/definitely/does/not/exist/12345"
	if PathExists(nonExistent) {
		t.Errorf("PathExists(%q) = true, want false", nonExistent)
	}
}

// =============================================================================
// ExpandHome Tests
// =============================================================================

func TestExpandHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get home dir: %v", err)
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Just tilde", "~", home},
		{"Tilde with path", "~/Documents", filepath.Join(home, "Documents")},
		{"Tilde with nested path", "~/foo/bar/baz", filepath.Join(home, "foo/bar/baz")},
		{"No tilde", "/usr/local/bin", "/usr/local/bin"},
		{"Relative path", "relative/path", "relative/path"},
		{"Empty string", "", ""},
		{"Tilde in middle", "/path/~/test", "/path/~/test"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExpandHome(tt.input)
			if err != nil {
				t.Fatalf("ExpandHome(%q) returned error: %v", tt.input, err)
			}
			if result != tt.expected {
				t.Errorf("ExpandHome(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// =============================================================================
// GetDirSize Tests
// =============================================================================

func TestGetDirSize(t *testing.T) {
	// Create a temporary directory with files
	tmpDir, err := os.MkdirTemp("", "getdirsize-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create files with known sizes
	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")
	subDir := filepath.Join(tmpDir, "subdir")
	file3 := filepath.Join(subDir, "file3.txt")

	// Create files
	os.WriteFile(file1, []byte("12345"), 0644)      // 5 bytes
	os.WriteFile(file2, []byte("1234567890"), 0644) // 10 bytes
	os.MkdirAll(subDir, 0755)
	os.WriteFile(file3, []byte("123"), 0644) // 3 bytes

	// Total: 18 bytes
	size, err := GetDirSize(tmpDir)
	if err != nil {
		t.Fatalf("GetDirSize returned error: %v", err)
	}

	expectedSize := int64(18)
	if size != expectedSize {
		t.Errorf("GetDirSize(%q) = %d, want %d", tmpDir, size, expectedSize)
	}
}

func TestGetDirSize_EmptyDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "getdirsize-empty-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	size, err := GetDirSize(tmpDir)
	if err != nil {
		t.Fatalf("GetDirSize returned error: %v", err)
	}

	if size != 0 {
		t.Errorf("GetDirSize(empty dir) = %d, want 0", size)
	}
}

func TestGetDirSize_SingleFile(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "getdirsize-file-*")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	content := "Hello, World!"
	os.WriteFile(tmpPath, []byte(content), 0644)

	size, err := GetDirSize(tmpPath)
	if err != nil {
		t.Fatalf("GetDirSize returned error: %v", err)
	}

	expectedSize := int64(len(content))
	if size != expectedSize {
		t.Errorf("GetDirSize(file) = %d, want %d", size, expectedSize)
	}
}

// =============================================================================
// SafeRemove Tests
// =============================================================================

func TestSafeRemove_DryRun(t *testing.T) {
	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "saferemove-dryrun-*")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	// Call SafeRemove with dryRun=true
	err = SafeRemove(tmpPath, true)
	if err != nil {
		t.Errorf("SafeRemove(dryRun=true) returned error: %v", err)
	}

	// File should still exist
	if !PathExists(tmpPath) {
		t.Error("SafeRemove(dryRun=true) deleted the file")
	}
}

func TestSafeRemove_Actual(t *testing.T) {
	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "saferemove-actual-*")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()

	// Call SafeRemove with dryRun=false
	err = SafeRemove(tmpPath, false)
	if err != nil {
		t.Errorf("SafeRemove(dryRun=false) returned error: %v", err)
	}

	// File should be deleted
	if PathExists(tmpPath) {
		t.Error("SafeRemove(dryRun=false) did not delete the file")
		os.Remove(tmpPath) // Clean up
	}
}

func TestSafeRemove_Directory(t *testing.T) {
	// Create a temporary directory with contents
	tmpDir, err := os.MkdirTemp("", "saferemove-dir-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Add some files
	os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("test"), 0644)
	os.MkdirAll(filepath.Join(tmpDir, "subdir"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "subdir", "file2.txt"), []byte("test"), 0644)

	// Call SafeRemove with dryRun=false
	err = SafeRemove(tmpDir, false)
	if err != nil {
		t.Errorf("SafeRemove(dir, dryRun=false) returned error: %v", err)
	}

	// Directory should be deleted
	if PathExists(tmpDir) {
		t.Error("SafeRemove did not delete the directory")
		os.RemoveAll(tmpDir) // Clean up
	}
}

func TestSafeRemove_NonExistent(t *testing.T) {
	// SafeRemove on non-existent path should not error (RemoveAll behavior)
	err := SafeRemove("/nonexistent/path/12345", false)
	if err != nil {
		t.Errorf("SafeRemove(nonexistent) returned error: %v", err)
	}
}

// =============================================================================
// CommandExists Tests
// =============================================================================

func TestCommandExists(t *testing.T) {
	tests := []struct {
		name     string
		cmd      string
		expected bool
	}{
		{"ls command", "ls", true},           // Should exist on all Unix systems
		{"cat command", "cat", true},         // Should exist on all Unix systems
		{"nonexistent", "nonexistent12345", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CommandExists(tt.cmd)
			if result != tt.expected {
				t.Errorf("CommandExists(%q) = %v, want %v", tt.cmd, result, tt.expected)
			}
		})
	}
}

// =============================================================================
// IsWritable Tests
// =============================================================================

func TestIsWritable_WritableDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "iswritable-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	if !IsWritable(tmpDir) {
		t.Errorf("IsWritable(%q) = false, want true", tmpDir)
	}
}

func TestIsWritable_NonExistent(t *testing.T) {
	nonExistent := "/nonexistent/path/12345"
	if IsWritable(nonExistent) {
		t.Errorf("IsWritable(%q) = true, want false", nonExistent)
	}
}

func TestIsWritable_File(t *testing.T) {
	// Create a temporary directory with a file
	tmpDir, err := os.MkdirTemp("", "iswritable-file-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tmpFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(tmpFile, []byte("test"), 0644)

	// IsWritable on a file should check the parent directory
	result := IsWritable(tmpFile)
	// Since the parent dir is writable, this should return true
	if !result {
		t.Errorf("IsWritable(%q) = false, want true", tmpFile)
	}
}

func TestIsWritable_ReadOnlyDir(t *testing.T) {
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "iswritable-readonly-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		// Restore permissions before cleanup
		os.Chmod(tmpDir, 0755)
		os.RemoveAll(tmpDir)
	}()

	// Make it read-only
	if err := os.Chmod(tmpDir, 0555); err != nil {
		t.Skipf("Could not set read-only permissions: %v", err)
	}

	if IsWritable(tmpDir) {
		t.Errorf("IsWritable(read-only dir) = true, want false")
	}
}
