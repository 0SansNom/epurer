package cleaner

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/0SansNom/epurer/internal/config"
	"github.com/0SansNom/epurer/pkg/utils"
)

// SystemCleaner handles system-level cleanup operations
type SystemCleaner struct {
	cleanerType string
}

// System cleaner types
const (
	TypeTrash      = "trash"
	TypeCache      = "cache"
	TypeLogs       = "logs"
	TypeTemp       = "temp"
	TypeDNS        = "dns"
	TypeHomebrew   = "homebrew"
	TypeXcode      = "xcode"
	TypeLaunchpad  = "launchpad"
	TypeIOSBackups = "ios_backups"
)

// Factory functions for each system cleaner type

func NewTrashCleaner() Cleaner {
	return &SystemCleaner{cleanerType: TypeTrash}
}

func NewCacheCleaner() Cleaner {
	return &SystemCleaner{cleanerType: TypeCache}
}

func NewLogCleaner() Cleaner {
	return &SystemCleaner{cleanerType: TypeLogs}
}

func NewTempFilesCleaner() Cleaner {
	return &SystemCleaner{cleanerType: TypeTemp}
}

func NewDNSCacheCleaner() Cleaner {
	return &SystemCleaner{cleanerType: TypeDNS}
}

func NewHomebrewCleaner() Cleaner {
	return &SystemCleaner{cleanerType: TypeHomebrew}
}

func NewXcodeCleaner() Cleaner {
	return &SystemCleaner{cleanerType: TypeXcode}
}

func NewLaunchpadCleaner() Cleaner {
	return &SystemCleaner{cleanerType: TypeLaunchpad}
}

func NewIOSBackupCleaner() Cleaner {
	return &SystemCleaner{cleanerType: TypeIOSBackups}
}

// Implement Cleaner interface

func (s *SystemCleaner) Name() string {
	switch s.cleanerType {
	case TypeTrash:
		return "Trash"
	case TypeCache:
		return "System Caches"
	case TypeLogs:
		return "System Logs"
	case TypeTemp:
		return "Temp Files"
	case TypeDNS:
		return "DNS Cache"
	case TypeHomebrew:
		return "Homebrew Cache"
	case TypeXcode:
		return "Xcode DerivedData"
	case TypeLaunchpad:
		return "Launchpad Database"
	case TypeIOSBackups:
		return "iOS Backups"
	default:
		return "Unknown"
	}
}

func (s *SystemCleaner) Domain() config.Domain {
	return config.DomainSystem
}

func (s *SystemCleaner) Detect(ctx context.Context) (bool, error) {
	switch s.cleanerType {
	case TypeHomebrew:
		// Only applicable if Homebrew is installed
		return utils.CommandExists("brew"), nil
	case TypeXcode:
		// Only applicable if Xcode is installed
		return utils.PathExists("/Applications/Xcode.app"), nil
	case TypeDNS, TypeTrash, TypeCache, TypeLogs, TypeTemp, TypeLaunchpad, TypeIOSBackups:
		// Always applicable on macOS
		return true, nil
	default:
		return false, fmt.Errorf("unknown cleaner type: %s", s.cleanerType)
	}
}

func (s *SystemCleaner) Scan(ctx context.Context, cfg *config.Config) ([]CleanTarget, error) {
	switch s.cleanerType {
	case TypeTrash:
		return s.scanTrash()
	case TypeCache:
		return s.scanCaches(cfg)
	case TypeLogs:
		return s.scanLogs(cfg)
	case TypeTemp:
		return s.scanTemp()
	case TypeDNS:
		return s.scanDNS()
	case TypeHomebrew:
		return s.scanHomebrew()
	case TypeXcode:
		return s.scanXcode()
	case TypeLaunchpad:
		return s.scanLaunchpad()
	case TypeIOSBackups:
		return s.scanIOSBackups(cfg)
	default:
		return nil, fmt.Errorf("unknown cleaner type: %s", s.cleanerType)
	}
}

func (s *SystemCleaner) Clean(ctx context.Context, targets []CleanTarget, dryRun bool) ([]CleanResult, error) {
	results := make([]CleanResult, 0, len(targets))

	for _, target := range targets {
		var result CleanResult
		result.Target = target

		// Special handling for DNS cache (uses command)
		if s.cleanerType == TypeDNS {
			err := s.cleanDNSCache(dryRun)
			result.Success = err == nil
			result.Error = err
			result.BytesFreed = 0 // DNS cache doesn't have measurable size
		} else if s.cleanerType == TypeHomebrew {
			// Homebrew uses its own cleanup command
			err := s.cleanHomebrew(dryRun)
			result.Success = err == nil
			result.Error = err
			result.BytesFreed = target.SizeBytes // Estimate
		} else {
			// Standard file/directory removal
			err := utils.SafeRemove(target.Path, dryRun)
			result.Success = err == nil
			result.Error = err
			if result.Success {
				result.BytesFreed = target.SizeBytes
			}
		}

		results = append(results, result)
	}

	return results, nil
}

// Private scan methods

func (s *SystemCleaner) scanTrash() ([]CleanTarget, error) {
	targets := []CleanTarget{}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	// User trash
	trashPath := filepath.Join(home, ".Trash")
	if utils.PathExists(trashPath) {
		size, _ := utils.GetDirSize(trashPath)
		if size > 0 {
			targets = append(targets, CleanTarget{
				Path:        trashPath,
				Description: "User trash",
				SizeBytes:   size,
				Safety:      config.Safe,
			})
		}
	}

	// External volumes trash
	volumesPattern := "/Volumes/*/.Trashes"
	matches, err := filepath.Glob(volumesPattern)
	if err == nil {
		for _, match := range matches {
			if utils.PathExists(match) {
				size, _ := utils.GetDirSize(match)
				if size > 0 {
					targets = append(targets, CleanTarget{
						Path:        match,
						Description: fmt.Sprintf("External volume trash: %s", filepath.Dir(match)),
						SizeBytes:   size,
						Safety:      config.Safe,
					})
				}
			}
		}
	}

	return targets, nil
}

func (s *SystemCleaner) scanCaches(cfg *config.Config) ([]CleanTarget, error) {
	targets := []CleanTarget{}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	// User caches (always safe)
	userCachePath := filepath.Join(home, "Library", "Caches")
	if utils.PathExists(userCachePath) {
		size, _ := utils.GetDirSize(userCachePath)
		if size > 0 {
			targets = append(targets, CleanTarget{
				Path:        userCachePath,
				Description: "User caches",
				SizeBytes:   size,
				Safety:      config.Safe,
			})
		}
	}

	// System caches (moderate - requires sudo)
	if cfg.CleanLevel.AllowsSafety(config.Moderate) {
		systemCachePath := "/Library/Caches"
		if utils.PathExists(systemCachePath) {
			size, _ := utils.GetDirSize(systemCachePath)
			if size > 0 {
				targets = append(targets, CleanTarget{
					Path:        systemCachePath,
					Description: "System caches",
					SizeBytes:   size,
					Safety:      config.Moderate,
				})
			}
		}
	}

	return targets, nil
}

func (s *SystemCleaner) scanLogs(cfg *config.Config) ([]CleanTarget, error) {
	targets := []CleanTarget{}

	// Only scan logs if moderate or higher
	if !cfg.CleanLevel.AllowsSafety(config.Moderate) {
		return targets, nil
	}

	// ASL logs
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
			targets = append(targets, CleanTarget{
				Path:        "/private/var/log/asl",
				Description: "ASL log files",
				SizeBytes:   totalSize,
				Safety:      config.Moderate,
			})
		}
	}

	// System logs
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
			targets = append(targets, CleanTarget{
				Path:        "/private/var/log",
				Description: "System log files",
				SizeBytes:   totalSize,
				Safety:      config.Moderate,
			})
		}
	}

	return targets, nil
}

func (s *SystemCleaner) scanTemp() ([]CleanTarget, error) {
	targets := []CleanTarget{}

	tempPaths := []string{
		"/private/var/tmp",
		"/private/tmp",
	}

	for _, path := range tempPaths {
		if utils.PathExists(path) {
			size, _ := utils.GetDirSize(path)
			if size > 0 {
				targets = append(targets, CleanTarget{
					Path:        path,
					Description: fmt.Sprintf("Temporary files in %s", path),
					SizeBytes:   size,
					Safety:      config.Safe,
				})
			}
		}
	}

	return targets, nil
}

func (s *SystemCleaner) scanDNS() ([]CleanTarget, error) {
	// DNS cache doesn't have a measurable size, but we report it as cleanable
	return []CleanTarget{
		{
			Path:        "system:dns_cache",
			Description: "DNS cache (via dscacheutil)",
			SizeBytes:   0,
			Safety:      config.Safe,
		},
	}, nil
}

func (s *SystemCleaner) scanHomebrew() ([]CleanTarget, error) {
	// Homebrew cache location
	cmd := exec.Command("brew", "--cache")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	cachePath := string(output)
	cachePath = filepath.Clean(cachePath[:len(cachePath)-1]) // Remove newline

	if utils.PathExists(cachePath) {
		size, _ := utils.GetDirSize(cachePath)
		if size > 0 {
			return []CleanTarget{
				{
					Path:        cachePath,
					Description: "Homebrew cache",
					SizeBytes:   size,
					Safety:      config.Safe,
				},
			}, nil
		}
	}

	return []CleanTarget{}, nil
}

func (s *SystemCleaner) scanXcode() ([]CleanTarget, error) {
	targets := []CleanTarget{}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	// DerivedData
	derivedDataPath := filepath.Join(home, "Library", "Developer", "Xcode", "DerivedData")
	if utils.PathExists(derivedDataPath) {
		size, _ := utils.GetDirSize(derivedDataPath)
		if size > 0 {
			targets = append(targets, CleanTarget{
				Path:        derivedDataPath,
				Description: "Xcode DerivedData",
				SizeBytes:   size,
				Safety:      config.Safe,
			})
		}
	}

	// Archives (optional, only if size is significant)
	archivesPath := filepath.Join(home, "Library", "Developer", "Xcode", "Archives")
	if utils.PathExists(archivesPath) {
		size, _ := utils.GetDirSize(archivesPath)
		if size > 0 {
			targets = append(targets, CleanTarget{
				Path:        archivesPath,
				Description: "Xcode Archives",
				SizeBytes:   size,
				Safety:      config.Moderate,
			})
		}
	}

	return targets, nil
}

func (s *SystemCleaner) scanLaunchpad() ([]CleanTarget, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	dbPath := filepath.Join(home, "Library", "Application Support", "Dock")
	if utils.PathExists(dbPath) {
		// Launchpad DB is a special case - we rebuild it, not delete it
		return []CleanTarget{
			{
				Path:        dbPath,
				Description: "Launchpad database (will be rebuilt)",
				SizeBytes:   0, // Negligible size
				Safety:      config.Dangerous,
			},
		}, nil
	}

	return []CleanTarget{}, nil
}

func (s *SystemCleaner) scanIOSBackups(cfg *config.Config) ([]CleanTarget, error) {
	// Only show iOS backups in aggressive mode
	if !cfg.CleanLevel.AllowsSafety(config.Dangerous) {
		return []CleanTarget{}, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	backupPath := filepath.Join(home, "Library", "Application Support", "MobileSync", "Backup")
	if utils.PathExists(backupPath) {
		size, _ := utils.GetDirSize(backupPath)
		if size > 0 {
			return []CleanTarget{
				{
					Path:        backupPath,
					Description: "iOS device backups (DANGEROUS - may contain important data)",
					SizeBytes:   size,
					Safety:      config.Dangerous,
				},
			}, nil
		}
	}

	return []CleanTarget{}, nil
}

// Private clean methods

func (s *SystemCleaner) cleanDNSCache(dryRun bool) error {
	if dryRun {
		return nil
	}

	// Flush DNS cache
	cmd := exec.Command("dscacheutil", "-flushcache")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to flush DNS cache: %w", err)
	}

	// Kill mDNSResponder (requires sudo for full effect)
	cmd = exec.Command("killall", "-HUP", "mDNSResponder")
	cmd.Run() // Ignore error if we don't have sudo

	return nil
}

func (s *SystemCleaner) cleanHomebrew(dryRun bool) error {
	if dryRun {
		return nil
	}

	// Run brew cleanup
	cmd := exec.Command("brew", "cleanup", "--prune=all")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run brew cleanup: %w", err)
	}

	return nil
}
