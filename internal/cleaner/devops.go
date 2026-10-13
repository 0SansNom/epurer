package cleaner

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/0SansNom/epurer/internal/config"
	"github.com/0SansNom/epurer/internal/ignorelist"
	"github.com/0SansNom/epurer/internal/scanner"
	"github.com/0SansNom/epurer/internal/scantask"
	"github.com/0SansNom/epurer/pkg/utils"
)

// DevOpsCleaner handles DevOps cleanup (Docker, Kubernetes, Terraform, Cloud CLIs)
type DevOpsCleaner struct {
	scanner *scanner.Scanner
	fsCleaner
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

// SetIgnoreList makes this cleaner respect a persistent ignore list.
func (d *DevOpsCleaner) SetIgnoreList(l *ignorelist.List) {
	d.fsCleaner.SetIgnoreList(l)
	d.scanner.SetIgnoreList(l)
}

func (d *DevOpsCleaner) Name() string {
	return "DevOps"
}

func (d *DevOpsCleaner) Domain() config.Domain {
	return config.DomainDevOps
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

	walks := []scantask.Walk{
		{Root: filepath.Join(home, ".kube", "cache"), Description: "Kubernetes cache", Safety: config.Safe},
		{Root: filepath.Join(home, ".aws", "cli", "cache"), Description: "AWS CLI cache", Safety: config.Safe},
		{Root: filepath.Join(home, ".cache", "helm"), Description: "Helm cache", Safety: config.Safe},
	}
	if cfg.CleanLevel.AllowsSafety(config.Moderate) {
		walks = append(walks,
			scantask.Walk{Root: filepath.Join(home, ".minikube"), Description: "Minikube", Safety: config.Moderate},
			scantask.Walk{Root: filepath.Join(home, ".vagrant.d", "boxes"), Description: "Vagrant boxes", Safety: config.Moderate},
		)
	}
	found, err := d.collect(ctx, walks...)
	if err != nil {
		return nil, err
	}
	targets = append(targets, found...)

	// .terraform folders (Moderate - providers and modules)
	if cfg.CleanLevel.AllowsSafety(config.Moderate) {
		terraformTargets := d.scanTerraform(ctx)
		targets = append(targets, terraformTargets...)
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
		} else if !utils.CanRemove(target.Path) {
			result.Success = false
			result.Error = utils.ErrNotRemovable
		} else if !dryRun {
			if err := utils.SafeRemove(target.Path, false); err != nil {
				result.Success = false
				result.Error = err
			} else {
				result.BytesFreed = target.SizeBytes
			}
		} else {
			result.BytesFreed = target.SizeBytes
		}

		results = append(results, result)
		var freed int64
		for _, done := range results {
			if done.Success {
				freed += done.BytesFreed
			}
		}
		noteCleanProgress(d.progress, target.Path, len(results), freed)

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
