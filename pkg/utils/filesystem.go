package utils

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

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

// GetDirSize calculates the total size of a directory recursively
func GetDirSize(path string) (int64, error) {
	var size int64

	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			// Continue on permission errors
			return nil
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})

	return size, err
}

// SafeRemove removes a path, respecting the dryRun flag
func SafeRemove(path string, dryRun bool) error {
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
