// Package engine exposes a JSON host protocol for the Épurer macOS app.
// The GUI should launch the epurer binary as a subprocess and speak this
// protocol on stdin/stdout. The engine never elevates and never installs
// a privileged helper.
package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/0SansNom/epurer/internal/cleaner"
	"github.com/0SansNom/epurer/internal/config"
	"github.com/0SansNom/epurer/internal/ignorelist"
	"github.com/0SansNom/epurer/internal/scantask"
	"github.com/0SansNom/epurer/internal/version"
	"github.com/0SansNom/epurer/pkg/utils"
)

// ProtocolVersion is bumped when the JSON shape changes incompatibly.
const ProtocolVersion = 1

// Target is one cleanable item exchanged with the host UI.
type Target struct {
	Path        string `json:"path"`
	Description string `json:"description"`
	SizeBytes   int64  `json:"size_bytes"`
	Safety      string `json:"safety"`
	Domain      string `json:"domain"`
	Cleaner     string `json:"cleaner"`
}

// Result is the outcome of cleaning one target.
type Result struct {
	Path       string `json:"path"`
	Success    bool   `json:"success"`
	BytesFreed int64  `json:"bytes_freed"`
	Error      string `json:"error,omitempty"`
	Status     string `json:"status"` // ok | failed | left_in_place
}

// InfoResponse describes the engine's security posture for the host UI.
type InfoResponse struct {
	Protocol          int      `json:"protocol"`
	Command           string   `json:"command"`
	Version           string   `json:"version"`
	PrivilegedHelper  bool     `json:"privileged_helper"`
	AsksAdminPassword bool     `json:"asks_admin_password"`
	DiskAccess        string   `json:"disk_access"`
	HardenedUserSpace bool     `json:"hardened_user_space"`
	SupportedCommands []string `json:"supported_commands"`
}

// ScanResponse is written by `epurer engine scan`.
type ScanResponse struct {
	Protocol   int      `json:"protocol"`
	Command    string   `json:"command"`
	DiskAccess string   `json:"disk_access"`
	Targets    []Target `json:"targets"`
	Locked     []Target `json:"locked"`
}

// CleanRequest is read by `epurer engine clean` from stdin.
type CleanRequest struct {
	Protocol int      `json:"protocol"`
	DryRun   bool     `json:"dry_run"`
	Targets  []Target `json:"targets"`
}

// CleanResponse is written by `epurer engine clean`.
type CleanResponse struct {
	Protocol int      `json:"protocol"`
	Command  string   `json:"command"`
	Results  []Result `json:"results"`
}

// ProgressEvent is one NDJSON line on stderr while the host watches a job.
type ProgressEvent struct {
	Protocol int    `json:"protocol"`
	Event    string `json:"event"` // progress | found | roots
	Phase    string `json:"phase"` // scan | clean | analyze | purge
	Path     string `json:"path"`
	Items    int    `json:"items"`
	Bytes    int64  `json:"bytes"`
	Total    int    `json:"total,omitempty"`
	Detail   string `json:"detail,omitempty"`
}

// WriteEvent writes one compact event line; a nil writer is a no-op.
func WriteEvent(w io.Writer, e ProgressEvent) {
	if w == nil {
		return
	}
	e.Protocol = ProtocolVersion
	_ = json.NewEncoder(w).Encode(e)
}

// WriteJSON encodes v as a single JSON document on w.
func WriteJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// WriteProgressLine writes one compact JSON object followed by a newline.
func WriteProgressLine(w io.Writer, phase string, p scantask.Progress) error {
	if w == nil {
		return nil
	}
	enc := json.NewEncoder(w)
	return enc.Encode(ProgressEvent{
		Protocol: ProtocolVersion,
		Event:    "progress",
		Phase:    phase,
		Path:     p.Path,
		Items:    p.Items,
		Bytes:    p.Bytes,
	})
}

// AttachProgress wires ProgressAware cleaners to emit events for phase.
func AttachProgress(cleaners []cleaner.Cleaner, w io.Writer, phase string) {
	if w == nil {
		return
	}
	for _, c := range cleaners {
		watcher, ok := c.(cleaner.ProgressAware)
		if !ok {
			continue
		}
		watcher.SetProgress(func(p scantask.Progress) {
			_ = WriteProgressLine(w, phase, p)
		})
	}
}

// Info builds the static security posture payload.
func Info() InfoResponse {
	return InfoResponse{
		Protocol:          ProtocolVersion,
		Command:           "info",
		Version:           version.Version,
		PrivilegedHelper:  false,
		AsksAdminPassword: false,
		DiskAccess:        diskAccessString(utils.FullDiskAccess()),
		HardenedUserSpace: true,
		SupportedCommands: []string{"info", "scan", "clean", "analyze", "purge", "ignore", "detect"},
	}
}

// Scan runs every cleaner and splits removable vs locked targets.
func Scan(ctx context.Context, cleaners []cleaner.Cleaner, cfg *config.Config) (ScanResponse, error) {
	resp := ScanResponse{
		Protocol:   ProtocolVersion,
		Command:    "scan",
		DiskAccess: diskAccessString(utils.FullDiskAccess()),
		Targets:    []Target{},
		Locked:     []Target{},
	}

	for _, c := range cleaners {
		ok, err := c.Detect(ctx)
		if err != nil || !ok {
			continue
		}
		found, err := c.Scan(ctx, cfg)
		if err != nil {
			return resp, fmt.Errorf("%s: %w", c.Name(), err)
		}
		domain := c.Domain().String()
		name := c.Name()
		for _, t := range found {
			item := toTarget(t, domain, name)
			if utils.CanRemove(t.Path) {
				resp.Targets = append(resp.Targets, item)
			} else {
				resp.Locked = append(resp.Locked, item)
			}
		}
	}
	return resp, nil
}

// Clean executes a host-supplied target list. Targets are grouped by cleaner name.
// Each filesystem path must pass the ignore list and CanRemove before the
// cleaner runs; pseudo targets (system:…, docker:…) are left to their cleaner.
func Clean(ctx context.Context, cleaners []cleaner.Cleaner, req CleanRequest, ignore *ignorelist.List) (CleanResponse, error) {
	resp := CleanResponse{
		Protocol: ProtocolVersion,
		Command:  "clean",
		Results:  []Result{},
	}
	if req.Protocol != ProtocolVersion {
		return resp, fmt.Errorf("unsupported protocol %d (want %d)", req.Protocol, ProtocolVersion)
	}

	byCleaner := map[string][]Target{}
	order := []string{}
	for _, t := range req.Targets {
		if _, seen := byCleaner[t.Cleaner]; !seen {
			order = append(order, t.Cleaner)
		}
		byCleaner[t.Cleaner] = append(byCleaner[t.Cleaner], t)
	}

	index := map[string]cleaner.Cleaner{}
	for _, c := range cleaners {
		index[c.Name()] = c
	}

	for _, name := range order {
		c, ok := index[name]
		if !ok {
			for _, t := range byCleaner[name] {
				resp.Results = append(resp.Results, Result{
					Path:    t.Path,
					Success: false,
					Error:   fmt.Sprintf("unknown cleaner %q", name),
					Status:  "failed",
				})
			}
			continue
		}
		targets := make([]cleaner.CleanTarget, 0, len(byCleaner[name]))
		for _, t := range byCleaner[name] {
			if ignore != nil && ignore.IsIgnored(t.Path) {
				resp.Results = append(resp.Results, Result{
					Path:    t.Path,
					Success: false,
					Error:   fmt.Sprintf("%s: on the ignore list", utils.ErrNotRemovable.Error()),
					Status:  "left_in_place",
				})
				continue
			}
			if !utils.CanRemove(t.Path) {
				resp.Results = append(resp.Results, Result{
					Path:    t.Path,
					Success: false,
					Error:   utils.ErrNotRemovable.Error(),
					Status:  "left_in_place",
				})
				continue
			}
			ct, err := fromTarget(t)
			if err != nil {
				resp.Results = append(resp.Results, Result{
					Path:    t.Path,
					Success: false,
					Error:   err.Error(),
					Status:  "failed",
				})
				continue
			}
			targets = append(targets, ct)
		}
		if len(targets) == 0 {
			continue
		}
		results, err := c.Clean(ctx, targets, req.DryRun)
		if err != nil {
			return resp, err
		}
		for _, r := range results {
			resp.Results = append(resp.Results, fromCleanResult(r))
		}
	}
	return resp, nil
}

// DecodeCleanRequest reads a CleanRequest from r.
func DecodeCleanRequest(r io.Reader) (CleanRequest, error) {
	var req CleanRequest
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		return req, err
	}
	return req, nil
}

func toTarget(t cleaner.CleanTarget, domain, name string) Target {
	return Target{
		Path:        t.Path,
		Description: t.Description,
		SizeBytes:   t.SizeBytes,
		Safety:      t.Safety.String(),
		Domain:      domain,
		Cleaner:     name,
	}
}

func fromTarget(t Target) (cleaner.CleanTarget, error) {
	safety, err := parseSafety(t.Safety)
	if err != nil {
		return cleaner.CleanTarget{}, err
	}
	return cleaner.CleanTarget{
		Path:        t.Path,
		Description: t.Description,
		SizeBytes:   t.SizeBytes,
		Safety:      safety,
	}, nil
}

func fromCleanResult(r cleaner.CleanResult) Result {
	out := Result{
		Path:       r.Target.Path,
		Success:    r.Success,
		BytesFreed: r.BytesFreed,
		Status:     "ok",
	}
	if r.Success {
		return out
	}
	if r.Error != nil {
		out.Error = r.Error.Error()
	}
	if errors.Is(r.Error, utils.ErrNotRemovable) || os.IsPermission(r.Error) {
		out.Status = "left_in_place"
	} else {
		out.Status = "failed"
	}
	return out
}

func parseSafety(s string) (config.SafetyLevel, error) {
	switch s {
	case "", "Safe":
		return config.Safe, nil
	case "Moderate":
		return config.Moderate, nil
	case "Dangerous":
		return config.Dangerous, nil
	default:
		return config.Safe, fmt.Errorf("invalid safety %q", s)
	}
}

func diskAccessString(a utils.DiskAccess) string {
	switch a {
	case utils.DiskAccessGranted:
		return "granted"
	case utils.DiskAccessDenied:
		return "denied"
	default:
		return "unknown"
	}
}
