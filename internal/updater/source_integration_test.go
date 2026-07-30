package updater

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGitHubSourceChannelsAndAssetSelection(t *testing.T) {
	var sawAuth bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "Bearer secret" {
			sawAuth = true
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{"tag_name": "v1.26.7291930", "draft": false, "prerelease": false, "assets": []map[string]any{{"name": "app_linux_amd64", "browser_download_url": srvURL(r, "/stable"), "size": 10}}},
			{"tag_name": "v1.26.7300100", "draft": true, "prerelease": false, "assets": []map[string]any{{"name": "app_linux_amd64", "browser_download_url": srvURL(r, "/draft"), "size": 10}}},
		})
	}))
	defer srv.Close()
	g := &GitHubSource{Owner: "o", Repo: "r", APIBase: srv.URL, Client: srv.Client()}
	stable, err := g.List(context.Background(), ListOptions{Channel: Stable})
	if err != nil || len(stable) != 1 || stable[0].Version != "1.26.0729.1930" {
		t.Fatalf("stable: %+v %v", stable, err)
	}
	internal, err := g.List(context.Background(), ListOptions{Channel: Internal, Token: "secret"})
	if err != nil || len(internal) != 2 || !sawAuth {
		t.Fatalf("internal: %+v %v auth=%v", internal, err, sawAuth)
	}
}

func TestAzureBlobSource(t *testing.T) {
	doc := struct {
		Releases []Release `json:"releases"`
	}{Releases: []Release{{Version: "1.26.0729.1930", Assets: []Asset{{Name: "app_linux_amd64", URL: "https://example/asset"}}}}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _ = json.NewEncoder(w).Encode(doc) }))
	defer srv.Close()
	a := &AzureBlobSource{IndexURL: srv.URL, Client: srv.Client()}
	rels, err := a.List(context.Background(), ListOptions{Channel: Stable})
	if err != nil || len(rels) != 1 || rels[0].Source != "azure" {
		t.Fatalf("azure: %+v %v", rels, err)
	}
}

func srvURL(r *http.Request, path string) string { return "http://" + r.Host + path }
