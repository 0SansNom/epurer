package cleaner

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/0SansNom/epurer/internal/config"
	"github.com/0SansNom/epurer/internal/scantask"
	"github.com/0SansNom/epurer/pkg/utils"
)

// systemCleaner is the shared shell of a system-family cleaner: a name,
// the system domain, a detect that always applies, and a file delete.
// A family overrides Detect or Clean only when its behavior differs.
type systemCleaner struct {
	fsCleaner
	name string
}

func (s systemCleaner) Name() string        { return s.name }
func (systemCleaner) Domain() config.Domain { return config.DomainSystem }
func (systemCleaner) Detect(context.Context) (bool, error) {
	return true, nil
}

func (s *systemCleaner) Clean(ctx context.Context, targets []CleanTarget, dryRun bool) ([]CleanResult, error) {
	return CleanTargets(ctx, targets, dryRun, s.progress, s.ignore)
}

// fixed keeps a target that did not come from a walk, unless the ignore list covers it.
func (s *systemCleaner) fixed(target CleanTarget) []CleanTarget {
	return filterIgnored([]CleanTarget{target}, s.ignore)
}

func NewTrashCleaner() Cleaner {
	return &trashCleaner{systemCleaner: systemCleaner{name: "Trash"}}
}

func NewCacheCleaner() Cleaner {
	return &cacheCleaner{systemCleaner: systemCleaner{name: "System Caches"}}
}

func NewLogCleaner() Cleaner {
	return &logCleaner{systemCleaner: systemCleaner{name: "System Logs"}}
}

func NewTempFilesCleaner() Cleaner {
	return &tempCleaner{systemCleaner: systemCleaner{name: "Temp Files"}}
}

func NewDNSCacheCleaner() Cleaner {
	return &dnsCleaner{systemCleaner: systemCleaner{name: "DNS Cache"}}
}

func NewHomebrewCleaner() Cleaner {
	return &homebrewCleaner{systemCleaner: systemCleaner{name: "Homebrew Cache"}}
}

func NewXcodeCleaner() Cleaner {
	return &xcodeCleaner{systemCleaner: systemCleaner{name: "Xcode DerivedData"}}
}

func NewLaunchpadCleaner() Cleaner {
	return &launchpadCleaner{systemCleaner: systemCleaner{name: "Launchpad Database"}}
}

func NewIOSBackupCleaner() Cleaner {
	return &iosBackupCleaner{systemCleaner: systemCleaner{name: "iOS Backups"}}
}

type trashCleaner struct{ systemCleaner }

func (c *trashCleaner) Scan(ctx context.Context, cfg *config.Config) ([]CleanTarget, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	walks := []scantask.Walk{{
		Root:        filepath.Join(home, ".Trash"),
		Description: "User trash",
		Safety:      config.Safe,
	}}

	matches, err := filepath.Glob("/Volumes/*/.Trashes")
	if err == nil {
		for _, match := range matches {
			walks = append(walks, scantask.Walk{
				Root:        match,
				Description: fmt.Sprintf("External volume trash: %s", filepath.Dir(match)),
				Safety:      config.Safe,
			})
		}
	}

	return c.collect(ctx, walks...)
}

type cacheCleaner struct{ systemCleaner }

func (c *cacheCleaner) Scan(ctx context.Context, cfg *config.Config) ([]CleanTarget, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	targets, err := c.collect(ctx, scantask.Walk{
		Root:        filepath.Join(home, "Library", "Caches"),
		Description: "User caches",
		Safety:      config.Safe,
	})
	if err != nil {
		return nil, err
	}

	// System caches stay one target: cleaning them needs a higher safety level.
	if cfg.CleanLevel.AllowsSafety(config.Moderate) {
		systemCachePath := "/Library/Caches"
		if utils.PathExists(systemCachePath) {
			size, _ := utils.GetDirSize(systemCachePath)
			if size > 0 {
				targets = append(targets, c.fixed(CleanTarget{
					Path:        systemCachePath,
					Description: "System caches",
					SizeBytes:   size,
					Safety:      config.Moderate,
				})...)
			}
		}
	}

	return targets, nil
}

type logCleaner struct{ systemCleaner }

func (c *logCleaner) Scan(ctx context.Context, cfg *config.Config) ([]CleanTarget, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	targets, err := c.collect(ctx, scantask.Walk{
		Root:        filepath.Join(home, "Library", "Logs"),
		Description: "User logs",
		Safety:      config.Safe,
	})
	if err != nil {
		return nil, err
	}

	// System logs stay grouped: cleaning them needs a higher safety level.
	if !cfg.CleanLevel.AllowsSafety(config.Moderate) {
		return targets, nil
	}

	aslPattern := "/private/var/log/asl/*.asl"
	matches, err := filepath.Glob(aslPattern)
	if err == nil && len(matches) > 0 {
		var totalSize int64
		for _, match := range matches {
			if info, err := os.Stat(match); err == nil {
				totalSize += info.Size()
			}
		}
		if totalSize > 0 {
			targets = append(targets, c.fixed(CleanTarget{
				Path:        "/private/var/log/asl",
				Description: "ASL log files",
				SizeBytes:   totalSize,
				Safety:      config.Moderate,
			})...)
		}
	}

	logPattern := "/private/var/log/*.log"
	matches, err = filepath.Glob(logPattern)
	if err == nil && len(matches) > 0 {
		var totalSize int64
		for _, match := range matches {
			if info, err := os.Stat(match); err == nil {
				totalSize += info.Size()
			}
		}
		if totalSize > 0 {
			targets = append(targets, c.fixed(CleanTarget{
				Path:        "/private/var/log",
				Description: "System log files",
				SizeBytes:   totalSize,
				Safety:      config.Moderate,
			})...)
		}
	}

	return targets, nil
}

type tempCleaner struct{ systemCleaner }

func (c *tempCleaner) Scan(ctx context.Context, cfg *config.Config) ([]CleanTarget, error) {
	// /tmp holds in-progress work. Offer it only at aggressive, and do not preselect it.
	if !cfg.CleanLevel.AllowsSafety(config.Dangerous) {
		return nil, nil
	}
	var walks []scantask.Walk
	for _, path := range []string{"/private/var/tmp", "/private/tmp"} {
		walks = append(walks, scantask.Walk{
			Root:        path,
			Description: fmt.Sprintf("Temporary files in %s", path),
			Safety:      config.Dangerous,
		})
	}
	return c.collect(ctx, walks...)
}

type dnsCleaner struct{ systemCleaner }

func (c *dnsCleaner) Scan(ctx context.Context, cfg *config.Config) ([]CleanTarget, error) {
	return c.fixed(CleanTarget{
		Path:        "system:dns_cache",
		Description: "DNS cache (via dscacheutil)",
		SizeBytes:   0,
		Safety:      config.Safe,
	}), nil
}

func (c *dnsCleaner) Clean(ctx context.Context, targets []CleanTarget, dryRun bool) ([]CleanResult, error) {
	results := make([]CleanResult, 0, len(targets))
	for _, target := range targets {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}

		var err error
		if target.Path != "system:dns_cache" {
			err = fmt.Errorf("%w: unexpected DNS target %q", utils.ErrNotRemovable, target.Path)
		} else {
			err = flushDNSCache(ctx, dryRun)
		}
		results = append(results, CleanResult{
			Target:  target,
			Success: err == nil,
			Error:   err,
		})
		noteCleanProgress(c.progress, target.Path, len(results), 0)
	}
	return results, nil
}

func flushDNSCache(ctx context.Context, dryRun bool) error {
	if dryRun {
		return nil
	}

	cmd := exec.CommandContext(ctx, "dscacheutil", "-flushcache")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to flush DNS cache: %w", err)
	}

	cmd = exec.CommandContext(ctx, "killall", "-HUP", "mDNSResponder")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to signal mDNSResponder: %w", err)
	}
	return nil
}

type homebrewCleaner struct{ systemCleaner }

func (c *homebrewCleaner) Detect(context.Context) (bool, error) {
	return utils.CommandExists("brew"), nil
}

func (c *homebrewCleaner) Scan(ctx context.Context, cfg *config.Config) ([]CleanTarget, error) {
	cachePath, err := homebrewCachePath()
	if err != nil {
		return nil, err
	}

	if utils.PathExists(cachePath) {
		size, _ := utils.GetDirSize(cachePath)
		if size > 0 {
			return c.fixed(CleanTarget{
				Path:        cachePath,
				Description: "Homebrew cache",
				SizeBytes:   size,
				Safety:      config.Safe,
			}), nil
		}
	}

	return []CleanTarget{}, nil
}

func (c *homebrewCleaner) Clean(ctx context.Context, targets []CleanTarget, dryRun bool) ([]CleanResult, error) {
	results := make([]CleanResult, 0, len(targets))
	for _, target := range targets {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}

		err := c.cleanupHomebrewTarget(ctx, target.Path, dryRun)
		result := CleanResult{
			Target:  target,
			Success: err == nil,
			Error:   err,
		}
		if err == nil {
			result.BytesFreed = target.SizeBytes
		}
		results = append(results, result)
		var freed int64
		for _, done := range results {
			if done.Success {
				freed += done.BytesFreed
			}
		}
		noteCleanProgress(c.progress, target.Path, len(results), freed)
	}
	return results, nil
}

func (c *homebrewCleaner) cleanupHomebrewTarget(ctx context.Context, path string, dryRun bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if c.ignore != nil && c.ignore.IsIgnored(path) {
		return fmt.Errorf("%w: on the ignore list", utils.ErrNotRemovable)
	}
	cachePath, err := homebrewCachePath()
	if err != nil {
		return err
	}
	if filepath.Clean(path) != cachePath {
		return fmt.Errorf("%w: unexpected Homebrew target %q", utils.ErrNotRemovable, path)
	}
	return removeChildren(cachePath, dryRun)
}

func homebrewCachePath() (string, error) {
	cmd := exec.Command("brew", "--cache")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to resolve brew --cache: %w", err)
	}
	return filepath.Clean(strings.TrimSpace(string(output))), nil
}

// removeChildren deletes the entries inside path and leaves path itself in place.
// The scan measured this directory, so only its contents are removed.
func removeChildren(path string, dryRun bool) error {
	entries, err := os.ReadDir(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, ent := range entries {
		if err := utils.SafeRemove(filepath.Join(path, ent.Name()), dryRun); err != nil {
			return err
		}
	}
	return nil
}

type xcodeCleaner struct{ systemCleaner }

func (c *xcodeCleaner) Detect(context.Context) (bool, error) {
	return utils.PathExists("/Applications/Xcode.app"), nil
}

func (c *xcodeCleaner) Scan(ctx context.Context, cfg *config.Config) ([]CleanTarget, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	xcode := filepath.Join(home, "Library", "Developer", "Xcode")
	developer := filepath.Join(home, "Library", "Developer")
	walks := []scantask.Walk{
		{Root: filepath.Join(xcode, "DerivedData"), Description: "Xcode DerivedData", Safety: config.Safe},
		{Root: filepath.Join(xcode, "iOS DeviceSupport"), Description: "iOS Device Support", Safety: config.Safe},
		{Root: filepath.Join(xcode, "watchOS DeviceSupport"), Description: "watchOS Device Support", Safety: config.Safe},
		{Root: filepath.Join(xcode, "tvOS DeviceSupport"), Description: "tvOS Device Support", Safety: config.Safe},
		{Root: filepath.Join(xcode, "macOS DeviceSupport"), Description: "macOS Device Support", Safety: config.Safe},
		{Root: filepath.Join(xcode, "iOS Device Logs"), Description: "iOS Device Logs", Safety: config.Safe},
	}

	sim := filepath.Join(developer, "CoreSimulator")
	if cfg.CleanLevel.AllowsSafety(config.Moderate) {
		walks = append(walks,
			scantask.Walk{Root: filepath.Join(sim, "Caches"), Description: "iOS Simulator caches", Safety: config.Moderate},
		)
	}
	// Archives and simulator devices are not rebuilt by the next compile.
	if cfg.CleanLevel.AllowsSafety(config.Dangerous) {
		walks = append(walks,
			scantask.Walk{Root: filepath.Join(xcode, "Archives"), Description: "Xcode Archives", Safety: config.Dangerous},
			scantask.Walk{Root: filepath.Join(sim, "Devices"), Description: "iOS Simulator devices", Safety: config.Dangerous},
		)
	}

	return c.collect(ctx, walks...)
}

type launchpadCleaner struct{ systemCleaner }

func (c *launchpadCleaner) Scan(ctx context.Context, cfg *config.Config) ([]CleanTarget, error) {
	if !cfg.CleanLevel.AllowsSafety(config.Dangerous) {
		return nil, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	dbPath := filepath.Join(home, "Library", "Application Support", "Dock")
	if utils.PathExists(dbPath) {
		return c.fixed(CleanTarget{
			Path:        dbPath,
			Description: "Launchpad database (will be rebuilt)",
			SizeBytes:   0,
			Safety:      config.Dangerous,
		}), nil
	}

	return []CleanTarget{}, nil
}

type iosBackupCleaner struct{ systemCleaner }

func (c *iosBackupCleaner) Scan(ctx context.Context, cfg *config.Config) ([]CleanTarget, error) {
	if !cfg.CleanLevel.AllowsSafety(config.Dangerous) {
		return []CleanTarget{}, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	return c.collect(ctx, scantask.Walk{
		Root:        filepath.Join(home, "Library", "Application Support", "MobileSync", "Backup"),
		Description: "iOS device backups (DANGEROUS - may contain important data)",
		Safety:      config.Dangerous,
	})
}
