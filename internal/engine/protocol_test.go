package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0SansNom/epurer/internal/cleaner"
	"github.com/0SansNom/epurer/internal/config"
	"github.com/0SansNom/epurer/internal/scantask"
	"github.com/0SansNom/epurer/pkg/utils"
)

func TestWriteProgressLine(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteProgressLine(&buf, "scan", scantask.Progress{Path: "/a", Items: 2, Bytes: 9}); err != nil {
		t.Fatal(err)
	}
	var ev ProgressEvent
	if err := json.Unmarshal(buf.Bytes(), &ev); err != nil {
		t.Fatal(err)
	}
	if ev.Event != "progress" || ev.Phase != "scan" || ev.Items != 2 || ev.Path != "/a" {
		t.Fatalf("%#v", ev)
	}
}

func TestInfo_NoPrivilege(t *testing.T) {
	info := Info()
	if info.Protocol != ProtocolVersion {
		t.Fatalf("protocol = %d", info.Protocol)
	}
	if info.PrivilegedHelper || info.AsksAdminPassword {
		t.Fatalf("info claims elevation: %#v", info)
	}
	if !info.HardenedUserSpace {
		t.Fatal("expected hardened_user_space")
	}
}

func TestWriteJSON_RoundTripScanShape(t *testing.T) {
	var buf bytes.Buffer
	resp := ScanResponse{
		Protocol:   1,
		Command:    "scan",
		DiskAccess: "unknown",
		Targets:    []Target{{Path: "/tmp/a", Safety: "Safe", Cleaner: "Caches", Domain: "System"}},
		Locked:     []Target{},
	}
	if err := WriteJSON(&buf, resp); err != nil {
		t.Fatal(err)
	}
	var got ScanResponse
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Targets) != 1 || got.Targets[0].Path != "/tmp/a" {
		t.Fatalf("got %#v", got)
	}
}

func TestClean_UnknownCleaner(t *testing.T) {
	resp, err := Clean(context.Background(), nil, CleanRequest{
		Protocol: 1,
		DryRun:   true,
		Targets:  []Target{{Path: "/tmp/x", Cleaner: "Missing", Safety: "Safe"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Results) != 1 || resp.Results[0].Status != "failed" {
		t.Fatalf("results = %#v", resp.Results)
	}
}

func TestClean_RejectsRelativePath(t *testing.T) {
	resp, err := Clean(context.Background(), []cleaner.Cleaner{&stubCleaner{name: "Stub", domain: config.DomainSystem}}, CleanRequest{
		Protocol: 1,
		DryRun:   true,
		Targets:  []Target{{Path: "relative/cache", Cleaner: "Stub", Safety: "Safe"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Results) != 1 || resp.Results[0].Status != "left_in_place" {
		t.Fatalf("results = %#v", resp.Results)
	}
}

func TestClean_RejectsProtectedPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	resp, err := Clean(context.Background(), []cleaner.Cleaner{&stubCleaner{name: "Stub", domain: config.DomainSystem}}, CleanRequest{
		Protocol: 1,
		DryRun:   false,
		Targets:  []Target{{Path: home, Cleaner: "Stub", Safety: "Safe", SizeBytes: 99}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Results) != 1 || resp.Results[0].Status != "left_in_place" || resp.Results[0].Success {
		t.Fatalf("protected home result = %#v, want left_in_place", resp.Results)
	}
}

func TestClean_DryRunRemovable(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "cache")
	if err := os.WriteFile(file, []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}

	c := &stubCleaner{name: "Stub", domain: config.DomainSystem}
	resp, err := Clean(context.Background(), []cleaner.Cleaner{c}, CleanRequest{
		Protocol: 1,
		DryRun:   true,
		Targets: []Target{{
			Path:        file,
			Description: "stub",
			SizeBytes:   2,
			Safety:      "Safe",
			Domain:      "System",
			Cleaner:     "Stub",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Results) != 1 || !resp.Results[0].Success || resp.Results[0].BytesFreed != 2 {
		t.Fatalf("results = %#v", resp.Results)
	}
	if _, err := os.Stat(file); err != nil {
		t.Fatal("dry-run deleted the file")
	}
}

func TestDecodeCleanRequest_RejectsUnknownField(t *testing.T) {
	_, err := DecodeCleanRequest(strings.NewReader(`{"protocol":1,"nope":true,"targets":[]}`))
	if err == nil {
		t.Fatal("expected error for unknown field")
	}
}

func TestClean_RejectsMissingProtocol(t *testing.T) {
	_, err := Clean(context.Background(), nil, CleanRequest{
		Protocol: 0,
		DryRun:   true,
		Targets:  nil,
	})
	if err == nil {
		t.Fatal("expected error for protocol 0")
	}
}

func TestFromCleanResult_LeftInPlace(t *testing.T) {
	r := fromCleanResult(cleaner.CleanResult{
		Target:  cleaner.CleanTarget{Path: "/Library/Caches"},
		Success: false,
		Error:   utils.ErrNotRemovable,
	})
	if r.Status != "left_in_place" || r.Success {
		t.Fatalf("%#v", r)
	}
}

type stubCleaner struct {
	name   string
	domain config.Domain
}

func (s *stubCleaner) Name() string                         { return s.name }
func (s *stubCleaner) Domain() config.Domain                { return s.domain }
func (s *stubCleaner) Detect(context.Context) (bool, error) { return true, nil }
func (s *stubCleaner) Scan(context.Context, *config.Config) ([]cleaner.CleanTarget, error) {
	return nil, nil
}
func (s *stubCleaner) Clean(ctx context.Context, targets []cleaner.CleanTarget, dryRun bool) ([]cleaner.CleanResult, error) {
	return cleaner.CleanTargets(ctx, targets, dryRun, nil)
}
