package analyzer

import (
	"os"
	"path/filepath"
	"testing"
)

// TestListDir_MarksIncompleteSizeOnUnreadableChild verifies that a child
// directory epurer can't fully read (e.g. blocked by macOS's TCC
// protections without Full Disk Access) is flagged Incomplete instead of
// silently reporting a misleadingly small size.
func TestListDir_MarksIncompleteSizeOnUnreadableChild(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root - permission bits don't block access")
	}

	dir := t.TempDir()
	locked := filepath.Join(dir, "locked")
	if err := os.Mkdir(locked, 0755); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(locked, "secret.txt"), []byte("hidden content"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := os.Chmod(locked, 0); err != nil {
		t.Fatalf("Chmod(0) error = %v", err)
	}
	defer os.Chmod(locked, 0755)

	entries, err := ListDir(dir)
	if err != nil {
		t.Fatalf("ListDir() error = %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
	if !entries[0].Incomplete {
		t.Error("Incomplete = false, want true for a directory with an unreadable child")
	}
}

func TestListDir_SortsBySizeDescending(t *testing.T) {
	dir := t.TempDir()

	mustWrite(t, filepath.Join(dir, "small.txt"), 10)
	mustWrite(t, filepath.Join(dir, "big.txt"), 1000)
	mustWrite(t, filepath.Join(dir, "medium.txt"), 100)

	entries, err := ListDir(dir)
	if err != nil {
		t.Fatalf("ListDir() error = %v", err)
	}

	if len(entries) != 3 {
		t.Fatalf("got %d entries, want 3", len(entries))
	}

	for i := 1; i < len(entries); i++ {
		if entries[i-1].Size < entries[i].Size {
			t.Errorf("entries not sorted descending: %+v", entries)
		}
	}

	if entries[0].Name != "big.txt" {
		t.Errorf("largest entry = %q, want big.txt", entries[0].Name)
	}
}

func TestListDir_ComputesDirSize(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "subdir")

	mustWrite(t, filepath.Join(sub, "a.txt"), 50)
	mustWrite(t, filepath.Join(sub, "b.txt"), 50)

	entries, err := ListDir(dir)
	if err != nil {
		t.Fatalf("ListDir() error = %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
	if !entries[0].IsDir {
		t.Errorf("IsDir = false, want true")
	}
	if entries[0].Size != 100 {
		t.Errorf("Size = %d, want 100", entries[0].Size)
	}
}

func mustWrite(t *testing.T, path string, size int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(path, make([]byte, size), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}
