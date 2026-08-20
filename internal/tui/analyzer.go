package tui

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/0SansNom/epurer/internal/analyzer"
	"github.com/0SansNom/epurer/pkg/utils"
)

type analyzerState int

const (
	analyzerLoading analyzerState = iota
	analyzerBrowse
	analyzerConfirmDelete
)

// AnalyzerModel is the Bubble Tea model behind `epurer analyze`. It's a
// read-only directory browser sorted by size until the user explicitly
// requests deletion of a specific entry, which always requires confirmation.
type AnalyzerModel struct {
	state    analyzerState
	path     string
	entries  []analyzer.Entry
	cursor   int
	spinner  spinner.Model
	dryRun   bool
	err      error
	message  string
	quitting bool
}

// NewAnalyzerModel creates a new AnalyzerModel rooted at path.
func NewAnalyzerModel(path string, dryRun bool) AnalyzerModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(primaryColor)

	return AnalyzerModel{
		state:   analyzerLoading,
		path:    path,
		spinner: s,
		dryRun:  dryRun,
	}
}

func (m AnalyzerModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, loadDir(m.path))
}

type dirLoadedMsg struct {
	path    string
	entries []analyzer.Entry
	err     error
}

func loadDir(path string) tea.Cmd {
	return func() tea.Msg {
		entries, err := analyzer.ListDir(path)
		return dirLoadedMsg{path: path, entries: entries, err: err}
	}
}

type revealedMsg struct{ err error }

type deletedMsg struct {
	path string
	err  error
}

func revealInFinder(path string) tea.Cmd {
	return func() tea.Msg {
		err := exec.Command("open", "-R", path).Run()
		return revealedMsg{err: err}
	}
}

func deleteEntry(path string, dryRun bool) tea.Cmd {
	return func() tea.Msg {
		err := utils.SafeRemove(path, dryRun)
		return deletedMsg{path: path, err: err}
	}
}

func (m AnalyzerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case dirLoadedMsg:
		m.path = msg.path
		m.entries = msg.entries
		m.err = msg.err
		m.cursor = 0
		m.state = analyzerBrowse
		return m, nil

	case revealedMsg:
		if msg.err != nil {
			m.message = "Failed to reveal in Finder"
		} else {
			m.message = "Revealed in Finder"
		}
		return m, nil

	case deletedMsg:
		if msg.err != nil {
			m.message = fmt.Sprintf("Failed to remove: %v", msg.err)
		} else if m.dryRun {
			m.message = fmt.Sprintf("Would remove %s", filepath.Base(msg.path))
		} else {
			m.message = fmt.Sprintf("Removed %s", filepath.Base(msg.path))
		}
		m.state = analyzerLoading
		return m, loadDir(m.path)

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		switch m.state {
		case analyzerBrowse:
			switch msg.String() {
			case "q", "ctrl+c":
				m.quitting = true
				return m, tea.Quit
			case "up", "k":
				if m.cursor > 0 {
					m.cursor--
				}
			case "down", "j":
				if m.cursor < len(m.entries)-1 {
					m.cursor++
				}
			case "enter", "right", "l":
				if len(m.entries) == 0 {
					return m, nil
				}
				sel := m.entries[m.cursor]
				if sel.IsDir {
					m.state = analyzerLoading
					m.message = ""
					return m, loadDir(sel.Path)
				}
			case "backspace", "left", "h":
				parent := filepath.Dir(m.path)
				if parent != m.path {
					m.state = analyzerLoading
					m.message = ""
					return m, loadDir(parent)
				}
			case "r":
				if len(m.entries) == 0 {
					return m, nil
				}
				return m, revealInFinder(m.entries[m.cursor].Path)
			case "d":
				if len(m.entries) > 0 {
					m.state = analyzerConfirmDelete
				}
			}
		case analyzerConfirmDelete:
			switch msg.String() {
			case "y", "Y":
				sel := m.entries[m.cursor]
				m.state = analyzerLoading
				return m, deleteEntry(sel.Path, m.dryRun)
			case "n", "N", "q", "ctrl+c":
				m.state = analyzerBrowse
			}
		}
	}

	return m, nil
}

func (m AnalyzerModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	header := lipgloss.JoinVertical(
		lipgloss.Center,
		titleStyle.Render("🧹 Épurer — Analyze"),
		subtitleStyle.Render(m.path),
	)
	b.WriteString(headerBox.Render(header))
	b.WriteString("\n")

	switch m.state {
	case analyzerLoading:
		b.WriteString(m.spinner.View())
		b.WriteString(" Scanning...")
		return b.String()

	case analyzerConfirmDelete:
		sel := m.entries[m.cursor]
		confirmMsg := fmt.Sprintf("Delete %q (%s)?", sel.Name, utils.FormatBytes(sel.Size))
		if m.dryRun {
			confirmMsg += " (DRY RUN)"
		}
		b.WriteString(lipgloss.NewStyle().Foreground(warningColor).Bold(true).Render("⚠️  " + confirmMsg))
		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render("y: yes • n: no"))
		return b.String()
	}

	if m.err != nil {
		b.WriteString(lipgloss.NewStyle().Foreground(dangerColor).Render(fmt.Sprintf("Error: %v", m.err)))
		b.WriteString("\n")
	} else if len(m.entries) == 0 {
		b.WriteString(mutedStyle.Render("(empty directory)"))
		b.WriteString("\n")
	} else {
		var maxSize int64
		for _, e := range m.entries {
			if e.Size > maxSize {
				maxSize = e.Size
			}
		}

		const barWidth = 20
		for i, e := range m.entries {
			cursor := "  "
			style := lipgloss.NewStyle()
			if i == m.cursor {
				cursor = "▸ "
				style = style.Foreground(primaryColor).Bold(true)
			}

			icon := "📄"
			if e.IsDir {
				icon = "📁"
			}

			filled := 0
			if maxSize > 0 {
				filled = int(float64(e.Size) / float64(maxSize) * float64(barWidth))
			}
			bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

			line := fmt.Sprintf("%s%s %-30s %10s  %s",
				cursor, icon, truncateName(e.Name, 30), utils.FormatBytes(e.Size), bar)
			b.WriteString(style.Render(line))
			b.WriteString("\n")
		}
	}

	if m.message != "" {
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Foreground(successColor).Render(m.message))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	help := "↑/↓: navigate • enter: open • backspace: up • r: reveal in Finder • d: delete • q: quit"
	b.WriteString(helpStyle.Render(help))

	return b.String()
}

func truncateName(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}

// RunAnalyzer starts the interactive disk-usage explorer rooted at path.
func RunAnalyzer(path string, dryRun bool) error {
	m := NewAnalyzerModel(path, dryRun)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
