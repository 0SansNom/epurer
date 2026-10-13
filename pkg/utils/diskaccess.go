package utils

import (
	"os"
	"os/exec"
	"path/filepath"
)

// DiskAccess is whether this process can read files protected by Full Disk Access.
type DiskAccess int

const (
	// DiskAccessUnknown means every probe was missing or failed for another reason.
	DiskAccessUnknown DiskAccess = iota
	// DiskAccessDenied means a protected file exists and macOS refused to open it.
	DiskAccessDenied
	// DiskAccessGranted means at least one protected file opened.
	DiskAccessGranted
)

// fullDiskAccessURL opens Privacy & Security → Full Disk Access.
// macOS has no Allow API for this permission.
const fullDiskAccessURL = "x-apple.systempreferences:com.apple.preference.security?Privacy_AllFiles"

// FullDiskAccess probes files that stay unreadable until Full Disk Access is granted.
// One successful open means granted. A permission error means denied.
// A missing file is ignored, same as a file that failed for another reason.
func FullDiskAccess() DiskAccess {
	home, err := os.UserHomeDir()
	if err != nil {
		return DiskAccessUnknown
	}
	return probeDiskAccess([]string{
		filepath.Join(home, "Library", "Safari", "CloudTabs.db"),
		filepath.Join(home, "Library", "Safari", "Bookmarks.plist"),
		filepath.Join(home, "Library", "Application Support", "com.apple.TCC", "TCC.db"),
		"/Library/Preferences/com.apple.TimeMachine.plist",
	})
}

func probeDiskAccess(paths []string) DiskAccess {
	denied := false
	for _, path := range paths {
		f, err := os.Open(path)
		if err == nil {
			f.Close()
			return DiskAccessGranted
		}
		if os.IsPermission(err) {
			denied = true
		}
	}
	if denied {
		return DiskAccessDenied
	}
	return DiskAccessUnknown
}

// OpenFullDiskAccessSettings opens the Full Disk Access pane in System Settings.
func OpenFullDiskAccessSettings() error {
	return exec.Command("open", fullDiskAccessURL).Run()
}
