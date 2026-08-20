package ignorelist

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func newTestList(t *testing.T) *List {
	t.Helper()
	dir := t.TempDir()
	return &List{path: filepath.Join(dir, "ignore.json")}
}

func TestAdd_DedupesAndPersists(t *testing.T) {
	l := newTestList(t)
	dir := t.TempDir()

	if err := l.Add(dir); err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if err := l.Add(dir); err != nil {
		t.Fatalf("Add() (dup) error = %v", err)
	}

	if len(l.Paths) != 1 {
		t.Fatalf("Paths = %v, want 1 entry", l.Paths)
	}

	reloaded := &List{path: l.path}
	data, err := os.ReadFile(l.path)
	if err != nil {
		t.Fatalf("failed reading persisted file: %v", err)
	}
	if err := json.Unmarshal(data, reloaded); err != nil {
		t.Fatalf("failed unmarshalling persisted file: %v", err)
	}
	if len(reloaded.Paths) != 1 || reloaded.Paths[0] != l.Paths[0] {
		t.Fatalf("persisted Paths = %v, want %v", reloaded.Paths, l.Paths)
	}
}

func TestRemove(t *testing.T) {
	l := newTestList(t)
	dir := t.TempDir()

	if err := l.Add(dir); err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if err := l.Remove(dir); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}

	if len(l.Paths) != 0 {
		t.Fatalf("Paths = %v, want empty", l.Paths)
	}
}

func TestIsIgnored(t *testing.T) {
	l := newTestList(t)
	dir := t.TempDir()
	ignored := filepath.Join(dir, "important-project")

	if err := l.Add(ignored); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	tests := []struct {
		path string
		want bool
	}{
		{ignored, true},
		{filepath.Join(ignored, "node_modules"), true},
		{dir, false},
		{filepath.Join(dir, "important-project-2"), false},
	}

	for _, tt := range tests {
		if got := l.IsIgnored(tt.path); got != tt.want {
			t.Errorf("IsIgnored(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}
