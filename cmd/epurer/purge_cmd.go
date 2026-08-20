package main

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/0SansNom/epurer/internal/ignorelist"
	"github.com/0SansNom/epurer/internal/purge"
	"github.com/0SansNom/epurer/internal/reporter"
	"github.com/0SansNom/epurer/internal/scanner"
	"github.com/0SansNom/epurer/internal/tui"
	"github.com/0SansNom/epurer/pkg/utils"
)

var (
	purgeMinAge time.Duration
	purgeForce  bool
	purgeAll    bool
	purgeYes    bool
)

// newPurgeCmd creates the purge command
func newPurgeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "purge [path]",
		Short: "Find and remove old project build artifacts, grouped by project",
		Long: `Scans project directories (~/Projects, ~/Code, ~/GitHub, ~/Workspace, ~/dev, and more)
for removable build artifacts - node_modules, target, .venv, Pods, vendor,
.next, .turbo, and similar - and groups them by the project they belong to.

Artifacts older than --min-age (default 7 days) are preselected. Pass a path
to scan a specific directory instead of the default project roots.`,
		Args: cobra.MaximumNArgs(1),
		RunE: runPurge,
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be purged without actually deleting")
	cmd.Flags().DurationVar(&purgeMinAge, "min-age", 7*24*time.Hour, "Preselect artifacts older than this duration")
	cmd.Flags().BoolVar(&purgeForce, "force", false, "Skip the interactive TUI and purge preselected artifacts (still asks for confirmation unless --yes is also set)")
	cmd.Flags().BoolVar(&purgeAll, "all", false, "Ignore age preselection and consider every found artifact")
	cmd.Flags().BoolVar(&purgeYes, "yes", false, "With --force, skip the confirmation prompt too (fully non-interactive)")

	return cmd
}

func runPurge(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	rep := reporter.NewReporter(verbose)

	rep.PrintHeader()

	var s *scanner.Scanner
	var err error
	if len(args) == 1 {
		s, err = scanner.NewScannerWithDirs([]string{args[0]})
	} else {
		s, err = scanner.NewScanner()
	}
	if err != nil {
		rep.PrintError(fmt.Sprintf("Failed to create scanner: %v", err))
		return err
	}

	if ignList, err := ignorelist.Load(); err == nil {
		s.SetIgnoreList(ignList)
	}

	minAge := purgeMinAge
	if purgeAll {
		minAge = 0
	}

	rep.PrintInfo("Scanning for project artifacts (this may take a while)...")

	projects, err := purge.Scan(ctx, s, minAge)
	if err != nil {
		rep.PrintError(fmt.Sprintf("Scan failed: %v", err))
		return err
	}

	if len(projects) == 0 {
		rep.PrintInfo("No project artifacts found!")
		return nil
	}

	rep.PrintPurgeReport(projects)

	if purgeForce {
		return runPurgeForce(ctx, rep, projects)
	}

	return tui.RunPurge(projects, dryRun)
}

// candidateArtifacts returns the artifacts --force would act on: every
// artifact when --all is set, otherwise only the age-preselected ones.
func candidateArtifacts(projects []purge.Project) []purge.Artifact {
	var candidates []purge.Artifact
	for _, p := range projects {
		for _, a := range p.Artifacts {
			if purgeAll || a.Selected {
				candidates = append(candidates, a)
			}
		}
	}
	return candidates
}

// runPurgeForce removes every preselected artifact (or, with --all, every
// found artifact) without launching the interactive TUI. Unless --yes is
// also set, it still asks for a single y/N confirmation before deleting
// anything - --force only skips the per-project review, not confirmation.
func runPurgeForce(ctx context.Context, rep *reporter.Reporter, projects []purge.Project) error {
	candidates := candidateArtifacts(projects)

	if len(candidates) == 0 {
		rep.PrintInfo("Nothing to purge (no preselected artifacts - try --all or a shorter --min-age)")
		return nil
	}

	if !dryRun && !purgeYes {
		var total int64
		for _, a := range candidates {
			total += a.SizeBytes
		}
		if !rep.AskConfirmation(fmt.Sprintf("Purge %d artifacts (%s)? This cannot be undone", len(candidates), utils.FormatBytes(total))) {
			rep.PrintInfo("Cancelled")
			return nil
		}
	}

	var freed int64
	var count int
	var failures int

	for _, a := range candidates {
		if err := utils.SafeRemove(a.Path, dryRun); err != nil {
			failures++
			if verbose {
				rep.PrintWarning(fmt.Sprintf("Failed to remove %s: %v", a.Path, err))
			}
			continue
		}

		freed += a.SizeBytes
		count++

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}

	if dryRun {
		rep.PrintInfo("DRY RUN - No files will be deleted")
	}

	rep.PrintSuccess(fmt.Sprintf("Purged %d artifacts, freed %s", count, utils.FormatBytes(freed)))
	if failures > 0 {
		rep.PrintWarning(fmt.Sprintf("%d artifacts failed to remove", failures))
	}

	return nil
}
