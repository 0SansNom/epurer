package cleaner

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/0SansNom/epurer/internal/config"
	"github.com/0SansNom/epurer/internal/ignorelist"
)

func TestScanCaches_SplitsUserCacheEntries(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	caches := filepath.Join(home, "Library", "Caches")
	if err := os.MkdirAll(filepath.Join(caches, "keep"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(caches, "keep", "a.bin"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	skipped := filepath.Join(caches, "skip")
	if err := os.MkdirAll(skipped, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skipped, "b.bin"), []byte("xyz"), 0o644); err != nil {
		t.Fatal(err)
	}

	cacheCleaner := NewCacheCleaner()
	cacheCleaner.(IgnoreAware).SetIgnoreList(&ignorelist.List{Paths: []string{skipped}})

	cfg := config.NewDefaultConfig()
	cfg.CleanLevel = config.Conservative

	targets, err := cacheCleaner.Scan(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if len(targets) != 1 {
		t.Fatalf("Scan() = %+v, want the keep entry only", targets)
	}
	if targets[0].Path != filepath.Join(caches, "keep") {
		t.Errorf("path = %s", targets[0].Path)
	}
	if targets[0].Description != "User caches: keep" {
		t.Errorf("description = %q", targets[0].Description)
	}
	if targets[0].SizeBytes != 5 || targets[0].Safety != config.Safe {
		t.Errorf("target = %+v", targets[0])
	}
}

func TestScanLogs_SplitsUserLogEntries(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	logs := filepath.Join(home, "Library", "Logs")
	if err := os.WriteFile(mustDir(t, filepath.Join(logs, "App")), []byte("log"), 0o644); err != nil {
		t.Fatal(err)
	}

	logCleaner := NewLogCleaner()
	cfg := config.NewDefaultConfig()
	cfg.CleanLevel = config.Conservative

	targets, err := logCleaner.Scan(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if len(targets) != 1 {
		t.Fatalf("Scan() = %+v, want the user log file only", targets)
	}
	if targets[0].Path != filepath.Join(logs, "App") {
		t.Errorf("path = %s", targets[0].Path)
	}
	if targets[0].Description != "User logs: App" || targets[0].Safety != config.Safe {
		t.Errorf("target = %+v", targets[0])
	}
}

func TestScanXcode_SplitsDerivedDataAndArchives(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	xcode := filepath.Join(home, "Library", "Developer", "Xcode")
	if err := os.WriteFile(mustDir(t, filepath.Join(xcode, "DerivedData", "Proj")), []byte("dd"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mustDir(t, filepath.Join(xcode, "Archives", "App.xcarchive")), []byte("arc"), 0o644); err != nil {
		t.Fatal(err)
	}

	xcodeCleaner := NewXcodeCleaner()
	targets, err := xcodeCleaner.Scan(context.Background(), config.NewDefaultConfig())
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if len(targets) != 2 {
		t.Fatalf("Scan() = %+v, want derived data and archives", targets)
	}
	if targets[0].Description != "Xcode DerivedData: Proj" || targets[0].Safety != config.Safe {
		t.Errorf("derived = %+v", targets[0])
	}
	if targets[1].Description != "Xcode Archives: App.xcarchive" || targets[1].Safety != config.Moderate {
		t.Errorf("archives = %+v", targets[1])
	}
}

func TestScanXcode_DeviceSupportAlwaysAndSimulatorsWhenModerate(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	dev := filepath.Join(home, "Library", "Developer")
	if err := os.WriteFile(mustDir(t, filepath.Join(dev, "Xcode", "iOS DeviceSupport", "16.4", "symbols")), []byte("sym"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mustDir(t, filepath.Join(dev, "CoreSimulator", "Devices", "UDID", "device.plist")), []byte("sim"), 0o644); err != nil {
		t.Fatal(err)
	}

	xcodeCleaner := NewXcodeCleaner()
	cfg := config.NewDefaultConfig()
	cfg.CleanLevel = config.Conservative

	targets, err := xcodeCleaner.Scan(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if len(targets) != 1 || targets[0].Description != "iOS Device Support: 16.4" || targets[0].Safety != config.Safe {
		t.Fatalf("conservative = %+v", targets)
	}

	cfg.CleanLevel = config.Standard
	targets, err = xcodeCleaner.Scan(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if len(targets) != 2 {
		t.Fatalf("standard = %+v, want device support and one simulator", targets)
	}
	if targets[1].Description != "iOS Simulator devices: UDID" || targets[1].Safety != config.Moderate {
		t.Errorf("simulator = %+v", targets[1])
	}
}

func TestMobileCleaner_LeavesXcodeTreesToXcodeCleaner(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	xcode := filepath.Join(home, "Library", "Developer", "Xcode")
	if err := os.WriteFile(mustDir(t, filepath.Join(xcode, "DerivedData", "Proj")), []byte("dd"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mustDir(t, filepath.Join(home, "Library", "Caches", "CocoaPods", "Pods")), []byte("pod"), 0o644); err != nil {
		t.Fatal(err)
	}

	mobile, err := NewMobileCleaner()
	if err != nil {
		t.Fatal(err)
	}
	targets, err := mobile.Scan(context.Background(), config.NewDefaultConfig())
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if len(targets) != 0 {
		t.Fatalf("Scan() = %+v, want no Xcode or CocoaPods targets", targets)
	}
}

func TestMobileCleaner_SplitsBuildCacheAndAVD(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := os.WriteFile(mustDir(t, filepath.Join(home, ".gradle", "caches", "modules-2", "a.jar")), []byte("jar"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mustDir(t, filepath.Join(home, "Library", "Android", "sdk", "build-cache", "hash", "out")), []byte("apk"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mustDir(t, filepath.Join(home, ".android", "avd", "Pixel.avd", "config.ini")), []byte("hw"), 0o644); err != nil {
		t.Fatal(err)
	}

	mobile, err := NewMobileCleaner()
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.NewDefaultConfig()
	cfg.CleanLevel = config.Conservative

	targets, err := mobile.Scan(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if len(targets) != 1 || targets[0].Description != "Android SDK build cache: hash" || targets[0].Safety != config.Safe {
		t.Fatalf("conservative = %+v, want the SDK build cache only", targets)
	}

	cfg.CleanLevel = config.Standard
	targets, err = mobile.Scan(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if len(targets) != 2 || targets[1].Description != "Android Virtual Devices: Pixel.avd" || targets[1].Safety != config.Moderate {
		t.Fatalf("standard = %+v", targets)
	}
}

func TestBackendCleaner_SplitsMavenGoAndCargo(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := os.WriteFile(mustDir(t, filepath.Join(home, ".m2", "repository", "com", "example.jar")), []byte("mvn"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mustDir(t, filepath.Join(home, "go", "pkg", "mod", "github.com", "mod")), []byte("mod"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mustDir(t, filepath.Join(home, ".cargo", "registry", "cache", "crate")), []byte("crate"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mustDir(t, filepath.Join(home, ".gradle", "caches", "modules-2", "a.jar")), []byte("jar"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mustDir(t, filepath.Join(home, "Library", "Caches", "pip", "http", "x")), []byte("whl"), 0o644); err != nil {
		t.Fatal(err)
	}

	backend, err := NewBackendCleaner()
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.NewDefaultConfig()
	cfg.CleanLevel = config.Conservative

	targets, err := backend.Scan(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if len(targets) != 2 {
		t.Fatalf("conservative = %+v, want cargo and gradle", targets)
	}
	if targets[0].Description != "Cargo registry cache: cache" || targets[0].Safety != config.Safe {
		t.Errorf("cargo = %+v", targets[0])
	}
	if targets[1].Description != "Gradle cache: modules-2" || targets[1].Safety != config.Safe {
		t.Errorf("gradle = %+v", targets[1])
	}

	cfg.CleanLevel = config.Standard
	targets, err = backend.Scan(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if len(targets) != 4 {
		t.Fatalf("standard = %+v, want cargo, gradle, maven, and the Go module cache", targets)
	}
	if targets[2].Description != "Maven local repository: com" || targets[2].Safety != config.Moderate {
		t.Errorf("maven = %+v", targets[2])
	}
	if targets[3].Description != "Go module cache: github.com" || targets[3].Safety != config.Moderate {
		t.Errorf("modules = %+v", targets[3])
	}
}

func TestBackendCleaner_SplitsComposerGemAndBundler(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := os.WriteFile(mustDir(t, filepath.Join(home, ".composer", "cache", "files", "pkg.zip")), []byte("php"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mustDir(t, filepath.Join(home, ".gem", "cache", "rake.gem")), []byte("gem"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mustDir(t, filepath.Join(home, ".bundle", "cache", "compact_index", "info")), []byte("idx"), 0o644); err != nil {
		t.Fatal(err)
	}

	backend, err := NewBackendCleaner()
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.NewDefaultConfig()
	cfg.CleanLevel = config.Conservative

	targets, err := backend.Scan(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if len(targets) != 3 {
		t.Fatalf("Scan() = %+v, want composer, gems, and bundler", targets)
	}
	if targets[0].Description != "Composer cache: files" || targets[0].Safety != config.Safe {
		t.Errorf("composer = %+v", targets[0])
	}
	if targets[1].Description != "Ruby gem cache: rake.gem" || targets[1].Safety != config.Safe {
		t.Errorf("gem = %+v", targets[1])
	}
	if targets[2].Description != "Bundler cache: compact_index" || targets[2].Safety != config.Safe {
		t.Errorf("bundler = %+v", targets[2])
	}
}

func TestFrontendCleaner_SplitsNpmYarnAndPnpm(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := os.WriteFile(mustDir(t, filepath.Join(home, ".npm", "_cacache", "content")), []byte("npm"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mustDir(t, filepath.Join(home, ".cache", "yarn", "v6", "pkg")), []byte("yarn"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mustDir(t, filepath.Join(home, ".pnpm-store", "v3", "files")), []byte("pnpm"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mustDir(t, filepath.Join(home, "Library", "Caches", "Yarn", "pkg")), []byte("global"), 0o644); err != nil {
		t.Fatal(err)
	}

	frontend, err := NewFrontendCleaner()
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.NewDefaultConfig()
	cfg.CleanLevel = config.Conservative

	targets, err := frontend.Scan(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if len(targets) != 3 {
		t.Fatalf("Scan() = %+v, want npm, yarn, and pnpm", targets)
	}
	if targets[0].Description != "npm cache: _cacache" || targets[0].Safety != config.Safe {
		t.Errorf("npm = %+v", targets[0])
	}
	if targets[1].Description != "Yarn cache: v6" || targets[1].Safety != config.Safe {
		t.Errorf("yarn = %+v", targets[1])
	}
	if targets[2].Description != "pnpm store: v3" || targets[2].Safety != config.Safe {
		t.Errorf("pnpm = %+v", targets[2])
	}
}

func TestDataMLCleaner_SplitsCondaHuggingFaceAndWandb(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := os.WriteFile(mustDir(t, filepath.Join(home, ".conda", "pkgs", "numpy-1.26", "info")), []byte("pkg"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mustDir(t, filepath.Join(home, ".cache", "huggingface", "hub", "model")), []byte("hf"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mustDir(t, filepath.Join(home, ".cache", "wandb", "artifacts", "run")), []byte("wb"), 0o644); err != nil {
		t.Fatal(err)
	}

	data, err := NewDataMLCleaner()
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.NewDefaultConfig()
	cfg.CleanLevel = config.Conservative

	targets, err := data.Scan(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if len(targets) != 3 {
		t.Fatalf("Scan() = %+v, want conda, Hugging Face, and Weights & Biases", targets)
	}
	if targets[0].Description != "Conda package cache: numpy-1.26" || targets[0].Safety != config.Safe {
		t.Errorf("conda = %+v", targets[0])
	}
	if targets[1].Description != "Hugging Face cache: hub" || targets[1].Safety != config.Safe {
		t.Errorf("huggingface = %+v", targets[1])
	}
	if targets[2].Description != "Weights & Biases cache: artifacts" || targets[2].Safety != config.Safe {
		t.Errorf("wandb = %+v", targets[2])
	}
}

func TestDataMLCleaner_SplitsMambaTorchAndKeras(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := os.WriteFile(mustDir(t, filepath.Join(home, ".mamba", "pkgs", "numpy-1.26", "info")), []byte("pkg"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mustDir(t, filepath.Join(home, ".cache", "torch", "hub", "checkpoints", "resnet.pth")), []byte("pt"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mustDir(t, filepath.Join(home, ".keras", "datasets", "mnist.npz")), []byte("ds"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mustDir(t, filepath.Join(home, ".keras", "models", "resnet.keras")), []byte("model"), 0o644); err != nil {
		t.Fatal(err)
	}

	data, err := NewDataMLCleaner()
	if err != nil {
		t.Fatal(err)
	}
	targets, err := data.Scan(context.Background(), config.NewDefaultConfig())
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if len(targets) != 4 {
		t.Fatalf("Scan() = %+v, want mamba, PyTorch, and both Keras caches", targets)
	}
	if targets[0].Description != "Mamba/Miniforge package cache: numpy-1.26" || targets[0].Safety != config.Safe {
		t.Errorf("mamba = %+v", targets[0])
	}
	if targets[1].Description != "PyTorch Hub cache: checkpoints" || targets[1].Safety != config.Safe {
		t.Errorf("torch = %+v", targets[1])
	}
	if targets[2].Description != "Keras/TensorFlow datasets cache: mnist.npz" || targets[2].Safety != config.Safe {
		t.Errorf("datasets = %+v", targets[2])
	}
	if targets[3].Description != "Keras/TensorFlow models cache: resnet.keras" || targets[3].Safety != config.Safe {
		t.Errorf("models = %+v", targets[3])
	}
}

func TestDataMLCleaner_SplitsCondaArchivesJupyterRuntimeAndKernels(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := os.WriteFile(mustDir(t, filepath.Join(home, ".conda", "envs", ".pkgs", "numpy.tar.bz2")), []byte("tar"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mustDir(t, filepath.Join(home, "Library", "Jupyter", "runtime", "kernel.json")), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mustDir(t, filepath.Join(home, "Library", "Jupyter", "kernels", "python3", "kernel.json")), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	data, err := NewDataMLCleaner()
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.NewDefaultConfig()
	cfg.CleanLevel = config.Conservative

	targets, err := data.Scan(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if len(targets) != 2 {
		t.Fatalf("conservative = %+v", targets)
	}
	if targets[0].Description != "Conda environments tarball cache: numpy.tar.bz2" || targets[0].Safety != config.Safe {
		t.Errorf("conda = %+v", targets[0])
	}
	if targets[1].Description != "Jupyter runtime files: kernel.json" || targets[1].Safety != config.Safe {
		t.Errorf("runtime = %+v", targets[1])
	}

	cfg.CleanLevel = config.Standard
	targets, err = data.Scan(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if len(targets) != 2 {
		t.Fatalf("standard = %+v, want no small kernel", targets)
	}
	for _, target := range targets {
		if target.Description == "Jupyter kernels cache: python3" {
			t.Fatalf("standard included a small kernel: %+v", target)
		}
	}
}

func TestDevOpsCleaner_SplitsCachesMinikubeAndBoxes(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	files := []string{
		filepath.Join(home, ".kube", "cache", "http", "index"),
		filepath.Join(home, ".aws", "cli", "cache", "session.json"),
		filepath.Join(home, ".cache", "helm", "repository", "index.yaml"),
		filepath.Join(home, ".minikube", "cache", "iso"),
		filepath.Join(home, ".minikube", "machines", "disk"),
		filepath.Join(home, ".vagrant.d", "boxes", "ubuntu", "box"),
	}
	for _, path := range files {
		if err := os.WriteFile(mustDir(t, path), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	devops, err := NewDevOpsCleaner()
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.NewDefaultConfig()
	cfg.CleanLevel = config.Conservative

	targets, err := devops.Scan(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	for _, desc := range []string{"Kubernetes cache: http", "AWS CLI cache: session.json", "Helm cache: repository"} {
		got, ok := findTarget(targets, desc)
		if !ok || got.Safety != config.Safe {
			t.Errorf("missing %s in %+v", desc, targets)
		}
	}
	if _, ok := findTarget(targets, "Minikube: cache"); ok {
		t.Errorf("conservative included Minikube: %+v", targets)
	}

	cfg.CleanLevel = config.Standard
	targets, err = devops.Scan(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	for _, desc := range []string{"Minikube: cache", "Minikube: machines", "Vagrant boxes: ubuntu"} {
		got, ok := findTarget(targets, desc)
		if !ok || got.Safety != config.Moderate {
			t.Errorf("missing %s in %+v", desc, targets)
		}
	}
}

func TestScanTrash_SplitsEntries(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	notes := filepath.Join(home, ".Trash", "notes.txt")
	if err := os.WriteFile(mustDir(t, notes), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	targets, err := NewTrashCleaner().Scan(context.Background(), config.NewDefaultConfig())
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	got, ok := findTarget(targets, "User trash: notes.txt")
	if !ok || got.Path != notes || got.Safety != config.Safe {
		t.Fatalf("trash = %+v, targets = %+v", got, targets)
	}
	for _, target := range targets {
		if target.Path == filepath.Join(home, ".Trash") {
			t.Fatalf("Scan() targeted the trash directory itself: %+v", target)
		}
	}
}

func TestScanTemp_TargetsChildrenOnly(t *testing.T) {
	targets, err := NewTempFilesCleaner().Scan(context.Background(), config.NewDefaultConfig())
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	for _, target := range targets {
		parent := filepath.Dir(target.Path)
		if parent != "/private/tmp" && parent != "/private/var/tmp" {
			t.Errorf("target = %+v, want a child of a temp directory", target)
		}
		if target.Safety != config.Safe {
			t.Errorf("safety = %v", target.Safety)
		}
	}
}

func TestAICleaner_SplitsCachesAndLeavesUserCacheToCacheCleaner(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := os.WriteFile(mustDir(t, filepath.Join(home, "Library", "Caches", "Claude", "blob")), []byte("c"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mustDir(t, filepath.Join(home, "Library", "Application Support", "Claude", "Cache", "Cache_Data", "data")), []byte("c"), 0o644); err != nil {
		t.Fatal(err)
	}

	ai, err := NewAICleaner()
	if err != nil {
		t.Fatal(err)
	}
	targets, err := ai.Scan(context.Background(), config.NewDefaultConfig())
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if len(targets) != 1 || targets[0].Description != "Claude desktop application cache: Cache_Data" || targets[0].Safety != config.Safe {
		t.Fatalf("Scan() = %+v", targets)
	}
}

func TestScanIOSBackups_SplitsDevicesInAggressiveMode(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	backup := filepath.Join(home, "Library", "Application Support", "MobileSync", "Backup", "DEVICE", "Info.plist")
	if err := os.WriteFile(mustDir(t, backup), []byte("plist"), 0o644); err != nil {
		t.Fatal(err)
	}

	ios := NewIOSBackupCleaner()
	cfg := config.NewDefaultConfig()
	cfg.CleanLevel = config.Conservative

	targets, err := ios.Scan(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if len(targets) != 0 {
		t.Fatalf("conservative = %+v", targets)
	}

	cfg.CleanLevel = config.Aggressive
	targets, err = ios.Scan(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if len(targets) != 1 || targets[0].Description != "iOS device backups (DANGEROUS - may contain important data): DEVICE" || targets[0].Safety != config.Dangerous {
		t.Fatalf("aggressive = %+v", targets)
	}
}

func findTarget(targets []CleanTarget, description string) (CleanTarget, bool) {
	for _, target := range targets {
		if target.Description == description {
			return target, true
		}
	}
	return CleanTarget{}, false
}

func mustDir(t *testing.T, path string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}
