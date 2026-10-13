package scantask

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/0SansNom/epurer/internal/config"
	"github.com/0SansNom/epurer/internal/ignorelist"
)

func TestWalk_TopLevelEntries(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "keep", "a.bin"), "hello")
	writeFile(t, filepath.Join(root, "skip-name", "b.bin"), "x")
	writeFile(t, filepath.Join(root, "huge.bin"), "0123456789")
	writeFile(t, filepath.Join(root, "empty", "z"), "")

	var last Progress
	got, err := (Walk{
		Root:        root,
		Description: "User caches",
		Safety:      config.Safe,
		Predicate: Ignored{List: &ignorelist.List{Paths: []string{
			filepath.Join(root, "skip-name"),
			filepath.Join(root, "huge.bin"),
		}}},
		OnProgress: func(p Progress) { last = p },
	}).Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("Scan() = %+v, want only keep", got)
	}
	if got[0].Path != filepath.Join(root, "keep") {
		t.Errorf("path = %s", got[0].Path)
	}
	if got[0].Description != "User caches: keep" {
		t.Errorf("description = %q", got[0].Description)
	}
	if got[0].SizeBytes != 5 || got[0].Safety != config.Safe {
		t.Errorf("target = %+v", got[0])
	}
	if last.Items != 4 || last.Bytes != 16 {
		t.Errorf("progress = %+v, want 4 items and 16 bytes", last)
	}
}

func TestWalk_UnreadableRoot(t *testing.T) {
	blocked := filepath.Join(t.TempDir(), "blocked")
	if err := os.Mkdir(blocked, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(blocked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(blocked, 0o755) })
	if _, err := os.ReadDir(blocked); err == nil {
		t.Skip("process can read a mode 000 directory")
	}

	got, err := (Walk{Root: blocked}).Scan(context.Background())
	if err != nil || got != nil {
		t.Fatalf("unreadable root = (%v, %v), want nil, nil", got, err)
	}
}

func TestWalk_IgnoreListAndMissingRoot(t *testing.T) {
	root := t.TempDir()
	skipped := filepath.Join(root, "skip")
	writeFile(t, filepath.Join(skipped, "a.bin"), "abc")
	writeFile(t, filepath.Join(root, "keep", "b.bin"), "de")

	got, err := (Walk{
		Root:      root,
		Predicate: Ignored{List: &ignorelist.List{Paths: []string{skipped}}},
	}).Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if len(got) != 1 || filepath.Base(got[0].Path) != "keep" {
		t.Fatalf("Scan() = %+v, want keep", got)
	}

	missing, err := (Walk{Root: filepath.Join(root, "absent")}).Scan(context.Background())
	if err != nil || missing != nil {
		t.Fatalf("missing root = (%v, %v), want nil, nil", missing, err)
	}
}

func TestWalk_Cancel(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "a.bin"), "a")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	got, err := (Walk{Root: root}).Scan(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Scan() error = %v, want context.Canceled", err)
	}
	if len(got) != 0 {
		t.Fatalf("Scan() = %+v, want no targets after cancel", got)
	}
}

type staticTask struct {
	targets []Target
	err     error
	calls   int
}

func (s *staticTask) Scan(ctx context.Context) ([]Target, error) {
	s.calls++
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.targets, s.err
}

func TestCompound_ConcatenatesAndStops(t *testing.T) {
	first := &staticTask{targets: []Target{{Path: "/a", SizeBytes: 1}}}
	second := &staticTask{err: errors.New("boom")}
	third := &staticTask{targets: []Target{{Path: "/c"}}}

	got, err := (Compound{Tasks: []Task{first, second, third}}).Scan(context.Background())
	if err == nil || err.Error() != "boom" {
		t.Fatalf("Scan() error = %v, want boom", err)
	}
	if len(got) != 1 || got[0].Path != "/a" {
		t.Fatalf("Scan() = %+v, want the first task's target", got)
	}
	if third.calls != 0 {
		t.Fatalf("third task calls = %d, want 0", third.calls)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	later := &staticTask{targets: []Target{{Path: "/z"}}}
	got, err = (Compound{Tasks: []Task{later}}).Scan(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled Scan() error = %v", err)
	}
	if later.calls != 0 || len(got) != 0 {
		t.Fatalf("cancelled Scan() ran the task: calls=%d targets=%+v", later.calls, got)
	}
}

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
