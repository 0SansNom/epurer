package cleaner

import (
	"context"
	"os"
	"path/filepath"

	"github.com/0SansNom/epurer/internal/config"
	"github.com/0SansNom/epurer/internal/ignorelist"
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

// SetIgnoreList makes this cleaner respect a persistent ignore list.
func (f *FrontendCleaner) SetIgnoreList(l *ignorelist.List) {
	f.scanner.SetIgnoreList(l)
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

	if cfg.CleanLevel.AllowsSafety(config.Moderate) {
		nodeModulesTargets := f.scanNodeModules(ctx)
		targets = append(targets, nodeModulesTargets...)
	}

	targets = append(targets, f.scanPattern(ctx, "dist")...)
	targets = append(targets, f.scanPattern(ctx, "build")...)
	targets = append(targets, f.scanPattern(ctx, "out")...)
	targets = append(targets, f.scanPattern(ctx, ".next")...)

	targets = append(targets, f.scanPattern(ctx, ".vite")...)
	targets = append(targets, f.scanPattern(ctx, ".parcel-cache")...)
	targets = append(targets, f.scanNestedCache(ctx, ".cache/webpack")...)
	targets = append(targets, f.scanNestedCache(ctx, ".cache/turbo")...)

	targets = append(targets, f.scanPattern(ctx, "coverage")...)
	targets = append(targets, f.scanPattern(ctx, ".nyc_output")...)

	targets = append(targets, f.scanPattern(ctx, ".eslintcache")...)

	targets = append(targets, f.scanPattern(ctx, "storybook-static")...)

	targets = append(targets, f.scanPattern(ctx, "npm-debug.log*")...)
	targets = append(targets, f.scanPattern(ctx, "yarn-error.log*")...)
	targets = append(targets, f.scanPattern(ctx, "yarn-debug.log*")...)

	return targets, nil
}

func (f *FrontendCleaner) Clean(ctx context.Context, targets []CleanTarget, dryRun bool) ([]CleanResult, error) {
	return CleanTargets(ctx, targets, dryRun)
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
