package cleaner

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/0SansNom/epurer/internal/config"
)

// testTimeout is the default timeout for tests
const testTimeout = 10 * time.Second

// setupTestDir creates a temporary directory for testing
func setupTestDir(t *testing.T) string {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "cleaner-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	return tmpDir
}

// createTestFile creates a file with content in the given directory
func createTestFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("Failed to create parent dirs: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	return path
}

// createTestDir creates a directory with optional files
func createTestDir(t *testing.T, parent, name string, files map[string]string) string {
	t.Helper()
	dir := filepath.Join(parent, name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("Failed to create test dir: %v", err)
	}
	for fileName, content := range files {
		createTestFile(t, dir, fileName, content)
	}
	return dir
}

// =============================================================================
// FrontendCleaner Tests
// =============================================================================

func TestNewFrontendCleaner(t *testing.T) {
	cleaner, err := NewFrontendCleaner()
	if err != nil {
		t.Fatalf("Failed to create FrontendCleaner: %v", err)
	}
	if cleaner == nil {
		t.Fatal("FrontendCleaner is nil")
	}
}

func TestFrontendCleaner_Name(t *testing.T) {
	cleaner, _ := NewFrontendCleaner()
	if cleaner.Name() != "Frontend" {
		t.Errorf("Expected name 'Frontend', got '%s'", cleaner.Name())
	}
}

func TestFrontendCleaner_Domain(t *testing.T) {
	cleaner, _ := NewFrontendCleaner()
	if cleaner.Domain() != config.DomainFrontend {
		t.Errorf("Expected domain DomainFrontend, got %v", cleaner.Domain())
	}
}

func TestFrontendCleaner_Detect(t *testing.T) {
	cleaner, _ := NewFrontendCleaner()
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	detected, err := cleaner.Detect(ctx)
	if err != nil {
		t.Fatalf("Detect() returned error: %v", err)
	}
	// Result depends on whether node/npm/yarn/pnpm is installed
	// We just verify no error is returned
	t.Logf("Frontend tools detected: %v", detected)
}

func TestFrontendCleaner_Clean_DryRun(t *testing.T) {
	cleaner, _ := NewFrontendCleaner()
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	// Create a test target
	tmpDir := setupTestDir(t)
	defer os.RemoveAll(tmpDir)

	testFile := createTestFile(t, tmpDir, "test.txt", "test content")

	targets := []CleanTarget{
		{
			Path:        testFile,
			Description: "Test file",
			SizeBytes:   12,
			Safety:      config.Safe,
		},
	}

	// Clean in dry-run mode
	results, err := cleaner.Clean(ctx, targets, true)
	if err != nil {
		t.Fatalf("Clean() returned error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	if !results[0].Success {
		t.Error("Expected success in dry-run mode")
	}

	// Verify file still exists (dry-run should not delete)
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Error("File was deleted in dry-run mode")
	}
}

func TestFrontendCleaner_Clean_Actual(t *testing.T) {
	cleaner, _ := NewFrontendCleaner()
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	// Create a test target
	tmpDir := setupTestDir(t)
	defer os.RemoveAll(tmpDir)

	testDir := createTestDir(t, tmpDir, "node_modules", map[string]string{
		"package/index.js": "module.exports = {}",
	})

	targets := []CleanTarget{
		{
			Path:        testDir,
			Description: "Test node_modules",
			SizeBytes:   100,
			Safety:      config.Moderate,
		},
	}

	// Clean in actual mode
	results, err := cleaner.Clean(ctx, targets, false)
	if err != nil {
		t.Fatalf("Clean() returned error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	if !results[0].Success {
		t.Errorf("Expected success, got error: %v", results[0].Error)
	}

	// Verify directory was deleted
	if _, err := os.Stat(testDir); !os.IsNotExist(err) {
		t.Error("Directory was not deleted")
	}
}

func TestFrontendCleaner_Clean_ContextCancellation(t *testing.T) {
	cleaner, _ := NewFrontendCleaner()
	ctx, cancel := context.WithCancel(context.Background())

	tmpDir := setupTestDir(t)
	defer os.RemoveAll(tmpDir)

	// Create multiple targets
	targets := make([]CleanTarget, 5)
	for i := 0; i < 5; i++ {
		dir := createTestDir(t, tmpDir, filepath.Join("dir", string(rune('a'+i))), nil)
		targets[i] = CleanTarget{
			Path:        dir,
			Description: "Test dir",
			SizeBytes:   100,
			Safety:      config.Safe,
		}
	}

	// Cancel immediately
	cancel()

	results, err := cleaner.Clean(ctx, targets, true)
	if err != context.Canceled {
		t.Logf("Clean returned: results=%d, err=%v", len(results), err)
	}
}

// =============================================================================
// BackendCleaner Tests
// =============================================================================

func TestNewBackendCleaner(t *testing.T) {
	cleaner, err := NewBackendCleaner()
	if err != nil {
		t.Fatalf("Failed to create BackendCleaner: %v", err)
	}
	if cleaner == nil {
		t.Fatal("BackendCleaner is nil")
	}
}

func TestBackendCleaner_Name(t *testing.T) {
	cleaner, _ := NewBackendCleaner()
	if cleaner.Name() != "Backend" {
		t.Errorf("Expected name 'Backend', got '%s'", cleaner.Name())
	}
}

func TestBackendCleaner_Detect(t *testing.T) {
	cleaner, _ := NewBackendCleaner()
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	detected, err := cleaner.Detect(ctx)
	if err != nil {
		t.Fatalf("Detect() returned error: %v", err)
	}
	t.Logf("Backend tools detected: %v", detected)
}

func TestBackendCleaner_Clean_DryRun(t *testing.T) {
	cleaner, _ := NewBackendCleaner()
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	tmpDir := setupTestDir(t)
	defer os.RemoveAll(tmpDir)

	// Create a __pycache__ directory
	pycacheDir := createTestDir(t, tmpDir, "__pycache__", map[string]string{
		"module.cpython-39.pyc": "fake bytecode",
	})

	targets := []CleanTarget{
		{
			Path:        pycacheDir,
			Description: "Python bytecode cache",
			SizeBytes:   100,
			Safety:      config.Safe,
		},
	}

	results, err := cleaner.Clean(ctx, targets, true)
	if err != nil {
		t.Fatalf("Clean() returned error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	if !results[0].Success {
		t.Error("Expected success in dry-run mode")
	}

	// Verify directory still exists
	if _, err := os.Stat(pycacheDir); os.IsNotExist(err) {
		t.Error("Directory was deleted in dry-run mode")
	}
}

func TestBackendCleaner_Clean_Actual(t *testing.T) {
	cleaner, _ := NewBackendCleaner()
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	tmpDir := setupTestDir(t)
	defer os.RemoveAll(tmpDir)

	pycacheDir := createTestDir(t, tmpDir, "__pycache__", map[string]string{
		"module.cpython-39.pyc": "fake bytecode",
	})

	targets := []CleanTarget{
		{
			Path:        pycacheDir,
			Description: "Python bytecode cache",
			SizeBytes:   100,
			Safety:      config.Safe,
		},
	}

	results, err := cleaner.Clean(ctx, targets, false)
	if err != nil {
		t.Fatalf("Clean() returned error: %v", err)
	}

	if !results[0].Success {
		t.Errorf("Expected success, got error: %v", results[0].Error)
	}

	// Verify directory was deleted
	if _, err := os.Stat(pycacheDir); !os.IsNotExist(err) {
		t.Error("Directory was not deleted")
	}
}

// =============================================================================
// MobileCleaner Tests
// =============================================================================

func TestNewMobileCleaner(t *testing.T) {
	cleaner, err := NewMobileCleaner()
	if err != nil {
		t.Fatalf("Failed to create MobileCleaner: %v", err)
	}
	if cleaner == nil {
		t.Fatal("MobileCleaner is nil")
	}
}

func TestMobileCleaner_Name(t *testing.T) {
	cleaner, _ := NewMobileCleaner()
	if cleaner.Name() != "Mobile" {
		t.Errorf("Expected name 'Mobile', got '%s'", cleaner.Name())
	}
}

func TestMobileCleaner_Detect(t *testing.T) {
	cleaner, _ := NewMobileCleaner()
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	detected, err := cleaner.Detect(ctx)
	if err != nil {
		t.Fatalf("Detect() returned error: %v", err)
	}
	t.Logf("Mobile tools detected: %v", detected)
}

func TestMobileCleaner_Clean_DryRun(t *testing.T) {
	cleaner, _ := NewMobileCleaner()
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	tmpDir := setupTestDir(t)
	defer os.RemoveAll(tmpDir)

	// Simulate a Gradle build directory
	buildDir := createTestDir(t, tmpDir, "app/build", map[string]string{
		"outputs/apk/debug/app-debug.apk": "fake apk",
	})

	targets := []CleanTarget{
		{
			Path:        buildDir,
			Description: "Android build output",
			SizeBytes:   1000,
			Safety:      config.Safe,
		},
	}

	results, err := cleaner.Clean(ctx, targets, true)
	if err != nil {
		t.Fatalf("Clean() returned error: %v", err)
	}

	if !results[0].Success {
		t.Error("Expected success in dry-run mode")
	}

	// Verify directory still exists
	if _, err := os.Stat(buildDir); os.IsNotExist(err) {
		t.Error("Directory was deleted in dry-run mode")
	}
}

func TestMobileCleaner_Clean_Actual(t *testing.T) {
	cleaner, _ := NewMobileCleaner()
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	tmpDir := setupTestDir(t)
	defer os.RemoveAll(tmpDir)

	buildDir := createTestDir(t, tmpDir, "app/build", map[string]string{
		"outputs/apk/debug/app-debug.apk": "fake apk",
	})

	targets := []CleanTarget{
		{
			Path:        buildDir,
			Description: "Android build output",
			SizeBytes:   1000,
			Safety:      config.Safe,
		},
	}

	results, err := cleaner.Clean(ctx, targets, false)
	if err != nil {
		t.Fatalf("Clean() returned error: %v", err)
	}

	if !results[0].Success {
		t.Errorf("Expected success, got error: %v", results[0].Error)
	}

	// Verify directory was deleted
	if _, err := os.Stat(buildDir); !os.IsNotExist(err) {
		t.Error("Directory was not deleted")
	}
}

// =============================================================================
// DevOpsCleaner Tests
// =============================================================================

func TestNewDevOpsCleaner(t *testing.T) {
	cleaner, err := NewDevOpsCleaner()
	if err != nil {
		t.Fatalf("Failed to create DevOpsCleaner: %v", err)
	}
	if cleaner == nil {
		t.Fatal("DevOpsCleaner is nil")
	}
}

func TestDevOpsCleaner_Name(t *testing.T) {
	cleaner, _ := NewDevOpsCleaner()
	if cleaner.Name() != "DevOps" {
		t.Errorf("Expected name 'DevOps', got '%s'", cleaner.Name())
	}
}

func TestDevOpsCleaner_Detect(t *testing.T) {
	cleaner, _ := NewDevOpsCleaner()
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	detected, err := cleaner.Detect(ctx)
	if err != nil {
		t.Fatalf("Detect() returned error: %v", err)
	}
	t.Logf("DevOps tools detected: %v", detected)
}

func TestDevOpsCleaner_Clean_DryRun(t *testing.T) {
	cleaner, _ := NewDevOpsCleaner()
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	tmpDir := setupTestDir(t)
	defer os.RemoveAll(tmpDir)

	// Simulate a .terraform directory
	terraformDir := createTestDir(t, tmpDir, ".terraform", map[string]string{
		"providers/registry.terraform.io/hashicorp/aws/5.0.0/darwin_amd64/terraform-provider-aws": "fake provider",
	})

	targets := []CleanTarget{
		{
			Path:        terraformDir,
			Description: "Terraform providers",
			SizeBytes:   5000,
			Safety:      config.Safe,
		},
	}

	results, err := cleaner.Clean(ctx, targets, true)
	if err != nil {
		t.Fatalf("Clean() returned error: %v", err)
	}

	if !results[0].Success {
		t.Error("Expected success in dry-run mode")
	}

	if _, err := os.Stat(terraformDir); os.IsNotExist(err) {
		t.Error("Directory was deleted in dry-run mode")
	}
}

func TestDevOpsCleaner_Clean_Actual(t *testing.T) {
	cleaner, _ := NewDevOpsCleaner()
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	tmpDir := setupTestDir(t)
	defer os.RemoveAll(tmpDir)

	terraformDir := createTestDir(t, tmpDir, ".terraform", map[string]string{
		"providers/test": "fake provider",
	})

	targets := []CleanTarget{
		{
			Path:        terraformDir,
			Description: "Terraform providers",
			SizeBytes:   5000,
			Safety:      config.Safe,
		},
	}

	results, err := cleaner.Clean(ctx, targets, false)
	if err != nil {
		t.Fatalf("Clean() returned error: %v", err)
	}

	if !results[0].Success {
		t.Errorf("Expected success, got error: %v", results[0].Error)
	}

	if _, err := os.Stat(terraformDir); !os.IsNotExist(err) {
		t.Error("Directory was not deleted")
	}
}

// =============================================================================
// DataMLCleaner Tests
// =============================================================================

func TestNewDataMLCleaner(t *testing.T) {
	cleaner, err := NewDataMLCleaner()
	if err != nil {
		t.Fatalf("Failed to create DataMLCleaner: %v", err)
	}
	if cleaner == nil {
		t.Fatal("DataMLCleaner is nil")
	}
}

func TestDataMLCleaner_Name(t *testing.T) {
	cleaner, _ := NewDataMLCleaner()
	if cleaner.Name() != "Data/ML" {
		t.Errorf("Expected name 'Data/ML', got '%s'", cleaner.Name())
	}
}

func TestDataMLCleaner_Detect(t *testing.T) {
	cleaner, _ := NewDataMLCleaner()
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	detected, err := cleaner.Detect(ctx)
	if err != nil {
		t.Fatalf("Detect() returned error: %v", err)
	}
	t.Logf("Data/ML tools detected: %v", detected)
}

func TestDataMLCleaner_Clean_DryRun(t *testing.T) {
	cleaner, _ := NewDataMLCleaner()
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	tmpDir := setupTestDir(t)
	defer os.RemoveAll(tmpDir)

	// Simulate a Jupyter checkpoint directory
	checkpointDir := createTestDir(t, tmpDir, ".ipynb_checkpoints", map[string]string{
		"notebook-checkpoint.ipynb": "fake checkpoint",
	})

	targets := []CleanTarget{
		{
			Path:        checkpointDir,
			Description: "Jupyter checkpoints",
			SizeBytes:   500,
			Safety:      config.Safe,
		},
	}

	results, err := cleaner.Clean(ctx, targets, true)
	if err != nil {
		t.Fatalf("Clean() returned error: %v", err)
	}

	if !results[0].Success {
		t.Error("Expected success in dry-run mode")
	}

	if _, err := os.Stat(checkpointDir); os.IsNotExist(err) {
		t.Error("Directory was deleted in dry-run mode")
	}
}

func TestDataMLCleaner_Clean_Actual(t *testing.T) {
	cleaner, _ := NewDataMLCleaner()
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	tmpDir := setupTestDir(t)
	defer os.RemoveAll(tmpDir)

	checkpointDir := createTestDir(t, tmpDir, ".ipynb_checkpoints", map[string]string{
		"notebook-checkpoint.ipynb": "fake checkpoint",
	})

	targets := []CleanTarget{
		{
			Path:        checkpointDir,
			Description: "Jupyter checkpoints",
			SizeBytes:   500,
			Safety:      config.Safe,
		},
	}

	results, err := cleaner.Clean(ctx, targets, false)
	if err != nil {
		t.Fatalf("Clean() returned error: %v", err)
	}

	if !results[0].Success {
		t.Errorf("Expected success, got error: %v", results[0].Error)
	}

	if _, err := os.Stat(checkpointDir); !os.IsNotExist(err) {
		t.Error("Directory was not deleted")
	}
}

// =============================================================================
// SystemCleaner Tests
// =============================================================================

func TestNewSystemCleaners(t *testing.T) {
	tests := []struct {
		name     string
		factory  func() Cleaner
		expected string
	}{
		{"TrashCleaner", NewTrashCleaner, "Trash"},
		{"CacheCleaner", NewCacheCleaner, "System Caches"},
		{"LogCleaner", NewLogCleaner, "System Logs"},
		{"TempFilesCleaner", NewTempFilesCleaner, "Temp Files"},
		{"DNSCacheCleaner", NewDNSCacheCleaner, "DNS Cache"},
		{"HomebrewCleaner", NewHomebrewCleaner, "Homebrew Cache"},
		{"XcodeCleaner", NewXcodeCleaner, "Xcode DerivedData"},
		{"LaunchpadCleaner", NewLaunchpadCleaner, "Launchpad Database"},
		{"IOSBackupCleaner", NewIOSBackupCleaner, "iOS Backups"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleaner := tt.factory()
			if cleaner == nil {
				t.Fatal("Cleaner is nil")
			}
			if cleaner.Name() != tt.expected {
				t.Errorf("Expected name '%s', got '%s'", tt.expected, cleaner.Name())
			}
			if cleaner.Domain() != config.DomainSystem {
				t.Errorf("Expected domain DomainSystem, got %v", cleaner.Domain())
			}
		})
	}
}

func TestSystemCleaner_Detect(t *testing.T) {
	cleaners := []struct {
		name    string
		cleaner Cleaner
	}{
		{"Trash", NewTrashCleaner()},
		{"Cache", NewCacheCleaner()},
		{"Logs", NewLogCleaner()},
		{"Temp", NewTempFilesCleaner()},
		{"DNS", NewDNSCacheCleaner()},
		{"Homebrew", NewHomebrewCleaner()},
		{"Xcode", NewXcodeCleaner()},
	}

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	for _, tc := range cleaners {
		t.Run(tc.name, func(t *testing.T) {
			detected, err := tc.cleaner.Detect(ctx)
			if err != nil {
				t.Fatalf("Detect() returned error: %v", err)
			}
			t.Logf("%s detected: %v", tc.name, detected)
		})
	}
}

// =============================================================================
// CleanTarget and CleanResult Tests
// =============================================================================

func TestCleanTarget_Fields(t *testing.T) {
	target := CleanTarget{
		Path:        "/tmp/test",
		Description: "Test target",
		SizeBytes:   1024,
		Safety:      config.Safe,
	}

	if target.Path != "/tmp/test" {
		t.Errorf("Expected path '/tmp/test', got '%s'", target.Path)
	}
	if target.Description != "Test target" {
		t.Errorf("Expected description 'Test target', got '%s'", target.Description)
	}
	if target.SizeBytes != 1024 {
		t.Errorf("Expected size 1024, got %d", target.SizeBytes)
	}
	if target.Safety != config.Safe {
		t.Errorf("Expected safety Safe, got %v", target.Safety)
	}
}

func TestCleanResult_Fields(t *testing.T) {
	target := CleanTarget{
		Path:        "/tmp/test",
		Description: "Test target",
		SizeBytes:   1024,
		Safety:      config.Safe,
	}

	result := CleanResult{
		Target:     target,
		Success:    true,
		BytesFreed: 1024,
		Error:      nil,
	}

	if result.Target.Path != "/tmp/test" {
		t.Errorf("Expected target path '/tmp/test', got '%s'", result.Target.Path)
	}
	if !result.Success {
		t.Error("Expected success to be true")
	}
	if result.BytesFreed != 1024 {
		t.Errorf("Expected bytes freed 1024, got %d", result.BytesFreed)
	}
	if result.Error != nil {
		t.Errorf("Expected no error, got %v", result.Error)
	}
}

// =============================================================================
// Safety Level Filtering Tests
// =============================================================================

func TestCleanLevel_AllowsSafety(t *testing.T) {
	tests := []struct {
		level    config.CleanLevel
		safety   config.SafetyLevel
		expected bool
	}{
		// Conservative only allows Safe
		{config.Conservative, config.Safe, true},
		{config.Conservative, config.Moderate, false},
		{config.Conservative, config.Dangerous, false},
		// Standard allows Safe and Moderate
		{config.Standard, config.Safe, true},
		{config.Standard, config.Moderate, true},
		{config.Standard, config.Dangerous, false},
		// Aggressive allows all
		{config.Aggressive, config.Safe, true},
		{config.Aggressive, config.Moderate, true},
		{config.Aggressive, config.Dangerous, true},
	}

	for _, tt := range tests {
		name := tt.level.String() + "_" + tt.safety.String()
		t.Run(name, func(t *testing.T) {
			result := tt.level.AllowsSafety(tt.safety)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// =============================================================================
// Multiple Targets Cleaning Tests
// =============================================================================

func TestCleaner_CleanMultipleTargets(t *testing.T) {
	cleaner, _ := NewFrontendCleaner()
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	tmpDir := setupTestDir(t)
	defer os.RemoveAll(tmpDir)

	// Create multiple targets
	targets := make([]CleanTarget, 3)
	for i := 0; i < 3; i++ {
		dir := createTestDir(t, tmpDir, filepath.Join("project"+string(rune('1'+i)), "node_modules"), map[string]string{
			"package/index.js": "module.exports = {}",
		})
		targets[i] = CleanTarget{
			Path:        dir,
			Description: "node_modules",
			SizeBytes:   100,
			Safety:      config.Moderate,
		}
	}

	// Clean all targets
	results, err := cleaner.Clean(ctx, targets, false)
	if err != nil {
		t.Fatalf("Clean() returned error: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("Expected 3 results, got %d", len(results))
	}

	for i, result := range results {
		if !result.Success {
			t.Errorf("Target %d failed: %v", i, result.Error)
		}
		// Verify each directory was deleted
		if _, err := os.Stat(targets[i].Path); !os.IsNotExist(err) {
			t.Errorf("Target %d was not deleted", i)
		}
	}
}

// =============================================================================
// Error Handling Tests
// =============================================================================

func TestCleaner_CleanNonExistentPath(t *testing.T) {
	cleaner, _ := NewFrontendCleaner()
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	targets := []CleanTarget{
		{
			Path:        "/nonexistent/path/that/does/not/exist",
			Description: "Nonexistent",
			SizeBytes:   100,
			Safety:      config.Safe,
		},
	}

	results, err := cleaner.Clean(ctx, targets, false)
	if err != nil {
		t.Fatalf("Clean() returned error: %v", err)
	}

	// os.RemoveAll succeeds even for non-existent paths
	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}
}

func TestCleaner_EmptyTargets(t *testing.T) {
	cleaner, _ := NewFrontendCleaner()
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	results, err := cleaner.Clean(ctx, []CleanTarget{}, false)
	if err != nil {
		t.Fatalf("Clean() returned error: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Expected 0 results, got %d", len(results))
	}
}
