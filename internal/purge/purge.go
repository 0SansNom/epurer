// Package purge scans common project directories for removable build
// artifacts (node_modules, target, .venv, Pods, ...) and groups them by the
// project they belong to, so cleanup can be reviewed one project at a time
// rather than as a flat list of paths.
package purge

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/0SansNom/epurer/internal/cleaner"
	"github.com/0SansNom/epurer/internal/config"
	"github.com/0SansNom/epurer/internal/scanner"
)

// Artifact is a single removable directory found inside a project.
type Artifact struct {
	cleaner.CleanTarget
	Kind        string    // e.g. "node_modules", "target", ".venv"
	ModTime     time.Time // last-modified time of the artifact itself
	ProjectRoot string
	Selected    bool // preselected based on age
}

// Project groups every artifact found directly under the same directory.
type Project struct {
	Root      string
	Name      string
	Type      string // detected from marker files, e.g. "Node.js", "Rust", "Unknown"
	Artifacts []Artifact
	TotalSize int64
}

// artifactSpec describes one pattern to search for and how to decide whether
// a match is really a build artifact (vs. a coincidentally-named directory).
type artifactSpec struct {
	pattern string
	kind    string
	safety  config.SafetyLevel
	// marker returns true if the match at artifactPath should be treated as
	// a genuine artifact. nil means the name is unambiguous - always true.
	marker func(artifactPath string) bool
}

func siblingExists(artifactPath string, names ...string) bool {
	parent := filepath.Dir(artifactPath)
	for _, name := range names {
		if _, err := os.Stat(filepath.Join(parent, name)); err == nil {
			return true
		}
	}
	return false
}

func specs() []artifactSpec {
	return []artifactSpec{
		{pattern: "node_modules", kind: "node_modules", safety: config.Moderate},
		{pattern: ".next", kind: ".next", safety: config.Safe},
		{pattern: ".turbo", kind: ".turbo", safety: config.Safe},
		{pattern: "cmake-build-*", kind: "cmake-build", safety: config.Safe},
		{
			pattern: "Pods", kind: "Pods", safety: config.Moderate,
			marker: func(p string) bool { return siblingExists(p, "Podfile") },
		},
		{
			pattern: ".venv", kind: ".venv", safety: config.Moderate,
			marker: func(p string) bool { _, err := os.Stat(filepath.Join(p, "pyvenv.cfg")); return err == nil },
		},
		{
			pattern: "venv", kind: "venv", safety: config.Moderate,
			marker: func(p string) bool { _, err := os.Stat(filepath.Join(p, "pyvenv.cfg")); return err == nil },
		},
		{
			pattern: "target", kind: "target", safety: config.Moderate,
			marker: func(p string) bool { return siblingExists(p, "Cargo.toml", "pom.xml") },
		},
		{
			pattern: "vendor", kind: "vendor", safety: config.Moderate,
			marker: func(p string) bool { return siblingExists(p, "go.mod", "composer.json") },
		},
		{
			pattern: "build", kind: "build", safety: config.Safe,
			marker: func(p string) bool {
				return siblingExists(p, "package.json", "build.gradle", "build.gradle.kts", "Package.swift", "CMakeLists.txt")
			},
		},
		{
			pattern: ".build", kind: ".build", safety: config.Safe,
			marker: func(p string) bool { return siblingExists(p, "Package.swift") },
		},
	}
}

// projectMarkers maps a marker file present at a project root to a
// human-readable project type.
var projectMarkers = []struct {
	file string
	typ  string
}{
	{"package.json", "Node.js"},
	{"go.mod", "Go"},
	{"Cargo.toml", "Rust"},
	{"pyproject.toml", "Python"},
	{"requirements.txt", "Python"},
	{"Pipfile", "Python"},
	{"Podfile", "CocoaPods/iOS"},
	{"pubspec.yaml", "Flutter"},
	{"pom.xml", "Java (Maven)"},
	{"build.gradle", "Java (Gradle)"},
	{"build.gradle.kts", "Java (Gradle)"},
	{"composer.json", "PHP"},
	{"Package.swift", "Swift"},
	{"CMakeLists.txt", "C/C++ (CMake)"},
}

func detectProjectType(root string) string {
	for _, m := range projectMarkers {
		if _, err := os.Stat(filepath.Join(root, m.file)); err == nil {
			return m.typ
		}
	}
	return "Unknown"
}

// Scan searches s's configured directories for project artifacts and groups
// the results by project. Artifacts whose ModTime is older than minAge are
// preselected.
func Scan(ctx context.Context, s *scanner.Scanner, minAge time.Duration) ([]Project, error) {
	projects := make(map[string]*Project)
	cutoff := time.Now().Add(-minAge)

	for _, spec := range specs() {
		results := s.FindByPattern(ctx, spec.pattern)
		for r := range results {
			if r.Err != nil {
				continue
			}

			// Skip nested matches (e.g. node_modules inside node_modules).
			if filepath.Base(filepath.Dir(r.Path)) == spec.pattern {
				continue
			}

			if spec.marker != nil && !spec.marker(r.Path) {
				continue
			}

			info, err := os.Stat(r.Path)
			if err != nil {
				continue
			}

			root := filepath.Dir(r.Path)

			artifact := Artifact{
				CleanTarget: cleaner.CleanTarget{
					Path:        r.Path,
					Description: spec.kind,
					SizeBytes:   r.Size,
					Safety:      spec.safety,
				},
				Kind:        spec.kind,
				ModTime:     info.ModTime(),
				ProjectRoot: root,
				Selected:    info.ModTime().Before(cutoff),
			}

			p, ok := projects[root]
			if !ok {
				p = &Project{
					Root: root,
					Name: filepath.Base(root),
					Type: detectProjectType(root),
				}
				projects[root] = p
			}

			p.Artifacts = append(p.Artifacts, artifact)
			p.TotalSize += artifact.SizeBytes
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
	}

	result := make([]Project, 0, len(projects))
	for _, p := range projects {
		result = append(result, *p)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].TotalSize > result[j].TotalSize
	})

	return result, nil
}
