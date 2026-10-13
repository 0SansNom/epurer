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

// DataMLCleaner handles Data Science and ML cleanup (Conda, Jupyter, TensorFlow, PyTorch, etc.)
type DataMLCleaner struct {
	scanner *scanner.Scanner
	fsCleaner
}

// NewDataMLCleaner creates a new DataMLCleaner
func NewDataMLCleaner() (Cleaner, error) {
	s, err := scanner.NewScanner()
	if err != nil {
		return nil, err
	}

	return &DataMLCleaner{
		scanner: s,
	}, nil
}

// SetIgnoreList makes this cleaner respect a persistent ignore list.
func (d *DataMLCleaner) SetIgnoreList(l *ignorelist.List) {
	d.fsCleaner.SetIgnoreList(l)
	d.scanner.SetIgnoreList(l)
}

func (d *DataMLCleaner) Name() string {
	return "Data/ML"
}

func (d *DataMLCleaner) Domain() config.Domain {
	return config.DomainDataML
}

func (d *DataMLCleaner) Detect(ctx context.Context) (bool, error) {
	return utils.CommandExists("conda") ||
		utils.CommandExists("jupyter") ||
		utils.CommandExists("python3"), nil
}

func (d *DataMLCleaner) Scan(ctx context.Context, cfg *config.Config) ([]CleanTarget, error) {
	targets := []CleanTarget{}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	// Conda package cache, one target per package.
	found, err := d.collect(ctx,
		scantask.Walk{Root: filepath.Join(home, ".conda", "pkgs"), Description: "Conda package cache", Safety: config.Safe},
		scantask.Walk{Root: filepath.Join(home, ".cache", "huggingface"), Description: "Hugging Face cache", Safety: config.Safe},
		scantask.Walk{Root: filepath.Join(home, ".cache", "wandb"), Description: "Weights & Biases cache", Safety: config.Safe},
		scantask.Walk{Root: filepath.Join(home, ".mamba", "pkgs"), Description: "Mamba/Miniforge package cache", Safety: config.Safe},
		scantask.Walk{Root: filepath.Join(home, ".cache", "torch", "hub"), Description: "PyTorch Hub cache", Safety: config.Safe},
		scantask.Walk{Root: filepath.Join(home, ".keras", "datasets"), Description: "Keras/TensorFlow datasets cache", Safety: config.Safe},
		scantask.Walk{Root: filepath.Join(home, ".keras", "models"), Description: "Keras/TensorFlow models cache", Safety: config.Safe},
		scantask.Walk{Root: filepath.Join(home, ".conda", "envs", ".pkgs"), Description: "Conda environments tarball cache", Safety: config.Safe},
		scantask.Walk{Root: filepath.Join(home, "Library", "Jupyter", "runtime"), Description: "Jupyter runtime files", Safety: config.Safe},
	)
	if err != nil {
		return nil, err
	}
	targets = append(targets, found...)

	if cfg.CleanLevel.AllowsSafety(config.Moderate) {
		kernels, err := d.collect(ctx,
			scantask.Walk{Root: filepath.Join(home, "Library", "Jupyter", "kernels"), Description: "Jupyter kernels cache", Safety: config.Moderate},
		)
		if err != nil {
			return nil, err
		}
		// A normal kernel install is a few KB. Only offer it once the tree is large.
		var total int64
		for _, kernel := range kernels {
			total += kernel.SizeBytes
		}
		if total > 100*1024*1024 {
			targets = append(targets, kernels...)
		}
	}

	// .ipynb_checkpoints (Safe - automatically created)
	checkpointTargets := d.scanPattern(ctx, ".ipynb_checkpoints")
	targets = append(targets, checkpointTargets...)

	// wandb local logs (Moderate - may contain experiment data)
	if cfg.CleanLevel.AllowsSafety(config.Moderate) {
		wandbTargets := d.scanPattern(ctx, "wandb")
		for _, target := range wandbTargets {
			// Only include if it's a wandb directory with run logs
			if utils.PathExists(filepath.Join(target.Path, "run-*")) {
				target.Safety = config.Moderate
				target.Description = "W&B experiment logs"
				targets = append(targets, target)
			}
		}
	}

	// MLflow artifacts (Moderate - experiment data)
	if cfg.CleanLevel.AllowsSafety(config.Moderate) {
		mlflowTargets := d.scanPattern(ctx, "mlruns")
		targets = append(targets, mlflowTargets...)
	}

	// .DS_Store files (Safe)
	dsStoreTargets := d.scanPattern(ctx, ".DS_Store")
	targets = append(targets, dsStoreTargets...)

	return targets, nil
}

func (d *DataMLCleaner) Clean(ctx context.Context, targets []CleanTarget, dryRun bool) ([]CleanResult, error) {
	return CleanTargets(ctx, targets, dryRun, d.progress)
}

// scanPattern scans for a specific pattern
func (d *DataMLCleaner) scanPattern(ctx context.Context, pattern string) []CleanTarget {
	targets := []CleanTarget{}

	resultChan := d.scanner.FindByPattern(ctx, pattern)
	for result := range resultChan {
		if result.Err != nil {
			continue
		}

		desc := d.getDescriptionForPattern(pattern)
		safety := config.Safe

		// Special handling for certain patterns
		if pattern == "wandb" || pattern == "mlruns" {
			safety = config.Moderate
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
func (d *DataMLCleaner) getDescriptionForPattern(pattern string) string {
	descriptions := map[string]string{
		".ipynb_checkpoints": "Jupyter notebook checkpoints",
		".DS_Store":          "macOS metadata files",
		"wandb":              "W&B experiment logs",
		"mlruns":             "MLflow experiment runs",
	}

	if desc, ok := descriptions[pattern]; ok {
		return desc
	}

	return pattern
}
