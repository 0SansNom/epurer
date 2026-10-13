package engine

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0SansNom/epurer/internal/cleaner"
	"github.com/0SansNom/epurer/internal/config"
	"github.com/0SansNom/epurer/internal/detector"
	"github.com/0SansNom/epurer/internal/ignorelist"
)

func TestClean_PathCleanerRemovesPickedPaths(t *testing.T) {
	dir := t.TempDir()
	victim := filepath.Join(dir, "node_modules")
	if err := os.MkdirAll(victim, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(victim, "x.js"), []byte("1234"), 0o644); err != nil {
		t.Fatal(err)
	}

	resp, err := Clean(context.Background(), []cleaner.Cleaner{NewPathCleaner(nil)}, CleanRequest{
		Protocol: ProtocolVersion,
		Targets:  []Target{{Path: victim, SizeBytes: 4, Safety: "Moderate", Cleaner: PathCleanerName}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Results) != 1 || !resp.Results[0].Success || resp.Results[0].BytesFreed != 4 {
		t.Fatalf("results = %#v", resp.Results)
	}
	if _, err := os.Stat(victim); !os.IsNotExist(err) {
		t.Fatalf("expected %s removed, stat err = %v", victim, err)
	}
}

func TestClean_PathCleanerLeavesIgnoredPaths(t *testing.T) {
	dir := t.TempDir()
	kept := filepath.Join(dir, "keep")
	if err := os.MkdirAll(kept, 0o755); err != nil {
		t.Fatal(err)
	}

	ignore := &ignorelist.List{Paths: []string{dir}}
	resp, err := Clean(context.Background(), []cleaner.Cleaner{NewPathCleaner(ignore)}, CleanRequest{
		Protocol: ProtocolVersion,
		Targets:  []Target{{Path: kept, Cleaner: PathCleanerName}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Results) != 1 || resp.Results[0].Success || resp.Results[0].Status != "left_in_place" {
		t.Fatalf("results = %#v", resp.Results)
	}
	if _, err := os.Stat(kept); err != nil {
		t.Fatalf("ignored path was touched: %v", err)
	}
}

func TestAnalyze_ListsChildrenLargestFirst(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "small"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "big"), make([]byte, 64), 0o644); err != nil {
		t.Fatal(err)
	}

	var events bytes.Buffer
	resp, err := Analyze(dir, &ignorelist.List{Paths: []string{filepath.Join(dir, "small")}}, &events)
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Entries) != 2 || resp.Entries[0].Name != "big" {
		t.Fatalf("entries = %#v", resp.Entries)
	}
	// One progress line per child, counting up to the total.
	lines := strings.Split(strings.TrimSpace(events.String()), "\n")
	if len(lines) != 2 || !strings.Contains(lines[1], `"items":2`) || !strings.Contains(lines[1], `"total":2`) {
		t.Fatalf("progress = %q", events.String())
	}
	small := resp.Entries[1]
	if !small.Ignored || small.Removable {
		t.Fatalf("ignored entry = %#v", small)
	}
}

type fakeCleaner struct {
	name     string
	domain   config.Domain
	detected bool
}

func (f fakeCleaner) Name() string                         { return f.name }
func (f fakeCleaner) Domain() config.Domain                { return f.domain }
func (f fakeCleaner) Detect(context.Context) (bool, error) { return f.detected, nil }
func (f fakeCleaner) Scan(context.Context, *config.Config) ([]cleaner.CleanTarget, error) {
	return nil, nil
}
func (f fakeCleaner) Clean(context.Context, []cleaner.CleanTarget, bool) ([]cleaner.CleanResult, error) {
	return nil, nil
}

func TestDetect_GroupsCleanersByDomain(t *testing.T) {
	resp := Detect(context.Background(), []cleaner.Cleaner{
		fakeCleaner{"Trash", config.DomainSystem, true},
		fakeCleaner{"DevOps", config.DomainDevOps, false},
		fakeCleaner{"Frontend", config.DomainFrontend, true},
	}, detector.DetectionResult{Frontend: []string{"npm"}})

	if len(resp.Domains) != 3 {
		t.Fatalf("domains = %#v", resp.Domains)
	}
	front, devops, system := resp.Domains[0], resp.Domains[1], resp.Domains[2]
	if front.Domain != "Frontend" || !front.Detected || len(front.Tools) != 1 {
		t.Fatalf("frontend = %#v", front)
	}
	if devops.Detected {
		t.Fatalf("devops should be absent: %#v", devops)
	}
	if system.Domain != "System" || !system.Detected {
		t.Fatalf("system = %#v", system)
	}
}

func TestOutOfReach(t *testing.T) {
	if !OutOfReach("/System") {
		t.Fatal("/System should be out of reach for a normal user")
	}
	if OutOfReach(t.TempDir()) {
		t.Fatal("a temp dir owned by this user is cleanable")
	}
	if OutOfReach("/Volumes/epurer-test-not-mounted") {
		t.Fatal("a missing path must stay protectable (unplugged volume)")
	}
}
