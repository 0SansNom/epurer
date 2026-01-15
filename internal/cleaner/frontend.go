package cleaner

import (
	"context"
	"os"
	"path/filepath"

	"github.com/0SansNom/epurer/internal/config"
	"github.com/0SansNom/epurer/internal/scanner"
	"github.com/0SansNom/epurer/pkg/utils"
)

// FrontendCleaner handles frontend development cleanup (Node.js, npm, yarn, pnpm, etc.)
type FrontendCleaner struct {
	scanner *scanner.Scanner
}

// NewFrontendCleaner creates a new FrontendCleaner
func NewFrontendCleaner() (Cleaner, error) {
	s, err := scanner.NewScanner()
	if err != nil {
		return nil, err
	}

	return &FrontendCleaner{
		scanner: s,
	}, nil
}

func (f *FrontendCleaner) Name() string {
	return "Frontend"
}

func (f *FrontendCleaner) Domain() config.Domain {
	return config.DomainFrontend
}

func (f *FrontendCleaner) Detect(ctx context.Context) (bool, error) {
	// Check if Node.js ecosystem tools are installed
	return utils.CommandExists("node") ||
		utils.CommandExists("npm") ||
		utils.CommandExists("yarn") ||
		utils.CommandExists("pnpm"), nil
}

func (f *FrontendCleaner) Scan(ctx context.Context, cfg *config.Config) ([]CleanTarget, error) {
	targets := []CleanTarget{}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	// === Package manager caches (Safe - always can be rebuilt) ===

	// npm cache
	npmCachePath := filepath.Join(home, ".npm")
	if utils.PathExists(npmCachePath) {
		size, _ := utils.GetDirSize(npmCachePath)
		if size > 0 {
			targets = append(targets, CleanTarget{
				Path:        npmCachePath,
				Description: "npm cache",
				SizeBytes:   size,
				Safety:      config.Safe,
			})
		}
	}

	// yarn cache
	yarnCachePath := filepath.Join(home, ".cache", "yarn")
	if utils.PathExists(yarnCachePath) {
		size, _ := utils.GetDirSize(yarnCachePath)
		if size > 0 {
			targets = append(targets, CleanTarget{
				Path:        yarnCachePath,
				Description: "Yarn cache",
				SizeBytes:   size,
				Safety:      config.Safe,
			})
		}
	}

	// Yarn global cache (Library/Caches/Yarn on macOS)
	yarnGlobalCache := filepath.Join(home, "Library", "Caches", "Yarn")
	if utils.PathExists(yarnGlobalCache) {
		size, _ := utils.GetDirSize(yarnGlobalCache)
		if size > 0 {
			targets = append(targets, CleanTarget{
				Path:        yarnGlobalCache,
				Description: "Yarn global cache",
				SizeBytes:   size,
				Safety:      config.Safe,
			})
		}
	}

	// pnpm store
	pnpmStorePath := filepath.Join(home, ".pnpm-store")
	if utils.PathExists(pnpmStorePath) {
		size, _ := utils.GetDirSize(pnpmStorePath)
		if size > 0 {
			targets = append(targets, CleanTarget{
				Path:        pnpmStorePath,
				Description: "pnpm store",
				SizeBytes:   size,
				Safety:      config.Safe,
			})
		}
	}

	// === node_modules (Moderate - needs npm install) ===

	if cfg.CleanLevel.AllowsSafety(config.Moderate) {
		nodeModulesTargets := f.scanNodeModules(ctx)
		targets = append(targets, nodeModulesTargets...)
	}

	// === Build outputs (Safe - easily rebuilt) ===

	// dist folders
	distTargets := f.scanPattern(ctx, "dist")
	targets = append(targets, distTargets...)

	// build folders
	buildTargets := f.scanPattern(ctx, "build")
	targets = append(targets, buildTargets...)

	// out folders (Next.js, etc.)
	outTargets := f.scanPattern(ctx, "out")
	targets = append(targets, outTargets...)

	// .next (Next.js)
	nextTargets := f.scanPattern(ctx, ".next")
	targets = append(targets, nextTargets...)

	// === Bundler caches (Safe) ===

	// Vite cache
	viteTargets := f.scanPattern(ctx, ".vite")
	targets = append(targets, viteTargets...)

	// Parcel cache
	parcelTargets := f.scanPattern(ctx, ".parcel-cache")
	targets = append(targets, parcelTargets...)

	// Webpack cache (inside node_modules/.cache/webpack)
	// We'll get this with a more specific scan
	webpackCacheTargets := f.scanNestedCache(ctx, ".cache/webpack")
	targets = append(targets, webpackCacheTargets...)

	// Turbo cache
	turboCacheTargets := f.scanNestedCache(ctx, ".cache/turbo")
	targets = append(targets, turboCacheTargets...)

	// === Testing coverage (Safe) ===

	coverageTargets := f.scanPattern(ctx, "coverage")
	targets = append(targets, coverageTargets...)

	nycTargets := f.scanPattern(ctx, ".nyc_output")
	targets = append(targets, nycTargets...)

	// === Linter caches (Safe) ===

	eslintTargets := f.scanPattern(ctx, ".eslintcache")
	targets = append(targets, eslintTargets...)

	// === Storybook (Safe) ===

	storybookTargets := f.scanPattern(ctx, "storybook-static")
	targets = append(targets, storybookTargets...)

	// === Log files (Safe) ===

	npmLogTargets := f.scanPattern(ctx, "npm-debug.log*")
	targets = append(targets, npmLogTargets...)

	yarnLogTargets := f.scanPattern(ctx, "yarn-error.log*")
	targets = append(targets, yarnLogTargets...)

	yarnDebugTargets := f.scanPattern(ctx, "yarn-debug.log*")
	targets = append(targets, yarnDebugTargets...)

	return targets, nil
}

func (f *FrontendCleaner) Clean(ctx context.Context, targets []CleanTarget, dryRun bool) ([]CleanResult, error) {
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
			// In dry-run, just report what would be freed
			result.BytesFreed = target.SizeBytes
		}

		results = append(results, result)

		// Check for cancellation
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}
	}

	return results, nil
}

// scanNodeModules scans for node_modules directories
func (f *FrontendCleaner) scanNodeModules(ctx context.Context) []CleanTarget {
	targets := []CleanTarget{}

	resultChan := f.scanner.FindByPattern(ctx, "node_modules")
	for result := range resultChan {
		if result.Err != nil {
			continue
		}

		// Skip node_modules inside other node_modules (nested dependencies)
		if filepath.Base(filepath.Dir(result.Path)) == "node_modules" {
			continue
		}

		targets = append(targets, CleanTarget{
			Path:        result.Path,
			Description: "node_modules dependencies",
			SizeBytes:   result.Size,
			Safety:      config.Moderate,
		})
	}

	return targets
}

// scanPattern is a generic scanner for simple patterns
func (f *FrontendCleaner) scanPattern(ctx context.Context, pattern string) []CleanTarget {
	targets := []CleanTarget{}

	resultChan := f.scanner.FindByPattern(ctx, pattern)
	for result := range resultChan {
		if result.Err != nil {
			continue
		}

		// Determine description based on pattern
		desc := f.getDescriptionForPattern(pattern)

		targets = append(targets, CleanTarget{
			Path:        result.Path,
			Description: desc,
			SizeBytes:   result.Size,
			Safety:      config.Safe,
		})
	}

	return targets
}

// scanNestedCache scans for caches inside node_modules
func (f *FrontendCleaner) scanNestedCache(ctx context.Context, subPath string) []CleanTarget {
	targets := []CleanTarget{}

	// First find all node_modules
	nodeModulesChan := f.scanner.FindByPattern(ctx, "node_modules")
	for nmResult := range nodeModulesChan {
		if nmResult.Err != nil {
			continue
		}

		// Check if the cache exists inside this node_modules
		cachePath := filepath.Join(nmResult.Path, subPath)
		if utils.PathExists(cachePath) {
			size, _ := utils.GetDirSize(cachePath)
			if size > 0 {
				targets = append(targets, CleanTarget{
					Path:        cachePath,
					Description: filepath.Base(subPath) + " cache",
					SizeBytes:   size,
					Safety:      config.Safe,
				})
			}
		}
	}

	return targets
}

// getDescriptionForPattern returns a human-readable description for a pattern
func (f *FrontendCleaner) getDescriptionForPattern(pattern string) string {
	descriptions := map[string]string{
		"dist":              "Build output (dist)",
		"build":             "Build output (build)",
		"out":               "Build output (out)",
		".next":             "Next.js build cache",
		".vite":             "Vite cache",
		".parcel-cache":     "Parcel cache",
		"coverage":          "Test coverage reports",
		".nyc_output":       "NYC coverage output",
		".eslintcache":      "ESLint cache",
		"storybook-static":  "Storybook static build",
		"npm-debug.log*":    "npm debug logs",
		"yarn-error.log*":   "Yarn error logs",
		"yarn-debug.log*":   "Yarn debug logs",
	}

	if desc, ok := descriptions[pattern]; ok {
		return desc
	}

	return pattern
}
