package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/0SansNom/epurer/internal/cleaner"
	"github.com/0SansNom/epurer/internal/config"
)

// =============================================================================
// CleanItem Tests
// =============================================================================

func TestCleanItem_Title_Selected(t *testing.T) {
	item := CleanItem{
		domain:   "Frontend",
		selected: true,
	}

	title := item.Title()
	if !strings.Contains(title, "[✓]") {
		t.Error("Selected item title should contain checkmark")
	}
	if !strings.Contains(title, "Frontend") {
		t.Error("Title should contain domain name")
	}
}

func TestCleanItem_Title_Unselected(t *testing.T) {
	item := CleanItem{
		domain:   "Backend",
		selected: false,
	}

	title := item.Title()
	if !strings.Contains(title, "[ ]") {
		t.Error("Unselected item title should contain empty checkbox")
	}
	if !strings.Contains(title, "Backend") {
		t.Error("Title should contain domain name")
	}
}

func TestCleanItem_Description(t *testing.T) {
	item := CleanItem{
		domain: "Frontend",
		size:   1024 * 1024 * 100, // 100 MB
		targets: []cleaner.CleanTarget{
			{Path: "/path/1"},
			{Path: "/path/2"},
			{Path: "/path/3"},
		},
	}

	desc := item.Description()
	if !strings.Contains(desc, "3 items") {
		t.Errorf("Description should contain item count, got: %s", desc)
	}
	// Should contain formatted size
	if !strings.Contains(desc, "MB") {
		t.Errorf("Description should contain size, got: %s", desc)
	}
}

func TestCleanItem_FilterValue(t *testing.T) {
	item := CleanItem{
		domain: "DevOps",
	}

	if item.FilterValue() != "DevOps" {
		t.Errorf("FilterValue() = %q, want %q", item.FilterValue(), "DevOps")
	}
}

// =============================================================================
// State Tests
// =============================================================================

func TestState_Constants(t *testing.T) {
	// Verify state constants are distinct
	states := []State{StateSelect, StateConfirm, StateCleaning, StateDone}
	seen := make(map[State]bool)

	for _, s := range states {
		if seen[s] {
			t.Errorf("Duplicate state value: %d", s)
		}
		seen[s] = true
	}

	// Verify expected values
	if StateSelect != 0 {
		t.Error("StateSelect should be 0")
	}
	if StateConfirm != 1 {
		t.Error("StateConfirm should be 1")
	}
	if StateCleaning != 2 {
		t.Error("StateCleaning should be 2")
	}
	if StateDone != 3 {
		t.Error("StateDone should be 3")
	}
}

// =============================================================================
// NewModel Tests
// =============================================================================

func TestNewModel(t *testing.T) {
	targets := map[string][]cleaner.CleanTarget{
		"Frontend": {
			{Path: "/path/1", Description: "npm cache", SizeBytes: 1024, Safety: config.Safe},
			{Path: "/path/2", Description: "node_modules", SizeBytes: 2048, Safety: config.Moderate},
		},
		"Backend": {
			{Path: "/path/3", Description: "pip cache", SizeBytes: 512, Safety: config.Safe},
		},
	}

	model := NewModel(targets, false)

	// Check initial state
	if model.state != StateSelect {
		t.Errorf("Initial state should be StateSelect, got %d", model.state)
	}

	// Check items were created
	if len(model.items) != 2 {
		t.Errorf("Expected 2 items, got %d", len(model.items))
	}

	// Check dryRun flag
	if model.dryRun {
		t.Error("dryRun should be false")
	}

	// Check items are selected by default
	for _, item := range model.items {
		if !item.selected {
			t.Errorf("Item %s should be selected by default", item.domain)
		}
	}
}

func TestNewModel_DryRun(t *testing.T) {
	targets := map[string][]cleaner.CleanTarget{
		"System": {
			{Path: "/path/1", SizeBytes: 1024},
		},
	}

	model := NewModel(targets, true)

	if !model.dryRun {
		t.Error("dryRun should be true")
	}
}

func TestNewModel_EmptyTargets(t *testing.T) {
	model := NewModel(map[string][]cleaner.CleanTarget{}, false)

	if len(model.items) != 0 {
		t.Errorf("Expected 0 items for empty targets, got %d", len(model.items))
	}
}

func TestNewModel_SkipsEmptyDomains(t *testing.T) {
	targets := map[string][]cleaner.CleanTarget{
		"Frontend": {
			{Path: "/path/1", SizeBytes: 1024},
		},
		"Backend": {}, // Empty domain should be skipped
	}

	model := NewModel(targets, false)

	if len(model.items) != 1 {
		t.Errorf("Expected 1 item (empty domain skipped), got %d", len(model.items))
	}
}

func TestNewModel_CalculatesTotalSize(t *testing.T) {
	targets := map[string][]cleaner.CleanTarget{
		"Frontend": {
			{Path: "/path/1", SizeBytes: 1000},
			{Path: "/path/2", SizeBytes: 2000},
			{Path: "/path/3", SizeBytes: 3000},
		},
	}

	model := NewModel(targets, false)

	if len(model.items) != 1 {
		t.Fatalf("Expected 1 item, got %d", len(model.items))
	}

	expectedSize := int64(6000)
	if model.items[0].size != expectedSize {
		t.Errorf("Expected total size %d, got %d", expectedSize, model.items[0].size)
	}
}

// =============================================================================
// Model.Init Tests
// =============================================================================

func TestModel_Init(t *testing.T) {
	model := NewModel(map[string][]cleaner.CleanTarget{}, false)

	cmd := model.Init()

	// Init should return a command (spinner tick)
	if cmd == nil {
		t.Error("Init() should return a command")
	}
}

// =============================================================================
// Model.Update Tests
// =============================================================================

func TestModel_Update_Quit(t *testing.T) {
	model := NewModel(map[string][]cleaner.CleanTarget{
		"Test": {{Path: "/test", SizeBytes: 1024}},
	}, false)

	// Test quit with 'q'
	newModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m := newModel.(Model)

	if !m.quitting {
		t.Error("Model should be quitting after 'q' press")
	}
	if cmd == nil {
		t.Error("Should return tea.Quit command")
	}
}

func TestModel_Update_CtrlC(t *testing.T) {
	model := NewModel(map[string][]cleaner.CleanTarget{
		"Test": {{Path: "/test", SizeBytes: 1024}},
	}, false)

	// Test quit with Ctrl+C
	newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m := newModel.(Model)

	if !m.quitting {
		t.Error("Model should be quitting after Ctrl+C")
	}
}

func TestModel_Update_ToggleSelection(t *testing.T) {
	model := NewModel(map[string][]cleaner.CleanTarget{
		"Frontend": {{Path: "/test", SizeBytes: 1024}},
	}, false)

	// Items are selected by default
	if !model.items[0].selected {
		t.Fatal("Item should be selected by default")
	}

	// Toggle with space
	newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeySpace})
	m := newModel.(Model)

	if m.items[0].selected {
		t.Error("Item should be unselected after space press")
	}

	// Toggle again
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m = newModel.(Model)

	if !m.items[0].selected {
		t.Error("Item should be selected after second space press")
	}
}

func TestModel_Update_SelectAll(t *testing.T) {
	model := NewModel(map[string][]cleaner.CleanTarget{
		"Frontend": {{Path: "/test1", SizeBytes: 1024}},
		"Backend":  {{Path: "/test2", SizeBytes: 1024}},
	}, false)

	// Deselect all first
	for i := range model.items {
		model.items[i].selected = false
	}

	// Press 'a' to select all
	newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m := newModel.(Model)

	for _, item := range m.items {
		if !item.selected {
			t.Errorf("Item %s should be selected after 'a' press", item.domain)
		}
	}
}

func TestModel_Update_SelectNone(t *testing.T) {
	model := NewModel(map[string][]cleaner.CleanTarget{
		"Frontend": {{Path: "/test1", SizeBytes: 1024}},
		"Backend":  {{Path: "/test2", SizeBytes: 1024}},
	}, false)

	// All items should be selected by default
	// Press 'n' to select none
	newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m := newModel.(Model)

	for _, item := range m.items {
		if item.selected {
			t.Errorf("Item %s should be unselected after 'n' press", item.domain)
		}
	}
}

func TestModel_Update_EnterWithSelection(t *testing.T) {
	model := NewModel(map[string][]cleaner.CleanTarget{
		"Frontend": {{Path: "/test", SizeBytes: 1024}},
	}, false)

	// Press enter with selected items
	newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m := newModel.(Model)

	if m.state != StateConfirm {
		t.Errorf("State should be StateConfirm after enter, got %d", m.state)
	}
}

func TestModel_Update_EnterWithoutSelection(t *testing.T) {
	model := NewModel(map[string][]cleaner.CleanTarget{
		"Frontend": {{Path: "/test", SizeBytes: 1024}},
	}, false)

	// Deselect all
	model.items[0].selected = false

	// Press enter without selected items
	newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m := newModel.(Model)

	// Should stay in StateSelect
	if m.state != StateSelect {
		t.Errorf("State should remain StateSelect when nothing selected, got %d", m.state)
	}
}

func TestModel_Update_ConfirmYes(t *testing.T) {
	model := NewModel(map[string][]cleaner.CleanTarget{
		"Frontend": {{Path: "/test", SizeBytes: 1024}},
	}, false)
	model.state = StateConfirm

	// Press 'y' to confirm
	newModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m := newModel.(Model)

	if m.state != StateCleaning {
		t.Errorf("State should be StateCleaning after 'y', got %d", m.state)
	}
	if !m.cleaning {
		t.Error("cleaning flag should be true")
	}
	if cmd == nil {
		t.Error("Should return a command to start cleaning")
	}
}

func TestModel_Update_ConfirmNo(t *testing.T) {
	model := NewModel(map[string][]cleaner.CleanTarget{
		"Frontend": {{Path: "/test", SizeBytes: 1024}},
	}, false)
	model.state = StateConfirm

	// Press 'n' to cancel
	newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m := newModel.(Model)

	if m.state != StateSelect {
		t.Errorf("State should be StateSelect after 'n', got %d", m.state)
	}
}

func TestModel_Update_DoneQuit(t *testing.T) {
	model := NewModel(map[string][]cleaner.CleanTarget{}, false)
	model.state = StateDone

	// Press 'q' to quit from done state
	newModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m := newModel.(Model)

	if !m.quitting {
		t.Error("Should be quitting after 'q' in done state")
	}
	if cmd == nil {
		t.Error("Should return quit command")
	}
}

func TestModel_Update_DoneEnter(t *testing.T) {
	model := NewModel(map[string][]cleaner.CleanTarget{}, false)
	model.state = StateDone

	// Press enter to quit from done state
	newModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m := newModel.(Model)

	if !m.quitting {
		t.Error("Should be quitting after enter in done state")
	}
	if cmd == nil {
		t.Error("Should return quit command")
	}
}

func TestModel_Update_WindowSize(t *testing.T) {
	model := NewModel(map[string][]cleaner.CleanTarget{
		"Test": {{Path: "/test", SizeBytes: 1024}},
	}, false)

	// Send window size message
	newModel, _ := model.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
	m := newModel.(Model)

	if m.width != 100 {
		t.Errorf("Width should be 100, got %d", m.width)
	}
	if m.height != 50 {
		t.Errorf("Height should be 50, got %d", m.height)
	}
}

func TestModel_Update_CleanedMsg(t *testing.T) {
	model := NewModel(map[string][]cleaner.CleanTarget{
		"Frontend": {{Path: "/test", SizeBytes: 1024}},
	}, false)
	model.state = StateCleaning
	model.totalItems = 2
	model.cleanIndex = 0

	// Send cleaned message
	newModel, _ := model.Update(cleanedMsg{size: 512})
	m := newModel.(Model)

	if m.cleanedSize != 512 {
		t.Errorf("cleanedSize should be 512, got %d", m.cleanedSize)
	}
	if m.cleanIndex != 1 {
		t.Errorf("cleanIndex should be 1, got %d", m.cleanIndex)
	}
}

func TestModel_Update_CleanedMsg_Complete(t *testing.T) {
	model := NewModel(map[string][]cleaner.CleanTarget{
		"Frontend": {{Path: "/test", SizeBytes: 1024}},
	}, false)
	model.state = StateCleaning
	model.totalItems = 1
	model.cleanIndex = 0

	// Send final cleaned message
	newModel, _ := model.Update(cleanedMsg{size: 1024})
	m := newModel.(Model)

	if m.state != StateDone {
		t.Errorf("State should be StateDone after all items cleaned, got %d", m.state)
	}
	if m.cleaning {
		t.Error("cleaning should be false after completion")
	}
}

// =============================================================================
// Model.View Tests
// =============================================================================

func TestModel_View_Quitting(t *testing.T) {
	model := NewModel(map[string][]cleaner.CleanTarget{}, false)
	model.quitting = true

	view := model.View()

	if view != "" {
		t.Error("View should be empty when quitting")
	}
}

func TestModel_View_StateSelect(t *testing.T) {
	model := NewModel(map[string][]cleaner.CleanTarget{
		"Frontend": {{Path: "/test", SizeBytes: 1024 * 1024}},
	}, false)
	model.state = StateSelect

	view := model.View()

	// Should contain header
	if !strings.Contains(view, "Épurer") {
		t.Error("View should contain app name")
	}
	// Should contain help text
	if !strings.Contains(view, "navigate") {
		t.Error("View should contain help text")
	}
	// Should contain status bar info
	if !strings.Contains(view, "Selected") {
		t.Error("View should contain selection info")
	}
}

func TestModel_View_StateConfirm(t *testing.T) {
	model := NewModel(map[string][]cleaner.CleanTarget{
		"Frontend": {{Path: "/test", SizeBytes: 1024 * 1024}},
	}, false)
	model.state = StateConfirm

	view := model.View()

	// Should contain confirmation prompt
	if !strings.Contains(view, "Clean") {
		t.Error("View should contain clean confirmation")
	}
	if !strings.Contains(view, "y:") || !strings.Contains(view, "n:") {
		t.Error("View should contain yes/no options")
	}
}

func TestModel_View_StateConfirm_DryRun(t *testing.T) {
	model := NewModel(map[string][]cleaner.CleanTarget{
		"Frontend": {{Path: "/test", SizeBytes: 1024}},
	}, true)
	model.state = StateConfirm

	view := model.View()

	if !strings.Contains(view, "DRY RUN") {
		t.Error("View should indicate dry run mode")
	}
}

func TestModel_View_StateCleaning(t *testing.T) {
	model := NewModel(map[string][]cleaner.CleanTarget{
		"Frontend": {{Path: "/test", SizeBytes: 1024}},
	}, false)
	model.state = StateCleaning
	model.totalItems = 10
	model.cleanIndex = 5
	model.cleanedSize = 512

	view := model.View()

	// Should contain progress info
	if !strings.Contains(view, "Cleaning") {
		t.Error("View should contain cleaning message")
	}
	if !strings.Contains(view, "5") && !strings.Contains(view, "10") {
		t.Error("View should contain progress numbers")
	}
}

func TestModel_View_StateDone(t *testing.T) {
	model := NewModel(map[string][]cleaner.CleanTarget{}, false)
	model.state = StateDone
	model.cleanedSize = 1024 * 1024
	model.cleanIndex = 5

	view := model.View()

	// Should contain completion message
	if !strings.Contains(view, "complete") {
		t.Error("View should contain completion message")
	}
	// Should contain summary
	if !strings.Contains(view, "freed") {
		t.Error("View should contain freed space info")
	}
}

func TestModel_View_StateDone_DryRun(t *testing.T) {
	model := NewModel(map[string][]cleaner.CleanTarget{}, true)
	model.state = StateDone

	view := model.View()

	if !strings.Contains(view, "Dry run") {
		t.Error("View should indicate dry run completion")
	}
}

// =============================================================================
// keyMap Tests
// =============================================================================

func TestNewKeyMap(t *testing.T) {
	km := newKeyMap()

	// Verify all keybindings are set
	bindings := []struct {
		name    string
		binding interface{}
	}{
		{"Up", km.Up},
		{"Down", km.Down},
		{"Toggle", km.Toggle},
		{"All", km.All},
		{"None", km.None},
		{"Enter", km.Enter},
		{"Quit", km.Quit},
	}

	for _, b := range bindings {
		t.Run(b.name, func(t *testing.T) {
			// Just verify they exist and don't panic
			_ = b.binding
		})
	}
}

// =============================================================================
// cleanedMsg Tests
// =============================================================================

func TestCleanedMsg(t *testing.T) {
	msg := cleanedMsg{size: 2048}

	if msg.size != 2048 {
		t.Errorf("Expected size 2048, got %d", msg.size)
	}
}

// =============================================================================
// Style Tests
// =============================================================================

func TestStyles_Initialized(t *testing.T) {
	// Verify that all style variables are initialized and don't panic
	styles := []struct {
		name  string
		style interface{}
	}{
		{"primaryColor", primaryColor},
		{"secondaryColor", secondaryColor},
		{"successColor", successColor},
		{"warningColor", warningColor},
		{"dangerColor", dangerColor},
		{"mutedColor", mutedColor},
		{"titleStyle", titleStyle},
		{"subtitleStyle", subtitleStyle},
		{"selectedStyle", selectedStyle},
		{"mutedStyle", mutedStyle},
		{"helpStyle", helpStyle},
		{"headerBox", headerBox},
		{"statusBar", statusBar},
	}

	for _, s := range styles {
		t.Run(s.name, func(t *testing.T) {
			_ = s.style
		})
	}
}

// =============================================================================
// Integration Tests
// =============================================================================

func TestModel_FullSelectWorkflow(t *testing.T) {
	// Create model with multiple domains
	model := NewModel(map[string][]cleaner.CleanTarget{
		"Frontend": {
			{Path: "/npm", SizeBytes: 1000},
			{Path: "/yarn", SizeBytes: 2000},
		},
		"Backend": {
			{Path: "/pip", SizeBytes: 500},
		},
	}, false)

	// 1. Initial state should be StateSelect
	if model.state != StateSelect {
		t.Fatalf("Initial state should be StateSelect")
	}

	// 2. All items selected by default
	for _, item := range model.items {
		if !item.selected {
			t.Errorf("Item %s should be selected by default", item.domain)
		}
	}

	// 3. Deselect all with 'n'
	newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	model = newModel.(Model)

	for _, item := range model.items {
		if item.selected {
			t.Errorf("Item %s should be unselected after 'n'", item.domain)
		}
	}

	// 4. Select all with 'a'
	newModel, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	model = newModel.(Model)

	for _, item := range model.items {
		if !item.selected {
			t.Errorf("Item %s should be selected after 'a'", item.domain)
		}
	}

	// 5. Press enter to go to confirm
	newModel, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = newModel.(Model)

	if model.state != StateConfirm {
		t.Errorf("State should be StateConfirm, got %d", model.state)
	}

	// 6. Press 'n' to go back
	newModel, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	model = newModel.(Model)

	if model.state != StateSelect {
		t.Errorf("State should be StateSelect after cancel, got %d", model.state)
	}
}

func TestModel_CleaningWorkflow(t *testing.T) {
	model := NewModel(map[string][]cleaner.CleanTarget{
		"Frontend": {
			{Path: "/test1", SizeBytes: 1000},
			{Path: "/test2", SizeBytes: 2000},
		},
	}, true) // dry run

	// Go to confirm state
	model.state = StateConfirm

	// Confirm with 'y'
	newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	model = newModel.(Model)

	if model.state != StateCleaning {
		t.Fatalf("State should be StateCleaning")
	}

	// Simulate cleaning completion
	model.cleanIndex = model.totalItems
	model.state = StateDone

	// View should show dry run completion
	view := model.View()
	if !strings.Contains(view, "Dry run") {
		t.Error("Should show dry run completion")
	}
}

// =============================================================================
// Edge Cases
// =============================================================================

func TestModel_LargeNumberOfItems(t *testing.T) {
	targets := make(map[string][]cleaner.CleanTarget)
	for i := 0; i < 100; i++ {
		domain := string(rune('A' + (i % 26)))
		targets[domain] = append(targets[domain], cleaner.CleanTarget{
			Path:      "/path/" + string(rune('0'+i)),
			SizeBytes: int64(i * 1000),
		})
	}

	model := NewModel(targets, false)

	// Should not panic
	_ = model.View()
}

func TestModel_ZeroSizeItems(t *testing.T) {
	model := NewModel(map[string][]cleaner.CleanTarget{
		"Frontend": {
			{Path: "/test", SizeBytes: 0},
		},
	}, false)

	// Should handle zero-size items
	if len(model.items) != 1 {
		t.Errorf("Should have 1 item, got %d", len(model.items))
	}

	view := model.View()
	if !strings.Contains(view, "0 B") {
		t.Error("Should display zero bytes")
	}
}
