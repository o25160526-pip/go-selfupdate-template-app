package updater

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	appversion "github.com/your-org/go-selfupdate-template/internal/version"
)

const (
	ExitUpdated            = 0
	ExitUpToDate           = 10
	ExitNotFound           = 20
	ExitVerify             = 30
	ExitApplyRollback      = 40
	ExitSourcesUnavailable = 50
)

type ExitError struct {
	Code int
	Err  error
}

func (e *ExitError) Error() string { return e.Err.Error() }
func (e *ExitError) Unwrap() error { return e.Err }

var ErrCancelled = errors.New("update cancelled")

type Engine struct {
	AppName, CurrentVersion, TargetPath, Channel, Token, ManifestURL, MachineID string
	PublicKeys                                                                  []string
	Sources                                                                     []Source
	Cache                                                                       Cache
	Metadata                                                                    MetadataCache
	Client                                                                      *http.Client
	Log                                                                         io.Writer
}
type UpdateOptions struct {
	Version string
	DryRun  bool
	Timeout time.Duration
	// Confirm is called after policy evaluation and before downloading. It is
	// skipped for forced updates and dry-runs.
	Confirm func(UpdateResult) bool
}
type UpdateResult struct {
	From      string          `json:"from"`
	To        string          `json:"to"`
	Source    string          `json:"source"`
	Asset     string          `json:"asset"`
	CachePath string          `json:"cache_path,omitempty"`
	DryRun    bool            `json:"dry_run"`
	Forced    bool            `json:"forced"`
	Failures  []SourceFailure `json:"failures,omitempty"`
}

func (e Engine) logger() io.Writer {
	if e.Log != nil {
		return e.Log
	}
	return io.Discard
}
func (e Engine) List(ctx context.Context) ([]Release, []SourceFailure) {
	return e.listFrom(ctx, e.Sources)
}

// ListWithPolicy verifies the manifest and honors its per-channel source allow
// list before exposing releases to list/menu/prefetch callers.
func (e Engine) ListWithPolicy(ctx context.Context) ([]Release, []SourceFailure, error) {
	sources := e.Sources
	if e.ManifestURL != "" {
		manifest, err := e.fetchManifest(ctx)
		if err != nil {
			return nil, nil, err
		}
		policy, ok := manifest.Channels[e.Channel]
		if !ok {
			return nil, nil, fmt.Errorf("unknown manifest channel %q", e.Channel)
		}
		if len(policy.Sources) > 0 {
			sources = filterSources(sources, policy.Sources)
			if len(sources) == 0 {
				return nil, nil, fmt.Errorf("manifest permits no configured sources: %v", policy.Sources)
			}
		}
	}
	returnValues, failures := e.listFrom(ctx, sources)
	return returnValues, failures, nil
}

func (e Engine) listFrom(ctx context.Context, sources []Source) ([]Release, []SourceFailure) {
	type x struct {
		r []Release
		f *SourceFailure
	}
	ch := make(chan x, len(sources))
	for _, s := range sources {
		go func(s Source) {
			start := time.Now()
			r, err := s.List(ctx, ListOptions{Channel: Channel(e.Channel), Token: e.Token})
			if err != nil {
				f := SourceFailure{Name: s.Name(), Err: err}
				ch <- x{f: &f}
				return
			}
			for i := range r {
				r[i].Source = s.Name()
				r[i].Latency = time.Since(start)
			}
			ch <- x{r: r}
		}(s)
	}
	var out []Release
	var fail []SourceFailure
	for range sources {
		v := <-ch
		if v.f != nil {
			fail = append(fail, *v.f)
		} else {
			out = append(out, v.r...)
		}
	}
	sortReleasesDesc(out)
	return out, fail
}

func (e Engine) Update(parent context.Context, opt UpdateOptions) (UpdateResult, error) {
	timeout := opt.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()

	requestedVersion := opt.Version
	sources := e.Sources
	var policy *Manifest
	if e.ManifestURL != "" {
		m, err := e.fetchManifest(ctx)
		if err != nil {
			return UpdateResult{}, &ExitError{Code: ExitVerify, Err: err}
		}
		policy = &m
		p, ok := m.Channels[e.Channel]
		if !ok {
			return UpdateResult{}, &ExitError{Code: ExitVerify, Err: fmt.Errorf("unknown manifest channel %q", e.Channel)}
		}
		if requestedVersion == "" {
			requestedVersion = p.Latest
		}
		if len(p.Sources) > 0 {
			sources = filterSources(e.Sources, p.Sources)
			if len(sources) == 0 {
				return UpdateResult{}, &ExitError{Code: ExitSourcesUnavailable, Err: fmt.Errorf("manifest permits no configured sources: %v", p.Sources)}
			}
		}
	}

	res, err := (Resolver{Sources: sources, AppName: e.AppName, GOOS: runtime.GOOS, GOARCH: runtime.GOARCH}).Resolve(ctx, ListOptions{Channel: Channel(e.Channel), Token: e.Token, Version: requestedVersion})
	if err != nil {
		if errors.Is(err, ErrAllSourcesUnavailable) {
			return UpdateResult{}, &ExitError{Code: ExitSourcesUnavailable, Err: err}
		}
		return UpdateResult{}, &ExitError{Code: ExitNotFound, Err: err}
	}
	result := UpdateResult{From: e.CurrentVersion, To: res.Release.Version, Source: res.Release.Source, Asset: res.Asset.Name, Failures: res.Failures, DryRun: opt.DryRun}
	if policy != nil {
		decision, err := policy.Evaluate(e.Channel, e.CurrentVersion, res.Release.Version, e.machineID())
		result.Forced = decision.Forced
		if err != nil {
			return result, &ExitError{Code: ExitVerify, Err: err}
		}
	}
	cmp, err := appversion.Compare(res.Release.Version, e.CurrentVersion)
	if err != nil {
		return result, &ExitError{Code: ExitNotFound, Err: err}
	}
	if cmp == 0 {
		return result, &ExitError{Code: ExitUpToDate, Err: fmt.Errorf("already at latest version %s", e.CurrentVersion)}
	}
	if opt.DryRun {
		return result, nil
	}
	if !result.Forced && opt.Confirm != nil && !opt.Confirm(result) {
		return result, ErrCancelled
	}

	candidates := res.Candidates
	if len(candidates) == 0 {
		candidates = []Candidate{{Release: res.Release, Asset: res.Asset}}
	}
	var lastDownload error
	for _, candidate := range candidates {
		asset, err := e.enrichAsset(ctx, candidate.Release, candidate.Asset)
		if err != nil {
			// Missing or invalid integrity metadata is a release integrity failure,
			// not a transient mirror outage.
			return result, &ExitError{Code: ExitVerify, Err: err}
		}
		staging := filepath.Join(e.Cache.Root, "staging", candidate.Release.Source, asset.Name)
		d := Downloader{Client: e.Client, Retries: 3, Token: e.Token}
		sha, err := d.Download(ctx, asset, staging, e.PublicKeys)
		if err != nil {
			if errors.Is(err, ErrVerification) {
				return result, &ExitError{Code: ExitVerify, Err: err}
			}
			lastDownload = err
			result.Failures = append(result.Failures, SourceFailure{Name: candidate.Release.Source, Err: err})
			continue
		}
		cached, err := e.Cache.Import(staging, sha, candidate.Release.Version, asset.Name)
		if err != nil {
			return result, &ExitError{Code: ExitApplyRollback, Err: err}
		}
		result.Source = candidate.Release.Source
		result.Asset = asset.Name
		result.CachePath = cached
		lock := filepath.Join(e.Cache.Root, "update.lock")
		backup := e.TargetPath + ".rollback"
		if err := WithLock(lock, func() error { return ApplyBinary(cached, e.TargetPath, backup) }); err != nil {
			return result, &ExitError{Code: ExitApplyRollback, Err: err}
		}
		fmt.Fprintf(e.logger(), "%s updated %s -> %s from %s\n", time.Now().UTC().Format(time.RFC3339), e.CurrentVersion, candidate.Release.Version, candidate.Release.Source)
		return result, nil
	}
	if lastDownload == nil {
		lastDownload = ErrAllSourcesUnavailable
	}
	return result, &ExitError{Code: ExitSourcesUnavailable, Err: fmt.Errorf("all candidate downloads failed: %w", lastDownload)}
}

func filterSources(all []Source, allowed []string) []Source {
	set := make(map[string]struct{}, len(allowed))
	for _, name := range allowed {
		set[strings.ToLower(strings.TrimSpace(name))] = struct{}{}
	}
	out := make([]Source, 0, len(all))
	for _, source := range all {
		if _, ok := set[strings.ToLower(source.Name())]; ok {
			out = append(out, source)
		}
	}
	return out
}

func (e Engine) fetchManifest(ctx context.Context) (Manifest, error) {
	b, err := e.Metadata.Get(ctx, e.client(), e.ManifestURL, "", nil, 2<<20)
	if err != nil {
		return Manifest{}, err
	}
	m, err := ParseManifest(b)
	if err != nil {
		return m, err
	}
	if len(e.PublicKeys) > 0 {
		if err := m.Verify(e.PublicKeys); err != nil {
			return m, err
		}
	}
	return m, nil
}

func (e Engine) client() *http.Client {
	if e.Client != nil {
		return e.Client
	}
	return http.DefaultClient
}
func (e Engine) machineID() string {
	if e.MachineID != "" {
		return e.MachineID
	}
	for _, p := range []string{"/etc/machine-id", "/var/lib/dbus/machine-id"} {
		if b, err := os.ReadFile(p); err == nil && strings.TrimSpace(string(b)) != "" {
			return strings.TrimSpace(string(b))
		}
	}
	h, _ := os.Hostname()
	return h
}
func ResultJSON(v any) string { b, _ := json.MarshalIndent(v, "", "  "); return string(b) }

// PrepareAsset resolves checksum and detached signature metadata for an asset.
func (e Engine) PrepareAsset(ctx context.Context, release Release, asset Asset) (Asset, error) {
	return e.enrichAsset(ctx, release, asset)
}

func (e Engine) enrichAsset(ctx context.Context, release Release, asset Asset) (Asset, error) {
	if asset.SHA256 == "" {
		for _, meta := range release.Assets {
			if meta.Name != "checksums.txt" && !strings.HasPrefix(meta.Name, "checksums_") {
				continue
			}
			b, err := e.fetchSmall(ctx, meta.URL, 2<<20)
			if err != nil {
				continue
			}
			for _, line := range strings.Split(string(b), "\n") {
				fields := strings.Fields(line)
				if len(fields) < 2 {
					continue
				}
				name := strings.TrimPrefix(fields[len(fields)-1], "*")
				if name == asset.Name {
					asset.SHA256 = fields[0]
					break
				}
			}
			if asset.SHA256 != "" {
				break
			}
		}
	}
	if asset.Signature == "" {
		for _, meta := range release.Assets {
			if meta.Name != asset.Name+".sig" {
				continue
			}
			b, err := e.fetchSmall(ctx, meta.URL, 128<<10)
			if err != nil {
				return asset, err
			}
			asset.Signature = strings.TrimSpace(string(b))
			break
		}
	}
	if asset.SHA256 == "" {
		return asset, fmt.Errorf("%w: no SHA256 digest or checksums.txt entry for %s", ErrVerification, asset.Name)
	}
	if len(e.PublicKeys) > 0 && asset.Signature == "" {
		return asset, fmt.Errorf("%w: missing detached Ed25519 signature for %s", ErrVerification, asset.Name)
	}
	return asset, nil
}

func (e Engine) fetchSmall(ctx context.Context, url string, limit int64) ([]byte, error) {
	return e.Metadata.Get(ctx, e.client(), url, e.Token, nil, limit)
}
