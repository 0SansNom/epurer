package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/0SansNom/epurer/internal/purge"
	"github.com/0SansNom/epurer/pkg/utils"
)

// ProjectItem represents one project's worth of removable artifacts in the
// purge selection list.
type ProjectItem struct {
	project  purge.Project
	selected bool
}

func (i ProjectItem) Title() string {
	checkbox := "[ ]"
	if i.selected {
		checkbox = "[✓]"
	}
	return fmt.Sprintf("%s %s (%s)", checkbox, i.project.Name, i.project.Type)
}

func (i ProjectItem) Description() string {
	kinds := make([]string, 0, len(i.project.Artifacts))
	for _, a := range i.project.Artifacts {
		kinds = append(kinds, a.Kind)
	}
	return fmt.Sprintf("%s • %s", utils.FormatBytes(i.project.TotalSize), strings.Join(kinds, ", "))
}

func (i ProjectItem) FilterValue() string {
	return i.project.Name
}

// PurgeModel is the Bubble Tea model driving `epurer purge`'s interactive
// project selection. It mirrors Model's flow (select -> confirm -> clean ->
// done) but operates on whole projects rather than domain-level items.
type PurgeModel struct {
	state            State
	list             list.Model
	items            []ProjectItem
	spinner          spinner.Model
	progress         progress.Model
	cleaning         bool
	cleanIndex       int
	totalItems       int
	cleanedSize      int64
	pendingArtifacts []purge.Artifact
	dryRun           bool
	quitting         bool
	width            int
	height           int
}

// NewPurgeModel creates a new PurgeModel. Projects that contain at least one
// artifact older than the scan's min-age cutoff (Artifact.Selected) start
// preselected; picking a project queues its entire artifact set for removal.
func NewPurgeModel(projects []purge.Project, dryRun bool) PurgeModel {
	items := make([]ProjectItem, 0, len(projects))
	for _, p := range projects {
		preselect := false
		for _, a := range p.Artifacts {
			if a.Selected {
				preselect = true
				break
			}
		}
		items = append(items, ProjectItem{project: p, selected: preselect})
	}

	listItems := make([]list.Item, len(items))
	for i, item := range items {
		listItems[i] = item
	}

	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.Foreground(primaryColor)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.Foreground(secondaryColor)

	l := list.New(listItems, delegate, 0, 0)
	l.Title = "Select projects to purge"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = titleStyle

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(primaryColor)

	p := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(40),
	)

	return PurgeModel{
		state:    StateSelect,
		list:     l,
		items:    items,
		spinner:  s,
		progress: p,
		dryRun:   dryRun,
	}
}

func (m PurgeModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick)
}

func (m PurgeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.state {
		case StateSelect:
			switch msg.String() {
			case "q", "ctrl+c":
				m.quitting = true
				return m, tea.Quit
			case " ":
				if i := m.list.Index(); i >= 0 && i < len(m.items) {
					m.items[i].selected = !m.items[i].selected
					m.syncListItems()
				}
			case "enter":
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
			case "a":
				for i := range m.items {
					m.items[i].selected = true
				}
				m.syncListItems()
			case "n":
				for i := range m.items {
					m.items[i].selected = false
				}
				m.syncListItems()
			}
		case StateConfirm:
			switch msg.String() {
			case "y", "Y":
				m.state = StateCleaning
				m.cleaning = true
				for _, item := range m.items {
					if item.selected {
						m.pendingArtifacts = append(m.pendingArtifacts, item.project.Artifacts...)
					}
				}
				m.totalItems = len(m.pendingArtifacts)
				if m.totalItems == 0 {
					m.state = StateDone
					m.cleaning = false
					return m, nil
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
		if msg.err == nil {
			m.cleanedSize += msg.size
		}
		m.cleanIndex++
		if m.cleanIndex >= m.totalItems {
			m.state = StateDone
			m.cleaning = false
		} else {
			return m, m.cleanNext()
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *PurgeModel) syncListItems() {
	listItems := make([]list.Item, len(m.items))
	for j, item := range m.items {
		listItems[j] = item
	}
	m.list.SetItems(listItems)
}

// cleanNext removes the artifact at m.cleanIndex from disk (or just reports
// its size in dry-run mode) and returns the outcome as a cleanedMsg.
func (m PurgeModel) cleanNext() tea.Cmd {
	if m.cleanIndex >= len(m.pendingArtifacts) {
		return func() tea.Msg { return cleanedMsg{} }
	}

	artifact := m.pendingArtifacts[m.cleanIndex]
	dryRun := m.dryRun

	return func() tea.Msg {
		if err := utils.SafeRemove(artifact.Path, dryRun); err != nil {
			return cleanedMsg{err: err}
		}
		return cleanedMsg{size: artifact.SizeBytes}
	}
}

func (m PurgeModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	header := lipgloss.JoinVertical(
		lipgloss.Center,
		titleStyle.Render("🧹 Épurer — Purge"),
		subtitleStyle.Render("Project artifact cleanup"),
	)
	b.WriteString(headerBox.Render(header))
	b.WriteString("\n")

	switch m.state {
	case StateSelect:
		b.WriteString(m.list.View())
		b.WriteString("\n")

		var totalSize int64
		var selectedCount int
		for _, item := range m.items {
			if item.selected {
				totalSize += item.project.TotalSize
				selectedCount++
			}
		}

		status := fmt.Sprintf(" Selected: %d projects • %s ", selectedCount, utils.FormatBytes(totalSize))
		b.WriteString(statusBar.Render(status))
		b.WriteString("\n")

		help := "↑/↓: navigate • space: toggle • a: all • n: none • enter: confirm • q: quit"
		b.WriteString(helpStyle.Render(help))

	case StateConfirm:
		var totalSize int64
		var selectedCount int
		for _, item := range m.items {
			if item.selected {
				totalSize += item.project.TotalSize
				selectedCount++
			}
		}

		confirmMsg := fmt.Sprintf("Purge %d projects (%s)?", selectedCount, utils.FormatBytes(totalSize))
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
		b.WriteString(" Purging...")
		b.WriteString("\n\n")

		percent := float64(m.cleanIndex) / float64(m.totalItems)
		b.WriteString(m.progress.ViewAs(percent))
		b.WriteString("\n")

		status := fmt.Sprintf("Cleaned: %d/%d artifacts • %s freed",
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
				Render("✅ Purge complete!"))
		}
		b.WriteString("\n\n")

		summary := fmt.Sprintf("💾 Space freed: %s\n📁 Artifacts removed: %d",
			utils.FormatBytes(m.cleanedSize),
			m.cleanIndex)
		b.WriteString(summary)
		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render("Press enter or q to exit"))
	}

	return b.String()
}

// RunPurge starts the interactive purge TUI.
func RunPurge(projects []purge.Project, dryRun bool) error {
	m := NewPurgeModel(projects, dryRun)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
