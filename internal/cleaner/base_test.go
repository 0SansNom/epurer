package cleaner

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/0SansNom/epurer/internal/config"
	"github.com/0SansNom/epurer/internal/ignorelist"
	"github.com/0SansNom/epurer/internal/scantask"
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

// TestSystemCleaner_Scan_RespectsIgnoreList verifies the trash cleaner
// honors the ignore list. Trash is a plain file target, unlike DNS and
// Homebrew, which run a command.
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

func TestCleanTargets_ReportsProgress(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a")
	b := filepath.Join(dir, "b")
	if err := os.WriteFile(a, []byte("abcdefghij"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte("abcdefghijklmnopqrst"), 0o644); err != nil {
		t.Fatal(err)
	}

	var got []scantask.Progress
	targets := []CleanTarget{
		{Path: a, SizeBytes: 10},
		{Path: b, SizeBytes: 20},
	}
	_, err := CleanTargets(context.Background(), targets, true, func(p scantask.Progress) {
		got = append(got, p)
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[1].Items != 2 || got[1].Bytes != 30 || got[1].Path != b {
		t.Fatalf("progress = %#v", got)
	}

	got = nil
	_, err = CleanTargets(context.Background(), []CleanTarget{{Path: "/", SizeBytes: 99}}, false, func(p scantask.Progress) {
		got = append(got, p)
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Bytes != 0 || got[0].Items != 1 {
		t.Fatalf("failed remove progress = %#v", got)
	}
}

func TestCleanTargets_MissingPathFreesZeroBytes(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "already-gone")
	results, err := CleanTargets(context.Background(), []CleanTarget{
		{Path: missing, SizeBytes: 2 << 30},
	}, false, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || !results[0].Success || results[0].BytesFreed != 0 {
		t.Fatalf("missing path result = %#v, want success with 0 bytes freed", results)
	}
}

func TestCleanTargets_RefusesIgnoredPath(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "cache")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	ignore := &ignorelist.List{Paths: []string{dir}}
	results, err := CleanTargets(context.Background(), []CleanTarget{
		{Path: file, SizeBytes: 1},
	}, false, nil, ignore)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Success || results[0].Error == nil {
		t.Fatalf("ignored result = %#v", results)
	}
	if _, err := os.Stat(file); err != nil {
		t.Fatalf("ignored path was touched: %v", err)
	}
}
