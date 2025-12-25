package cleaner

import (
	"context"
	"os"
	"path/filepath"

	"github.com/0SansNom/mac-dev-clean/internal/config"
	"github.com/0SansNom/mac-dev-clean/internal/scanner"
	"github.com/0SansNom/mac-dev-clean/pkg/utils"
)

// DataMLCleaner handles Data Science and ML cleanup (Conda, Jupyter, TensorFlow, PyTorch, etc.)
type DataMLCleaner struct {
	scanner *scanner.Scanner
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

func (d *DataMLCleaner) Name() string {
	return "Data/ML"
}

func (d *DataMLCleaner) Domain() config.Domain {
	return config.DomainFrontend // TODO: Add DomainDataML to config
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

	// === Conda ===

	// Conda package cache (Safe - can be re-downloaded)
	condaPkgsPath := filepath.Join(home, ".conda", "pkgs")
	if utils.PathExists(condaPkgsPath) {
		size, _ := utils.GetDirSize(condaPkgsPath)
		if size > 0 {
			targets = append(targets, CleanTarget{
				Path:        condaPkgsPath,
				Description: "Conda package cache",
				SizeBytes:   size,
				Safety:      config.Safe,
			})
		}
	}

	// Conda environments tarball cache
	condaEnvsTarPath := filepath.Join(home, ".conda", "envs", ".pkgs")
	if utils.PathExists(condaEnvsTarPath) {
		size, _ := utils.GetDirSize(condaEnvsTarPath)
		if size > 0 {
			targets = append(targets, CleanTarget{
				Path:        condaEnvsTarPath,
				Description: "Conda environments tarball cache",
				SizeBytes:   size,
				Safety:      config.Safe,
			})
		}
	}

	// Miniforge/Mambaforge cache
	miniforgeCache := filepath.Join(home, ".mamba", "pkgs")
	if utils.PathExists(miniforgeCache) {
		size, _ := utils.GetDirSize(miniforgeCache)
		if size > 0 {
			targets = append(targets, CleanTarget{
				Path:        miniforgeCache,
				Description: "Mamba/Miniforge package cache",
				SizeBytes:   size,
				Safety:      config.Safe,
			})
		}
	}

	// === Jupyter ===

	// Jupyter runtime files (Safe)
	jupyterRuntimePath := filepath.Join(home, "Library", "Jupyter", "runtime")
	if utils.PathExists(jupyterRuntimePath) {
		size, _ := utils.GetDirSize(jupyterRuntimePath)
		if size > 0 {
			targets = append(targets, CleanTarget{
				Path:        jupyterRuntimePath,
				Description: "Jupyter runtime files",
				SizeBytes:   size,
				Safety:      config.Safe,
			})
		}
	}

	// Jupyter kernels cache
	jupyterKernelsPath := filepath.Join(home, "Library", "Jupyter", "kernels")
	if utils.PathExists(jupyterKernelsPath) {
		// Only suggest cleaning if it's large
		size, _ := utils.GetDirSize(jupyterKernelsPath)
		if size > 100*1024*1024 { // > 100MB
			targets = append(targets, CleanTarget{
				Path:        jupyterKernelsPath,
				Description: "Jupyter kernels cache",
				SizeBytes:   size,
				Safety:      config.Moderate,
			})
		}
	}

	// .ipynb_checkpoints (Safe - automatically created)
	checkpointTargets := d.scanPattern(ctx, ".ipynb_checkpoints")
	targets = append(targets, checkpointTargets...)

	// === TensorFlow ===

	// TensorFlow cache
	tfCachePath := filepath.Join(home, ".keras")
	if utils.PathExists(tfCachePath) {
		datasetsPath := filepath.Join(tfCachePath, "datasets")
		if utils.PathExists(datasetsPath) {
			size, _ := utils.GetDirSize(datasetsPath)
			if size > 0 {
				targets = append(targets, CleanTarget{
					Path:        datasetsPath,
					Description: "Keras/TensorFlow datasets cache",
					SizeBytes:   size,
					Safety:      config.Safe,
				})
			}
		}

		modelsPath := filepath.Join(tfCachePath, "models")
		if utils.PathExists(modelsPath) {
			size, _ := utils.GetDirSize(modelsPath)
			if size > 0 {
				targets = append(targets, CleanTarget{
					Path:        modelsPath,
					Description: "Keras/TensorFlow models cache",
					SizeBytes:   size,
					Safety:      config.Safe,
				})
			}
		}
	}

	// === PyTorch ===

	// PyTorch hub cache
	torchHubPath := filepath.Join(home, ".cache", "torch", "hub")
	if utils.PathExists(torchHubPath) {
		size, _ := utils.GetDirSize(torchHubPath)
		if size > 0 {
			targets = append(targets, CleanTarget{
				Path:        torchHubPath,
				Description: "PyTorch Hub cache (pretrained models)",
				SizeBytes:   size,
				Safety:      config.Safe,
			})
		}
	}

	// === Hugging Face ===

	// Hugging Face transformers cache
	hfCachePath := filepath.Join(home, ".cache", "huggingface")
	if utils.PathExists(hfCachePath) {
		size, _ := utils.GetDirSize(hfCachePath)
		if size > 0 {
			targets = append(targets, CleanTarget{
				Path:        hfCachePath,
				Description: "Hugging Face transformers cache",
				SizeBytes:   size,
				Safety:      config.Safe,
			})
		}
	}

	// === Weights & Biases ===

	// W&B cache
	wandbCachePath := filepath.Join(home, ".cache", "wandb")
	if utils.PathExists(wandbCachePath) {
		size, _ := utils.GetDirSize(wandbCachePath)
		if size > 0 {
			targets = append(targets, CleanTarget{
				Path:        wandbCachePath,
				Description: "Weights & Biases cache",
				SizeBytes:   size,
				Safety:      config.Safe,
			})
		}
	}

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

	// === MLflow ===

	// MLflow artifacts (Moderate - experiment data)
	if cfg.CleanLevel.AllowsSafety(config.Moderate) {
		mlflowTargets := d.scanPattern(ctx, "mlruns")
		targets = append(targets, mlflowTargets...)
	}

	// === General Data Science ===

	// .DS_Store files (Safe)
	dsStoreTargets := d.scanPattern(ctx, ".DS_Store")
	targets = append(targets, dsStoreTargets...)

	return targets, nil
}

func (d *DataMLCleaner) Clean(ctx context.Context, targets []CleanTarget, dryRun bool) ([]CleanResult, error) {
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
