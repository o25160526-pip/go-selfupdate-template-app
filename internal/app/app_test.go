package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestVersionJSON(t *testing.T) {
	var out, errOut bytes.Buffer
	code := Run([]string{"version", "--json"}, strings.NewReader(""), &out, &errOut)
	if code != 0 {
		t.Fatalf("code=%d err=%s", code, errOut.String())
	}
	var doc map[string]string
	if err := json.Unmarshal(out.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	if doc["version"] == "" || doc["tag"] == "" {
		t.Fatalf("bad output %v", doc)
	}
}

func TestUpdateDryRunFromGitHub(t *testing.T) {
	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}
	asset := fmt.Sprintf("%s_%s_%s%s", appName, runtime.GOOS, runtime.GOARCH, ext)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]any{{
			"tag_name": "v1.26.7291930", "draft": false, "prerelease": false,
			"assets": []map[string]any{{"name": asset, "browser_download_url": "https://example.invalid/asset", "size": 10}},
		}})
	}))
	defer srv.Close()
	t.Setenv("APP_GITHUB_OWNER", "test")
	t.Setenv("APP_GITHUB_REPO", "app")
	t.Setenv("APP_GITHUB_API", srv.URL)
	t.Setenv("APP_CACHE_DIR", filepath.Join(t.TempDir(), "cache"))
	var out, errOut bytes.Buffer
	code := Run([]string{"--config", filepath.Join(t.TempDir(), "missing.yaml"), "update", "--silent", "--dry-run", "--version", "1.26.0729.1930"}, strings.NewReader(""), &out, &errOut)
	if code != 0 {
		t.Fatalf("code=%d out=%s err=%s", code, out.String(), errOut.String())
	}
	if !strings.Contains(out.String(), `"to": "1.26.0729.1930"`) {
		t.Fatalf("unexpected output %s", out.String())
	}
}

func TestInternalChannelRequiresToken(t *testing.T) {
	t.Setenv("APP_UPDATE_TOKEN", "")
	var out, errOut bytes.Buffer
	code := Run([]string{"update", "--silent", "--dry-run", "--channel", "internal"}, strings.NewReader(""), &out, &errOut)
	if code != 2 || !strings.Contains(errOut.String(), "requires APP_UPDATE_TOKEN") {
		t.Fatalf("code=%d err=%s", code, errOut.String())
	}
}

func TestUpdateRejectsLatestAndVersionTogether(t *testing.T) {
	var out, errOut bytes.Buffer
	code := Run([]string{"update", "--latest", "--version", "1.26.0729.2000", "--silent"}, strings.NewReader(""), &out, &errOut)
	if code != 2 || !strings.Contains(errOut.String(), "mutually exclusive") {
		t.Fatalf("code=%d out=%q err=%q", code, out.String(), errOut.String())
	}
}
