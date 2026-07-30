package updater

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"
)

type fakeSource struct {
	name  string
	rels  []Release
	err   error
	delay time.Duration
}

func (f fakeSource) Name() string { return f.name }
func (f fakeSource) List(context.Context, ListOptions) ([]Release, error) {
	time.Sleep(f.delay)
	return f.rels, f.err
}
func (f fakeSource) Fetch(context.Context, Release, io.Writer) error { return nil }
func rel(v, src string) Release {
	return Release{Version: v, Source: src, Assets: []Asset{{Name: "app_linux_amd64", URL: "https://x/" + src}}}
}
func TestResolveLatestAndFallback(t *testing.T) {
	r := Resolver{AppName: "app", GOOS: "linux", GOARCH: "amd64", Sources: []Source{fakeSource{name: "dead", err: errors.New("down")}, fakeSource{name: "ok", rels: []Release{rel("1.26.0729.1930", "ok"), rel("1.26.0730.0100", "ok")}}}}
	x, err := r.Resolve(context.Background(), ListOptions{Channel: Stable})
	if err != nil {
		t.Fatal(err)
	}
	if x.Release.Version != "1.26.0730.0100" || len(x.Failures) != 1 {
		t.Fatalf("bad result %+v", x)
	}
}
func TestResolveFastestForSameVersion(t *testing.T) {
	r := Resolver{AppName: "app", GOOS: "linux", GOARCH: "amd64", Sources: []Source{fakeSource{name: "slow", delay: 20 * time.Millisecond, rels: []Release{rel("1.26.0730.0100", "slow")}}, fakeSource{name: "fast", rels: []Release{rel("1.26.0730.0100", "fast")}}}}
	x, err := r.Resolve(context.Background(), ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if x.Release.Source != "fast" {
		t.Fatalf("got %s", x.Release.Source)
	}
}
func TestResolveAllDead(t *testing.T) {
	r := Resolver{AppName: "app", GOOS: "linux", GOARCH: "amd64", Sources: []Source{fakeSource{name: "x", err: errors.New("down")}}}
	_, err := r.Resolve(context.Background(), ListOptions{})
	if !errors.Is(err, ErrAllSourcesUnavailable) {
		t.Fatalf("got %v", err)
	}
}
