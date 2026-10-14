package engine

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/0SansNom/epurer/internal/analyzer"
	"github.com/0SansNom/epurer/internal/cleaner"
	"github.com/0SansNom/epurer/internal/config"
	"github.com/0SansNom/epurer/internal/detector"
	"github.com/0SansNom/epurer/internal/ignorelist"
	"github.com/0SansNom/epurer/internal/purge"
	"github.com/0SansNom/epurer/internal/scanner"
	"github.com/0SansNom/epurer/internal/scantask"
	"github.com/0SansNom/epurer/pkg/utils"
	"golang.org/x/sys/unix"
)

// PathCleanerName is the cleaner a host names in a CleanRequest to remove
// paths the user picked by hand (analyzer entries, project artifacts).
const PathCleanerName = "paths"

// PathCleaner removes explicit, host-chosen paths. It refuses anything on
// the ignore list; CanRemove/SafeRemove refuse protected and locked paths.
type PathCleaner struct {
	ignore   *ignorelist.List
	progress func(scantask.Progress)
}

// NewPathCleaner builds a PathCleaner. A nil list ignores nothing.
func NewPathCleaner(l *ignorelist.List) *PathCleaner {
	return &PathCleaner{ignore: l}
}

func (p *PathCleaner) Name() string                         { return PathCleanerName }
func (p *PathCleaner) Domain() config.Domain                { return config.DomainSystem }
func (p *PathCleaner) Detect(context.Context) (bool, error) { return false, nil }

// Scan finds nothing: targets always come from the host.
func (p *PathCleaner) Scan(context.Context, *config.Config) ([]cleaner.CleanTarget, error) {
	return nil, nil
}

// SetProgress implements cleaner.ProgressAware.
func (p *PathCleaner) SetProgress(fn func(scantask.Progress)) { p.progress = fn }

// Clean removes allowed targets and reports ignored ones as left in place.
func (p *PathCleaner) Clean(ctx context.Context, targets []cleaner.CleanTarget, dryRun bool) ([]cleaner.CleanResult, error) {
	var allowed []cleaner.CleanTarget
	var refused []cleaner.CleanResult
	for _, t := range targets {
		if p.ignore != nil && p.ignore.IsIgnored(t.Path) {
			refused = append(refused, cleaner.CleanResult{
				Target: t,
				Error:  fmt.Errorf("%w: on the ignore list", utils.ErrNotRemovable),
			})
			continue
		}
		allowed = append(allowed, t)
	}
	results, err := cleaner.CleanTargets(ctx, allowed, dryRun, p.progress, p.ignore)
	return append(results, refused...), err
}

// Entry is one child of an analyzed directory.
type Entry struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	SizeBytes  int64  `json:"size_bytes"`
	IsDir      bool   `json:"is_dir"`
	Incomplete bool   `json:"incomplete"`
	Removable  bool   `json:"removable"`
	Ignored    bool   `json:"ignored"`
}

// AnalyzeResponse is written by `epurer engine analyze`.
type AnalyzeResponse struct {
	Protocol   int     `json:"protocol"`
	Command    string  `json:"command"`
	DiskAccess string  `json:"disk_access"`
	Path       string  `json:"path"`
	Entries    []Entry `json:"entries"`
}

// Analyze lists path's immediate children, largest first.
// With progress set, one "progress" event per measured child goes there.
func Analyze(path string, ignore *ignorelist.List, progress io.Writer) (AnalyzeResponse, error) {
	resp := AnalyzeResponse{
		Protocol:   ProtocolVersion,
		Command:    "analyze",
		DiskAccess: diskAccessString(utils.FullDiskAccess()),
		Path:       path,
		Entries:    []Entry{},
	}
	entries, err := analyzer.ListDirFunc(path, func(e analyzer.Entry, done, total int) {
		WriteEvent(progress, ProgressEvent{Event: "progress", Phase: "analyze", Path: e.Path, Items: done, Total: total, Bytes: e.Size})
	})
	if err != nil {
		return resp, err
	}
	for _, e := range entries {
		ignored := ignore != nil && ignore.IsIgnored(e.Path)
		resp.Entries = append(resp.Entries, Entry{
			Name:       e.Name,
			Path:       e.Path,
			SizeBytes:  e.Size,
			IsDir:      e.IsDir,
			Incomplete: e.Incomplete,
			Removable:  !ignored && utils.CanRemove(e.Path),
			Ignored:    ignored,
		})
	}
	return resp, nil
}

// Artifact is one build artifact inside a project.
type Artifact struct {
	Path      string    `json:"path"`
	Kind      string    `json:"kind"`
	SizeBytes int64     `json:"size_bytes"`
	Safety    string    `json:"safety"`
	ModTime   time.Time `json:"mod_time"`
	Selected  bool      `json:"selected"`
}

// Project groups the artifacts found directly under one directory.
type Project struct {
	Root       string     `json:"root"`
	Name       string     `json:"name"`
	Type       string     `json:"type"`
	TotalBytes int64      `json:"total_bytes"`
	Artifacts  []Artifact `json:"artifacts"`
}

// PurgeResponse is written by `epurer engine purge`. It never deletes:
// the host sends chosen artifacts back through `engine clean` with the
// "paths" cleaner.
type PurgeResponse struct {
	Protocol int       `json:"protocol"`
	Command  string    `json:"command"`
	MinAge   string    `json:"min_age"`
	Roots    []string  `json:"roots"`
	Projects []Project `json:"projects"`
}

// PurgeScan finds project artifacts with s. Artifacts older than minAge are
// preselected.
// With progress set: a "roots" event (search roots, one per line in
// detail), a "found" event per artifact (project type in detail) and a
// "progress" event after each pattern (items of total).
func PurgeScan(ctx context.Context, s *scanner.Scanner, minAge time.Duration, progress io.Writer) (PurgeResponse, error) {
	resp := PurgeResponse{
		Protocol: ProtocolVersion,
		Command:  "purge",
		MinAge:   minAge.String(),
		Roots:    append([]string{}, s.GetSearchDirs()...),
		Projects: []Project{},
	}
	WriteEvent(progress, ProgressEvent{Event: "roots", Phase: "purge", Detail: strings.Join(resp.Roots, "\n")})
	found := 0
	projects, err := purge.ScanFunc(ctx, s, minAge,
		func(a purge.Artifact, projectType string) {
			found++
			WriteEvent(progress, ProgressEvent{Event: "found", Phase: "purge", Path: a.Path, Items: found, Bytes: a.SizeBytes, Detail: projectType})
		},
		func(done, total int) {
			WriteEvent(progress, ProgressEvent{Event: "progress", Phase: "purge", Items: done, Total: total})
		})
	if err != nil {
		return resp, err
	}
	for _, p := range projects {
		out := Project{Root: p.Root, Name: p.Name, Type: p.Type, TotalBytes: p.TotalSize, Artifacts: []Artifact{}}
		for _, a := range p.Artifacts {
			out.Artifacts = append(out.Artifacts, Artifact{
				Path:      a.Path,
				Kind:      a.Kind,
				SizeBytes: a.SizeBytes,
				Safety:    a.Safety.String(),
				ModTime:   a.ModTime,
				Selected:  a.Selected,
			})
		}
		resp.Projects = append(resp.Projects, out)
	}
	return resp, nil
}

// IgnoreResponse is written by every `epurer engine ignore` subcommand.
type IgnoreResponse struct {
	Protocol int      `json:"protocol"`
	Command  string   `json:"command"`
	Paths    []string `json:"paths"`
	// Skipped is "system" when an add was refused because Épurer can
	// never clean that path anyway.
	Skipped string `json:"skipped,omitempty"`
}

// OutOfReach reports a path Épurer can never clean without an admin
// password: it exists, this user cannot remove it, and cannot write inside
// it. Protecting such a path changes nothing. A missing path (an unplugged
// volume) is never out of reach, so its protection is kept.
func OutOfReach(path string) bool {
	if _, err := os.Stat(path); err != nil {
		return false
	}
	return !utils.CanRemove(path) && unix.Access(path, unix.W_OK) != nil
}

// IgnorePaths reports the current ignore list.
func IgnorePaths(l *ignorelist.List) IgnoreResponse {
	paths := []string{}
	if l != nil {
		paths = append(paths, l.Paths...)
	}
	return IgnoreResponse{Protocol: ProtocolVersion, Command: "ignore", Paths: paths}
}

// DomainStatus says whether a cleaning domain applies to this machine.
type DomainStatus struct {
	Domain   string   `json:"domain"`
	Detected bool     `json:"detected"`
	Tools    []string `json:"tools"`
	Cleaners []string `json:"cleaners"`
}

// DetectResponse is written by `epurer engine detect`.
type DetectResponse struct {
	Protocol int            `json:"protocol"`
	Command  string         `json:"command"`
	Domains  []DomainStatus `json:"domains"`
}

// Detect reports, per domain, whether any cleaner applies and which tools
// were found. Domains without cleaners are left out.
func Detect(ctx context.Context, cleaners []cleaner.Cleaner, tools detector.DetectionResult) DetectResponse {
	order := []config.Domain{
		config.DomainFrontend, config.DomainBackend, config.DomainMobile,
		config.DomainDevOps, config.DomainDataML, config.DomainAI, config.DomainSystem,
	}
	found := map[config.Domain][]string{
		config.DomainFrontend: tools.Frontend,
		config.DomainBackend:  tools.Backend,
		config.DomainMobile:   tools.Mobile,
		config.DomainDevOps:   tools.DevOps,
		config.DomainDataML:   tools.DataML,
	}

	byDomain := map[config.Domain]*DomainStatus{}
	for _, c := range cleaners {
		d := c.Domain()
		st, ok := byDomain[d]
		if !ok {
			st = &DomainStatus{Domain: d.String(), Tools: append([]string{}, found[d]...), Cleaners: []string{}}
			byDomain[d] = st
		}
		st.Cleaners = append(st.Cleaners, c.Name())
		if ok, err := c.Detect(ctx); err == nil && ok {
			st.Detected = true
		}
	}

	resp := DetectResponse{Protocol: ProtocolVersion, Command: "detect", Domains: []DomainStatus{}}
	for _, d := range order {
		if st, ok := byDomain[d]; ok {
			resp.Domains = append(resp.Domains, *st)
		}
	}
	return resp
}
