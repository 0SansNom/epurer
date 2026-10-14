package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/0SansNom/epurer/internal/config"
	"github.com/0SansNom/epurer/internal/detector"
	"github.com/0SansNom/epurer/internal/engine"
	"github.com/0SansNom/epurer/internal/ignorelist"
	"github.com/0SansNom/epurer/internal/scanner"
	"github.com/0SansNom/epurer/pkg/utils"
)

func newEngineCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "engine",
		Short: "JSON host protocol for a native UI (subprocess)",
		Long: `Machine-readable engine API for a SwiftUI (or other) host.

The host launches this binary as a subprocess. JSON goes on stdout.
Progress and human text stay off stdout so the host can parse one document.

Security posture: no privileged helper, no admin password prompt.
The process only deletes what the current user can unlink.`,
	}

	cmd.AddCommand(
		newEngineInfoCmd(),
		newEngineScanCmd(),
		newEngineCleanCmd(),
		newEngineAnalyzeCmd(),
		newEnginePurgeCmd(),
		newEngineIgnoreCmd(),
		newEngineDetectCmd(),
	)
	return cmd
}

func newEngineInfoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "info",
		Short: "Print engine version and security posture as JSON",
		RunE: func(cmd *cobra.Command, args []string) error {
			return engine.WriteJSON(os.Stdout, engine.Info())
		},
	}
}

func newEngineScanCmd() *cobra.Command {
	var level string
	var domainFlags []string
	var progress bool

	cmd := &cobra.Command{
		Use:   "scan",
		Short: "Scan and print removable vs locked targets as JSON",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			cl, err := config.ParseCleanLevel(level)
			if err != nil {
				return err
			}
			cfg := config.NewDefaultConfig()
			cfg.CleanLevel = cl
			cfg.Verbose = false
			cfg.Interactive = false

			cleaners, err := initAllCleaners()
			if err != nil {
				return err
			}
			if len(domainFlags) > 0 {
				cleaners = filterCleanersByDomain(cleaners, domainFlags)
			}
			if progress {
				engine.AttachProgress(cleaners, os.Stderr, "scan")
			}

			resp, err := engine.Scan(ctx, cleaners, cfg)
			if err != nil {
				return err
			}
			return engine.WriteJSON(os.Stdout, resp)
		},
	}

	cmd.Flags().StringVarP(&level, "level", "l", "standard", "Clean level (conservative|standard|aggressive)")
	cmd.Flags().StringSliceVarP(&domainFlags, "domain", "d", []string{}, "Domains to scan (comma-separated, empty = all)")
	cmd.Flags().BoolVar(&progress, "progress", false, "Emit NDJSON progress events on stderr")
	return cmd
}

func newEngineCleanCmd() *cobra.Command {
	var progress bool

	cmd := &cobra.Command{
		Use:   "clean",
		Short: "Read a JSON clean request on stdin and print results as JSON",
		Long: `Reads one CleanRequest JSON document from stdin.

Example:
  epurer engine scan | jq '{protocol:1,dry_run:true,targets:.targets}' | epurer engine clean

Targets with cleaner "paths" are removed as-is (user-picked paths from
analyze or purge); ignored and protected paths are left in place.

With --progress, NDJSON progress lines go to stderr; the final result stays on stdout.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			req, err := engine.DecodeCleanRequest(os.Stdin)
			if err != nil {
				return fmt.Errorf("invalid clean request: %w", err)
			}
			cleaners, err := initAllCleaners()
			if err != nil {
				return err
			}
			ignore, err := loadIgnoreList()
			if err != nil {
				return err
			}
			cleaners = append(cleaners, engine.NewPathCleaner(ignore))
			if progress {
				engine.AttachProgress(cleaners, os.Stderr, "clean")
			}
			resp, err := engine.Clean(ctx, cleaners, req, ignore)
			if err != nil {
				return err
			}
			return engine.WriteJSON(os.Stdout, resp)
		},
	}

	cmd.Flags().BoolVar(&progress, "progress", false, "Emit NDJSON progress events on stderr")
	return cmd
}

// loadIgnoreList returns the persistent ignore list. A missing file is an
// empty list; any other read/parse error fails closed so clean never runs
// without the user's protections.
func loadIgnoreList() (*ignorelist.List, error) {
	l, err := ignorelist.Load()
	if err != nil {
		return nil, fmt.Errorf("cannot load ignore list: %w", err)
	}
	return l, nil
}

func newEngineAnalyzeCmd() *cobra.Command {
	var progress bool

	cmd := &cobra.Command{
		Use:   "analyze [path]",
		Short: "List a directory's children with sizes as JSON (default: home)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := os.UserHomeDir()
			if err != nil {
				return err
			}
			if len(args) == 1 {
				path = args[0]
			}
			info, err := os.Stat(path)
			if err != nil {
				return fmt.Errorf("cannot access %s: %w", path, err)
			}
			if !info.IsDir() {
				return fmt.Errorf("%s is not a directory", path)
			}
			ignore, err := loadIgnoreList()
			if err != nil {
				return err
			}
			resp, err := engine.Analyze(path, ignore, progressWriter(progress))
			if err != nil {
				return err
			}
			return engine.WriteJSON(os.Stdout, resp)
		},
	}

	cmd.Flags().BoolVar(&progress, "progress", false, "Emit NDJSON progress events on stderr")
	return cmd
}

// progressWriter is stderr when --progress is set, nil otherwise.
func progressWriter(on bool) io.Writer {
	if !on {
		return nil
	}
	return os.Stderr
}

func newEnginePurgeCmd() *cobra.Command {
	var minAge time.Duration
	var progress bool

	cmd := &cobra.Command{
		Use:   "purge [path]",
		Short: "Find project build artifacts and print them grouped by project as JSON",
		Long: `Scans project directories (or [path]) for build artifacts. Nothing is deleted:
send chosen artifacts to "engine clean" with cleaner "paths".`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var s *scanner.Scanner
			var err error
			if len(args) == 1 {
				s, err = scanner.NewScannerWithDirs([]string{args[0]})
			} else {
				s, err = scanner.NewScanner()
			}
			if err != nil {
				return err
			}
			l, err := loadIgnoreList()
			if err != nil {
				return err
			}
			s.SetIgnoreList(l)
			resp, err := engine.PurgeScan(context.Background(), s, minAge, progressWriter(progress))
			if err != nil {
				return err
			}
			return engine.WriteJSON(os.Stdout, resp)
		},
	}

	cmd.Flags().DurationVar(&minAge, "min-age", 7*24*time.Hour, "Preselect artifacts older than this duration")
	cmd.Flags().BoolVar(&progress, "progress", false, "Emit NDJSON progress events on stderr")
	return cmd
}

func newEngineIgnoreCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ignore",
		Short: "Read or change the ignore list; always prints the resulting list as JSON",
	}

	edit := func(use, short string, apply func(*ignorelist.List, string) error) *cobra.Command {
		return &cobra.Command{
			Use:   use + " <path>",
			Short: short,
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				l, err := ignorelist.Load()
				if err != nil {
					return err
				}
				if err := apply(l, args[0]); err != nil {
					return err
				}
				return engine.WriteJSON(os.Stdout, engine.IgnorePaths(l))
			},
		}
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "list",
			Short: "Print the ignore list",
			RunE: func(cmd *cobra.Command, args []string) error {
				l, err := ignorelist.Load()
				if err != nil {
					return err
				}
				return engine.WriteJSON(os.Stdout, engine.IgnorePaths(l))
			},
		},
		&cobra.Command{
			Use:   "add <path>",
			Short: "Add a path to the ignore list (refused with skipped=system when it is out of reach anyway)",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				l, err := ignorelist.Load()
				if err != nil {
					return err
				}
				path, err := utils.ExpandHome(args[0])
				if err != nil {
					return err
				}
				if path, err = filepath.Abs(path); err != nil {
					return err
				}
				if engine.OutOfReach(path) {
					resp := engine.IgnorePaths(l)
					resp.Skipped = "system"
					return engine.WriteJSON(os.Stdout, resp)
				}
				if err := l.Add(path); err != nil {
					return err
				}
				return engine.WriteJSON(os.Stdout, engine.IgnorePaths(l))
			},
		},
		edit("remove", "Remove a path from the ignore list", (*ignorelist.List).Remove),
	)
	return cmd
}

func newEngineDetectCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "detect",
		Short: "Print which cleaning domains apply to this machine as JSON",
		RunE: func(cmd *cobra.Command, args []string) error {
			cleaners, err := initAllCleaners()
			if err != nil {
				return err
			}
			d, err := detector.NewDetector()
			if err != nil {
				return err
			}
			resp := engine.Detect(context.Background(), cleaners, d.DetectAll())
			return engine.WriteJSON(os.Stdout, resp)
		},
	}
}
