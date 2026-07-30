package updater

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// AzureBlobSource uses a static index.json hosted in Azure Blob Storage.
// This is intentionally selected over Universal Packages so public clients do
// not need Azure DevOps credentials.
type AzureBlobSource struct {
	IndexURL string
	Client   *http.Client
	Token    string
	Metadata MetadataCache
}

func (a *AzureBlobSource) Name() string { return "azure" }
func (a *AzureBlobSource) client() *http.Client {
	if a.Client != nil {
		return a.Client
	}
	return http.DefaultClient
}
func (a *AzureBlobSource) List(ctx context.Context, opt ListOptions) ([]Release, error) {
	if a.IndexURL == "" {
		return nil, fmt.Errorf("azure index URL is empty")
	}
	token := a.Token
	if token == "" {
		token = opt.Token
	}
	body, err := a.Metadata.Get(ctx, a.client(), a.IndexURL, token, nil, 8<<20)
	if err != nil {
		return nil, fmt.Errorf("azure index: %w", err)
	}
	var doc struct {
		Releases []Release `json:"releases"`
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil, err
	}
	var out []Release
	for _, r := range doc.Releases {
		if r.Draft && opt.Channel != Internal {
			continue
		}
		if r.Prerelease && opt.Channel == Stable {
			continue
		}
		if opt.Version != "" && r.Version != opt.Version {
			continue
		}
		r.Source = a.Name()
		out = append(out, r)
	}
	sortReleasesDesc(out)
	return out, nil
}
func (a *AzureBlobSource) Fetch(ctx context.Context, r Release, dst io.Writer) error {
	return fetchReleaseAsset(ctx, a.client(), r, dst, a.Token)
}
