package main

import (
	"path/filepath"
	"testing"

	"github.com/0SansNom/epurer/internal/ignorelist"
)

func TestIgnoreAddRemoveList(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	target := filepath.Join(home, "Projects", "important")

	addCmd := newIgnoreAddCmd()
	if err := addCmd.RunE(addCmd, []string{target}); err != nil {
		t.Fatalf("ignore add RunE error = %v", err)
	}

	l, err := ignorelist.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(l.Paths) != 1 {
		t.Fatalf("Paths = %v, want 1 entry after add", l.Paths)
	}

	listCmd := newIgnoreListCmd()
	if err := listCmd.RunE(listCmd, nil); err != nil {
		t.Fatalf("ignore list RunE error = %v", err)
	}

	removeCmd := newIgnoreRemoveCmd()
	if err := removeCmd.RunE(removeCmd, []string{target}); err != nil {
		t.Fatalf("ignore remove RunE error = %v", err)
	}

	l, err = ignorelist.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(l.Paths) != 0 {
		t.Errorf("Paths = %v, want empty after remove", l.Paths)
	}
}

func TestIgnoreList_EmptyDoesNotError(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	listCmd := newIgnoreListCmd()
	if err := listCmd.RunE(listCmd, nil); err != nil {
		t.Fatalf("ignore list RunE on an empty list error = %v", err)
	}
}
