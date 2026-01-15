package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/0SansNom/epurer/internal/cleaner"
	"github.com/0SansNom/epurer/pkg/utils"
)

// Colors
var (
	primaryColor   = lipgloss.Color("#7C3AED")
	secondaryColor = lipgloss.Color("#06B6D4")
	successColor   = lipgloss.Color("#10B981")
	warningColor   = lipgloss.Color("#F59E0B")
	dangerColor    = lipgloss.Color("#EF4444")
	mutedColor     = lipgloss.Color("#6B7280")
)

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			MarginBottom(1)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(secondaryColor)

	selectedStyle = lipgloss.NewStyle().
			Foreground(successColor).
			Bold(true)

	mutedStyle = lipgloss.NewStyle().
			Foreground(mutedColor)

	helpStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			MarginTop(1)

	headerBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(0, 2).
			Align(lipgloss.Center).
			MarginBottom(1)

	statusBar = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(primaryColor).
			Padding(0, 1).
			MarginTop(1)
)

// CleanItem represents a cleanable item in the list
type CleanItem struct {
	domain      string
	description string
	size        int64
	selected    bool
	targets     []cleaner.CleanTarget
}

func (i CleanItem) Title() string {
	checkbox := "[ ]"
	if i.selected {
		checkbox = "[✓]"
	}
	return fmt.Sprintf("%s %s", checkbox, i.domain)
}

func (i CleanItem) Description() string {
	return fmt.Sprintf("%s • %d items", utils.FormatBytes(i.size), len(i.targets))
}

func (i CleanItem) FilterValue() string {
	return i.domain
}

// State represents the current state of the TUI
type State int

const (
	StateSelect State = iota
	StateConfirm
	StateCleaning
	StateDone
)

// Model is the main Bubble Tea model
type Model struct {
	state       State
	list        list.Model
	items       []CleanItem
	spinner     spinner.Model
	progress    progress.Model
	cleaning    bool
	cleanIndex  int
	totalItems  int
	cleanedSize int64
	dryRun      bool
	quitting    bool
	err         error
	width       int
	height      int
}

// NewModel creates a new TUI model
func NewModel(targetsByDomain map[string][]cleaner.CleanTarget, dryRun bool) Model {
	// Create list items from targets
	var items []CleanItem
	for domain, targets := range targetsByDomain {
		if len(targets) == 0 {
			continue
		}

		totalSize := int64(0)
		for _, t := range targets {
			totalSize += t.SizeBytes
		}

		items = append(items, CleanItem{
			domain:      domain,
			description: fmt.Sprintf("%d items", len(targets)),
			size:        totalSize,
			selected:    true, // Selected by default
			targets:     targets,
		})
	}

	// Convert to list.Item slice
	listItems := make([]list.Item, len(items))
	for i, item := range items {
		listItems[i] = item
	}

	// Create list
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.Foreground(primaryColor)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.Foreground(secondaryColor)

	l := list.New(listItems, delegate, 0, 0)
	l.Title = "Select domains to clean"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = titleStyle

	// Create spinner
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(primaryColor)

	// Create progress bar
	p := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(40),
	)

	return Model{
		state:    StateSelect,
		list:     l,
		items:    items,
		spinner:  s,
		progress: p,
		dryRun:   dryRun,
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick)
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.state {
		case StateSelect:
			switch msg.String() {
			case "q", "ctrl+c":
				m.quitting = true
				return m, tea.Quit
			case " ": // Space to toggle selection
				if i := m.list.Index(); i >= 0 && i < len(m.items) {
					m.items[i].selected = !m.items[i].selected
					// Update list item
					listItems := make([]list.Item, len(m.items))
					for j, item := range m.items {
						listItems[j] = item
					}
					m.list.SetItems(listItems)
				}
			case "enter":
				// Check if any items are selected
				hasSelected := false
				for _, item := range m.items {
					if item.selected {
						hasSelected = true
						break
					}
				}
				if hasSelected {
					m.state = StateConfirm
				}
			case "a": // Select all
				for i := range m.items {
					m.items[i].selected = true
				}
				listItems := make([]list.Item, len(m.items))
				for j, item := range m.items {
					listItems[j] = item
				}
				m.list.SetItems(listItems)
			case "n": // Select none
				for i := range m.items {
					m.items[i].selected = false
				}
				listItems := make([]list.Item, len(m.items))
				for j, item := range m.items {
					listItems[j] = item
				}
				m.list.SetItems(listItems)
			}
		case StateConfirm:
			switch msg.String() {
			case "y", "Y":
				m.state = StateCleaning
				m.cleaning = true
				// Count total items
				for _, item := range m.items {
					if item.selected {
						m.totalItems += len(item.targets)
					}
				}
				return m, m.cleanNext()
			case "n", "N", "q", "ctrl+c":
				m.state = StateSelect
			}
		case StateDone:
			if msg.String() == "q" || msg.String() == "ctrl+c" || msg.String() == "enter" {
				m.quitting = true
				return m, tea.Quit
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(msg.Width-4, msg.Height-10)

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case progress.FrameMsg:
		progressModel, cmd := m.progress.Update(msg)
		m.progress = progressModel.(progress.Model)
		return m, cmd

	case cleanedMsg:
		m.cleanedSize += msg.size
		m.cleanIndex++
		if m.cleanIndex >= m.totalItems {
			m.state = StateDone
			m.cleaning = false
		} else {
			return m, m.cleanNext()
		}
	}

	// Update list
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

type cleanedMsg struct {
	size int64
}

func (m Model) cleanNext() tea.Cmd {
	return func() tea.Msg {
		// Simulate cleaning (in real implementation, this would actually clean)
		// For now, just return a message
		return cleanedMsg{size: 1024}
	}
}

// View renders the model
func (m Model) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	// Header
	header := lipgloss.JoinVertical(
		lipgloss.Center,
		titleStyle.Render("🧹 Épurer"),
		subtitleStyle.Render("Interactive cleanup mode"),
	)
	b.WriteString(headerBox.Render(header))
	b.WriteString("\n")

	switch m.state {
	case StateSelect:
		b.WriteString(m.list.View())
		b.WriteString("\n")

		// Calculate total selected size
		var totalSize int64
		var selectedCount int
		for _, item := range m.items {
			if item.selected {
				totalSize += item.size
				selectedCount++
			}
		}

		// Status bar
		status := fmt.Sprintf(" Selected: %d domains • %s ", selectedCount, utils.FormatBytes(totalSize))
		b.WriteString(statusBar.Render(status))
		b.WriteString("\n")

		// Help
		help := "↑/↓: navigate • space: toggle • a: all • n: none • enter: confirm • q: quit"
		b.WriteString(helpStyle.Render(help))

	case StateConfirm:
		var totalSize int64
		var selectedCount int
		for _, item := range m.items {
			if item.selected {
				totalSize += item.size
				selectedCount++
			}
		}

		confirmMsg := fmt.Sprintf(
			"Clean %d domains (%s)?",
			selectedCount,
			utils.FormatBytes(totalSize),
		)
		if m.dryRun {
			confirmMsg += " (DRY RUN)"
		}

		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().
			Foreground(warningColor).
			Bold(true).
			Render("⚠️  " + confirmMsg))
		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render("y: yes • n: no"))

	case StateCleaning:
		b.WriteString("\n")
		b.WriteString(m.spinner.View())
		b.WriteString(" Cleaning...")
		b.WriteString("\n\n")

		percent := float64(m.cleanIndex) / float64(m.totalItems)
		b.WriteString(m.progress.ViewAs(percent))
		b.WriteString("\n")

		status := fmt.Sprintf("Cleaned: %d/%d items • %s freed",
			m.cleanIndex, m.totalItems, utils.FormatBytes(m.cleanedSize))
		b.WriteString(mutedStyle.Render(status))

	case StateDone:
		b.WriteString("\n")
		if m.dryRun {
			b.WriteString(lipgloss.NewStyle().
				Foreground(secondaryColor).
				Bold(true).
				Render("✨ Dry run complete!"))
		} else {
			b.WriteString(lipgloss.NewStyle().
				Foreground(successColor).
				Bold(true).
				Render("✅ Cleaning complete!"))
		}
		b.WriteString("\n\n")

		summary := fmt.Sprintf("💾 Space freed: %s\n📁 Items cleaned: %d",
			utils.FormatBytes(m.cleanedSize),
			m.cleanIndex)
		b.WriteString(summary)
		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render("Press enter or q to exit"))
	}

	return b.String()
}

// Run starts the TUI
func Run(targetsByDomain map[string][]cleaner.CleanTarget, dryRun bool) error {
	m := NewModel(targetsByDomain, dryRun)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// Keymap for custom key bindings
type keyMap struct {
	Up     key.Binding
	Down   key.Binding
	Toggle key.Binding
	All    key.Binding
	None   key.Binding
	Enter  key.Binding
	Quit   key.Binding
}

func newKeyMap() keyMap {
	return keyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Toggle: key.NewBinding(
			key.WithKeys(" "),
			key.WithHelp("space", "toggle"),
		),
		All: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "select all"),
		),
		None: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "select none"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "confirm"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}
}
