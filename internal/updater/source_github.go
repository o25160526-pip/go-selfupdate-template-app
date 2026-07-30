package updater

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	appversion "github.com/your-org/go-selfupdate-template/internal/version"
)

type GitHubSource struct {
	Owner, Repo, APIBase string
	Client               *http.Client
	Metadata             MetadataCache
}

func (g *GitHubSource) Name() string { return "github" }
func (g *GitHubSource) client() *http.Client {
	if g.Client != nil {
		return g.Client
	}
	return http.DefaultClient
}
func (g *GitHubSource) List(ctx context.Context, opt ListOptions) ([]Release, error) {
	if opt.Channel == Internal && opt.Token == "" {
		return nil, fmt.Errorf("internal channel requires APP_UPDATE_TOKEN")
	}
	base := normalizeGitHubAPIBase(g.APIBase)
	u := fmt.Sprintf("%s/repos/%s/%s/releases?per_page=100", base, g.Owner, g.Repo)
	headers := make(http.Header)
	headers.Set("Accept", "application/vnd.github+json")
	headers.Set("X-GitHub-Api-Version", "2022-11-28")
	body, err := g.Metadata.Get(ctx, g.client(), u, opt.Token, headers, 8<<20)
	if err != nil {
		return nil, fmt.Errorf("github releases: %w", err)
	}
	var raw []struct {
		TagName     string    `json:"tag_name"`
		Draft       bool      `json:"draft"`
		Prerelease  bool      `json:"prerelease"`
		Body        string    `json:"body"`
		PublishedAt time.Time `json:"published_at"`
		Assets      []struct {
			Name       string `json:"name"`
			APIURL     string `json:"url"`
			BrowserURL string `json:"browser_download_url"`
			Size       int64  `json:"size"`
			Digest     string `json:"digest"`
		} `json:"assets"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}
	var out []Release
	for _, x := range raw {
		if x.Draft && opt.Channel != Internal {
			continue
		}
		if x.Prerelease && opt.Channel == Stable {
			continue
		}
		v, err := appversion.Parse(x.TagName)
		if err != nil {
			continue
		}
		if opt.Version != "" && v.String() != opt.Version {
			continue
		}
		r := Release{Version: v.String(), Tag: x.TagName, Draft: x.Draft, Prerelease: x.Prerelease, Changelog: x.Body, PublishedAt: x.PublishedAt, Source: g.Name()}
		for _, a := range x.Assets {
			sha := strings.TrimPrefix(a.Digest, "sha256:")
			assetURL := normalizeGitHubAssetAPIURL(a.APIURL, a.BrowserURL)
			r.Assets = append(r.Assets, Asset{Name: a.Name, URL: assetURL, Size: a.Size, SHA256: sha})
		}
		out = append(out, r)
	}
	sortReleasesDesc(out)
	return out, nil
}
func (g *GitHubSource) Fetch(ctx context.Context, r Release, dst io.Writer) error {
	return fetchReleaseAsset(ctx, g.client(), r, dst, "")
}

func normalizeGitHubAPIBase(raw string) string {
	base := strings.TrimRight(strings.TrimSpace(raw), "/")
	if base == "" {
		return "https://api.github.com"
	}
	u, err := url.Parse(base)
	if err == nil && (u.Host == "github.com" || u.Host == "www.github.com") {
		u.Scheme = "https"
		u.Host = "api.github.com"
		if u.Path == "" || u.Path == "/" {
			u.Path = ""
		}
		return strings.TrimRight(u.String(), "/")
	}
	return base
}

func normalizeGitHubAssetAPIURL(apiURL, browserURL string) string {
	if strings.TrimSpace(apiURL) != "" {
		return apiURL
	}
	u, err := url.Parse(browserURL)
	if err != nil || (u.Host != "github.com" && u.Host != "www.github.com") {
		return browserURL
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	for i := range parts {
		if parts[i] == "releases" && i+2 < len(parts) && parts[i+1] == "assets" {
			return fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/assets/%s", parts[0], parts[1], parts[i+2])
		}
	}
	return browserURL
}
