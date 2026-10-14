package cleaner

import (
	"context"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
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
		images, containers, volumes, buildCache := dockerReclaimableSizes()
		if cfg.CleanLevel.AllowsSafety(config.Moderate) && images > 0 {
			targets = append(targets, CleanTarget{
				Path:        "docker:images:dangling",
				Description: "Docker dangling images",
				SizeBytes:   images,
				Safety:      config.Moderate,
			})
		}
		if cfg.CleanLevel.AllowsSafety(config.Moderate) && containers > 0 {
			targets = append(targets, CleanTarget{
				Path:        "docker:containers:stopped",
				Description: "Docker stopped containers",
				SizeBytes:   containers,
				Safety:      config.Moderate,
			})
		}
		if buildCache > 0 {
			targets = append(targets, CleanTarget{
				Path:        "docker:buildcache",
				Description: "Docker build cache",
				SizeBytes:   buildCache,
				Safety:      config.Safe,
			})
		}
		if cfg.CleanLevel.AllowsSafety(config.Dangerous) && volumes > 0 {
			targets = append(targets, CleanTarget{
				Path:        "docker:volumes:unused",
				Description: "Docker unused volumes (DANGEROUS - may contain data)",
				SizeBytes:   volumes,
				Safety:      config.Dangerous,
			})
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
		} else if d.ignore != nil && d.ignore.IsIgnored(target.Path) {
			result.Success = false
			result.Error = fmt.Errorf("%w: on the ignore list", utils.ErrNotRemovable)
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

// dockerReclaimableSizes reads the RECLAIMABLE column of `docker system df`.
// A missing row, a zero size, or a size that cannot be parsed yields 0, so the target is skipped.
func dockerReclaimableSizes() (images, containers, volumes, buildCache int64) {
	output, err := exec.Command("docker", "system", "df").Output()
	if err != nil {
		return 0, 0, 0, 0
	}
	return parseDockerReclaimable(string(output))
}

func parseDockerReclaimable(output string) (images, containers, volumes, buildCache int64) {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "Images"):
			images = dockerReclaimableSize(line)
		case strings.HasPrefix(line, "Containers"):
			containers = dockerReclaimableSize(line)
		case strings.HasPrefix(line, "Local Volumes"):
			volumes = dockerReclaimableSize(line)
		case strings.HasPrefix(line, "Build Cache"):
			buildCache = dockerReclaimableSize(line)
		}
	}
	return images, containers, volumes, buildCache
}

func dockerReclaimableSize(line string) int64 {
	if i := strings.Index(line, "("); i >= 0 {
		line = line[:i]
	}
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return 0
	}
	return parseDockerSize(fields[len(fields)-1])
}

func parseDockerSize(raw string) int64 {
	raw = strings.TrimSpace(raw)
	i := 0
	for i < len(raw) && (raw[i] == '.' || (raw[i] >= '0' && raw[i] <= '9')) {
		i++
	}
	if i == 0 {
		return 0
	}
	n, err := strconv.ParseFloat(raw[:i], 64)
	if err != nil || n <= 0 {
		return 0
	}
	var mul float64
	switch strings.ToUpper(strings.TrimSpace(raw[i:])) {
	case "B":
		mul = 1
	case "KB":
		mul = 1e3
	case "KIB":
		mul = 1024
	case "MB":
		mul = 1e6
	case "MIB":
		mul = 1024 * 1024
	case "GB":
		mul = 1e9
	case "GIB":
		mul = 1024 * 1024 * 1024
	case "TB":
		mul = 1e12
	case "TIB":
		mul = 1024 * 1024 * 1024 * 1024
	default:
		return 0
	}
	return int64(math.Round(n * mul))
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
	default:
		return fmt.Errorf("unknown docker target %q", target)
	}
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
