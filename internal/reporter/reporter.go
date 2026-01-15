package reporter

import (
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/schollz/progressbar/v3"

	"github.com/0SansNom/epurer/internal/cleaner"
	"github.com/0SansNom/epurer/internal/config"
	"github.com/0SansNom/epurer/pkg/utils"
)

// Reporter handles all output formatting and display
type Reporter struct {
	verbose bool
}

// NewReporter creates a new Reporter
func NewReporter(verbose bool) *Reporter {
	return &Reporter{
		verbose: verbose,
	}
}

// PrintHeader prints the application header
func (r *Reporter) PrintHeader() {
	header := `
╔═══════════════════════════════════════════╗
║   🧹 Épurer v1.0                          ║
║   Intelligent cache cleanup for macOS     ║
╚═══════════════════════════════════════════╝
`
	fmt.Println(color.CyanString(header))
}

// PrintDetection prints the detection results
func (r *Reporter) PrintDetection(detected map[string]bool) {
	fmt.Println(color.YellowString("\n🔍 Detecting development tools...\n"))

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
			fmt.Printf("  %s %s\n", color.GreenString("✓"), description)
		} else {
			if r.verbose {
				fmt.Printf("  %s %s\n", color.RedString("✗"), description)
			}
		}
	}

	fmt.Println()
}

// PrintEstimation prints a table of estimated cleanup sizes
func (r *Reporter) PrintEstimation(targetsByDomain map[string][]cleaner.CleanTarget) {
	fmt.Println(color.YellowString("📊 Cleanup Estimation:\n"))

	table := tablewriter.NewWriter(os.Stdout)
	table.Header([]string{"Domain", "Items", "Size", "Safety", "Impact"})

	totalSize := int64(0)
	totalItems := 0

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

		// Build safety string
		safetyStr := ""
		if safetyIcons[config.Safe] {
			safetyStr += "🟢 "
		}
		if safetyIcons[config.Moderate] {
			safetyStr += "🟡 "
		}
		if safetyIcons[config.Dangerous] {
			safetyStr += "🔴 "
		}

		// Determine impact
		impact := getImpactString(domainSize)

		table.Append([]string{
			domain,
			utils.FormatCount(len(targets)),
			utils.FormatBytes(domainSize),
			safetyStr,
			impact,
		})

		totalSize += domainSize
		totalItems += len(targets)
	}

	// Footer with totals
	table.Footer([]string{
		"Total",
		utils.FormatCount(totalItems),
		utils.FormatBytes(totalSize),
		"",
		"",
	})

	table.Render()
	fmt.Println()
}

// PrintTargetDetails prints detailed information about targets
func (r *Reporter) PrintTargetDetails(targets []cleaner.CleanTarget) {
	if !r.verbose {
		return
	}

	fmt.Println(color.YellowString("\n📋 Detailed Breakdown:\n"))

	for _, target := range targets {
		safetyIcon := target.Safety.Icon()
		fmt.Printf("  %s %s - %s (%s)\n",
			safetyIcon,
			target.Description,
			utils.FormatBytes(target.SizeBytes),
			target.Path,
		)
	}

	fmt.Println()
}

// PrintCleaningProgress creates and returns a progress bar
func (r *Reporter) PrintCleaningProgress(total int, description string) *progressbar.ProgressBar {
	return progressbar.NewOptions(total,
		progressbar.OptionSetDescription(description),
		progressbar.OptionSetWidth(40),
		progressbar.OptionShowCount(),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "█",
			SaucerPadding: "░",
			BarStart:      "[",
			BarEnd:        "]",
		}),
	)
}

// PrintCleanResults prints the results of a cleaning operation
func (r *Reporter) PrintCleanResults(results []cleaner.CleanResult, dryRun bool) {
	if dryRun {
		fmt.Println(color.BlueString("\n✨ Dry Run Complete - No files were deleted\n"))
	} else {
		fmt.Println(color.GreenString("\n✅ Cleaning Complete!\n"))
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

	// Print summary
	fmt.Printf("Space %s: %s\n",
		getActionVerb(dryRun),
		color.GreenString(utils.FormatBytes(totalFreed)),
	)
	fmt.Printf("Items %s: %s\n",
		getActionVerb(dryRun),
		color.GreenString(utils.FormatCount(totalFiles)),
	)

	if failures > 0 {
		fmt.Printf("Failures: %s\n",
			color.RedString(utils.FormatCount(failures)),
		)
	}

	// Print failures if any
	if failures > 0 && r.verbose {
		fmt.Println(color.RedString("\n❌ Failed Items:\n"))
		for _, result := range results {
			if !result.Success {
				fmt.Printf("  • %s: %v\n", result.Target.Path, result.Error)
			}
		}
	}

	fmt.Println()
}

// PrintWarning prints a warning message
func (r *Reporter) PrintWarning(message string) {
	fmt.Println(color.YellowString("⚠️  " + message))
}

// PrintError prints an error message
func (r *Reporter) PrintError(message string) {
	fmt.Println(color.RedString("❌ " + message))
}

// PrintSuccess prints a success message
func (r *Reporter) PrintSuccess(message string) {
	fmt.Println(color.GreenString("✅ " + message))
}

// PrintInfo prints an info message
func (r *Reporter) PrintInfo(message string) {
	fmt.Println(color.CyanString("ℹ️  " + message))
}

// AskConfirmation asks the user for confirmation
func (r *Reporter) AskConfirmation(message string) bool {
	fmt.Printf("\n%s [y/N]: ", color.YellowString(message))

	var response string
	fmt.Scanln(&response)

	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes"
}

// PrintSafetyLegend prints the safety level legend
func (r *Reporter) PrintSafetyLegend() {
	fmt.Println(color.YellowString("\n🔐 Safety Levels:\n"))
	fmt.Printf("  🟢 %s - No risk, easily rebuilt (caches, logs)\n", color.GreenString("Safe"))
	fmt.Printf("  🟡 %s - Rebuild needed (dependencies, build outputs)\n", color.YellowString("Moderate"))
	fmt.Printf("  🔴 %s - Potential data loss (backups, databases)\n", color.RedString("Dangerous"))
	fmt.Println()
}

// Helper functions

func getImpactString(size int64) string {
	const (
		low      = 500 * 1024 * 1024        // 500 MB
		medium   = 5 * 1024 * 1024 * 1024   // 5 GB
		high     = 20 * 1024 * 1024 * 1024  // 20 GB
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
