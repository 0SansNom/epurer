package utils

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
)

// ErrIncompleteSize is wrapped into GetDirSize's returned error when one or
// more entries under path couldn't be read (most commonly: macOS blocking
// access to a TCC-protected folder like Photos Library without Full Disk
// Access). The returned size is a lower bound, not the true total - callers
// that surface sizes to a user should check for this rather than silently
// trusting a possibly-tiny number for what's actually a large, protected
// directory.
var ErrIncompleteSize = errors.New("directory size is incomplete: some entries could not be read")

// GetDirSize calculates the total size of a directory recursively. If any
// entry under path couldn't be read, the returned size is only a partial
// sum and the error wraps ErrIncompleteSize (check with errors.Is).
func GetDirSize(path string) (int64, error) {
	var size int64
	var incomplete bool

	walkErr := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			// Skip what we can't read, but remember that we did.
			incomplete = true
			return nil
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})

	if walkErr != nil {
		return size, walkErr
	}
	if incomplete {
		return size, ErrIncompleteSize
	}

	return size, nil
}

// protectedPaths are locations SafeRemove refuses to touch, regardless of
// what a caller asks for. This is a last line of defense against a bug
// upstream (a bad filepath.Dir/Join, an over-broad glob) turning into
// deleting a system directory or the user's entire home folder.
var protectedPaths = map[string]bool{
	"/":             true,
	"/System":       true,
	"/Library":      true,
	"/Applications": true,
	"/Users":        true,
	"/bin":          true,
	"/sbin":         true,
	"/usr":          true,
	"/etc":          true,
	"/var":          true,
	"/private":      true,
	"/opt":          true,
	"/Volumes":      true,
}

// isProtectedPath reports whether path is empty, a known system root, or
// the user's home directory itself. Symlink targets are resolved when they
// exist so a link into a protected root cannot bypass the check.
func isProtectedPath(path string) bool {
	if path == "" {
		return true
	}
	return isProtectedCleanPath(filepath.Clean(path)) || isProtectedCleanPath(resolveExisting(path))
}

func isProtectedCleanPath(clean string) bool {
	if protectedPaths[clean] {
		return true
	}
	if home, err := os.UserHomeDir(); err == nil && clean == filepath.Clean(home) {
		return true
	}
	return false
}

// resolveExisting returns the symlink-resolved path when it exists.
// Missing paths keep their cleaned form so CanRemove can still reason
// about the parent directory.
func resolveExisting(path string) string {
	clean := filepath.Clean(path)
	resolved, err := filepath.EvalSymlinks(clean)
	if err != nil {
		return clean
	}
	return resolved
}

// PathExists checks if a path exists on the filesystem
func PathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// ExpandHome expands ~ to the user's home directory
func ExpandHome(path string) (string, error) {
	if !strings.HasPrefix(path, "~") {
		return path, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	if path == "~" {
		return home, nil
	}

	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, path[2:]), nil
	}

	return path, nil
}

// ErrNotRemovable means this user cannot unlink the path. epurer does not
// ask for an admin password to get around that.
var ErrNotRemovable = errors.New("not removable by this user; epurer does not ask for an admin password")

// IsPseudoPath reports command-style targets that are not filesystem paths
// (DNS flush, Docker prune, …). They must never reach os.RemoveAll.
func IsPseudoPath(path string) bool {
	return strings.HasPrefix(path, "system:") || strings.HasPrefix(path, "docker:")
}

// CanRemove reports whether this process can unlink path.
// Known pseudo-targets (system:…, docker:…) are left to their own cleaner.
// Other non-absolute paths are refused so a relative RemoveAll cannot
// follow the process CWD. A missing absolute path is removable when its
// parent is writable. On a sticky parent, only a file owned by this user
// is removable.
func CanRemove(path string) bool {
	if path == "" || isProtectedPath(path) {
		return false
	}
	if IsPseudoPath(path) {
		return true
	}
	if !filepath.IsAbs(path) {
		return false
	}

	// Unlink permission is about the directory that holds the name, not the
	// symlink target. isProtectedPath already refused resolved protected roots.
	clean := filepath.Clean(path)
	parent := filepath.Dir(clean)
	if unix.Access(parent, unix.W_OK) != nil {
		return false
	}

	info, err := os.Lstat(clean)
	if err != nil {
		return os.IsNotExist(err)
	}
	parentInfo, err := os.Lstat(parent)
	if err != nil || parentInfo.Mode()&os.ModeSticky == 0 {
		return err == nil
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return true
	}
	return int(stat.Uid) == os.Getuid()
}

// SafeRemove removes a path, respecting the dryRun flag. It refuses
// protected roots, pseudo-targets, and anything CanRemove rejects.
func SafeRemove(path string, dryRun bool) error {
	if isProtectedPath(path) {
		return fmt.Errorf("refusing to remove protected path: %q", path)
	}
	if IsPseudoPath(path) {
		return fmt.Errorf("refusing to remove command target as a file path: %q", path)
	}
	if !CanRemove(path) {
		return ErrNotRemovable
	}

	if dryRun {
		return nil
	}
	return os.RemoveAll(path)
}

// CommandExists checks if a command is available in PATH
func CommandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// IsWritable checks if a path is writable by the current user
func IsWritable(path string) bool {
	// Try to create a temp file in the directory
	if !PathExists(path) {
		return false
	}

	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	// If it's a directory, try to create a temp file in it
	if info.IsDir() {
		testFile := filepath.Join(path, ".write_test")
		file, err := os.Create(testFile)
		if err != nil {
			return false
		}
		file.Close()
		os.Remove(testFile)
		return true
	}

	// If it's a file, check the directory
	return IsWritable(filepath.Dir(path))
}
