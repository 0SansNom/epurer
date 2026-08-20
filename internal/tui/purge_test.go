package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/0SansNom/epurer/internal/cleaner"
	"github.com/0SansNom/epurer/internal/config"
	"github.com/0SansNom/epurer/internal/purge"
)

func testProject(name string, size int64, artifacts ...purge.Artifact) purge.Project {
	return purge.Project{
		Root:      "/tmp/" + name,
		Name:      name,
		Type:      "Node.js",
		Artifacts: artifacts,
		TotalSize: size,
	}
}

func testArtifact(path string, size int64, selected bool) purge.Artifact {
	return purge.Artifact{
		CleanTarget: cleaner.CleanTarget{
			Path:      path,
			SizeBytes: size,
			Safety:    config.Moderate,
		},
		Kind:     "node_modules",
		Selected: selected,
	}
}

func TestProjectItem_Title(t *testing.T) {
	item := ProjectItem{project: testProject("my-app", 100, testArtifact("/tmp/my-app/node_modules", 100, true)), selected: true}
	if !strings.Contains(item.Title(), "[✓]") || !strings.Contains(item.Title(), "my-app") {
		t.Errorf("Title() = %q, want checkbox and project name", item.Title())
	}

	item.selected = false
	if !strings.Contains(item.Title(), "[ ]") {
		t.Errorf("Title() = %q, want empty checkbox when unselected", item.Title())
	}
}

func TestProjectItem_Description(t *testing.T) {
	item := ProjectItem{project: testProject("my-app", 1024*1024, testArtifact("/tmp/my-app/node_modules", 1024*1024, true))}
	desc := item.Description()
	if !strings.Contains(desc, "node_modules") {
		t.Errorf("Description() = %q, want artifact kind listed", desc)
	}
}

func TestNewPurgeModel_PreselectsAgedProjects(t *testing.T) {
	projects := []purge.Project{
		testProject("old-app", 100, testArtifact("/tmp/old-app/node_modules", 100, true)),
		testProject("new-app", 100, testArtifact("/tmp/new-app/node_modules", 100, false)),
	}

	m := NewPurgeModel(projects, false)

	if !m.items[0].selected {
		t.Error("project with a preselected artifact should start selected")
	}
	if m.items[1].selected {
		t.Error("project with no preselected artifact should start unselected")
	}
}

func TestPurgeModel_ToggleAndSelectAllNone(t *testing.T) {
	projects := []purge.Project{
		testProject("a", 10, testArtifact("/tmp/a/node_modules", 10, false)),
		testProject("b", 10, testArtifact("/tmp/b/node_modules", 10, false)),
	}
	m := NewPurgeModel(projects, false)

	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m = newModel.(PurgeModel)
	if !m.items[0].selected {
		t.Error("space should toggle the item under the cursor")
	}

	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = newModel.(PurgeModel)
	for i, item := range m.items {
		if !item.selected {
			t.Errorf("item %d not selected after 'a'", i)
		}
	}

	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m = newModel.(PurgeModel)
	for i, item := range m.items {
		if item.selected {
			t.Errorf("item %d still selected after 'n'", i)
		}
	}
}

func TestPurgeModel_EnterRequiresSelection(t *testing.T) {
	projects := []purge.Project{testProject("a", 10, testArtifact("/tmp/a/node_modules", 10, false))}
	m := NewPurgeModel(projects, false)

	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(PurgeModel)
	if m.state != StateSelect {
		t.Error("enter with nothing selected should not advance state")
	}

	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = newModel.(PurgeModel)
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(PurgeModel)
	if m.state != StateConfirm {
		t.Error("enter with a selection should advance to StateConfirm")
	}
}

func TestPurgeModel_ConfirmNoGoesBackToSelect(t *testing.T) {
	m := NewPurgeModel([]purge.Project{testProject("a", 10, testArtifact("/tmp/a/node_modules", 10, true))}, false)
	m.state = StateConfirm

	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m = newModel.(PurgeModel)
	if m.state != StateSelect {
		t.Error("'n' at confirm should return to StateSelect")
	}
}

// TestPurgeModel_CleanNext_DryRun_DoesNotDelete verifies the dry-run path
// never touches the filesystem, driving the model through cleanNext exactly
// as the real Update loop does.
func TestPurgeModel_CleanNext_DryRun_DoesNotDelete(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "node_modules")
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	m := NewPurgeModel([]purge.Project{testProject("a", 10, testArtifact(path, 10, true))}, true)

	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = newModel.(PurgeModel)
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(PurgeModel)
	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m = newModel.(PurgeModel)

	if cmd == nil {
		t.Fatal("expected a command to run cleanNext")
	}
	msg := cmd()
	newModel, _ = m.Update(msg)
	m = newModel.(PurgeModel)

	if m.state != StateDone {
		t.Fatalf("state = %v, want StateDone", m.state)
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("dry-run purge deleted the artifact, it should not have")
	}
}

// TestPurgeModel_CleanNext_Actual_Deletes verifies real (non-dry-run)
// deletion actually removes the artifact from disk.
func TestPurgeModel_CleanNext_Actual_Deletes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "node_modules")
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	m := NewPurgeModel([]purge.Project{testProject("a", 10, testArtifact(path, 10, true))}, false)

	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = newModel.(PurgeModel)
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(PurgeModel)
	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m = newModel.(PurgeModel)

	msg := cmd()
	newModel, _ = m.Update(msg)
	m = newModel.(PurgeModel)

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("artifact was not deleted")
	}
	if m.cleanedSize != 10 {
		t.Errorf("cleanedSize = %d, want 10", m.cleanedSize)
	}
}

func TestPurgeModel_DoneEnterQuits(t *testing.T) {
	m := NewPurgeModel(nil, false)
	m.state = StateDone

	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(PurgeModel)

	if !m.quitting {
		t.Error("enter in StateDone should set quitting")
	}
	if cmd == nil {
		t.Error("enter in StateDone should return tea.Quit")
	}
}

func TestPurgeModel_View_DoesNotPanic(t *testing.T) {
	m := NewPurgeModel([]purge.Project{testProject("a", 10, testArtifact("/tmp/a/node_modules", 10, true))}, true)
	for _, state := range []State{StateSelect, StateConfirm, StateCleaning, StateDone} {
		m.state = state
		m.totalItems = 1
		_ = m.View()
	}

	m.quitting = true
	if m.View() != "" {
		t.Error("View() should be empty when quitting")
	}
}

var _ tea.Model = PurgeModel{}
