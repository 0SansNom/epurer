package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProbeDiskAccess(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root - permission bits don't block access")
	}

	dir := t.TempDir()
	readable := filepath.Join(dir, "readable")
	if err := os.WriteFile(readable, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	locked := filepath.Join(dir, "locked")
	if err := os.WriteFile(locked, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(locked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(locked, 0o644) })
	missing := filepath.Join(dir, "missing")

	if got := probeDiskAccess([]string{missing}); got != DiskAccessUnknown {
		t.Errorf("missing = %v, want unknown", got)
	}
	if got := probeDiskAccess([]string{locked}); got != DiskAccessDenied {
		t.Errorf("locked = %v, want denied", got)
	}
	if got := probeDiskAccess([]string{locked, readable}); got != DiskAccessGranted {
		t.Errorf("locked then readable = %v, want granted", got)
	}
	if got := probeDiskAccess([]string{readable, locked}); got != DiskAccessGranted {
		t.Errorf("readable = %v, want granted", got)
	}
}
