package cleaner

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/0SansNom/epurer/internal/config"
	"github.com/0SansNom/epurer/internal/ignorelist"
)

func TestFilterIgnored_NilListReturnsAll(t *testing.T) {
	targets := []CleanTarget{{Path: "/a"}, {Path: "/b"}}
	got := filterIgnored(targets, nil)
	if len(got) != 2 {
		t.Errorf("filterIgnored(nil list) = %d targets, want all 2 untouched", len(got))
	}
}

func TestFilterIgnored_RemovesIgnoredPaths(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	ignList, err := ignorelist.Load()
	if err != nil {
		t.Fatalf("ignorelist.Load() error = %v", err)
	}
	if err := ignList.Add("/a"); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	targets := []CleanTarget{{Path: "/a"}, {Path: "/b"}}
	got := filterIgnored(targets, ignList)

	if len(got) != 1 || got[0].Path != "/b" {
		t.Errorf("filterIgnored() = %+v, want only /b", got)
	}
}

// TestSystemCleaner_Scan_RespectsIgnoreList verifies SystemCleaner - which
// scans fixed, well-known paths rather than a filesystem walk - still
// honors the ignore list. Trash is used because it's a plain SafeRemove
// target (unlike DNS/Homebrew, which use commands and have no path to
// ignore).
func TestSystemCleaner_Scan_RespectsIgnoreList(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	trashPath := filepath.Join(home, ".Trash")
	if err := os.MkdirAll(trashPath, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(trashPath, "deleted.txt"), []byte("gone"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	ignList, err := ignorelist.Load()
	if err != nil {
		t.Fatalf("ignorelist.Load() error = %v", err)
	}
	if err := ignList.Add(trashPath); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	c := NewTrashCleaner()
	c.(IgnoreAware).SetIgnoreList(ignList)

	targets, err := c.Scan(context.Background(), config.NewDefaultConfig())
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	for _, tgt := range targets {
		if tgt.Path == trashPath {
			t.Errorf("Scan() still returned ignored path %q", trashPath)
		}
	}
}
