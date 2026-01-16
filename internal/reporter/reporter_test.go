package reporter

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/0SansNom/epurer/internal/cleaner"
	"github.com/0SansNom/epurer/internal/config"
)

// captureOutput captures stdout during function execution
func captureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

// =============================================================================
// NewReporter Tests
// =============================================================================

func TestNewReporter(t *testing.T) {
	r := NewReporter(false)
	if r == nil {
		t.Fatal("NewReporter(false) returned nil")
	}
	if r.verbose {
		t.Error("Expected verbose to be false")
	}
}

func TestNewReporter_Verbose(t *testing.T) {
	r := NewReporter(true)
	if r == nil {
		t.Fatal("NewReporter(true) returned nil")
	}
	if !r.verbose {
		t.Error("Expected verbose to be true")
	}
}

// =============================================================================
// Helper Function Tests
// =============================================================================

func TestGetImpactString(t *testing.T) {
	tests := []struct {
		name     string
		size     int64
		expected string
	}{
		{"Zero bytes", 0, "Low"},
		{"100 MB", 100 * 1024 * 1024, "Low"},
		{"499 MB", 499 * 1024 * 1024, "Low"},
		{"500 MB", 500 * 1024 * 1024, "Medium"},
		{"1 GB", 1024 * 1024 * 1024, "Medium"},
		{"5 GB", 5 * 1024 * 1024 * 1024, "High"},
		{"10 GB", 10 * 1024 * 1024 * 1024, "High"},
		{"20 GB", 20 * 1024 * 1024 * 1024, "Very High"},
		{"100 GB", 100 * 1024 * 1024 * 1024, "Very High"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getImpactString(tt.size)
			if result != tt.expected {
				t.Errorf("getImpactString(%d) = %q, want %q", tt.size, result, tt.expected)
			}
		})
	}
}

func TestGetActionVerb(t *testing.T) {
	tests := []struct {
		name     string
		dryRun   bool
		expected string
	}{
		{"Dry run", true, "would be freed"},
		{"Actual run", false, "freed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getActionVerb(tt.dryRun)
			if result != tt.expected {
				t.Errorf("getActionVerb(%v) = %q, want %q", tt.dryRun, result, tt.expected)
			}
		})
	}
}

// =============================================================================
// PrintHeader Tests
// =============================================================================

func TestPrintHeader(t *testing.T) {
	r := NewReporter(false)

	output := captureOutput(func() {
		r.PrintHeader()
	})

	// Check that header contains expected content
	if !strings.Contains(output, "Épurer") {
		t.Error("Header should contain 'Épurer'")
	}
	if !strings.Contains(output, "v1.1") {
		t.Error("Header should contain version")
	}
}

// =============================================================================
// PrintDetection Tests
// =============================================================================

func TestPrintDetection(t *testing.T) {
	r := NewReporter(false)

	detected := map[string]bool{
		"frontend": true,
		"backend":  false,
		"mobile":   true,
		"devops":   false,
		"dataml":   false,
		"system":   true,
	}

	output := captureOutput(func() {
		r.PrintDetection(detected)
	})

	// Check that detected items are shown
	if !strings.Contains(output, "Frontend") {
		t.Error("Output should contain detected Frontend")
	}
	if !strings.Contains(output, "Mobile") {
		t.Error("Output should contain detected Mobile")
	}
	if !strings.Contains(output, "System") {
		t.Error("Output should contain detected System")
	}
}

func TestPrintDetection_Verbose(t *testing.T) {
	r := NewReporter(true)

	detected := map[string]bool{
		"frontend": true,
		"backend":  false,
	}

	output := captureOutput(func() {
		r.PrintDetection(detected)
	})

	// In verbose mode, non-detected items should also be shown
	if !strings.Contains(output, "Backend") {
		t.Error("Verbose output should contain non-detected Backend")
	}
}

func TestPrintDetection_Empty(t *testing.T) {
	r := NewReporter(false)

	output := captureOutput(func() {
		r.PrintDetection(map[string]bool{})
	})

	// Should not panic, should contain detection message
	if !strings.Contains(output, "Detecting") {
		t.Error("Output should contain detection message")
	}
}

// =============================================================================
// PrintEstimation Tests
// =============================================================================

func TestPrintEstimation(t *testing.T) {
	r := NewReporter(false)

	targetsByDomain := map[string][]cleaner.CleanTarget{
		"Frontend": {
			{Path: "/path/1", Description: "npm cache", SizeBytes: 1024 * 1024, Safety: config.Safe},
			{Path: "/path/2", Description: "node_modules", SizeBytes: 500 * 1024 * 1024, Safety: config.Moderate},
		},
		"Backend": {
			{Path: "/path/3", Description: "pip cache", SizeBytes: 100 * 1024 * 1024, Safety: config.Safe},
		},
	}

	output := captureOutput(func() {
		r.PrintEstimation(targetsByDomain)
	})

	// Check that output contains expected content
	if !strings.Contains(output, "Frontend") {
		t.Error("Output should contain Frontend domain")
	}
	if !strings.Contains(output, "Backend") {
		t.Error("Output should contain Backend domain")
	}
	if !strings.Contains(output, "Total") {
		t.Error("Output should contain Total row")
	}
}

func TestPrintEstimation_Empty(t *testing.T) {
	r := NewReporter(false)

	output := captureOutput(func() {
		r.PrintEstimation(map[string][]cleaner.CleanTarget{})
	})

	// Should not panic
	if !strings.Contains(output, "Estimation") {
		t.Error("Output should contain estimation header")
	}
}

func TestPrintEstimation_AllSafetyLevels(t *testing.T) {
	r := NewReporter(false)

	targetsByDomain := map[string][]cleaner.CleanTarget{
		"System": {
			{Path: "/path/1", Description: "cache", SizeBytes: 1024, Safety: config.Safe},
			{Path: "/path/2", Description: "temp", SizeBytes: 1024, Safety: config.Moderate},
			{Path: "/path/3", Description: "backup", SizeBytes: 1024, Safety: config.Dangerous},
		},
	}

	output := captureOutput(func() {
		r.PrintEstimation(targetsByDomain)
	})

	// Check that safety levels are represented
	if !strings.Contains(output, "Safe") {
		t.Error("Output should contain Safe safety level")
	}
}

// =============================================================================
// PrintTargetDetails Tests
// =============================================================================

func TestPrintTargetDetails_NotVerbose(t *testing.T) {
	r := NewReporter(false)

	targets := []cleaner.CleanTarget{
		{Path: "/path/1", Description: "Test", SizeBytes: 1024, Safety: config.Safe},
	}

	output := captureOutput(func() {
		r.PrintTargetDetails(targets)
	})

	// Should not print anything in non-verbose mode
	if strings.Contains(output, "Test") {
		t.Error("Non-verbose mode should not print target details")
	}
}

func TestPrintTargetDetails_Verbose(t *testing.T) {
	r := NewReporter(true)

	targets := []cleaner.CleanTarget{
		{Path: "/path/to/cache", Description: "npm cache", SizeBytes: 1024 * 1024, Safety: config.Safe},
		{Path: "/path/to/modules", Description: "node_modules", SizeBytes: 500 * 1024 * 1024, Safety: config.Moderate},
	}

	output := captureOutput(func() {
		r.PrintTargetDetails(targets)
	})

	// Should print details in verbose mode
	if !strings.Contains(output, "npm cache") {
		t.Error("Verbose output should contain target description")
	}
	if !strings.Contains(output, "node_modules") {
		t.Error("Verbose output should contain all targets")
	}
}

func TestPrintTargetDetails_Empty(t *testing.T) {
	r := NewReporter(true)

	output := captureOutput(func() {
		r.PrintTargetDetails([]cleaner.CleanTarget{})
	})

	// Should not panic with empty slice
	_ = output
}

// =============================================================================
// PrintProgress Tests
// =============================================================================

func TestPrintProgress(t *testing.T) {
	r := NewReporter(false)

	output := captureOutput(func() {
		r.PrintProgress(1, 10, "Processing")
	})

	// Should contain progress info
	if !strings.Contains(output, "Processing") {
		t.Error("Output should contain description")
	}
	if !strings.Contains(output, "1") && !strings.Contains(output, "10") {
		t.Error("Output should contain progress numbers")
	}
}

func TestPrintProgress_Complete(t *testing.T) {
	r := NewReporter(false)

	output := captureOutput(func() {
		r.PrintProgress(10, 10, "Done")
	})

	// Should contain progress info
	if !strings.Contains(output, "Done") {
		t.Error("Output should contain description")
	}
}

// =============================================================================
// PrintCleanResults Tests
// =============================================================================

func TestPrintCleanResults_DryRun(t *testing.T) {
	r := NewReporter(false)

	results := []cleaner.CleanResult{
		{Target: cleaner.CleanTarget{Path: "/path/1"}, Success: true, BytesFreed: 1024},
		{Target: cleaner.CleanTarget{Path: "/path/2"}, Success: true, BytesFreed: 2048},
	}

	output := captureOutput(func() {
		r.PrintCleanResults(results, true)
	})

	// Should indicate dry run
	if !strings.Contains(output, "Dry Run") {
		t.Error("Output should indicate dry run")
	}
	if !strings.Contains(output, "would be freed") {
		t.Error("Output should use dry run language")
	}
}

func TestPrintCleanResults_Actual(t *testing.T) {
	r := NewReporter(false)

	results := []cleaner.CleanResult{
		{Target: cleaner.CleanTarget{Path: "/path/1"}, Success: true, BytesFreed: 1024 * 1024},
		{Target: cleaner.CleanTarget{Path: "/path/2"}, Success: true, BytesFreed: 2048 * 1024},
	}

	output := captureOutput(func() {
		r.PrintCleanResults(results, false)
	})

	// Should indicate completion
	if !strings.Contains(output, "Complete") {
		t.Error("Output should indicate completion")
	}
	if !strings.Contains(output, "freed") {
		t.Error("Output should mention freed space")
	}
}

func TestPrintCleanResults_WithFailures(t *testing.T) {
	r := NewReporter(false)

	results := []cleaner.CleanResult{
		{Target: cleaner.CleanTarget{Path: "/path/1"}, Success: true, BytesFreed: 1024},
		{Target: cleaner.CleanTarget{Path: "/path/2"}, Success: false, Error: errors.New("permission denied")},
	}

	output := captureOutput(func() {
		r.PrintCleanResults(results, false)
	})

	// Should show failure count
	if !strings.Contains(output, "Failure") {
		t.Error("Output should mention failures")
	}
}

func TestPrintCleanResults_WithFailures_Verbose(t *testing.T) {
	r := NewReporter(true)

	results := []cleaner.CleanResult{
		{Target: cleaner.CleanTarget{Path: "/path/success"}, Success: true, BytesFreed: 1024},
		{Target: cleaner.CleanTarget{Path: "/path/failed"}, Success: false, Error: errors.New("permission denied")},
	}

	output := captureOutput(func() {
		r.PrintCleanResults(results, false)
	})

	// Verbose mode should show failure details
	if !strings.Contains(output, "Failed") {
		t.Error("Verbose output should show failure section")
	}
	if !strings.Contains(output, "permission denied") {
		t.Error("Verbose output should show error message")
	}
}

func TestPrintCleanResults_Empty(t *testing.T) {
	r := NewReporter(false)

	output := captureOutput(func() {
		r.PrintCleanResults([]cleaner.CleanResult{}, false)
	})

	// Should not panic with empty results
	if !strings.Contains(output, "Complete") {
		t.Error("Output should indicate completion even with no results")
	}
}

// =============================================================================
// Print Message Tests
// =============================================================================

func TestPrintWarning(t *testing.T) {
	r := NewReporter(false)

	output := captureOutput(func() {
		r.PrintWarning("Test warning message")
	})

	if !strings.Contains(output, "Test warning message") {
		t.Error("Output should contain warning message")
	}
}

func TestPrintError(t *testing.T) {
	r := NewReporter(false)

	output := captureOutput(func() {
		r.PrintError("Test error message")
	})

	if !strings.Contains(output, "Test error message") {
		t.Error("Output should contain error message")
	}
}

func TestPrintSuccess(t *testing.T) {
	r := NewReporter(false)

	output := captureOutput(func() {
		r.PrintSuccess("Test success message")
	})

	if !strings.Contains(output, "Test success message") {
		t.Error("Output should contain success message")
	}
}

func TestPrintInfo(t *testing.T) {
	r := NewReporter(false)

	output := captureOutput(func() {
		r.PrintInfo("Test info message")
	})

	if !strings.Contains(output, "Test info message") {
		t.Error("Output should contain info message")
	}
}

// =============================================================================
// PrintSafetyLegend Tests
// =============================================================================

func TestPrintSafetyLegend(t *testing.T) {
	r := NewReporter(false)

	output := captureOutput(func() {
		r.PrintSafetyLegend()
	})

	// Should contain all safety levels
	if !strings.Contains(output, "Safe") {
		t.Error("Output should contain Safe level")
	}
	if !strings.Contains(output, "Mod") {
		t.Error("Output should contain Moderate level")
	}
	if !strings.Contains(output, "Risk") {
		t.Error("Output should contain Risk level")
	}
	if !strings.Contains(output, "Safety Levels") {
		t.Error("Output should contain legend title")
	}
}

// =============================================================================
// Style Tests (verify styles are initialized)
// =============================================================================

func TestStyles_Initialized(t *testing.T) {
	// Verify that all style variables are initialized
	styles := []struct {
		name  string
		style interface{}
	}{
		{"titleStyle", titleStyle},
		{"subtitleStyle", subtitleStyle},
		{"successStyle", successStyle},
		{"warningStyle", warningStyle},
		{"errorStyle", errorStyle},
		{"infoStyle", infoStyle},
		{"mutedStyle", mutedStyle},
		{"headerBox", headerBox},
		{"tableHeaderStyle", tableHeaderStyle},
		{"tableCellStyle", tableCellStyle},
		{"tableFooterStyle", tableFooterStyle},
	}

	for _, s := range styles {
		t.Run(s.name, func(t *testing.T) {
			// Just verify they can be used without panic
			_ = s.style
		})
	}
}

// =============================================================================
// Integration Tests
// =============================================================================

func TestReporter_FullWorkflow(t *testing.T) {
	r := NewReporter(true)

	// Simulate a full workflow
	output := captureOutput(func() {
		// 1. Print header
		r.PrintHeader()

		// 2. Print detection
		r.PrintDetection(map[string]bool{
			"frontend": true,
			"backend":  true,
		})

		// 3. Print estimation
		r.PrintEstimation(map[string][]cleaner.CleanTarget{
			"Frontend": {
				{Path: "/test", Description: "Test", SizeBytes: 1024, Safety: config.Safe},
			},
		})

		// 4. Print safety legend
		r.PrintSafetyLegend()

		// 5. Print results
		r.PrintCleanResults([]cleaner.CleanResult{
			{Target: cleaner.CleanTarget{}, Success: true, BytesFreed: 1024},
		}, true)
	})

	// Verify all sections are present
	sections := []string{"Épurer", "Detecting", "Estimation", "Safety Levels", "Dry Run"}
	for _, section := range sections {
		if !strings.Contains(output, section) {
			t.Errorf("Full workflow output missing section: %s", section)
		}
	}
}

// =============================================================================
// Edge Cases
// =============================================================================

func TestReporter_LargeNumbers(t *testing.T) {
	r := NewReporter(false)

	// Test with very large sizes
	targetsByDomain := map[string][]cleaner.CleanTarget{
		"System": {
			{Path: "/path", Description: "Large", SizeBytes: 100 * 1024 * 1024 * 1024, Safety: config.Safe}, // 100 GB
		},
	}

	output := captureOutput(func() {
		r.PrintEstimation(targetsByDomain)
	})

	// Should handle large numbers without panic
	if !strings.Contains(output, "GB") {
		t.Error("Output should format large sizes as GB")
	}
}

func TestReporter_SpecialCharacters(t *testing.T) {
	r := NewReporter(false)

	// Test with special characters in messages
	output := captureOutput(func() {
		r.PrintWarning("Path with spaces: /Users/test user/Documents")
		r.PrintError("Error: file not found (╯°□°)╯︵ ┻━┻")
	})

	// Should handle special characters without panic
	if !strings.Contains(output, "spaces") {
		t.Error("Output should contain message with spaces")
	}
}
