package tui

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/0SansNom/epurer/internal/analyzer"
)

func loadedModel(t *testing.T, dir string) AnalyzerModel {
	t.Helper()
	m := NewAnalyzerModel(dir, false)
	entries, err := analyzer.ListDir(dir)
	if err != nil {
		t.Fatalf("ListDir() error = %v", err)
	}
	newModel, _ := m.Update(dirLoadedMsg{path: dir, entries: entries})
	return newModel.(AnalyzerModel)
}

func TestAnalyzerModel_InitialStateIsLoading(t *testing.T) {
	m := NewAnalyzerModel("/tmp", false)
	if m.state != analyzerLoading {
		t.Errorf("initial state = %v, want analyzerLoading", m.state)
	}
}

func TestAnalyzerModel_DirLoadedTransitionsToBrowse(t *testing.T) {
	dir := t.TempDir()
	mustTouch(t, filepath.Join(dir, "a.txt"), 10)
	mustTouch(t, filepath.Join(dir, "b.txt"), 20)

	m := loadedModel(t, dir)

	if m.state != analyzerBrowse {
		t.Errorf("state = %v, want analyzerBrowse", m.state)
	}
	if len(m.entries) != 2 {
		t.Fatalf("entries = %d, want 2", len(m.entries))
	}
	// ListDir sorts descending by size - b.txt (20 bytes) should be first.
	if m.entries[0].Name != "b.txt" {
		t.Errorf("entries[0] = %q, want b.txt (largest first)", m.entries[0].Name)
	}
}

func TestAnalyzerModel_CursorBounds(t *testing.T) {
	dir := t.TempDir()
	mustTouch(t, filepath.Join(dir, "a.txt"), 10)
	mustTouch(t, filepath.Join(dir, "b.txt"), 20)
	m := loadedModel(t, dir)

	// down twice should clamp at len(entries)-1, not go out of bounds.
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = newModel.(AnalyzerModel)
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = newModel.(AnalyzerModel)
	if m.cursor != 1 {
		t.Errorf("cursor = %d, want clamped at 1", m.cursor)
	}

	// up past zero should clamp at 0.
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = newModel.(AnalyzerModel)
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = newModel.(AnalyzerModel)
	if m.cursor != 0 {
		t.Errorf("cursor = %d, want clamped at 0", m.cursor)
	}
}

func TestAnalyzerModel_EnterOnDirDrillsIn(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "subdir")
	mustTouch(t, filepath.Join(sub, "f.txt"), 5)
	m := loadedModel(t, dir)

	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(AnalyzerModel)

	if m.state != analyzerLoading {
		t.Errorf("state = %v, want analyzerLoading after entering a directory", m.state)
	}
	if cmd == nil {
		t.Fatal("expected a loadDir command")
	}
	msg := cmd().(dirLoadedMsg)
	if msg.path != sub {
		t.Errorf("loadDir path = %q, want %q", msg.path, sub)
	}
}

func TestAnalyzerModel_BackspaceGoesToParent(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "subdir")
	mustTouch(t, filepath.Join(sub, "f.txt"), 5)

	m := loadedModel(t, sub)
	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m = newModel.(AnalyzerModel)

	if cmd == nil {
		t.Fatal("expected a loadDir command for the parent")
	}
	msg := cmd().(dirLoadedMsg)
	if msg.path != dir {
		t.Errorf("loadDir path = %q, want parent %q", msg.path, dir)
	}
}

func TestAnalyzerModel_RevealTriggersCommandWithoutRunningIt(t *testing.T) {
	dir := t.TempDir()
	mustTouch(t, filepath.Join(dir, "a.txt"), 10)
	m := loadedModel(t, dir)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if cmd == nil {
		t.Fatal("'r' should return a reveal command")
	}
	// Deliberately not invoking cmd() here - it shells out to `open -R`,
	// which would pop a real Finder window. We only assert wiring, and
	// separately test the revealedMsg handler below.
}

func TestAnalyzerModel_RevealedMsgSetsMessage(t *testing.T) {
	dir := t.TempDir()
	m := loadedModel(t, dir)

	newModel, _ := m.Update(revealedMsg{err: nil})
	m = newModel.(AnalyzerModel)
	if m.message == "" {
		t.Error("expected a status message after a successful reveal")
	}
}

func TestAnalyzerModel_DeleteRequiresConfirmation(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "a.txt")
	mustTouch(t, target, 10)
	m := loadedModel(t, dir)

	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m = newModel.(AnalyzerModel)
	if m.state != analyzerConfirmDelete {
		t.Fatalf("state = %v, want analyzerConfirmDelete", m.state)
	}

	// 'n' should cancel back to browsing without touching the file.
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m = newModel.(AnalyzerModel)
	if m.state != analyzerBrowse {
		t.Errorf("state = %v, want analyzerBrowse after cancelling", m.state)
	}
	if _, err := os.Stat(target); err != nil {
		t.Errorf("file should still exist after cancelling delete: %v", err)
	}
}

func TestAnalyzerModel_ConfirmDelete_Actual_Deletes(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "a.txt")
	mustTouch(t, target, 10)
	m := loadedModel(t, dir)
	m.dryRun = false

	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m = newModel.(AnalyzerModel)
	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m = newModel.(AnalyzerModel)

	if cmd == nil {
		t.Fatal("expected a deleteEntry command")
	}
	msg := cmd().(deletedMsg)
	if msg.err != nil {
		t.Fatalf("delete failed: %v", msg.err)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Error("file was not deleted")
	}
}

func TestAnalyzerModel_ConfirmDelete_DryRun_DoesNotDelete(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "a.txt")
	mustTouch(t, target, 10)
	m := loadedModel(t, dir)
	m.dryRun = true

	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m = newModel.(AnalyzerModel)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})

	msg := cmd().(deletedMsg)
	if msg.err != nil {
		t.Fatalf("unexpected error: %v", msg.err)
	}
	if _, err := os.Stat(target); err != nil {
		t.Error("dry-run delete should not touch the file")
	}
}

func TestAnalyzerModel_Quit(t *testing.T) {
	m := loadedModel(t, t.TempDir())
	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m = newModel.(AnalyzerModel)

	if !m.quitting {
		t.Error("'q' should set quitting")
	}
	if cmd == nil {
		t.Error("'q' should return tea.Quit")
	}
}

func TestAnalyzerModel_View_DoesNotPanic(t *testing.T) {
	dir := t.TempDir()
	mustTouch(t, filepath.Join(dir, "a.txt"), 10)
	m := loadedModel(t, dir)

	for _, state := range []analyzerState{analyzerLoading, analyzerBrowse, analyzerConfirmDelete} {
		m.state = state
		_ = m.View()
	}

	m.entries = nil
	m.state = analyzerBrowse
	_ = m.View()

	m.quitting = true
	if m.View() != "" {
		t.Error("View() should be empty when quitting")
	}
}

func mustTouch(t *testing.T, path string, size int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(path, make([]byte, size), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}

var _ tea.Model = AnalyzerModel{}
