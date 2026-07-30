package updater

import (
	"context"
	"fmt"
	"io"
	"runtime"
	"sort"
	"strings"
	"time"

	appversion "github.com/your-org/go-selfupdate-template/internal/version"
)

type Channel string

const (
	Stable   Channel = "stable"
	Beta     Channel = "beta"
	Internal Channel = "internal"
)

type ListOptions struct {
	Channel Channel
	Token   string
	Version string
}
type Asset struct {
	Name      string `json:"name"`
	URL       string `json:"url"`
	SHA256    string `json:"sha256,omitempty"`
	Signature string `json:"signature,omitempty"`
	Size      int64  `json:"size,omitempty"`
}
type Release struct {
	Version     string        `json:"version"`
	Tag         string        `json:"tag,omitempty"`
	Draft       bool          `json:"draft,omitempty"`
	Prerelease  bool          `json:"prerelease,omitempty"`
	Changelog   string        `json:"changelog,omitempty"`
	PublishedAt time.Time     `json:"published_at,omitempty"`
	Assets      []Asset       `json:"assets"`
	Source      string        `json:"source,omitempty"`
	Latency     time.Duration `json:"latency,omitempty"`
}

type Source interface {
	Name() string
	List(context.Context, ListOptions) ([]Release, error)
	Fetch(context.Context, Release, io.Writer) error
}

type SourceFailure struct {
	Name string
	Err  error
}

func (e SourceFailure) Error() string { return fmt.Sprintf("%s: %v", e.Name, e.Err) }

type Resolved struct {
	Release    Release
	Asset      Asset
	Candidates []Candidate
	Failures   []SourceFailure
}

func AssetName(app, goos, goarch string) string {
	ext := ""
	if goos == "windows" {
		ext = ".exe"
	}
	return fmt.Sprintf("%s_%s_%s%s", app, goos, goarch, ext)
}
func SelectAsset(r Release, app, goos, goarch string) (Asset, error) {
	want := AssetName(app, goos, goarch)
	for _, a := range r.Assets {
		if a.Name == want {
			return a, nil
		}
	}
	return Asset{}, fmt.Errorf("release %s has no matching asset %s", r.Version, want)
}
func SelectCurrentAsset(r Release, app string) (Asset, error) {
	return SelectAsset(r, app, runtime.GOOS, runtime.GOARCH)
}

func sortReleasesDesc(in []Release) {
	sort.SliceStable(in, func(i, j int) bool {
		a, e1 := appversion.Parse(in[i].Version)
		b, e2 := appversion.Parse(in[j].Version)
		if e1 != nil || e2 != nil {
			return strings.Compare(in[i].Version, in[j].Version) > 0
		}
		return a.Compare(b) > 0
	})
}
