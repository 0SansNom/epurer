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

// MobileCleaner handles mobile development cleanup (iOS, Android, Flutter)
type MobileCleaner struct {
	scanner *scanner.Scanner
	fsCleaner
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

// SetIgnoreList makes this cleaner respect a persistent ignore list.
func (m *MobileCleaner) SetIgnoreList(l *ignorelist.List) {
	m.fsCleaner.SetIgnoreList(l)
	m.scanner.SetIgnoreList(l)
}

func (m *MobileCleaner) Name() string {
	return "Mobile"
}

func (m *MobileCleaner) Domain() config.Domain {
	return config.DomainMobile
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

	// Xcode DerivedData, archives, device support, and simulator data are
	// walked by the Xcode cleaner. Caches such as com.apple.dt.Xcode and
	// CocoaPods are top-level entries of ~/Library/Caches.

	// Android build folders (Safe - rebuilt)
	androidBuildTargets := m.scanAndroidBuildFolders(ctx)
	targets = append(targets, androidBuildTargets...)

	walks := []scantask.Walk{
		{Root: filepath.Join(home, "Library", "Android", "sdk", "build-cache"), Description: "Android SDK build cache", Safety: config.Safe},
	}
	if cfg.CleanLevel.AllowsSafety(config.Moderate) {
		walks = append(walks, scantask.Walk{
			Root:        filepath.Join(home, ".android", "avd"),
			Description: "Android Virtual Devices",
			Safety:      config.Moderate,
		})
	}
	found, err := m.collect(ctx, walks...)
	if err != nil {
		return nil, err
	}
	targets = append(targets, found...)

	// .dart_tool (Safe - rebuilt)
	dartToolTargets := m.scanDartTool(ctx)
	targets = append(targets, dartToolTargets...)

	// Flutter build (Safe)
	flutterBuildTargets := m.scanFlutterBuild(ctx)
	targets = append(targets, flutterBuildTargets...)

	return targets, nil
}

func (m *MobileCleaner) Clean(ctx context.Context, targets []CleanTarget, dryRun bool) ([]CleanResult, error) {
	return CleanTargets(ctx, targets, dryRun, m.progress, m.ignore)
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
