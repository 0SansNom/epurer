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
	analyzerBrowse analyzerState = iota
	analyzerConfirmDelete
)

// AnalyzerModel is the Bubble Tea model behind `epurer analyze`. It's a
// read-only directory browser sorted by size until the user explicitly
// requests deletion of a specific entry, which always requires confirmation.
//
// Directory loads run in the background rather than blocking the UI in a
// dedicated "loading" state: entering a directory with many/large children
// can take a while (ListDir sizes every subdirectory recursively), and if
// key handling only worked in a post-load state, the user would be stuck
// unable to even quit until the scan finished. The browse view - and its
// keybindings, including q and backspace - stay live throughout; loadGen
// discards any result for a directory the user has since navigated away
// from.
type AnalyzerModel struct {
	state       analyzerState
	path        string
	entries     []analyzer.Entry
	cursor      int
	spinner     spinner.Model
	dryRun      bool
	err         error
	message     string
	quitting    bool
	loading     bool
	pendingPath string
	loadGen     int
}

// NewAnalyzerModel creates a new AnalyzerModel rooted at path.
func NewAnalyzerModel(path string, dryRun bool) AnalyzerModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(primaryColor)

	return AnalyzerModel{
		state:       analyzerBrowse,
		path:        path,
		spinner:     s,
		dryRun:      dryRun,
		loading:     true,
		pendingPath: path,
	}
}

func (m AnalyzerModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, loadDir(m.path, m.loadGen))
}

type dirLoadedMsg struct {
	path    string
	entries []analyzer.Entry
	err     error
	gen     int
}

func loadDir(path string, gen int) tea.Cmd {
	return func() tea.Msg {
		entries, err := analyzer.ListDir(path)
		return dirLoadedMsg{path: path, entries: entries, err: err, gen: gen}
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

// startLoad kicks off a background load of path, invalidating any load
// already in flight (its result will arrive with a stale gen and be
// discarded when it lands).
func (m *AnalyzerModel) startLoad(path string) tea.Cmd {
	m.loadGen++
	m.loading = true
	m.pendingPath = path
	return loadDir(path, m.loadGen)
}

func (m AnalyzerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case dirLoadedMsg:
		if msg.gen != m.loadGen {
			return m, nil // stale - user navigated elsewhere before this landed
		}
		m.loading = false
		m.pendingPath = ""
		m.path = msg.path
		m.entries = msg.entries
		m.err = msg.err
		m.cursor = 0
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
		m.state = analyzerBrowse
		cmd := m.startLoad(m.path)
		return m, cmd

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
					m.message = ""
					cmd := m.startLoad(sel.Path)
					return m, cmd
				}
			case "backspace", "left", "h":
				parent := filepath.Dir(m.path)
				if parent != m.path {
					m.message = ""
					cmd := m.startLoad(parent)
					return m, cmd
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
				m.state = analyzerBrowse
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

	if m.state == analyzerConfirmDelete {
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

	switch {
	case m.err != nil:
		b.WriteString(lipgloss.NewStyle().Foreground(dangerColor).Render(fmt.Sprintf("Error: %v", m.err)))
		b.WriteString("\n")
	case m.loading && len(m.entries) == 0:
		b.WriteString(m.spinner.View())
		b.WriteString(fmt.Sprintf(" Scanning %s...", m.pendingPath))
		b.WriteString("\n")
	case len(m.entries) == 0:
		b.WriteString(mutedStyle.Render("(empty directory)"))
		b.WriteString("\n")
	default:
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

			sizeStr := utils.FormatBytes(e.Size)
			if e.Incomplete {
				// Size is a lower bound only - part of this directory
				// couldn't be read (commonly: no Full Disk Access on macOS).
				sizeStr = "≥" + sizeStr
			}

			line := fmt.Sprintf("%s%s %-30s %10s  %s",
				cursor, icon, truncateName(e.Name, 30), sizeStr, bar)
			if e.Incomplete && i != m.cursor {
				style = style.Foreground(warningColor)
			}
			b.WriteString(style.Render(line))
			b.WriteString("\n")
		}
	}

	if m.loading && len(m.entries) > 0 {
		b.WriteString("\n")
		b.WriteString(mutedStyle.Render(m.spinner.View() + fmt.Sprintf(" Loading %s...", m.pendingPath)))
		b.WriteString("\n")
	}

	if m.message != "" {
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Foreground(successColor).Render(m.message))
		b.WriteString("\n")
	}

	if hasIncompleteEntry(m.entries) {
		b.WriteString("\n")
		b.WriteString(mutedStyle.Render("≥ size is a lower bound - part of that folder is protected (no Full Disk Access)"))
	}

	b.WriteString("\n")
	help := "↑/↓: navigate • enter: open • backspace: up • r: reveal in Finder • d: delete • q: quit"
	b.WriteString(helpStyle.Render(help))

	return b.String()
}

func hasIncompleteEntry(entries []analyzer.Entry) bool {
	for _, e := range entries {
		if e.Incomplete {
			return true
		}
	}
	return false
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
