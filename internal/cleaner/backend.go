package cleaner

import (
	"context"
	"os"
	"path/filepath"

	"github.com/0SansNom/epurer/internal/config"
	"github.com/0SansNom/epurer/internal/scanner"
	"github.com/0SansNom/epurer/pkg/utils"
)

// BackendCleaner handles backend development cleanup (Python, Java, Go, Rust, PHP, Ruby)
type BackendCleaner struct {
	scanner *scanner.Scanner
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

func (b *BackendCleaner) Name() string {
	return "Backend"
}

func (b *BackendCleaner) Domain() config.Domain {
	return config.DomainFrontend // TODO: Add DomainBackend to config
}

func (b *BackendCleaner) Detect(ctx context.Context) (bool, error) {
	// Check for common backend tools
	return utils.CommandExists("python3") ||
		utils.CommandExists("python") ||
		utils.CommandExists("java") ||
		utils.CommandExists("go") ||
		utils.CommandExists("cargo") ||
		utils.CommandExists("php") ||
		utils.CommandExists("ruby"), nil
}

func (b *BackendCleaner) Scan(ctx context.Context, cfg *config.Config) ([]CleanTarget, error) {
	targets := []CleanTarget{}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	// === Python ===

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

	// pip cache (Safe)
	pipCachePath := filepath.Join(home, "Library", "Caches", "pip")
	if utils.PathExists(pipCachePath) {
		size, _ := utils.GetDirSize(pipCachePath)
		if size > 0 {
			targets = append(targets, CleanTarget{
				Path:        pipCachePath,
				Description: "pip cache",
				SizeBytes:   size,
				Safety:      config.Safe,
			})
		}
	}

	// Poetry cache (Safe)
	poetryCachePath := filepath.Join(home, "Library", "Caches", "pypoetry")
	if utils.PathExists(poetryCachePath) {
		size, _ := utils.GetDirSize(poetryCachePath)
		if size > 0 {
			targets = append(targets, CleanTarget{
				Path:        poetryCachePath,
				Description: "Poetry cache",
				SizeBytes:   size,
				Safety:      config.Safe,
			})
		}
	}

	// === Java / Maven / Gradle ===

	// Maven local repository (Moderate - can be large)
	if cfg.CleanLevel.AllowsSafety(config.Moderate) {
		mavenRepoPath := filepath.Join(home, ".m2", "repository")
		if utils.PathExists(mavenRepoPath) {
			size, _ := utils.GetDirSize(mavenRepoPath)
			if size > 0 {
				targets = append(targets, CleanTarget{
					Path:        mavenRepoPath,
					Description: "Maven local repository",
					SizeBytes:   size,
					Safety:      config.Moderate,
				})
			}
		}
	}

	// Gradle cache (Safe)
	gradleCachePath := filepath.Join(home, ".gradle", "caches")
	if utils.PathExists(gradleCachePath) {
		size, _ := utils.GetDirSize(gradleCachePath)
		if size > 0 {
			targets = append(targets, CleanTarget{
				Path:        gradleCachePath,
				Description: "Gradle cache",
				SizeBytes:   size,
				Safety:      config.Safe,
			})
		}
	}

	// target folders (Java/Scala build output - Safe)
	targetTargets := b.scanPattern(ctx, "target")
	targets = append(targets, targetTargets...)

	// === Go ===

	// Go build cache (Safe)
	goCachePath := filepath.Join(home, "Library", "Caches", "go-build")
	if utils.PathExists(goCachePath) {
		size, _ := utils.GetDirSize(goCachePath)
		if size > 0 {
			targets = append(targets, CleanTarget{
				Path:        goCachePath,
				Description: "Go build cache",
				SizeBytes:   size,
				Safety:      config.Safe,
			})
		}
	}

	// Go module cache (Moderate - can be large)
	if cfg.CleanLevel.AllowsSafety(config.Moderate) {
		goModCachePath := filepath.Join(home, "go", "pkg", "mod")
		if utils.PathExists(goModCachePath) {
			size, _ := utils.GetDirSize(goModCachePath)
			if size > 0 {
				targets = append(targets, CleanTarget{
					Path:        goModCachePath,
					Description: "Go module cache",
					SizeBytes:   size,
					Safety:      config.Moderate,
				})
			}
		}
	}

	// === Rust ===

	// Cargo cache (Safe)
	cargoCachePath := filepath.Join(home, ".cargo", "registry")
	if utils.PathExists(cargoCachePath) {
		size, _ := utils.GetDirSize(cargoCachePath)
		if size > 0 {
			targets = append(targets, CleanTarget{
				Path:        cargoCachePath,
				Description: "Cargo registry cache",
				SizeBytes:   size,
				Safety:      config.Safe,
			})
		}
	}

	// Rust target folders (Moderate - build artifacts)
	if cfg.CleanLevel.AllowsSafety(config.Moderate) {
		rustTargetTargets := b.scanRustTargets(ctx)
		targets = append(targets, rustTargetTargets...)
	}

	// === PHP ===

	// Composer cache (Safe)
	composerCachePath := filepath.Join(home, ".composer", "cache")
	if utils.PathExists(composerCachePath) {
		size, _ := utils.GetDirSize(composerCachePath)
		if size > 0 {
			targets = append(targets, CleanTarget{
				Path:        composerCachePath,
				Description: "Composer cache",
				SizeBytes:   size,
				Safety:      config.Safe,
			})
		}
	}

	// vendor folders (Moderate - PHP dependencies)
	if cfg.CleanLevel.AllowsSafety(config.Moderate) {
		vendorTargets := b.scanPHPVendor(ctx)
		targets = append(targets, vendorTargets...)
	}

	// === Ruby ===

	// Gem cache (Safe)
	gemCachePath := filepath.Join(home, ".gem")
	if utils.PathExists(gemCachePath) {
		cachePath := filepath.Join(gemCachePath, "cache")
		if utils.PathExists(cachePath) {
			size, _ := utils.GetDirSize(cachePath)
			if size > 0 {
				targets = append(targets, CleanTarget{
					Path:        cachePath,
					Description: "Ruby gem cache",
					SizeBytes:   size,
					Safety:      config.Safe,
				})
			}
		}
	}

	// Bundler cache (Safe)
	bundlerCachePath := filepath.Join(home, ".bundle", "cache")
	if utils.PathExists(bundlerCachePath) {
		size, _ := utils.GetDirSize(bundlerCachePath)
		if size > 0 {
			targets = append(targets, CleanTarget{
				Path:        bundlerCachePath,
				Description: "Bundler cache",
				SizeBytes:   size,
				Safety:      config.Safe,
			})
		}
	}

	return targets, nil
}

func (b *BackendCleaner) Clean(ctx context.Context, targets []CleanTarget, dryRun bool) ([]CleanResult, error) {
	results := make([]CleanResult, 0, len(targets))

	for _, target := range targets {
		result := CleanResult{
			Target:  target,
			Success: true,
		}

		if !dryRun {
			err := utils.SafeRemove(target.Path, false)
			if err != nil {
				result.Success = false
				result.Error = err
			} else {
				result.BytesFreed = target.SizeBytes
			}
		} else {
			result.BytesFreed = target.SizeBytes
		}

		results = append(results, result)

		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}
	}

	return results, nil
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
		"__pycache__":    "Python bytecode cache",
		"*.pyc":          "Python compiled files",
		".pytest_cache":  "pytest cache",
		".mypy_cache":    "mypy type checker cache",
		".tox":           "tox test environments",
		"target":         "Build output (target)",
	}

	if desc, ok := descriptions[pattern]; ok {
		return desc
	}

	return pattern
}
