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

// FrontendCleaner handles frontend development cleanup (Node.js, npm, yarn, pnpm, etc.)
type FrontendCleaner struct {
	scanner *scanner.Scanner
	fsCleaner
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
	f.fsCleaner.SetIgnoreList(l)
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

	// The Yarn cache under ~/Library/Caches is already a user-cache entry.
	found, err := f.collect(ctx,
		scantask.Walk{Root: filepath.Join(home, ".npm"), Description: "npm cache", Safety: config.Safe},
		scantask.Walk{Root: filepath.Join(home, ".cache", "yarn"), Description: "Yarn cache", Safety: config.Safe},
		scantask.Walk{Root: filepath.Join(home, ".pnpm-store"), Description: "pnpm store", Safety: config.Safe},
	)
	if err != nil {
		return nil, err
	}
	targets = append(targets, found...)

	if cfg.CleanLevel.AllowsSafety(config.Moderate) {
		nodeModulesTargets := f.scanNodeModules(ctx)
		targets = append(targets, nodeModulesTargets...)
	}

	targets = append(targets, f.scanProjectFiles(ctx, cfg)...)
	targets = append(targets, f.scanNestedCache(ctx, ".cache/webpack")...)
	targets = append(targets, f.scanNestedCache(ctx, ".cache/turbo")...)

	return targets, nil
}

func (f *FrontendCleaner) Clean(ctx context.Context, targets []CleanTarget, dryRun bool) ([]CleanResult, error) {
	return CleanTargets(ctx, targets, dryRun, f.progress, f.ignore)
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

// scanProjectFiles finds build outputs and caches in one walk.
// dist, build, and out are Moderate, and only when a project file sits beside them.
func (f *FrontendCleaner) scanProjectFiles(ctx context.Context, cfg *config.Config) []CleanTarget {
	patterns := []string{
		"dist", "build", "out",
		".next", ".vite", ".parcel-cache",
		"coverage", ".nyc_output", ".eslintcache", "storybook-static",
		"npm-debug.log*", "yarn-error.log*", "yarn-debug.log*",
	}
	moderate := cfg.CleanLevel.AllowsSafety(config.Moderate)
	var targets []CleanTarget
	for result := range f.scanner.FindByPatterns(ctx, patterns) {
		if result.Err != nil {
			continue
		}
		safety := config.Safe
		switch result.Pattern {
		case "dist", "out":
			if !moderate || !siblingExists(result.Path, "package.json") {
				continue
			}
			safety = config.Moderate
		case "build":
			if !moderate || !siblingExists(result.Path, "package.json", "build.gradle", "build.gradle.kts", "Package.swift", "CMakeLists.txt") {
				continue
			}
			safety = config.Moderate
		}
		targets = append(targets, CleanTarget{
			Path:        result.Path,
			Description: f.getDescriptionForPattern(result.Pattern),
			SizeBytes:   result.Size,
			Safety:      safety,
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
		"dist":             "Build output (dist)",
		"build":            "Build output (build)",
		"out":              "Build output (out)",
		".next":            "Next.js build cache",
		".vite":            "Vite cache",
		".parcel-cache":    "Parcel cache",
		"coverage":         "Test coverage reports",
		".nyc_output":      "NYC coverage output",
		".eslintcache":     "ESLint cache",
		"storybook-static": "Storybook static build",
		"npm-debug.log*":   "npm debug logs",
		"yarn-error.log*":  "Yarn error logs",
		"yarn-debug.log*":  "Yarn debug logs",
	}

	if desc, ok := descriptions[pattern]; ok {
		return desc
	}

	return pattern
}
