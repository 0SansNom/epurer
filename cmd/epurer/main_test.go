package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/0SansNom/epurer/internal/cleaner"
	"github.com/0SansNom/epurer/internal/config"
	"github.com/0SansNom/epurer/internal/ignorelist"
)

// fakeCleaner is a minimal cleaner.Cleaner used to test the orchestration
// helpers in main.go without touching the real filesystem/system tools.
type fakeCleaner struct {
	name   string
	domain config.Domain
}

func (f *fakeCleaner) Name() string          { return f.name }
func (f *fakeCleaner) Domain() config.Domain { return f.domain }
func (f *fakeCleaner) Detect(ctx context.Context) (bool, error) {
	return true, nil
}
func (f *fakeCleaner) Scan(ctx context.Context, cfg *config.Config) ([]cleaner.CleanTarget, error) {
	return nil, nil
}
func (f *fakeCleaner) Clean(ctx context.Context, targets []cleaner.CleanTarget, dryRun bool) ([]cleaner.CleanResult, error) {
	return nil, nil
}

// fakeIgnoreAwareCleaner additionally records whether SetIgnoreList was called.
type fakeIgnoreAwareCleaner struct {
	fakeCleaner
	ignoreList *ignorelist.List
}

func (f *fakeIgnoreAwareCleaner) SetIgnoreList(l *ignorelist.List) {
	f.ignoreList = l
}

func TestToLower(t *testing.T) {
	tests := []struct{ in, want string }{
		{"Frontend", "frontend"},
		{"AI", "ai"},
		{"already-lower", "already-lower"},
		{"", ""},
		{"Mixed123CASE", "mixed123case"},
	}
	for _, tt := range tests {
		if got := toLower(tt.in); got != tt.want {
			t.Errorf("toLower(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestContainsString(t *testing.T) {
	tests := []struct {
		s, substr string
		want      bool
	}{
		{"frontend", "front", true},
		{"frontend", "end", true},
		{"frontend", "frontend", true},
		{"frontend", "xyz", false},
		{"frontend", "", false},
		{"", "x", false},
		{"short", "muchlongersubstring", false},
	}
	for _, tt := range tests {
		if got := containsString(tt.s, tt.substr); got != tt.want {
			t.Errorf("containsString(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.want)
		}
	}
}

func TestMatchesDomain(t *testing.T) {
	tests := []struct {
		cleanerName, requestedDomain string
		want                         bool
	}{
		{"frontend", "frontend", true},
		{"ai", "ai", true},
		{"data/ml", "dataml", true},
		{"data/ml", "data/ml", true},
		{"trash", "system", true},
		{"homebrew", "system", true},
		{"xcode deriveddata", "mobile", true},
		{"frontend", "backend", false},
		{"mobile", "devops", false},
	}
	for _, tt := range tests {
		if got := matchesDomain(tt.cleanerName, tt.requestedDomain); got != tt.want {
			t.Errorf("matchesDomain(%q, %q) = %v, want %v", tt.cleanerName, tt.requestedDomain, got, tt.want)
		}
	}
}

func TestFilterCleanersByDomain_EmptyReturnsAll(t *testing.T) {
	cleaners := []cleaner.Cleaner{
		&fakeCleaner{name: "Frontend"},
		&fakeCleaner{name: "Backend"},
	}
	got := filterCleanersByDomain(cleaners, nil)
	if len(got) != 2 {
		t.Errorf("got %d cleaners, want all 2 when no domain filter is given", len(got))
	}
}

func TestFilterCleanersByDomain_FiltersByRequestedDomain(t *testing.T) {
	cleaners := []cleaner.Cleaner{
		&fakeCleaner{name: "Frontend"},
		&fakeCleaner{name: "Backend"},
		&fakeCleaner{name: "AI"},
	}
	got := filterCleanersByDomain(cleaners, []string{"frontend"})
	if len(got) != 1 || got[0].Name() != "Frontend" {
		t.Errorf("filterCleanersByDomain(frontend) = %+v, want only Frontend", got)
	}
}

func TestApplyIgnoreList_OnlyAppliesToIgnoreAwareCleaners(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	aware := &fakeIgnoreAwareCleaner{fakeCleaner: fakeCleaner{name: "Frontend"}}
	unaware := &fakeCleaner{name: "System"}

	if err := applyIgnoreList([]cleaner.Cleaner{aware, unaware}); err != nil {
		t.Fatalf("applyIgnoreList() error = %v", err)
	}

	if aware.ignoreList == nil {
		t.Error("expected the ignore-aware cleaner to receive a non-nil ignore list")
	}
}

func TestApplyIgnoreList_CorruptFileFailsClosed(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := filepath.Join(home, "Library", "Application Support", "epurer")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ignore.json"), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}

	aware := &fakeIgnoreAwareCleaner{fakeCleaner: fakeCleaner{name: "Frontend"}}
	if err := applyIgnoreList([]cleaner.Cleaner{aware}); err == nil {
		t.Fatal("expected corrupt ignore.json to fail closed")
	}
	if aware.ignoreList != nil {
		t.Fatal("cleaner must not receive a list when load fails")
	}
}
