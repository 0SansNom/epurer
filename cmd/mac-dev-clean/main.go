package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/prenomnom/mac-dev-clean/internal/cleaner"
	"github.com/prenomnom/mac-dev-clean/internal/config"
	"github.com/prenomnom/mac-dev-clean/internal/detector"
	"github.com/prenomnom/mac-dev-clean/internal/reporter"
)

var (
	// Global flags
	dryRun      bool
	verbose     bool
	interactive bool

	// Clean command flags
	cleanLevel string
	domains    []string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "mac-dev-clean",
		Short: "🧹 Mac Developer Cleaner - Intelligent cache cleanup",
		Long: `Mac Developer Cleaner intelligently scans and cleans development caches,
build artifacts, and temporary files on macOS.

Supports: Node.js, Python, Java, Go, Rust, PHP, Ruby, Docker, Kubernetes,
Xcode, Android, Flutter, TensorFlow, PyTorch, and more.`,
		Version: "1.0.0",
	}

	// Global flags
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")

	// Commands
	rootCmd.AddCommand(
		newCleanCmd(),
		newDetectCmd(),
		newReportCmd(),
		newSmartCmd(),
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// newCleanCmd creates the clean command
func newCleanCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clean",
		Short: "Clean development caches and temporary files",
		Long: `Scan and clean development caches, build artifacts, and temporary files.

Safety Levels:
  conservative - Only safe items (caches, logs)
  standard     - Safe + moderate items (node_modules, builds)
  aggressive   - All items including dangerous ones (backups, data)

Domains:
  frontend - Node.js, npm, yarn, pnpm
  backend  - Python, Java, Go, Rust, PHP, Ruby
  mobile   - Xcode, Android, Flutter
  devops   - Docker, Kubernetes, Terraform
  dataml   - Conda, Jupyter, TensorFlow, PyTorch
  system   - System caches, logs, Homebrew`,
		RunE: runClean,
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be cleaned without actually deleting")
	cmd.Flags().BoolVarP(&interactive, "interactive", "i", true, "Ask for confirmation before cleaning")
	cmd.Flags().StringVarP(&cleanLevel, "level", "l", "standard", "Clean level (conservative|standard|aggressive)")
	cmd.Flags().StringSliceVarP(&domains, "domain", "d", []string{}, "Domains to clean (comma-separated, empty = all)")

	return cmd
}

// newDetectCmd creates the detect command
func newDetectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "detect",
		Short: "Detect installed development tools",
		Long:  `Automatically detect which development tools and frameworks are installed on your system.`,
		RunE:  runDetect,
	}

	return cmd
}

// newReportCmd creates the report command
func newReportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "report",
		Short: "Generate a cleanup estimation report",
		Long:  `Scan the system and generate a detailed report of what can be cleaned, without actually cleaning anything.`,
		RunE:  runReport,
	}

	cmd.Flags().StringVarP(&cleanLevel, "level", "l", "standard", "Clean level (conservative|standard|aggressive)")
	cmd.Flags().StringSliceVarP(&domains, "domain", "d", []string{}, "Domains to scan (comma-separated, empty = all)")

	return cmd
}

// newSmartCmd creates the smart command
func newSmartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "smart",
		Short: "Smart automatic cleanup based on detected tools",
		Long: `Automatically detects installed development tools and performs an intelligent cleanup
using conservative settings. Perfect for quick, safe cleanup.`,
		RunE: runSmart,
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be cleaned without actually deleting")

	return cmd
}

// runClean executes the clean command
func runClean(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	rep := reporter.NewReporter(verbose)

	rep.PrintHeader()

	// Parse clean level
	level, err := config.ParseCleanLevel(cleanLevel)
	if err != nil {
		rep.PrintError(err.Error())
		return err
	}

	// Create config
	cfg := config.NewDefaultConfig()
	cfg.DryRun = dryRun
	cfg.Verbose = verbose
	cfg.CleanLevel = level
	cfg.Interactive = interactive

	// Initialize cleaners
	cleaners, err := initAllCleaners()
	if err != nil {
		rep.PrintError(fmt.Sprintf("Failed to initialize cleaners: %v", err))
		return err
	}

	// Filter by domain if specified
	if len(domains) > 0 {
		cleaners = filterCleanersByDomain(cleaners, domains)
	}

	// Detect and scan
	rep.PrintInfo("Scanning system...")

	targetsByDomain := make(map[string][]cleaner.CleanTarget)
	detected := make(map[string]bool)

	for _, c := range cleaners {
		isDetected, err := c.Detect(ctx)
		if err != nil {
			if verbose {
				rep.PrintWarning(fmt.Sprintf("Detection error for %s: %v", c.Name(), err))
			}
			continue
		}

		if !isDetected {
			continue
		}

		detected[c.Name()] = true

		targets, err := c.Scan(ctx, cfg)
		if err != nil {
			if verbose {
				rep.PrintWarning(fmt.Sprintf("Scan error for %s: %v", c.Name(), err))
			}
			continue
		}

		if len(targets) > 0 {
			targetsByDomain[c.Name()] = targets
		}
	}

	// Print estimation
	rep.PrintEstimation(targetsByDomain)
	rep.PrintSafetyLegend()

	// Calculate totals
	totalTargets := 0
	for _, targets := range targetsByDomain {
		totalTargets += len(targets)
	}

	if totalTargets == 0 {
		rep.PrintInfo("Nothing to clean!")
		return nil
	}

	// Ask for confirmation if interactive
	if interactive && !dryRun {
		if !rep.AskConfirmation(fmt.Sprintf("Proceed with cleaning %d items?", totalTargets)) {
			rep.PrintInfo("Cancelled")
			return nil
		}
	}

	// Execute cleanup
	if dryRun {
		rep.PrintInfo("DRY RUN - No files will be deleted")
	}

	allResults := []cleaner.CleanResult{}

	for domain, targets := range targetsByDomain {
		rep.PrintInfo(fmt.Sprintf("Cleaning %s...", domain))

		// Find the cleaner for this domain
		var domainCleaner cleaner.Cleaner
		for _, c := range cleaners {
			if c.Name() == domain {
				domainCleaner = c
				break
			}
		}

		if domainCleaner == nil {
			continue
		}

		results, err := domainCleaner.Clean(ctx, targets, dryRun)
		if err != nil {
			rep.PrintWarning(fmt.Sprintf("Error cleaning %s: %v", domain, err))
			continue
		}

		allResults = append(allResults, results...)
	}

	// Print results
	rep.PrintCleanResults(allResults, dryRun)

	return nil
}

// runDetect executes the detect command
func runDetect(cmd *cobra.Command, args []string) error {
	rep := reporter.NewReporter(verbose)
	rep.PrintHeader()

	det, err := detector.NewDetector()
	if err != nil {
		rep.PrintError(fmt.Sprintf("Failed to create detector: %v", err))
		return err
	}

	rep.PrintInfo("Detecting development tools...")
	fmt.Println()

	summary := det.GetSummary()
	fmt.Println(summary)

	return nil
}

// runReport executes the report command
func runReport(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	rep := reporter.NewReporter(verbose)

	rep.PrintHeader()

	// Parse clean level
	level, err := config.ParseCleanLevel(cleanLevel)
	if err != nil {
		rep.PrintError(err.Error())
		return err
	}

	// Create config
	cfg := config.NewDefaultConfig()
	cfg.Verbose = verbose
	cfg.CleanLevel = level

	// Initialize cleaners
	cleaners, err := initAllCleaners()
	if err != nil {
		rep.PrintError(fmt.Sprintf("Failed to initialize cleaners: %v", err))
		return err
	}

	// Filter by domain if specified
	if len(domains) > 0 {
		cleaners = filterCleanersByDomain(cleaners, domains)
	}

	// Scan
	rep.PrintInfo("Scanning system (this may take a while)...")
	startTime := time.Now()

	targetsByDomain := make(map[string][]cleaner.CleanTarget)

	for _, c := range cleaners {
		isDetected, err := c.Detect(ctx)
		if err != nil || !isDetected {
			continue
		}

		targets, err := c.Scan(ctx, cfg)
		if err != nil {
			if verbose {
				rep.PrintWarning(fmt.Sprintf("Scan error for %s: %v", c.Name(), err))
			}
			continue
		}

		if len(targets) > 0 {
			targetsByDomain[c.Name()] = targets
		}
	}

	scanDuration := time.Since(startTime)

	// Print report
	rep.PrintEstimation(targetsByDomain)
	rep.PrintSafetyLegend()

	// Print all targets if verbose
	if verbose {
		for domain, targets := range targetsByDomain {
			fmt.Printf("\n=== %s ===\n", domain)
			rep.PrintTargetDetails(targets)
		}
	}

	rep.PrintInfo(fmt.Sprintf("Scan completed in %v", scanDuration.Round(time.Second)))

	return nil
}

// runSmart executes the smart command
func runSmart(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	rep := reporter.NewReporter(verbose)

	rep.PrintHeader()
	rep.PrintInfo("Running smart cleanup with conservative settings...")

	// Use conservative level for smart mode
	cfg := config.NewDefaultConfig()
	cfg.DryRun = dryRun
	cfg.Verbose = verbose
	cfg.CleanLevel = config.Conservative
	cfg.Interactive = false // Smart mode is automatic

	// Detect tools first
	det, err := detector.NewDetector()
	if err != nil {
		rep.PrintError(fmt.Sprintf("Failed to create detector: %v", err))
		return err
	}

	detection := det.DetectAll()

	// Initialize only relevant cleaners
	cleaners := []cleaner.Cleaner{}

	if len(detection.Frontend) > 0 {
		if c, err := cleaner.NewFrontendCleaner(); err == nil {
			cleaners = append(cleaners, c)
		}
	}
	if len(detection.Backend) > 0 {
		if c, err := cleaner.NewBackendCleaner(); err == nil {
			cleaners = append(cleaners, c)
		}
	}
	if len(detection.Mobile) > 0 {
		if c, err := cleaner.NewMobileCleaner(); err == nil {
			cleaners = append(cleaners, c)
		}
	}
	if len(detection.DevOps) > 0 {
		if c, err := cleaner.NewDevOpsCleaner(); err == nil {
			cleaners = append(cleaners, c)
		}
	}
	if len(detection.DataML) > 0 {
		if c, err := cleaner.NewDataMLCleaner(); err == nil {
			cleaners = append(cleaners, c)
		}
	}

	// Always include system cleaner
	cleaners = append(cleaners,
		cleaner.NewTrashCleaner(),
		cleaner.NewCacheCleaner(),
		cleaner.NewLogCleaner(),
		cleaner.NewTempFilesCleaner(),
	)

	// Scan and clean
	targetsByDomain := make(map[string][]cleaner.CleanTarget)

	for _, c := range cleaners {
		isDetected, err := c.Detect(ctx)
		if err != nil || !isDetected {
			continue
		}

		targets, err := c.Scan(ctx, cfg)
		if err != nil {
			continue
		}

		if len(targets) > 0 {
			targetsByDomain[c.Name()] = targets
		}
	}

	// Print estimation
	rep.PrintEstimation(targetsByDomain)

	if dryRun {
		rep.PrintInfo("DRY RUN - No files will be deleted")
		return nil
	}

	// Execute cleanup
	allResults := []cleaner.CleanResult{}

	for domain, targets := range targetsByDomain {
		for _, c := range cleaners {
			if c.Name() == domain {
				results, err := c.Clean(ctx, targets, dryRun)
				if err == nil {
					allResults = append(allResults, results...)
				}
				break
			}
		}
	}

	// Print results
	rep.PrintCleanResults(allResults, dryRun)

	return nil
}

// Helper functions

func initAllCleaners() ([]cleaner.Cleaner, error) {
	cleaners := []cleaner.Cleaner{
		cleaner.NewTrashCleaner(),
		cleaner.NewCacheCleaner(),
		cleaner.NewLogCleaner(),
		cleaner.NewTempFilesCleaner(),
		cleaner.NewDNSCacheCleaner(),
		cleaner.NewHomebrewCleaner(),
		cleaner.NewXcodeCleaner(),
		cleaner.NewLaunchpadCleaner(),
		cleaner.NewIOSBackupCleaner(),
	}

	// Add cleaners that can return errors
	if c, err := cleaner.NewFrontendCleaner(); err == nil {
		cleaners = append(cleaners, c)
	}
	if c, err := cleaner.NewBackendCleaner(); err == nil {
		cleaners = append(cleaners, c)
	}
	if c, err := cleaner.NewMobileCleaner(); err == nil {
		cleaners = append(cleaners, c)
	}
	if c, err := cleaner.NewDevOpsCleaner(); err == nil {
		cleaners = append(cleaners, c)
	}
	if c, err := cleaner.NewDataMLCleaner(); err == nil {
		cleaners = append(cleaners, c)
	}

	return cleaners, nil
}

func filterCleanersByDomain(cleaners []cleaner.Cleaner, domains []string) []cleaner.Cleaner {
	if len(domains) == 0 {
		return cleaners
	}

	filtered := []cleaner.Cleaner{}

	// Normalize domains to lowercase
	domainMap := make(map[string]bool)
	for _, d := range domains {
		domainMap[toLower(d)] = true
	}

	for _, c := range cleaners {
		name := toLower(c.Name())

		// Check if cleaner name matches any requested domain
		for domain := range domainMap {
			if matchesDomain(name, domain) {
				filtered = append(filtered, c)
				break
			}
		}
	}

	return filtered
}

func toLower(s string) string {
	// Simple ASCII lowercase conversion
	result := ""
	for _, r := range s {
		if r >= 'A' && r <= 'Z' {
			result += string(r + 32)
		} else {
			result += string(r)
		}
	}
	return result
}

func matchesDomain(cleanerName, requestedDomain string) bool {
	// Map cleaner names to domain keywords
	domainMapping := map[string][]string{
		"frontend":    {"frontend"},
		"backend":     {"backend"},
		"mobile":      {"mobile"},
		"devops":      {"devops"},
		"dataml":      {"data/ml", "dataml"},
		"data/ml":     {"data/ml", "dataml"},
		"system":      {"trash", "cache", "log", "temp", "dns", "homebrew", "xcode", "launchpad", "ios"},
	}

	// Check direct match
	if cleanerName == requestedDomain {
		return true
	}

	// Check if requested domain has mapping
	if keywords, ok := domainMapping[requestedDomain]; ok {
		for _, keyword := range keywords {
			if cleanerName == keyword || containsString(cleanerName, keyword) {
				return true
			}
		}
	}

	// Check if cleaner name contains the requested domain
	return containsString(cleanerName, requestedDomain)
}

func containsString(s, substr string) bool {
	if len(substr) == 0 {
		return false
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
