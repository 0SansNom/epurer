package cleaner

import (
	"context"
	"os"
	"path/filepath"

	"github.com/0SansNom/epurer/internal/config"
	"github.com/0SansNom/epurer/internal/scanner"
	"github.com/0SansNom/epurer/pkg/utils"
)

// MobileCleaner handles mobile development cleanup (iOS, Android, Flutter)
type MobileCleaner struct {
	scanner *scanner.Scanner
}

// NewMobileCleaner creates a new MobileCleaner
func NewMobileCleaner() (Cleaner, error) {
	s, err := scanner.NewScanner()
	if err != nil {
		return nil, err
	}

	return &MobileCleaner{
		scanner: s,
	}, nil
}

func (m *MobileCleaner) Name() string {
	return "Mobile"
}

func (m *MobileCleaner) Domain() config.Domain {
	return config.DomainFrontend // TODO: Add DomainMobile to config
}

func (m *MobileCleaner) Detect(ctx context.Context) (bool, error) {
	// Check if Xcode, Android Studio, or Flutter are present
	hasXcode := utils.PathExists("/Applications/Xcode.app")
	hasAndroid := utils.CommandExists("adb") || utils.PathExists(filepath.Join(os.Getenv("HOME"), "Library/Android"))
	hasFlutter := utils.CommandExists("flutter")

	return hasXcode || hasAndroid || hasFlutter, nil
}

func (m *MobileCleaner) Scan(ctx context.Context, cfg *config.Config) ([]CleanTarget, error) {
	targets := []CleanTarget{}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	// === iOS / Xcode (VeryHigh impact - can be 50-100 GB) ===

	// DerivedData (Safe - always rebuilt)
	derivedDataPath := filepath.Join(home, "Library", "Developer", "Xcode", "DerivedData")
	if utils.PathExists(derivedDataPath) {
		size, _ := utils.GetDirSize(derivedDataPath)
		if size > 0 {
			targets = append(targets, CleanTarget{
				Path:        derivedDataPath,
				Description: "Xcode DerivedData (rebuilds automatically)",
				SizeBytes:   size,
				Safety:      config.Safe,
			})
		}
	}

	// Archives (Moderate - old app versions)
	if cfg.CleanLevel.AllowsSafety(config.Moderate) {
		archivesPath := filepath.Join(home, "Library", "Developer", "Xcode", "Archives")
		if utils.PathExists(archivesPath) {
			size, _ := utils.GetDirSize(archivesPath)
			if size > 0 {
				targets = append(targets, CleanTarget{
					Path:        archivesPath,
					Description: "Xcode Archives (old app versions)",
					SizeBytes:   size,
					Safety:      config.Moderate,
				})
			}
		}
	}

	// Module cache (Safe)
	moduleCachePath := filepath.Join(home, "Library", "Developer", "Xcode", "DerivedData", "ModuleCache.noindex")
	if utils.PathExists(moduleCachePath) {
		size, _ := utils.GetDirSize(moduleCachePath)
		if size > 0 {
			targets = append(targets, CleanTarget{
				Path:        moduleCachePath,
				Description: "Xcode Module Cache",
				SizeBytes:   size,
				Safety:      config.Safe,
			})
		}
	}

	// iOS Device Support (Safe - re-downloaded as needed)
	deviceSupportPath := filepath.Join(home, "Library", "Developer", "Xcode", "iOS DeviceSupport")
	if utils.PathExists(deviceSupportPath) {
		size, _ := utils.GetDirSize(deviceSupportPath)
		if size > 0 {
			targets = append(targets, CleanTarget{
				Path:        deviceSupportPath,
				Description: "iOS Device Support symbols",
				SizeBytes:   size,
				Safety:      config.Safe,
			})
		}
	}

	// watchOS Device Support
	watchDeviceSupportPath := filepath.Join(home, "Library", "Developer", "Xcode", "watchOS DeviceSupport")
	if utils.PathExists(watchDeviceSupportPath) {
		size, _ := utils.GetDirSize(watchDeviceSupportPath)
		if size > 0 {
			targets = append(targets, CleanTarget{
				Path:        watchDeviceSupportPath,
				Description: "watchOS Device Support symbols",
				SizeBytes:   size,
				Safety:      config.Safe,
			})
		}
	}

	// tvOS Device Support
	tvDeviceSupportPath := filepath.Join(home, "Library", "Developer", "Xcode", "tvOS DeviceSupport")
	if utils.PathExists(tvDeviceSupportPath) {
		size, _ := utils.GetDirSize(tvDeviceSupportPath)
		if size > 0 {
			targets = append(targets, CleanTarget{
				Path:        tvDeviceSupportPath,
				Description: "tvOS Device Support symbols",
				SizeBytes:   size,
				Safety:      config.Safe,
			})
		}
	}

	// CoreSimulator Caches (Moderate)
	if cfg.CleanLevel.AllowsSafety(config.Moderate) {
		simCachePath := filepath.Join(home, "Library", "Developer", "CoreSimulator", "Caches")
		if utils.PathExists(simCachePath) {
			size, _ := utils.GetDirSize(simCachePath)
			if size > 0 {
				targets = append(targets, CleanTarget{
					Path:        simCachePath,
					Description: "iOS Simulator caches",
					SizeBytes:   size,
					Safety:      config.Moderate,
				})
			}
		}

		// Old simulator devices (can be huge)
		devicesPath := filepath.Join(home, "Library", "Developer", "CoreSimulator", "Devices")
		if utils.PathExists(devicesPath) {
			size, _ := utils.GetDirSize(devicesPath)
			if size > 0 {
				targets = append(targets, CleanTarget{
					Path:        devicesPath,
					Description: "iOS Simulator devices (can be recreated)",
					SizeBytes:   size,
					Safety:      config.Moderate,
				})
			}
		}
	}

	// Xcode cache
	xcodeCachePath := filepath.Join(home, "Library", "Caches", "com.apple.dt.Xcode")
	if utils.PathExists(xcodeCachePath) {
		size, _ := utils.GetDirSize(xcodeCachePath)
		if size > 0 {
			targets = append(targets, CleanTarget{
				Path:        xcodeCachePath,
				Description: "Xcode general cache",
				SizeBytes:   size,
				Safety:      config.Safe,
			})
		}
	}

	// === Android ===

	// Android build folders (Safe - rebuilt)
	androidBuildTargets := m.scanAndroidBuildFolders(ctx)
	targets = append(targets, androidBuildTargets...)

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

	// Android SDK build cache (Safe)
	androidSDKPath := filepath.Join(home, "Library", "Android", "sdk")
	if utils.PathExists(androidSDKPath) {
		buildCachePath := filepath.Join(androidSDKPath, "build-cache")
		if utils.PathExists(buildCachePath) {
			size, _ := utils.GetDirSize(buildCachePath)
			if size > 0 {
				targets = append(targets, CleanTarget{
					Path:        buildCachePath,
					Description: "Android SDK build cache",
					SizeBytes:   size,
					Safety:      config.Safe,
				})
			}
		}
	}

	// AVD (Android Virtual Devices) - Moderate
	if cfg.CleanLevel.AllowsSafety(config.Moderate) {
		avdPath := filepath.Join(home, ".android", "avd")
		if utils.PathExists(avdPath) {
			size, _ := utils.GetDirSize(avdPath)
			if size > 0 {
				targets = append(targets, CleanTarget{
					Path:        avdPath,
					Description: "Android Virtual Devices",
					SizeBytes:   size,
					Safety:      config.Moderate,
				})
			}
		}
	}

	// === CocoaPods ===

	// CocoaPods cache (Safe)
	podsCachePath := filepath.Join(home, "Library", "Caches", "CocoaPods")
	if utils.PathExists(podsCachePath) {
		size, _ := utils.GetDirSize(podsCachePath)
		if size > 0 {
			targets = append(targets, CleanTarget{
				Path:        podsCachePath,
				Description: "CocoaPods cache",
				SizeBytes:   size,
				Safety:      config.Safe,
			})
		}
	}

	// === Flutter ===

	// .dart_tool (Safe - rebuilt)
	dartToolTargets := m.scanDartTool(ctx)
	targets = append(targets, dartToolTargets...)

	// Flutter build (Safe)
	flutterBuildTargets := m.scanFlutterBuild(ctx)
	targets = append(targets, flutterBuildTargets...)

	return targets, nil
}

func (m *MobileCleaner) Clean(ctx context.Context, targets []CleanTarget, dryRun bool) ([]CleanResult, error) {
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

// scanAndroidBuildFolders scans for Android build folders
func (m *MobileCleaner) scanAndroidBuildFolders(ctx context.Context) []CleanTarget {
	targets := []CleanTarget{}

	// Look for app/build or build folders in Android projects
	resultChan := m.scanner.FindByPattern(ctx, "build")
	for result := range resultChan {
		if result.Err != nil {
			continue
		}

		// Check if this is an Android build folder
		// (contains typical Android build artifacts)
		parent := filepath.Dir(result.Path)
		if filepath.Base(parent) == "app" || utils.PathExists(filepath.Join(parent, "gradle.properties")) {
			targets = append(targets, CleanTarget{
				Path:        result.Path,
				Description: "Android build output",
				SizeBytes:   result.Size,
				Safety:      config.Safe,
			})
		}
	}

	return targets
}

// scanDartTool scans for .dart_tool folders (Flutter/Dart)
func (m *MobileCleaner) scanDartTool(ctx context.Context) []CleanTarget {
	targets := []CleanTarget{}

	resultChan := m.scanner.FindByPattern(ctx, ".dart_tool")
	for result := range resultChan {
		if result.Err != nil {
			continue
		}

		targets = append(targets, CleanTarget{
			Path:        result.Path,
			Description: "Flutter/Dart build cache",
			SizeBytes:   result.Size,
			Safety:      config.Safe,
		})
	}

	return targets
}

// scanFlutterBuild scans for Flutter build folders
func (m *MobileCleaner) scanFlutterBuild(ctx context.Context) []CleanTarget {
	targets := []CleanTarget{}

	// Flutter projects have a "build" folder at the root
	resultChan := m.scanner.FindByPattern(ctx, "build")
	for result := range resultChan {
		if result.Err != nil {
			continue
		}

		// Check if this is a Flutter project by looking for pubspec.yaml
		parent := filepath.Dir(result.Path)
		pubspecPath := filepath.Join(parent, "pubspec.yaml")
		if utils.PathExists(pubspecPath) {
			targets = append(targets, CleanTarget{
				Path:        result.Path,
				Description: "Flutter build output",
				SizeBytes:   result.Size,
				Safety:      config.Safe,
			})
		}
	}

	return targets
}
