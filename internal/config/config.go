package config

import "fmt"

// SafetyLevel indicates the risk level of a cleanup operation
type SafetyLevel int

const (
	Safe       SafetyLevel = iota // 🟢 No risk - easily rebuilt (caches, logs)
	Moderate                      // 🟡 Rebuild needed (node_modules, builds)
	Dangerous                     // 🔴 Potential data loss (backups, databases)
)

// String returns human-readable representation
func (s SafetyLevel) String() string {
	switch s {
	case Safe:
		return "Safe"
	case Moderate:
		return "Moderate"
	case Dangerous:
		return "Dangerous"
	default:
		return "Unknown"
	}
}

// Icon returns emoji icon for safety level
func (s SafetyLevel) Icon() string {
	switch s {
	case Safe:
		return "🟢"
	case Moderate:
		return "🟡"
	case Dangerous:
		return "🔴"
	default:
		return "❓"
	}
}

// CleanLevel represents the aggressiveness of cleaning
type CleanLevel int

const (
	Conservative CleanLevel = iota // Only Safe items
	Standard                        // Safe + Moderate items
	Aggressive                      // All items including Dangerous
)

// String returns human-readable representation
func (cl CleanLevel) String() string {
	switch cl {
	case Conservative:
		return "conservative"
	case Standard:
		return "standard"
	case Aggressive:
		return "aggressive"
	default:
		return "unknown"
	}
}

// ParseCleanLevel converts string to CleanLevel
func ParseCleanLevel(s string) (CleanLevel, error) {
	switch s {
	case "conservative":
		return Conservative, nil
	case "standard":
		return Standard, nil
	case "aggressive":
		return Aggressive, nil
	default:
		return Standard, fmt.Errorf("invalid clean level: %s (must be conservative, standard, or aggressive)", s)
	}
}

// AllowsSafety checks if a safety level is allowed at this clean level
func (cl CleanLevel) AllowsSafety(safety SafetyLevel) bool {
	switch cl {
	case Conservative:
		return safety == Safe
	case Standard:
		return safety == Safe || safety == Moderate
	case Aggressive:
		return true
	default:
		return false
	}
}

// Domain represents a category of cleaners
type Domain int

const (
	DomainSystem   Domain = iota // System-level cleaners (trash, cache, logs)
	DomainFrontend               // Frontend development (node_modules, npm cache)
	DomainBackend                // Backend development (Python, Java, Go, Rust, PHP, Ruby)
	DomainMobile                 // Mobile development (Xcode, Android, Flutter)
	DomainDevOps                 // DevOps tooling (Docker, Kubernetes, Terraform)
	DomainDataML                 // Data science / ML tooling (Conda, Jupyter, TensorFlow)
	DomainAI                     // AI coding tool caches (Claude, Cursor, Copilot, ...)
)

// String returns human-readable representation
func (d Domain) String() string {
	switch d {
	case DomainSystem:
		return "System"
	case DomainFrontend:
		return "Frontend"
	case DomainBackend:
		return "Backend"
	case DomainMobile:
		return "Mobile"
	case DomainDevOps:
		return "DevOps"
	case DomainDataML:
		return "Data/ML"
	case DomainAI:
		return "AI"
	default:
		return "Unknown"
	}
}

// Config holds runtime configuration for the cleaner
type Config struct {
	DryRun        bool       // If true, don't actually delete anything
	Interactive   bool       // If true, ask for confirmation before cleaning
	Domains       []Domain   // Which domains to clean (empty = all)
	CleanLevel    CleanLevel // How aggressive to be
	MaxConcurrent int        // Max number of concurrent scans
	Verbose       bool       // Enable verbose output
}

// NewDefaultConfig returns a Config with sensible defaults
func NewDefaultConfig() *Config {
	return &Config{
		DryRun:        false,
		Interactive:   true,
		Domains:       []Domain{},
		CleanLevel:    Standard,
		MaxConcurrent: 4,
		Verbose:       false,
	}
}
