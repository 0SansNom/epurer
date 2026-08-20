package ignorelist

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/0SansNom/epurer/pkg/utils"
)

// List holds a set of filesystem paths the user never wants scanned or cleaned.
type List struct {
	Paths []string `json:"paths"`
	path  string   // path to the backing JSON file
}

// configPath returns the path to the ignore list file:
// ~/Library/Application Support/epurer/ignore.json
func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "Application Support", "epurer", "ignore.json"), nil
}

// Load reads the ignore list from disk, returning an empty list if the file
// doesn't exist yet.
func Load() (*List, error) {
	p, err := configPath()
	if err != nil {
		return nil, err
	}

	l := &List{path: p}

	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return l, nil
		}
		return nil, err
	}

	if err := json.Unmarshal(data, l); err != nil {
		return nil, err
	}
	l.path = p // json.Unmarshal doesn't touch unexported fields, but keep it explicit

	return l, nil
}

// Save persists the ignore list to disk.
func (l *List) Save() error {
	if err := os.MkdirAll(filepath.Dir(l.path), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(l.path, data, 0o644)
}

// normalize resolves a user-supplied path to an absolute, cleaned form so
// comparisons in IsIgnored are reliable regardless of how the path was typed.
func normalize(path string) (string, error) {
	expanded, err := utils.ExpandHome(path)
	if err != nil {
		return "", err
	}

	abs, err := filepath.Abs(expanded)
	if err != nil {
		return "", err
	}

	return filepath.Clean(abs), nil
}

// Add adds a path to the ignore list and persists it. Adding an already
// present path is a no-op.
func (l *List) Add(path string) error {
	norm, err := normalize(path)
	if err != nil {
		return err
	}

	for _, p := range l.Paths {
		if p == norm {
			return nil
		}
	}

	l.Paths = append(l.Paths, norm)
	sort.Strings(l.Paths)

	return l.Save()
}

// Remove removes a path from the ignore list and persists the change.
func (l *List) Remove(path string) error {
	norm, err := normalize(path)
	if err != nil {
		return err
	}

	filtered := l.Paths[:0]
	for _, p := range l.Paths {
		if p != norm {
			filtered = append(filtered, p)
		}
	}
	l.Paths = filtered

	return l.Save()
}

// IsIgnored reports whether path is equal to, or nested under, any entry in
// the ignore list.
func (l *List) IsIgnored(path string) bool {
	norm, err := normalize(path)
	if err != nil {
		return false
	}

	for _, p := range l.Paths {
		if norm == p || strings.HasPrefix(norm, p+string(filepath.Separator)) {
			return true
		}
	}

	return false
}
