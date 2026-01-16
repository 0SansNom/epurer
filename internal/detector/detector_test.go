package detector

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// =============================================================================
// NewDetector Tests
// =============================================================================

func TestNewDetector(t *testing.T) {
	detector, err := NewDetector()
	if err != nil {
		t.Fatalf("NewDetector() returned error: %v", err)
	}
	if detector == nil {
		t.Fatal("NewDetector() returned nil")
	}
	if detector.homePath == "" {
		t.Error("detector.homePath is empty")
	}
}

// =============================================================================
// DetectionResult Tests
// =============================================================================

func TestDetectionResult_Structure(t *testing.T) {
	result := DetectionResult{
		Frontend: []string{"node", "npm"},
		Backend:  []string{"python", "go"},
		Mobile:   []string{"xcode"},
		DevOps:   []string{"docker"},
		DataML:   []string{"jupyter"},
	}

	if len(result.Frontend) != 2 {
		t.Errorf("Expected 2 frontend tools, got %d", len(result.Frontend))
	}
	if len(result.Backend) != 2 {
		t.Errorf("Expected 2 backend tools, got %d", len(result.Backend))
	}
	if len(result.Mobile) != 1 {
		t.Errorf("Expected 1 mobile tool, got %d", len(result.Mobile))
	}
	if len(result.DevOps) != 1 {
		t.Errorf("Expected 1 devops tool, got %d", len(result.DevOps))
	}
	if len(result.DataML) != 1 {
		t.Errorf("Expected 1 dataml tool, got %d", len(result.DataML))
	}
}

// =============================================================================
// DetectAll Tests
// =============================================================================

func TestDetectAll(t *testing.T) {
	detector, err := NewDetector()
	if err != nil {
		t.Fatalf("NewDetector() returned error: %v", err)
	}

	result := detector.DetectAll()

	// Result should be initialized (not nil slices)
	if result.Frontend == nil {
		t.Error("result.Frontend is nil")
	}
	if result.Backend == nil {
		t.Error("result.Backend is nil")
	}
	if result.Mobile == nil {
		t.Error("result.Mobile is nil")
	}
	if result.DevOps == nil {
		t.Error("result.DevOps is nil")
	}
	if result.DataML == nil {
		t.Error("result.DataML is nil")
	}

	// Log what was detected (informational)
	t.Logf("Detected Frontend: %v", result.Frontend)
	t.Logf("Detected Backend: %v", result.Backend)
	t.Logf("Detected Mobile: %v", result.Mobile)
	t.Logf("Detected DevOps: %v", result.DevOps)
	t.Logf("Detected DataML: %v", result.DataML)
}

// =============================================================================
// HasXxx Tests
// =============================================================================

func TestHasFrontend(t *testing.T) {
	detector, _ := NewDetector()
	result := detector.HasFrontend()
	// Just verify no panic, result depends on system
	t.Logf("HasFrontend: %v", result)
}

func TestHasBackend(t *testing.T) {
	detector, _ := NewDetector()
	result := detector.HasBackend()
	t.Logf("HasBackend: %v", result)
}

func TestHasMobile(t *testing.T) {
	detector, _ := NewDetector()
	result := detector.HasMobile()
	t.Logf("HasMobile: %v", result)
}

func TestHasDevOps(t *testing.T) {
	detector, _ := NewDetector()
	result := detector.HasDevOps()
	t.Logf("HasDevOps: %v", result)
}

func TestHasDataML(t *testing.T) {
	detector, _ := NewDetector()
	result := detector.HasDataML()
	t.Logf("HasDataML: %v", result)
}

// =============================================================================
// GetSummary Tests
// =============================================================================

func TestGetSummary(t *testing.T) {
	detector, _ := NewDetector()
	summary := detector.GetSummary()

	// Summary should not be empty (at least "No development tools detected")
	if summary == "" {
		t.Error("GetSummary() returned empty string")
	}

	t.Logf("Summary:\n%s", summary)
}

func TestGetSummary_Format(t *testing.T) {
	detector, _ := NewDetector()
	summary := detector.GetSummary()

	// Check that summary ends with newline
	if !strings.HasSuffix(summary, "\n") {
		t.Error("Summary does not end with newline")
	}

	// If tools are detected, check format
	if !strings.Contains(summary, "No development tools detected") {
		// Should contain category labels
		hasCategory := strings.Contains(summary, "Frontend:") ||
			strings.Contains(summary, "Backend:") ||
			strings.Contains(summary, "Mobile:") ||
			strings.Contains(summary, "DevOps:") ||
			strings.Contains(summary, "Data/ML:")
		if !hasCategory {
			t.Error("Summary has no category labels")
		}
	}
}

// =============================================================================
// contains Helper Tests
// =============================================================================

func TestContains(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		value    string
		expected bool
	}{
		{"Found first", []string{"a", "b", "c"}, "a", true},
		{"Found middle", []string{"a", "b", "c"}, "b", true},
		{"Found last", []string{"a", "b", "c"}, "c", true},
		{"Not found", []string{"a", "b", "c"}, "d", false},
		{"Empty slice", []string{}, "a", false},
		{"Empty value found", []string{"", "a"}, "", true},
		{"Case sensitive", []string{"Node", "NPM"}, "node", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contains(tt.slice, tt.value)
			if result != tt.expected {
				t.Errorf("contains(%v, %q) = %v, want %v", tt.slice, tt.value, result, tt.expected)
			}
		})
	}
}

// =============================================================================
// hasPythonPackage Tests
// =============================================================================

func TestHasPythonPackage_NonExistent(t *testing.T) {
	detector, _ := NewDetector()

	// Test with a package that definitely doesn't exist
	result := detector.hasPythonPackage("nonexistent_package_xyz_12345")
	if result {
		t.Error("hasPythonPackage returned true for non-existent package")
	}
}

func TestHasPythonPackage_WithMockDir(t *testing.T) {
	// Create a mock Python package structure
	tmpDir, err := os.MkdirTemp("", "detector-python-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a detector with custom home path
	detector := &StackDetector{homePath: tmpDir}

	// Create mock site-packages directory
	sitePackages := filepath.Join(tmpDir, "Library", "Python", "3.9", "site-packages", "testpkg")
	if err := os.MkdirAll(sitePackages, 0755); err != nil {
		t.Fatalf("Failed to create mock site-packages: %v", err)
	}

	// Test detection
	result := detector.hasPythonPackage("testpkg")
	if !result {
		t.Error("hasPythonPackage did not detect mock package")
	}

	// Test non-existent in mock
	result = detector.hasPythonPackage("otherpkg")
	if result {
		t.Error("hasPythonPackage detected non-existent package")
	}
}

// =============================================================================
// Integration Tests
// =============================================================================

func TestDetector_Integration(t *testing.T) {
	detector, err := NewDetector()
	if err != nil {
		t.Fatalf("NewDetector() returned error: %v", err)
	}

	// Run DetectAll
	result := detector.DetectAll()

	// Check HasXxx matches DetectAll results
	if detector.HasFrontend() != (len(result.Frontend) > 0) {
		t.Error("HasFrontend() doesn't match DetectAll().Frontend")
	}
	if detector.HasBackend() != (len(result.Backend) > 0) {
		t.Error("HasBackend() doesn't match DetectAll().Backend")
	}
	if detector.HasMobile() != (len(result.Mobile) > 0) {
		t.Error("HasMobile() doesn't match DetectAll().Mobile")
	}
	if detector.HasDevOps() != (len(result.DevOps) > 0) {
		t.Error("HasDevOps() doesn't match DetectAll().DevOps")
	}
	if detector.HasDataML() != (len(result.DataML) > 0) {
		t.Error("HasDataML() doesn't match DetectAll().DataML")
	}
}

func TestDetector_SummaryMatchesDetection(t *testing.T) {
	detector, _ := NewDetector()
	result := detector.DetectAll()
	summary := detector.GetSummary()

	// If frontend tools detected, summary should contain "Frontend:"
	if len(result.Frontend) > 0 {
		if !strings.Contains(summary, "Frontend:") {
			t.Error("Summary missing Frontend section")
		}
		// Check that detected tools appear in summary
		for _, tool := range result.Frontend {
			if !strings.Contains(summary, tool) {
				t.Errorf("Summary missing frontend tool: %s", tool)
			}
		}
	}

	// Same for other categories
	if len(result.Backend) > 0 && !strings.Contains(summary, "Backend:") {
		t.Error("Summary missing Backend section")
	}
	if len(result.Mobile) > 0 && !strings.Contains(summary, "Mobile:") {
		t.Error("Summary missing Mobile section")
	}
	if len(result.DevOps) > 0 && !strings.Contains(summary, "DevOps:") {
		t.Error("Summary missing DevOps section")
	}
	if len(result.DataML) > 0 && !strings.Contains(summary, "Data/ML:") {
		t.Error("Summary missing Data/ML section")
	}
}

// =============================================================================
// Edge Cases
// =============================================================================

func TestDetector_NoTools(t *testing.T) {
	// Create detector with non-existent home (edge case)
	detector := &StackDetector{homePath: "/nonexistent/path/12345"}

	// DetectAll should still work (just won't find anything path-based)
	result := detector.DetectAll()

	// Should not panic
	if result.Frontend == nil || result.Backend == nil {
		t.Error("DetectAll returned nil slices")
	}
}

func TestDetector_EmptyHomePath(t *testing.T) {
	detector := &StackDetector{homePath: ""}

	// Should handle empty home path gracefully
	result := detector.DetectAll()

	// Should not panic
	_ = result
}
