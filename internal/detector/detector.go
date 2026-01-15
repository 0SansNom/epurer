package detector

import (
	"os"
	"path/filepath"

	"github.com/0SansNom/epurer/pkg/utils"
)

// StackDetector detects installed development tools and frameworks
type StackDetector struct {
	homePath string
}

// DetectionResult contains all detected tools organized by category
type DetectionResult struct {
	Frontend []string // Node.js, npm, yarn, pnpm, etc.
	Backend  []string // Python, Java, Go, Rust, PHP, Ruby
	Mobile   []string // Xcode, Android Studio, Flutter
	DevOps   []string // Docker, Kubernetes, Terraform, Helm
	DataML   []string // Conda, Jupyter, TensorFlow, PyTorch
}

// NewDetector creates a new StackDetector
func NewDetector() (*StackDetector, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	return &StackDetector{
		homePath: home,
	}, nil
}

// DetectAll detects all development tools on the system
func (d *StackDetector) DetectAll() DetectionResult {
	result := DetectionResult{
		Frontend: []string{},
		Backend:  []string{},
		Mobile:   []string{},
		DevOps:   []string{},
		DataML:   []string{},
	}

	// === Frontend Detection ===

	if utils.CommandExists("node") {
		result.Frontend = append(result.Frontend, "node")
	}
	if utils.CommandExists("npm") {
		result.Frontend = append(result.Frontend, "npm")
	}
	if utils.CommandExists("yarn") {
		result.Frontend = append(result.Frontend, "yarn")
	}
	if utils.CommandExists("pnpm") {
		result.Frontend = append(result.Frontend, "pnpm")
	}
	if utils.CommandExists("bun") {
		result.Frontend = append(result.Frontend, "bun")
	}
	if utils.CommandExists("deno") {
		result.Frontend = append(result.Frontend, "deno")
	}

	// === Backend Detection ===

	if utils.CommandExists("python3") || utils.CommandExists("python") {
		result.Backend = append(result.Backend, "python")
	}
	if utils.CommandExists("java") {
		result.Backend = append(result.Backend, "java")
	}
	if utils.CommandExists("go") {
		result.Backend = append(result.Backend, "go")
	}
	if utils.CommandExists("cargo") {
		result.Backend = append(result.Backend, "rust")
	}
	if utils.CommandExists("php") {
		result.Backend = append(result.Backend, "php")
	}
	if utils.CommandExists("ruby") {
		result.Backend = append(result.Backend, "ruby")
	}
	if utils.CommandExists("dotnet") {
		result.Backend = append(result.Backend, ".net")
	}
	if utils.CommandExists("mvn") {
		result.Backend = append(result.Backend, "maven")
	}
	if utils.CommandExists("gradle") {
		result.Backend = append(result.Backend, "gradle")
	}

	// === Mobile Detection ===

	if utils.PathExists("/Applications/Xcode.app") {
		result.Mobile = append(result.Mobile, "xcode")
	}
	if utils.CommandExists("adb") {
		result.Mobile = append(result.Mobile, "android")
	}
	androidHome := filepath.Join(d.homePath, "Library", "Android")
	if utils.PathExists(androidHome) {
		if !contains(result.Mobile, "android") {
			result.Mobile = append(result.Mobile, "android")
		}
	}
	if utils.CommandExists("flutter") {
		result.Mobile = append(result.Mobile, "flutter")
	}
	if utils.CommandExists("pod") {
		result.Mobile = append(result.Mobile, "cocoapods")
	}

	// === DevOps Detection ===

	if utils.CommandExists("docker") {
		result.DevOps = append(result.DevOps, "docker")
	}
	if utils.CommandExists("kubectl") {
		result.DevOps = append(result.DevOps, "kubernetes")
	}
	if utils.CommandExists("terraform") {
		result.DevOps = append(result.DevOps, "terraform")
	}
	if utils.CommandExists("helm") {
		result.DevOps = append(result.DevOps, "helm")
	}
	if utils.CommandExists("minikube") {
		result.DevOps = append(result.DevOps, "minikube")
	}
	if utils.CommandExists("vagrant") {
		result.DevOps = append(result.DevOps, "vagrant")
	}
	if utils.CommandExists("aws") {
		result.DevOps = append(result.DevOps, "aws-cli")
	}
	if utils.CommandExists("gcloud") {
		result.DevOps = append(result.DevOps, "gcloud")
	}
	if utils.CommandExists("az") {
		result.DevOps = append(result.DevOps, "azure-cli")
	}

	// === Data Science / ML Detection ===

	if utils.CommandExists("conda") {
		result.DataML = append(result.DataML, "conda")
	}
	if utils.CommandExists("jupyter") {
		result.DataML = append(result.DataML, "jupyter")
	}
	if utils.CommandExists("pip3") || utils.CommandExists("pip") {
		result.DataML = append(result.DataML, "pip")
	}
	// Check for common ML frameworks by looking for Python packages
	if d.hasPythonPackage("tensorflow") {
		result.DataML = append(result.DataML, "tensorflow")
	}
	if d.hasPythonPackage("torch") {
		result.DataML = append(result.DataML, "pytorch")
	}

	return result
}

// HasFrontend checks if any frontend tools are detected
func (d *StackDetector) HasFrontend() bool {
	result := d.DetectAll()
	return len(result.Frontend) > 0
}

// HasBackend checks if any backend tools are detected
func (d *StackDetector) HasBackend() bool {
	result := d.DetectAll()
	return len(result.Backend) > 0
}

// HasMobile checks if any mobile tools are detected
func (d *StackDetector) HasMobile() bool {
	result := d.DetectAll()
	return len(result.Mobile) > 0
}

// HasDevOps checks if any DevOps tools are detected
func (d *StackDetector) HasDevOps() bool {
	result := d.DetectAll()
	return len(result.DevOps) > 0
}

// HasDataML checks if any Data Science/ML tools are detected
func (d *StackDetector) HasDataML() bool {
	result := d.DetectAll()
	return len(result.DataML) > 0
}

// GetSummary returns a human-readable summary of detected tools
func (d *StackDetector) GetSummary() string {
	result := d.DetectAll()
	summary := ""

	if len(result.Frontend) > 0 {
		summary += "Frontend: "
		for i, tool := range result.Frontend {
			if i > 0 {
				summary += ", "
			}
			summary += tool
		}
		summary += "\n"
	}

	if len(result.Backend) > 0 {
		summary += "Backend: "
		for i, tool := range result.Backend {
			if i > 0 {
				summary += ", "
			}
			summary += tool
		}
		summary += "\n"
	}

	if len(result.Mobile) > 0 {
		summary += "Mobile: "
		for i, tool := range result.Mobile {
			if i > 0 {
				summary += ", "
			}
			summary += tool
		}
		summary += "\n"
	}

	if len(result.DevOps) > 0 {
		summary += "DevOps: "
		for i, tool := range result.DevOps {
			if i > 0 {
				summary += ", "
			}
			summary += tool
		}
		summary += "\n"
	}

	if len(result.DataML) > 0 {
		summary += "Data/ML: "
		for i, tool := range result.DataML {
			if i > 0 {
				summary += ", "
			}
			summary += tool
		}
		summary += "\n"
	}

	if summary == "" {
		summary = "No development tools detected\n"
	}

	return summary
}

// hasPythonPackage checks if a Python package is installed
func (d *StackDetector) hasPythonPackage(pkg string) bool {
	// Check in common site-packages locations
	pythonPaths := []string{
		filepath.Join(d.homePath, "Library", "Python"),
		filepath.Join(d.homePath, ".local", "lib"),
	}

	for _, basePath := range pythonPaths {
		if utils.PathExists(basePath) {
			// Look for the package in site-packages
			matches, _ := filepath.Glob(filepath.Join(basePath, "*/site-packages/"+pkg))
			if len(matches) > 0 {
				return true
			}
		}
	}

	return false
}

// contains checks if a string slice contains a value
func contains(slice []string, value string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}
