package config

import (
	"testing"
)

// =============================================================================
// SafetyLevel Tests
// =============================================================================

func TestSafetyLevel_String(t *testing.T) {
	tests := []struct {
		level    SafetyLevel
		expected string
	}{
		{Safe, "Safe"},
		{Moderate, "Moderate"},
		{Dangerous, "Dangerous"},
		{SafetyLevel(99), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.level.String(); got != tt.expected {
				t.Errorf("SafetyLevel.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestSafetyLevel_Icon(t *testing.T) {
	tests := []struct {
		level    SafetyLevel
		expected string
	}{
		{Safe, "🟢"},
		{Moderate, "🟡"},
		{Dangerous, "🔴"},
		{SafetyLevel(99), "❓"},
	}

	for _, tt := range tests {
		name := tt.level.String()
		t.Run(name, func(t *testing.T) {
			if got := tt.level.Icon(); got != tt.expected {
				t.Errorf("SafetyLevel.Icon() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// =============================================================================
// CleanLevel Tests
// =============================================================================

func TestCleanLevel_String(t *testing.T) {
	tests := []struct {
		level    CleanLevel
		expected string
	}{
		{Conservative, "conservative"},
		{Standard, "standard"},
		{Aggressive, "aggressive"},
		{CleanLevel(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.level.String(); got != tt.expected {
				t.Errorf("CleanLevel.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestParseCleanLevel(t *testing.T) {
	tests := []struct {
		input       string
		expected    CleanLevel
		expectError bool
	}{
		{"conservative", Conservative, false},
		{"standard", Standard, false},
		{"aggressive", Aggressive, false},
		{"invalid", Standard, true},
		{"STANDARD", Standard, true}, // case sensitive
		{"", Standard, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseCleanLevel(tt.input)
			if tt.expectError {
				if err == nil {
					t.Errorf("ParseCleanLevel(%q) expected error, got nil", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("ParseCleanLevel(%q) unexpected error: %v", tt.input, err)
				}
				if got != tt.expected {
					t.Errorf("ParseCleanLevel(%q) = %v, want %v", tt.input, got, tt.expected)
				}
			}
		})
	}
}

func TestCleanLevel_AllowsSafety(t *testing.T) {
	tests := []struct {
		cleanLevel  CleanLevel
		safetyLevel SafetyLevel
		expected    bool
	}{
		// Conservative only allows Safe
		{Conservative, Safe, true},
		{Conservative, Moderate, false},
		{Conservative, Dangerous, false},
		// Standard allows Safe and Moderate
		{Standard, Safe, true},
		{Standard, Moderate, true},
		{Standard, Dangerous, false},
		// Aggressive allows all
		{Aggressive, Safe, true},
		{Aggressive, Moderate, true},
		{Aggressive, Dangerous, true},
		// Unknown level (edge case)
		{CleanLevel(99), Safe, false},
	}

	for _, tt := range tests {
		name := tt.cleanLevel.String() + "_allows_" + tt.safetyLevel.String()
		t.Run(name, func(t *testing.T) {
			if got := tt.cleanLevel.AllowsSafety(tt.safetyLevel); got != tt.expected {
				t.Errorf("CleanLevel.AllowsSafety() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// =============================================================================
// Domain Tests
// =============================================================================

func TestDomain_String(t *testing.T) {
	tests := []struct {
		domain   Domain
		expected string
	}{
		{DomainSystem, "System"},
		{DomainFrontend, "Frontend"},
		{Domain(99), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.domain.String(); got != tt.expected {
				t.Errorf("Domain.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// =============================================================================
// Config Tests
// =============================================================================

func TestNewDefaultConfig(t *testing.T) {
	cfg := NewDefaultConfig()

	if cfg == nil {
		t.Fatal("NewDefaultConfig() returned nil")
	}

	// Verify default values
	if cfg.DryRun != false {
		t.Errorf("Expected DryRun to be false, got %v", cfg.DryRun)
	}

	if cfg.Interactive != true {
		t.Errorf("Expected Interactive to be true, got %v", cfg.Interactive)
	}

	if len(cfg.Domains) != 0 {
		t.Errorf("Expected empty Domains, got %v", cfg.Domains)
	}

	if cfg.CleanLevel != Standard {
		t.Errorf("Expected CleanLevel to be Standard, got %v", cfg.CleanLevel)
	}

	if cfg.MaxConcurrent != 4 {
		t.Errorf("Expected MaxConcurrent to be 4, got %d", cfg.MaxConcurrent)
	}

	if cfg.Verbose != false {
		t.Errorf("Expected Verbose to be false, got %v", cfg.Verbose)
	}
}

func TestConfig_Modification(t *testing.T) {
	cfg := NewDefaultConfig()

	// Modify config
	cfg.DryRun = true
	cfg.Interactive = false
	cfg.CleanLevel = Aggressive
	cfg.Domains = []Domain{DomainSystem, DomainFrontend}
	cfg.MaxConcurrent = 8
	cfg.Verbose = true

	// Verify modifications
	if !cfg.DryRun {
		t.Error("DryRun modification failed")
	}
	if cfg.Interactive {
		t.Error("Interactive modification failed")
	}
	if cfg.CleanLevel != Aggressive {
		t.Error("CleanLevel modification failed")
	}
	if len(cfg.Domains) != 2 {
		t.Error("Domains modification failed")
	}
	if cfg.MaxConcurrent != 8 {
		t.Error("MaxConcurrent modification failed")
	}
	if !cfg.Verbose {
		t.Error("Verbose modification failed")
	}
}

// =============================================================================
// Integration Tests
// =============================================================================

func TestConfigWithCleanLevel_Integration(t *testing.T) {
	// Test that config levels work correctly with safety filtering
	tests := []struct {
		name            string
		cleanLevel      CleanLevel
		safeCount       int // number of Safe items that should be included
		moderateCount   int // number of Moderate items that should be included
		dangerousCount  int // number of Dangerous items that should be included
	}{
		{"Conservative", Conservative, 1, 0, 0},
		{"Standard", Standard, 1, 1, 0},
		{"Aggressive", Aggressive, 1, 1, 1},
	}

	items := []SafetyLevel{Safe, Moderate, Dangerous}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := NewDefaultConfig()
			cfg.CleanLevel = tt.cleanLevel

			safeCount := 0
			moderateCount := 0
			dangerousCount := 0

			for _, item := range items {
				if cfg.CleanLevel.AllowsSafety(item) {
					switch item {
					case Safe:
						safeCount++
					case Moderate:
						moderateCount++
					case Dangerous:
						dangerousCount++
					}
				}
			}

			if safeCount != tt.safeCount {
				t.Errorf("Safe count = %d, want %d", safeCount, tt.safeCount)
			}
			if moderateCount != tt.moderateCount {
				t.Errorf("Moderate count = %d, want %d", moderateCount, tt.moderateCount)
			}
			if dangerousCount != tt.dangerousCount {
				t.Errorf("Dangerous count = %d, want %d", dangerousCount, tt.dangerousCount)
			}
		})
	}
}
