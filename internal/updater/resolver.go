package updater

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	appversion "github.com/your-org/go-selfupdate-template/internal/version"
)

var ErrAllSourcesUnavailable = errors.New("all update sources unavailable")
var ErrVersionNotFound = errors.New("version not found")

type Resolver struct {
	Sources               []Source
	AppName, GOOS, GOARCH string
}

type Candidate struct {
	Release Release
	Asset   Asset
}

func (r Resolver) Resolve(ctx context.Context, opt ListOptions) (Resolved, error) {
	if len(r.Sources) == 0 {
		return Resolved{}, ErrAllSourcesUnavailable
	}
	type result struct {
		rels    []Release
		failure *SourceFailure
	}
	ch := make(chan result, len(r.Sources))
	var wg sync.WaitGroup
	for _, s := range r.Sources {
		wg.Add(1)
		go func(s Source) {
			defer wg.Done()
			start := time.Now()
			rels, err := s.List(ctx, opt)
			lat := time.Since(start)
			if err != nil {
				f := SourceFailure{Name: s.Name(), Err: err}
				ch <- result{failure: &f}
				return
			}
			for i := range rels {
				rels[i].Source = s.Name()
				rels[i].Latency = lat
			}
			ch <- result{rels: rels}
		}(s)
	}
	wg.Wait()
	close(ch)

	var all []Release
	var failures []SourceFailure
	success := 0
	for x := range ch {
		if x.failure != nil {
			failures = append(failures, *x.failure)
		} else {
			success++
			all = append(all, x.rels...)
		}
	}
	if success == 0 {
		return Resolved{Failures: failures}, fmt.Errorf("%w: %v", ErrAllSourcesUnavailable, failures)
	}

	valid := all[:0]
	for _, rel := range all {
		if _, err := appversion.Parse(rel.Version); err != nil {
			continue
		}
		if opt.Version != "" && rel.Version != opt.Version {
			continue
		}
		valid = append(valid, rel)
	}
	all = valid
	if len(all) == 0 {
		return Resolved{Failures: failures}, ErrVersionNotFound
	}

	// Highest version wins. For the same version, the source with lower list
	// latency is attempted first, while every matching source is retained as a
	// download fallback.
	sort.SliceStable(all, func(i, j int) bool {
		a, _ := appversion.Parse(all[i].Version)
		b, _ := appversion.Parse(all[j].Version)
		if c := a.Compare(b); c != 0 {
			return c > 0
		}
		return all[i].Latency < all[j].Latency
	})
	goos, goarch := r.GOOS, r.GOARCH
	if goos == "" || goarch == "" {
		return Resolved{}, fmt.Errorf("GOOS and GOARCH are required")
	}
	targetVersion := all[0].Version
	var candidates []Candidate
	for _, rel := range all {
		if rel.Version != targetVersion {
			continue
		}
		asset, err := SelectAsset(rel, r.AppName, goos, goarch)
		if err != nil {
			failures = append(failures, SourceFailure{Name: rel.Source, Err: err})
			continue
		}
		candidates = append(candidates, Candidate{Release: rel, Asset: asset})
	}
	if len(candidates) == 0 {
		return Resolved{Release: all[0], Failures: failures}, fmt.Errorf("release %s has no matching asset for %s/%s", targetVersion, goos, goarch)
	}
	chosen := candidates[0]
	return Resolved{Release: chosen.Release, Asset: chosen.Asset, Candidates: candidates, Failures: failures}, nil
}
