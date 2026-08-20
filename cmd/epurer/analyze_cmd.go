package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/0SansNom/epurer/internal/tui"
)

// newAnalyzeCmd creates the analyze command
func newAnalyzeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "analyze [path]",
		Short: "Explore disk usage interactively",
		Long: `Open an interactive disk usage explorer, sorted by size. Browsing does not
modify your files. Press 'r' to reveal the selected entry in Finder, or 'd'
to delete it (always requires confirmation). Defaults to your home directory.`,
		Args: cobra.MaximumNArgs(1),
		RunE: runAnalyze,
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Report deletions from the analyzer without actually deleting")

	return cmd
}

func runAnalyze(cmd *cobra.Command, args []string) error {
	path := ""
	if len(args) == 1 {
		path = args[0]
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		path = home
	}

	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("cannot access %s: %w", path, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", path)
	}

	return tui.RunAnalyzer(path, dryRun)
}
