package purge

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/0SansNom/epurer/internal/scanner"
)

func mustMkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", path, err)
	}
}

func mustWriteFile(t *testing.T, path string, content string) {
	t.Helper()
	mustMkdirAll(t, filepath.Dir(path))
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func TestScan_GroupsByProjectAndDetectsType(t *testing.T) {
	root := t.TempDir()
	proj := filepath.Join(root, "my-node-app")

	mustWriteFile(t, filepath.Join(proj, "package.json"), `{"name":"app"}`)
	mustWriteFile(t, filepath.Join(proj, "node_modules", "lodash", "index.js"), "module.exports = {}")

	s, err := scanner.NewScannerWithDirs([]string{root})
	if err != nil {
		t.Fatalf("NewScannerWithDirs() error = %v", err)
	}

	projects, err := Scan(context.Background(), s, 0)
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	if len(projects) != 1 {
		t.Fatalf("got %d projects, want 1: %+v", len(projects), projects)
	}

	p := projects[0]
	if p.Type != "Node.js" {
		t.Errorf("Type = %q, want Node.js", p.Type)
	}
	if len(p.Artifacts) != 1 || p.Artifacts[0].Kind != "node_modules" {
		t.Errorf("Artifacts = %+v, want a single node_modules artifact", p.Artifacts)
	}
}

func TestScan_AmbiguousNameRequiresMarker(t *testing.T) {
	root := t.TempDir()

	// "target" with no Cargo.toml/pom.xml sibling should NOT be treated as
	// a Rust/Maven build artifact.
	mustMkdirAll(t, filepath.Join(root, "random-dir", "target", "stuff"))

	// "target" WITH a Cargo.toml sibling should be picked up.
	rustProj := filepath.Join(root, "rust-app")
	mustWriteFile(t, filepath.Join(rustProj, "Cargo.toml"), "[package]\nname=\"app\"")
	mustMkdirAll(t, filepath.Join(rustProj, "target", "debug"))

	s, err := scanner.NewScannerWithDirs([]string{root})
	if err != nil {
		t.Fatalf("NewScannerWithDirs() error = %v", err)
	}

	projects, err := Scan(context.Background(), s, 0)
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	if len(projects) != 1 {
		t.Fatalf("got %d projects, want 1 (only the Rust one): %+v", len(projects), projects)
	}
	if projects[0].Name != "rust-app" {
		t.Errorf("project = %q, want rust-app", projects[0].Name)
	}
}

func TestScan_AgePreselection(t *testing.T) {
	root := t.TempDir()
	proj := filepath.Join(root, "app")
	nm := filepath.Join(proj, "node_modules")
	mustMkdirAll(t, filepath.Join(nm, "pkg"))

	// Backdate the artifact so it's older than the cutoff.
	old := time.Now().Add(-30 * 24 * time.Hour)
	if err := os.Chtimes(nm, old, old); err != nil {
		t.Fatalf("Chtimes() error = %v", err)
	}

	s, err := scanner.NewScannerWithDirs([]string{root})
	if err != nil {
		t.Fatalf("NewScannerWithDirs() error = %v", err)
	}

	projects, err := Scan(context.Background(), s, 7*24*time.Hour)
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	if len(projects) != 1 || len(projects[0].Artifacts) != 1 {
		t.Fatalf("unexpected projects: %+v", projects)
	}
	if !projects[0].Artifacts[0].Selected {
		t.Errorf("Selected = false, want true for an artifact older than minAge")
	}
}
