package cleaner

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/0SansNom/epurer/internal/config"
	"github.com/0SansNom/epurer/internal/ignorelist"
)

func TestNewAICleaner(t *testing.T) {
	c, err := NewAICleaner()
	if err != nil {
		t.Fatalf("NewAICleaner() returned error: %v", err)
	}
	if c == nil {
		t.Fatal("NewAICleaner() returned nil")
	}
}

func TestAICleaner_Name(t *testing.T) {
	c, _ := NewAICleaner()
	if got := c.Name(); got != "AI" {
		t.Errorf("Name() = %q, want %q", got, "AI")
	}
}

func TestAICleaner_Domain(t *testing.T) {
	c, _ := NewAICleaner()
	if got := c.Domain(); got != config.DomainAI {
		t.Errorf("Domain() = %v, want %v", got, config.DomainAI)
	}
}

func TestAICleaner_Detect(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	c, _ := NewAICleaner()
	ctx := context.Background()

	detected, err := c.Detect(ctx)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if detected {
		t.Error("Detect() = true with no AI tool caches present, want false")
	}

	cursorCache := filepath.Join(home, "Library", "Application Support", "Cursor", "Cache")
	if err := os.MkdirAll(cursorCache, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	detected, err = c.Detect(ctx)
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if !detected {
		t.Error("Detect() = false with a known cache path present, want true")
	}
}

func TestAICleaner_Scan_RespectsIgnoreList(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cursorCache := filepath.Join(home, "Library", "Application Support", "Cursor", "Cache")
	if err := os.MkdirAll(cursorCache, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(cursorCache, "blob.bin"), []byte("cached data"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	ignList, err := ignorelist.Load()
	if err != nil {
		t.Fatalf("ignorelist.Load() error = %v", err)
	}
	if err := ignList.Add(cursorCache); err != nil {
		t.Fatalf("ignList.Add() error = %v", err)
	}

	c, _ := NewAICleaner()
	c.(IgnoreAware).SetIgnoreList(ignList)

	targets, err := c.Scan(context.Background(), config.NewDefaultConfig())
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	for _, tgt := range targets {
		if tgt.Path == cursorCache {
			t.Errorf("Scan() still returned ignored path %q", cursorCache)
		}
	}
}

func TestAICleaner_Clean_DryRun(t *testing.T) {
	c, _ := NewAICleaner()
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	tmpDir := setupTestDir(t)
	defer os.RemoveAll(tmpDir)

	testFile := createTestFile(t, tmpDir, "cache.bin", "cached data")

	targets := []CleanTarget{
		{Path: testFile, Description: "Test AI cache", SizeBytes: 11, Safety: config.Safe},
	}

	results, err := c.Clean(ctx, targets, true)
	if err != nil {
		t.Fatalf("Clean() returned error: %v", err)
	}
	if len(results) != 1 || !results[0].Success {
		t.Fatalf("unexpected results: %+v", results)
	}

	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Error("File was deleted in dry-run mode")
	}
}

func TestAICleaner_Clean_Actual(t *testing.T) {
	c, _ := NewAICleaner()
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	tmpDir := setupTestDir(t)
	defer os.RemoveAll(tmpDir)

	testDir := createTestDir(t, tmpDir, "Cache", map[string]string{
		"blob.bin": "cached data",
	})

	targets := []CleanTarget{
		{Path: testDir, Description: "Test AI cache dir", SizeBytes: 11, Safety: config.Safe},
	}

	results, err := c.Clean(ctx, targets, false)
	if err != nil {
		t.Fatalf("Clean() returned error: %v", err)
	}
	if len(results) != 1 || !results[0].Success {
		t.Fatalf("unexpected results: %+v", results)
	}

	if _, err := os.Stat(testDir); !os.IsNotExist(err) {
		t.Error("Directory was not deleted")
	}
}
