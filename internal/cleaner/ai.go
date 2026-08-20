package cleaner

import (
	"context"
	"os"
	"path/filepath"

	"github.com/0SansNom/epurer/internal/config"
	"github.com/0SansNom/epurer/internal/ignorelist"
	"github.com/0SansNom/epurer/pkg/utils"
)

// AICleaner handles cache and artifact cleanup for AI coding tools
// (Claude desktop, Cursor, GitHub Copilot, Continue.dev, ChatGPT desktop).
type AICleaner struct {
	ignoreList *ignorelist.List
}

// SetIgnoreList makes this cleaner respect a persistent ignore list.
func (a *AICleaner) SetIgnoreList(l *ignorelist.List) {
	a.ignoreList = l
}

// NewAICleaner creates a new AICleaner
func NewAICleaner() (Cleaner, error) {
	return &AICleaner{}, nil
}

func (a *AICleaner) Name() string {
	return "AI"
}

func (a *AICleaner) Domain() config.Domain {
	return config.DomainAI
}

func (a *AICleaner) Detect(ctx context.Context) (bool, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return false, err
	}

	for _, path := range a.candidatePaths(home) {
		if utils.PathExists(path.path) {
			return true, nil
		}
	}

	return false, nil
}

type aiCachePath struct {
	path        string
	description string
	safety      config.SafetyLevel
}

// candidatePaths returns every known AI-tool cache/artifact location.
func (a *AICleaner) candidatePaths(home string) []aiCachePath {
	return []aiCachePath{
		{
			path:        filepath.Join(home, "Library", "Caches", "Claude"),
			description: "Claude desktop cache",
			safety:      config.Safe,
		},
		{
			path:        filepath.Join(home, "Library", "Application Support", "Claude", "Cache"),
			description: "Claude desktop application cache",
			safety:      config.Safe,
		},
		{
			path:        filepath.Join(home, ".cursor", "cache"),
			description: "Cursor cache",
			safety:      config.Safe,
		},
		{
			path:        filepath.Join(home, "Library", "Application Support", "Cursor", "Cache"),
			description: "Cursor application cache",
			safety:      config.Safe,
		},
		{
			path:        filepath.Join(home, "Library", "Application Support", "Cursor", "CachedData"),
			description: "Cursor cached data",
			safety:      config.Safe,
		},
		{
			path:        filepath.Join(home, ".config", "github-copilot"),
			description: "GitHub Copilot cache",
			safety:      config.Safe,
		},
		{
			path:        filepath.Join(home, ".continue", "index"),
			description: "Continue.dev index cache",
			safety:      config.Safe,
		},
		{
			path:        filepath.Join(home, "Library", "Application Support", "com.openai.chat", "Cache"),
			description: "ChatGPT desktop cache",
			safety:      config.Safe,
		},
	}
}

func (a *AICleaner) Scan(ctx context.Context, cfg *config.Config) ([]CleanTarget, error) {
	targets := []CleanTarget{}

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	for _, candidate := range a.candidatePaths(home) {
		if !cfg.CleanLevel.AllowsSafety(candidate.safety) {
			continue
		}

		if !utils.PathExists(candidate.path) {
			continue
		}

		size, _ := utils.GetDirSize(candidate.path)
		if size == 0 {
			continue
		}

		targets = append(targets, CleanTarget{
			Path:        candidate.path,
			Description: candidate.description,
			SizeBytes:   size,
			Safety:      candidate.safety,
		})
	}

	return filterIgnored(targets, a.ignoreList), nil
}

func (a *AICleaner) Clean(ctx context.Context, targets []CleanTarget, dryRun bool) ([]CleanResult, error) {
	return CleanTargets(ctx, targets, dryRun)
}
