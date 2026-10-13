package cleaner

import (
	"context"
	"os"
	"path/filepath"

	"github.com/0SansNom/epurer/internal/config"
	"github.com/0SansNom/epurer/internal/ignorelist"
	"github.com/0SansNom/epurer/internal/scanner"
	"github.com/0SansNom/epurer/internal/scantask"
	"github.com/0SansNom/epurer/pkg/utils"
)

// BackendCleaner handles backend development cleanup (Python, Java, Go, Rust, PHP, Ruby)
type BackendCleaner struct {
	scanner *scanner.Scanner
	fsCleaner
}

// NewBackendCleaner creates a new BackendCleaner
func NewBackendCleaner() (Cleaner, error) {
	s, err := scanner.NewScanner()
	if err != nil {
		return nil, err
	}

	return &BackendCleaner{
		scanner: s,
	}, nil
}

// SetIgnoreList makes this cleaner respect a persistent ignore list.
func (b *BackendCleaner) SetIgnoreList(l *ignorelist.List) {
	b.fsCleaner.SetIgnoreList(l)
	b.scanner.SetIgnoreList(l)
}

func (b *BackendCleaner) Name() string {
	return "Backend"
}

func (b *BackendCleaner) Domain() config.Domain {
	return config.DomainBackend
}

func (b *BackendCleaner) Detect(ctx context.Context) (bool, error) {
	if utils.CommandExists("python3") ||
		utils.CommandExists("python") ||
		utils.CommandExists("java") ||
		utils.CommandExists("go") ||
		utils.CommandExists("cargo") ||
		utils.CommandExists("php") ||
		utils.CommandExists("ruby") {
		return true, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return false, err
	}
	// Android Studio often ships Java off PATH. The cache is still this cleaner's job.
	return utils.PathExists(filepath.Join(home, ".gradle", "caches")), nil
}

func (b *BackendCleaner) Scan(ctx context.Context, cfg *config.Config) ([]CleanTarget, error) {
	targets := []CleanTarget{}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	// __pycache__ (Safe - automatically rebuilt)
	pycacheTargets := b.scanPattern(ctx, "__pycache__")
	targets = append(targets, pycacheTargets...)

	// .pyc files (Safe)
	pycTargets := b.scanPattern(ctx, "*.pyc")
	targets = append(targets, pycTargets...)

	// .pytest_cache (Safe)
	pytestTargets := b.scanPattern(ctx, ".pytest_cache")
	targets = append(targets, pytestTargets...)

	// .mypy_cache (Safe)
	mypyTargets := b.scanPattern(ctx, ".mypy_cache")
	targets = append(targets, mypyTargets...)

	// .tox (Safe - test environments)
	toxTargets := b.scanPattern(ctx, ".tox")
	targets = append(targets, toxTargets...)

	// pip, Poetry, and the Go build cache are top-level entries of ~/Library/Caches.

	walks := []scantask.Walk{
		{Root: filepath.Join(home, ".cargo", "registry"), Description: "Cargo registry cache", Safety: config.Safe},
		{Root: filepath.Join(home, ".composer", "cache"), Description: "Composer cache", Safety: config.Safe},
		{Root: filepath.Join(home, ".gem", "cache"), Description: "Ruby gem cache", Safety: config.Safe},
		{Root: filepath.Join(home, ".bundle", "cache"), Description: "Bundler cache", Safety: config.Safe},
		{Root: filepath.Join(home, ".gradle", "caches"), Description: "Gradle cache", Safety: config.Safe},
	}
	if cfg.CleanLevel.AllowsSafety(config.Moderate) {
		walks = append(walks,
			scantask.Walk{Root: filepath.Join(home, ".m2", "repository"), Description: "Maven local repository", Safety: config.Moderate},
			scantask.Walk{Root: filepath.Join(home, "go", "pkg", "mod"), Description: "Go module cache", Safety: config.Moderate},
		)
	}
	found, err := b.collect(ctx, walks...)
	if err != nil {
		return nil, err
	}
	targets = append(targets, found...)

	// target folders (Java/Scala build output - Safe)
	targetTargets := b.scanPattern(ctx, "target")
	targets = append(targets, targetTargets...)

	// Rust target folders (Moderate - build artifacts)
	if cfg.CleanLevel.AllowsSafety(config.Moderate) {
		rustTargetTargets := b.scanRustTargets(ctx)
		targets = append(targets, rustTargetTargets...)
	}

	// vendor folders (Moderate - PHP dependencies)
	if cfg.CleanLevel.AllowsSafety(config.Moderate) {
		vendorTargets := b.scanPHPVendor(ctx)
		targets = append(targets, vendorTargets...)
	}

	return targets, nil
}

func (b *BackendCleaner) Clean(ctx context.Context, targets []CleanTarget, dryRun bool) ([]CleanResult, error) {
	return CleanTargets(ctx, targets, dryRun, b.progress)
}

// scanPattern scans for a specific pattern
func (b *BackendCleaner) scanPattern(ctx context.Context, pattern string) []CleanTarget {
	targets := []CleanTarget{}

	resultChan := b.scanner.FindByPattern(ctx, pattern)
	for result := range resultChan {
		if result.Err != nil {
			continue
		}

		desc := b.getDescriptionForPattern(pattern)

		targets = append(targets, CleanTarget{
			Path:        result.Path,
			Description: desc,
			SizeBytes:   result.Size,
			Safety:      config.Safe,
		})
	}

	return targets
}

// scanRustTargets scans for Rust target folders (build output)
func (b *BackendCleaner) scanRustTargets(ctx context.Context) []CleanTarget {
	targets := []CleanTarget{}

	resultChan := b.scanner.FindByPattern(ctx, "target")
	for result := range resultChan {
		if result.Err != nil {
			continue
		}

		// Check if this is a Rust project by looking for Cargo.toml
		parent := filepath.Dir(result.Path)
		cargoTomlPath := filepath.Join(parent, "Cargo.toml")
		if utils.PathExists(cargoTomlPath) {
			targets = append(targets, CleanTarget{
				Path:        result.Path,
				Description: "Rust build output (target)",
				SizeBytes:   result.Size,
				Safety:      config.Moderate,
			})
		}
	}

	return targets
}

// scanPHPVendor scans for PHP vendor folders
func (b *BackendCleaner) scanPHPVendor(ctx context.Context) []CleanTarget {
	targets := []CleanTarget{}

	resultChan := b.scanner.FindByPattern(ctx, "vendor")
	for result := range resultChan {
		if result.Err != nil {
			continue
		}

		// Check if this is a PHP project by looking for composer.json
		parent := filepath.Dir(result.Path)
		composerJsonPath := filepath.Join(parent, "composer.json")
		if utils.PathExists(composerJsonPath) {
			targets = append(targets, CleanTarget{
				Path:        result.Path,
				Description: "PHP vendor dependencies",
				SizeBytes:   result.Size,
				Safety:      config.Moderate,
			})
		}
	}

	return targets
}

// getDescriptionForPattern returns a human-readable description
func (b *BackendCleaner) getDescriptionForPattern(pattern string) string {
	descriptions := map[string]string{
		"__pycache__":   "Python bytecode cache",
		"*.pyc":         "Python compiled files",
		".pytest_cache": "pytest cache",
		".mypy_cache":   "mypy type checker cache",
		".tox":          "tox test environments",
		"target":        "Build output (target)",
	}

	if desc, ok := descriptions[pattern]; ok {
		return desc
	}

	return pattern
}
