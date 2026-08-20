package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunAnalyze_NonExistentPath(t *testing.T) {
	cmd := newAnalyzeCmd()
	err := cmd.RunE(cmd, []string{"/definitely/does/not/exist/xyz"})
	if err == nil {
		t.Fatal("expected an error for a non-existent path")
	}
}

func TestRunAnalyze_PathIsAFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "notadir.txt")
	if err := os.WriteFile(file, []byte("content"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cmd := newAnalyzeCmd()
	err := cmd.RunE(cmd, []string{file})
	if err == nil {
		t.Fatal("expected an error when the argument is a file, not a directory")
	}
}
