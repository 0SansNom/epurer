package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/0SansNom/epurer/internal/ignorelist"
	"github.com/0SansNom/epurer/internal/reporter"
)

// newIgnoreCmd creates the ignore command and its subcommands
func newIgnoreCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ignore",
		Short: "Manage the persistent ignore list",
		Long: `Protect specific paths from ever being scanned or cleaned by Épurer.
Paths added here are skipped by clean, report, smart, purge, and ui.`,
	}

	cmd.AddCommand(newIgnoreAddCmd(), newIgnoreRemoveCmd(), newIgnoreListCmd())

	return cmd
}

func newIgnoreAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add <path>",
		Short: "Add a path to the ignore list",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rep := reporter.NewReporter(verbose)

			l, err := ignorelist.Load()
			if err != nil {
				rep.PrintError(fmt.Sprintf("Failed to load ignore list: %v", err))
				return err
			}

			if err := l.Add(args[0]); err != nil {
				rep.PrintError(fmt.Sprintf("Failed to add path: %v", err))
				return err
			}

			rep.PrintSuccess(fmt.Sprintf("Added to ignore list: %s", args[0]))
			return nil
		},
	}
}

func newIgnoreRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <path>",
		Short: "Remove a path from the ignore list",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rep := reporter.NewReporter(verbose)

			l, err := ignorelist.Load()
			if err != nil {
				rep.PrintError(fmt.Sprintf("Failed to load ignore list: %v", err))
				return err
			}

			if err := l.Remove(args[0]); err != nil {
				rep.PrintError(fmt.Sprintf("Failed to remove path: %v", err))
				return err
			}

			rep.PrintSuccess(fmt.Sprintf("Removed from ignore list: %s", args[0]))
			return nil
		},
	}
}

func newIgnoreListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all ignored paths",
		RunE: func(cmd *cobra.Command, args []string) error {
			rep := reporter.NewReporter(verbose)

			l, err := ignorelist.Load()
			if err != nil {
				rep.PrintError(fmt.Sprintf("Failed to load ignore list: %v", err))
				return err
			}

			if len(l.Paths) == 0 {
				rep.PrintInfo("Ignore list is empty")
				return nil
			}

			fmt.Println()
			for _, p := range l.Paths {
				fmt.Printf("  • %s\n", p)
			}
			fmt.Println()

			return nil
		},
	}
}
