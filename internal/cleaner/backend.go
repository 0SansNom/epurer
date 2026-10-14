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

	targets = append(targets, b.scanProjectFiles(ctx, cfg)...)

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
	return targets, nil
}

func (b *BackendCleaner) Clean(ctx context.Context, targets []CleanTarget, dryRun bool) ([]CleanResult, error) {
	return CleanTargets(ctx, targets, dryRun, b.progress, b.ignore)
}

// scanProjectFiles finds caches and build folders in one walk.
// target and vendor are Moderate, and only next to their project file.
func (b *BackendCleaner) scanProjectFiles(ctx context.Context, cfg *config.Config) []CleanTarget {
	patterns := []string{"__pycache__", "*.pyc", ".pytest_cache", ".mypy_cache", ".tox", "target", "vendor"}
	moderate := cfg.CleanLevel.AllowsSafety(config.Moderate)
	var targets []CleanTarget
	for result := range b.scanner.FindByPatterns(ctx, patterns) {
		if result.Err != nil {
			continue
		}
		safety := config.Safe
		desc := b.getDescriptionForPattern(result.Pattern)
		switch result.Pattern {
		case "target":
			if !moderate || !siblingExists(result.Path, "Cargo.toml", "pom.xml", "build.sbt") {
				continue
			}
			safety = config.Moderate
		case "vendor":
			if !moderate || !siblingExists(result.Path, "composer.json") {
				continue
			}
			safety = config.Moderate
			desc = "PHP vendor dependencies"
		}
		targets = append(targets, CleanTarget{
			Path:        result.Path,
			Description: desc,
			SizeBytes:   result.Size,
			Safety:      safety,
		})
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
