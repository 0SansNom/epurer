package cleaner

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/0SansNom/epurer/internal/config"
	"github.com/0SansNom/epurer/internal/scanner"
	"github.com/0SansNom/epurer/pkg/utils"
)

// DevOpsCleaner handles DevOps cleanup (Docker, Kubernetes, Terraform, Cloud CLIs)
type DevOpsCleaner struct {
	scanner *scanner.Scanner
}

// NewDevOpsCleaner creates a new DevOpsCleaner
func NewDevOpsCleaner() (Cleaner, error) {
	s, err := scanner.NewScanner()
	if err != nil {
		return nil, err
	}

	return &DevOpsCleaner{
		scanner: s,
	}, nil
}

func (d *DevOpsCleaner) Name() string {
	return "DevOps"
}

func (d *DevOpsCleaner) Domain() config.Domain {
	return config.DomainFrontend // TODO: Add DomainDevOps to config
}

func (d *DevOpsCleaner) Detect(ctx context.Context) (bool, error) {
	return utils.CommandExists("docker") ||
		utils.CommandExists("kubectl") ||
		utils.CommandExists("terraform") ||
		utils.CommandExists("helm"), nil
}

func (d *DevOpsCleaner) Scan(ctx context.Context, cfg *config.Config) ([]CleanTarget, error) {
	targets := []CleanTarget{}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	// === Docker ===

	if utils.CommandExists("docker") {
		// Dangling images (Moderate)
		if cfg.CleanLevel.AllowsSafety(config.Moderate) {
			danglingSize := d.getDockerDanglingSize()
			if danglingSize > 0 {
				targets = append(targets, CleanTarget{
					Path:        "docker:images:dangling",
					Description: "Docker dangling images",
					SizeBytes:   danglingSize,
					Safety:      config.Moderate,
				})
			}
		}

		// Stopped containers (Moderate)
		if cfg.CleanLevel.AllowsSafety(config.Moderate) {
			stoppedSize := d.getDockerStoppedContainersSize()
			if stoppedSize > 0 {
				targets = append(targets, CleanTarget{
					Path:        "docker:containers:stopped",
					Description: "Docker stopped containers",
					SizeBytes:   stoppedSize,
					Safety:      config.Moderate,
				})
			}
		}

		// Build cache (Safe)
		buildCacheSize := d.getDockerBuildCacheSize()
		if buildCacheSize > 0 {
			targets = append(targets, CleanTarget{
				Path:        "docker:buildcache",
				Description: "Docker build cache",
				SizeBytes:   buildCacheSize,
				Safety:      config.Safe,
			})
		}

		// Unused volumes (Dangerous - may contain data)
		if cfg.CleanLevel.AllowsSafety(config.Dangerous) {
			volumeSize := d.getDockerUnusedVolumesSize()
			if volumeSize > 0 {
				targets = append(targets, CleanTarget{
					Path:        "docker:volumes:unused",
					Description: "Docker unused volumes (DANGEROUS - may contain data)",
					SizeBytes:   volumeSize,
					Safety:      config.Dangerous,
				})
			}
		}
	}

	// === Kubernetes ===

	// kubectl cache (Safe)
	kubeCachePath := filepath.Join(home, ".kube", "cache")
	if utils.PathExists(kubeCachePath) {
		size, _ := utils.GetDirSize(kubeCachePath)
		if size > 0 {
			targets = append(targets, CleanTarget{
				Path:        kubeCachePath,
				Description: "Kubernetes cache",
				SizeBytes:   size,
				Safety:      config.Safe,
			})
		}
	}

	// Minikube (Moderate - can be recreated)
	if cfg.CleanLevel.AllowsSafety(config.Moderate) {
		minikubePath := filepath.Join(home, ".minikube")
		if utils.PathExists(minikubePath) {
			size, _ := utils.GetDirSize(minikubePath)
			if size > 0 {
				targets = append(targets, CleanTarget{
					Path:        minikubePath,
					Description: "Minikube cache and VMs",
					SizeBytes:   size,
					Safety:      config.Moderate,
				})
			}
		}
	}

	// === Terraform ===

	// .terraform folders (Moderate - providers and modules)
	if cfg.CleanLevel.AllowsSafety(config.Moderate) {
		terraformTargets := d.scanTerraform(ctx)
		targets = append(targets, terraformTargets...)
	}

	// === Cloud CLIs ===

	// AWS CLI cache (Safe)
	awsCachePath := filepath.Join(home, ".aws", "cli", "cache")
	if utils.PathExists(awsCachePath) {
		size, _ := utils.GetDirSize(awsCachePath)
		if size > 0 {
			targets = append(targets, CleanTarget{
				Path:        awsCachePath,
				Description: "AWS CLI cache",
				SizeBytes:   size,
				Safety:      config.Safe,
			})
		}
	}

	// Helm cache (Safe)
	helmCachePath := filepath.Join(home, ".cache", "helm")
	if utils.PathExists(helmCachePath) {
		size, _ := utils.GetDirSize(helmCachePath)
		if size > 0 {
			targets = append(targets, CleanTarget{
				Path:        helmCachePath,
				Description: "Helm cache",
				SizeBytes:   size,
				Safety:      config.Safe,
			})
		}
	}

	// === Vagrant ===

	// Vagrant boxes (Moderate - can be large VMs)
	if cfg.CleanLevel.AllowsSafety(config.Moderate) {
		vagrantBoxesPath := filepath.Join(home, ".vagrant.d", "boxes")
		if utils.PathExists(vagrantBoxesPath) {
			size, _ := utils.GetDirSize(vagrantBoxesPath)
			if size > 0 {
				targets = append(targets, CleanTarget{
					Path:        vagrantBoxesPath,
					Description: "Vagrant boxes",
					SizeBytes:   size,
					Safety:      config.Moderate,
				})
			}
		}
	}

	return targets, nil
}

func (d *DevOpsCleaner) Clean(ctx context.Context, targets []CleanTarget, dryRun bool) ([]CleanResult, error) {
	results := make([]CleanResult, 0, len(targets))

	for _, target := range targets {
		result := CleanResult{
			Target:  target,
			Success: true,
		}

		// Docker commands are special
		if strings.HasPrefix(target.Path, "docker:") {
			err := d.cleanDockerTarget(target.Path, dryRun)
			if err != nil {
				result.Success = false
				result.Error = err
			} else {
				result.BytesFreed = target.SizeBytes
			}
		} else {
			// Regular file/directory removal
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

// Docker helper methods

func (d *DevOpsCleaner) getDockerDanglingSize() int64 {
	cmd := exec.Command("docker", "images", "-f", "dangling=true", "-q")
	output, err := cmd.Output()
	if err != nil {
		return 0
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 || (len(lines) == 1 && lines[0] == "") {
		return 0
	}

	// Estimate: ~100MB per dangling image (conservative)
	return int64(len(lines)) * 100 * 1024 * 1024
}

func (d *DevOpsCleaner) getDockerStoppedContainersSize() int64 {
	cmd := exec.Command("docker", "ps", "-a", "-f", "status=exited", "-q")
	output, err := cmd.Output()
	if err != nil {
		return 0
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 || (len(lines) == 1 && lines[0] == "") {
		return 0
	}

	// Estimate: ~50MB per stopped container (conservative)
	return int64(len(lines)) * 50 * 1024 * 1024
}

func (d *DevOpsCleaner) getDockerBuildCacheSize() int64 {
	cmd := exec.Command("docker", "system", "df", "-v")
	output, err := cmd.Output()
	if err != nil {
		return 0
	}

	// Parse output to find build cache size
	// For now, return a conservative estimate
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "Build Cache") {
			// Try to parse the size
			// Format: "Build Cache   X   Y   Z"
			// This is a simplified approach
			return 1024 * 1024 * 1024 // 1GB estimate
		}
	}

	return 0
}

func (d *DevOpsCleaner) getDockerUnusedVolumesSize() int64 {
	cmd := exec.Command("docker", "volume", "ls", "-f", "dangling=true", "-q")
	output, err := cmd.Output()
	if err != nil {
		return 0
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 || (len(lines) == 1 && lines[0] == "") {
		return 0
	}

	// Estimate: ~200MB per volume (very conservative)
	return int64(len(lines)) * 200 * 1024 * 1024
}

func (d *DevOpsCleaner) cleanDockerTarget(target string, dryRun bool) error {
	if dryRun {
		return nil
	}

	switch target {
	case "docker:images:dangling":
		return exec.Command("docker", "image", "prune", "-f").Run()
	case "docker:containers:stopped":
		return exec.Command("docker", "container", "prune", "-f").Run()
	case "docker:buildcache":
		return exec.Command("docker", "builder", "prune", "-f").Run()
	case "docker:volumes:unused":
		return exec.Command("docker", "volume", "prune", "-f").Run()
	}

	return nil
}

// scanTerraform scans for .terraform folders
func (d *DevOpsCleaner) scanTerraform(ctx context.Context) []CleanTarget {
	targets := []CleanTarget{}

	resultChan := d.scanner.FindByPattern(ctx, ".terraform")
	for result := range resultChan {
		if result.Err != nil {
			continue
		}

		targets = append(targets, CleanTarget{
			Path:        result.Path,
			Description: "Terraform providers and modules",
			SizeBytes:   result.Size,
			Safety:      config.Moderate,
		})
	}

	return targets
}
