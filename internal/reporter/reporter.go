package reporter

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/lipgloss"

	"github.com/0SansNom/epurer/internal/cleaner"
	"github.com/0SansNom/epurer/internal/config"
	"github.com/0SansNom/epurer/pkg/utils"
)

// Styles using Lip Gloss
var (
	// Colors
	primaryColor   = lipgloss.Color("#7C3AED") // Purple
	secondaryColor = lipgloss.Color("#06B6D4") // Cyan
	successColor   = lipgloss.Color("#10B981") // Green
	warningColor   = lipgloss.Color("#F59E0B") // Amber
	dangerColor    = lipgloss.Color("#EF4444") // Red
	mutedColor     = lipgloss.Color("#6B7280") // Gray

	// Text styles
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(secondaryColor)

	successStyle = lipgloss.NewStyle().
			Foreground(successColor)

	warningStyle = lipgloss.NewStyle().
			Foreground(warningColor)

	errorStyle = lipgloss.NewStyle().
			Foreground(dangerColor)

	infoStyle = lipgloss.NewStyle().
			Foreground(secondaryColor)

	mutedStyle = lipgloss.NewStyle().
			Foreground(mutedColor)

	// Box styles
	headerBox = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(primaryColor).
			Padding(0, 2).
			Align(lipgloss.Center)

	// Table styles
	tableHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(primaryColor).
				Padding(0, 1)

	tableCellStyle = lipgloss.NewStyle().
			Padding(0, 1)

	tableFooterStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(primaryColor).
				Padding(0, 1)
)

// Reporter handles all output formatting and display
type Reporter struct {
	verbose  bool
	progress progress.Model
}

// NewReporter creates a new Reporter
func NewReporter(verbose bool) *Reporter {
	p := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(40),
	)
	return &Reporter{
		verbose:  verbose,
		progress: p,
	}
}

// PrintHeader prints the application header
func (r *Reporter) PrintHeader() {
	title := "🧹 Épurer v1.1"
	subtitle := "Intelligent cache cleanup for macOS"

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		titleStyle.Render(title),
		subtitleStyle.Render(subtitle),
	)

	box := headerBox.Render(content)
	fmt.Println()
	fmt.Println(box)
	fmt.Println()
}

// PrintDetection prints the detection results
func (r *Reporter) PrintDetection(detected map[string]bool) {
	fmt.Println(warningStyle.Render("\n🔍 Detecting development tools...\n"))

	detectionMap := map[string]string{
		"frontend": "Frontend (Node.js, npm, yarn)",
		"backend":  "Backend (Python, Java, Go, Rust, PHP, Ruby)",
		"mobile":   "Mobile (Xcode, Android, Flutter)",
		"devops":   "DevOps (Docker, Kubernetes, Terraform)",
		"dataml":   "Data/ML (Conda, Jupyter, TensorFlow)",
		"system":   "System (Caches, Logs, Homebrew)",
	}

	for domain, description := range detectionMap {
		if detected[domain] {
			fmt.Printf("  %s %s\n", successStyle.Render("✓"), description)
		} else {
			if r.verbose {
				fmt.Printf("  %s %s\n", errorStyle.Render("✗"), mutedStyle.Render(description))
			}
		}
	}

	fmt.Println()
}

// PrintEstimation prints a table of estimated cleanup sizes
func (r *Reporter) PrintEstimation(targetsByDomain map[string][]cleaner.CleanTarget) {
	fmt.Println(warningStyle.Render("📊 Cleanup Estimation:\n"))

	totalSize := int64(0)
	totalItems := 0

	// Collect data first
	type rowData struct {
		domain  string
		items   string
		size    string
		safety  string
		impact  string
	}
	var rows []rowData

	// Sort domains for consistent output
	domains := []string{"Frontend", "Backend", "Mobile", "DevOps", "Data/ML", "System"}

	for _, domain := range domains {
		targets, exists := targetsByDomain[domain]
		if !exists || len(targets) == 0 {
			continue
		}

		domainSize := int64(0)
		safetyIcons := make(map[config.SafetyLevel]bool)

		for _, target := range targets {
			domainSize += target.SizeBytes
			safetyIcons[target.Safety] = true
		}

		// Build safety string (simple text, no emoji for alignment)
		safetyStr := ""
		if safetyIcons[config.Safe] {
			safetyStr += "Safe "
		}
		if safetyIcons[config.Moderate] {
			safetyStr += "Mod "
		}
		if safetyIcons[config.Dangerous] {
			safetyStr += "Risk "
		}

		impact := getImpactString(domainSize)

		rows = append(rows, rowData{
			domain:  domain,
			items:   utils.FormatCount(len(targets)),
			size:    utils.FormatBytes(domainSize),
			safety:  strings.TrimSpace(safetyStr),
			impact:  impact,
		})

		totalSize += domainSize
		totalItems += len(targets)
	}

	// Build table with lipgloss
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(primaryColor).Padding(0, 1)
	cellStyle := lipgloss.NewStyle().Padding(0, 1)

	// Print header
	fmt.Printf("%s%s%s%s%s\n",
		headerStyle.Width(12).Render("DOMAIN"),
		headerStyle.Width(8).Align(lipgloss.Right).Render("ITEMS"),
		headerStyle.Width(10).Align(lipgloss.Right).Render("SIZE"),
		headerStyle.Width(10).Render("SAFETY"),
		headerStyle.Width(10).Render("IMPACT"),
	)

	// Print separator
	fmt.Println(mutedStyle.Render(strings.Repeat("─", 50)))

	// Print rows
	for _, row := range rows {
		impactStyled := row.impact
		switch row.impact {
		case "Very High":
			impactStyled = errorStyle.Render(row.impact)
		case "High":
			impactStyled = warningStyle.Render(row.impact)
		case "Medium":
			impactStyled = infoStyle.Render(row.impact)
		default:
			impactStyled = mutedStyle.Render(row.impact)
		}

		safetyStyled := row.safety
		if strings.Contains(row.safety, "Safe") {
			safetyStyled = successStyle.Render(row.safety)
		} else if strings.Contains(row.safety, "Risk") {
			safetyStyled = errorStyle.Render(row.safety)
		}

		fmt.Printf("%s%s%s%s%s\n",
			cellStyle.Width(12).Render(row.domain),
			cellStyle.Width(8).Align(lipgloss.Right).Render(row.items),
			cellStyle.Width(10).Align(lipgloss.Right).Render(row.size),
			cellStyle.Width(10).Render(safetyStyled),
			cellStyle.Width(10).Render(impactStyled),
		)
	}

	// Print footer
	fmt.Println(mutedStyle.Render(strings.Repeat("─", 50)))
	fmt.Printf("%s%s%s%s%s\n",
		titleStyle.Padding(0, 1).Width(12).Render("Total"),
		successStyle.Padding(0, 1).Width(8).Align(lipgloss.Right).Render(utils.FormatCount(totalItems)),
		successStyle.Padding(0, 1).Width(10).Align(lipgloss.Right).Render(utils.FormatBytes(totalSize)),
		cellStyle.Width(10).Render(""),
		cellStyle.Width(10).Render(""),
	)
	fmt.Println()
}

// PrintTargetDetails prints detailed information about targets
func (r *Reporter) PrintTargetDetails(targets []cleaner.CleanTarget) {
	if !r.verbose {
		return
	}

	fmt.Println(warningStyle.Render("\n📋 Detailed Breakdown:\n"))

	for _, target := range targets {
		safetyIcon := target.Safety.Icon()
		fmt.Printf("  %s %s - %s (%s)\n",
			safetyIcon,
			target.Description,
			successStyle.Render(utils.FormatBytes(target.SizeBytes)),
			mutedStyle.Render(target.Path),
		)
	}

	fmt.Println()
}

// PrintProgress prints a progress indicator
func (r *Reporter) PrintProgress(current, total int, description string) {
	percent := float64(current) / float64(total)
	bar := r.progress.ViewAs(percent)
	fmt.Printf("\r%s %s [%d/%d]", description, bar, current, total)
	if current == total {
		fmt.Println()
	}
}

// PrintCleanResults prints the results of a cleaning operation
func (r *Reporter) PrintCleanResults(results []cleaner.CleanResult, dryRun bool) {
	if dryRun {
		fmt.Println(infoStyle.Render("\n✨ Dry Run Complete - No files were deleted\n"))
	} else {
		fmt.Println(successStyle.Render("\n✅ Cleaning Complete!\n"))
	}

	// Calculate statistics
	totalFreed := int64(0)
	totalFiles := 0
	failures := 0

	for _, result := range results {
		totalFreed += result.BytesFreed
		if result.Success {
			totalFiles++
		} else {
			failures++
		}
	}

	// Print summary with styled output
	actionVerb := getActionVerb(dryRun)

	fmt.Printf("  💾 Space %s: %s\n",
		actionVerb,
		successStyle.Render(utils.FormatBytes(totalFreed)),
	)
	fmt.Printf("  📁 Items %s: %s\n",
		actionVerb,
		successStyle.Render(utils.FormatCount(totalFiles)),
	)

	if failures > 0 {
		fmt.Printf("  ❌ Failures: %s\n",
			errorStyle.Render(utils.FormatCount(failures)),
		)
	}

	// Print failures if any
	if failures > 0 && r.verbose {
		fmt.Println(errorStyle.Render("\n❌ Failed Items:\n"))
		for _, result := range results {
			if !result.Success {
				fmt.Printf("  • %s: %v\n",
					mutedStyle.Render(result.Target.Path),
					errorStyle.Render(result.Error.Error()),
				)
			}
		}
	}

	fmt.Println()
}

// PrintWarning prints a warning message
func (r *Reporter) PrintWarning(message string) {
	fmt.Println(warningStyle.Render("⚠️  " + message))
}

// PrintError prints an error message
func (r *Reporter) PrintError(message string) {
	fmt.Println(errorStyle.Render("❌ " + message))
}

// PrintSuccess prints a success message
func (r *Reporter) PrintSuccess(message string) {
	fmt.Println(successStyle.Render("✅ " + message))
}

// PrintInfo prints an info message
func (r *Reporter) PrintInfo(message string) {
	fmt.Println(infoStyle.Render("ℹ️  " + message))
}

// AskConfirmation asks the user for confirmation
func (r *Reporter) AskConfirmation(message string) bool {
	prompt := warningStyle.Render(message + " [y/N]: ")
	fmt.Printf("\n%s", prompt)

	var response string
	fmt.Scanln(&response)

	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes"
}

// PrintSafetyLegend prints the safety level legend
func (r *Reporter) PrintSafetyLegend() {
	fmt.Println(warningStyle.Render("\n🔐 Safety Levels:\n"))

	safeBox := successStyle.Render("Safe")
	moderateBox := warningStyle.Render("Mod")
	dangerBox := errorStyle.Render("Risk")

	fmt.Printf("  %s - No risk, easily rebuilt (caches, logs)\n", safeBox)
	fmt.Printf("  %s  - Rebuild needed (dependencies, build outputs)\n", moderateBox)
	fmt.Printf("  %s - Potential data loss (backups, databases)\n", dangerBox)
	fmt.Println()
}

// Helper functions

func getImpactString(size int64) string {
	const (
		low    = 500 * 1024 * 1024       // 500 MB
		medium = 5 * 1024 * 1024 * 1024  // 5 GB
		high   = 20 * 1024 * 1024 * 1024 // 20 GB
	)

	switch {
	case size < low:
		return "Low"
	case size < medium:
		return "Medium"
	case size < high:
		return "High"
	default:
		return "Very High"
	}
}

func getActionVerb(dryRun bool) string {
	if dryRun {
		return "would be freed"
	}
	return "freed"
}

// Ensure Reporter doesn't use os.Stdout directly for tests
var _ = os.Stdout
