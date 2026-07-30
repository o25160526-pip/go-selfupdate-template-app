package updater

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestEngineFallsBackWhenPreferredDownloadFails(t *testing.T) {
	payload := []byte("new-binary")
	sum := sha256.Sum256(payload)
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { http.Error(w, "down", http.StatusServiceUnavailable) }))
	defer bad.Close()
	good := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(payload) }))
	defer good.Close()
	version := "1.26.0729.2000"
	asset := func(url string) Release {
		return Release{Version: version, Assets: []Asset{{Name: AssetName("app", "linux", "amd64"), URL: url, SHA256: hex.EncodeToString(sum[:])}}}
	}
	target := filepath.Join(t.TempDir(), "app")
	if err := os.WriteFile(target, []byte("old-binary"), 0700); err != nil {
		t.Fatal(err)
	}
	eng := Engine{
		AppName: "app", CurrentVersion: "1.26.0729.1900", TargetPath: target, Channel: "stable",
		Sources: []Source{
			fakeSource{name: "preferred", rels: []Release{asset(bad.URL)}},
			fakeSource{name: "fallback", delay: 15 * time.Millisecond, rels: []Release{asset(good.URL)}},
		},
		Cache: Cache{Root: filepath.Join(t.TempDir(), "cache")},
	}
	result, err := eng.Update(context.Background(), UpdateOptions{Timeout: 3 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	if result.Source != "fallback" {
		t.Fatalf("source=%s failures=%+v", result.Source, result.Failures)
	}
	got, _ := os.ReadFile(target)
	if string(got) != string(payload) {
		t.Fatalf("target=%q", got)
	}
	if len(result.Failures) == 0 || result.Failures[len(result.Failures)-1].Name != "preferred" {
		t.Fatalf("missing preferred source failure: %+v", result.Failures)
	}
}

func TestForcedPolicySkipsConfirmation(t *testing.T) {
	payload := []byte("forced-binary")
	sum := sha256.Sum256(payload)
	assetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(payload) }))
	defer assetServer.Close()
	manifest := Manifest{Schema: 1, Channels: map[string]ChannelPolicy{"stable": {
		Latest: "1.26.0729.2000", MinSupported: "1.26.0701.0000", ForceUpdate: true, RolloutPercent: 100,
	}}}
	manifestBytes, _ := json.Marshal(manifest)
	manifestServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(manifestBytes) }))
	defer manifestServer.Close()
	target := filepath.Join(t.TempDir(), "app")
	_ = os.WriteFile(target, []byte("old"), 0700)
	called := false
	eng := Engine{
		AppName: "app", CurrentVersion: "1.26.0729.1900", TargetPath: target, Channel: "stable", ManifestURL: manifestServer.URL,
		Sources: []Source{fakeSource{name: "github", rels: []Release{{Version: "1.26.0729.2000", Assets: []Asset{{Name: "app_linux_amd64", URL: assetServer.URL, SHA256: hex.EncodeToString(sum[:])}}}}}},
		Cache:   Cache{Root: filepath.Join(t.TempDir(), "cache")},
	}
	result, err := eng.Update(context.Background(), UpdateOptions{Confirm: func(UpdateResult) bool { called = true; return false }})
	if err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("forced update asked for confirmation")
	}
	if !result.Forced {
		t.Fatal("result was not marked forced")
	}
}

func TestNonForcedCancellationDoesNotWriteTarget(t *testing.T) {
	payload := []byte("new")
	sum := sha256.Sum256(payload)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(payload) }))
	defer srv.Close()
	target := filepath.Join(t.TempDir(), "app")
	_ = os.WriteFile(target, []byte("old"), 0700)
	eng := Engine{
		AppName: "app", CurrentVersion: "1.26.0729.1900", TargetPath: target, Channel: "stable",
		Sources: []Source{fakeSource{name: "github", rels: []Release{{Version: "1.26.0729.2000", Assets: []Asset{{Name: "app_linux_amd64", URL: srv.URL, SHA256: hex.EncodeToString(sum[:])}}}}}},
		Cache:   Cache{Root: filepath.Join(t.TempDir(), "cache")},
	}
	_, err := eng.Update(context.Background(), UpdateOptions{Confirm: func(UpdateResult) bool { return false }})
	if !errors.Is(err, ErrCancelled) {
		t.Fatalf("err=%v", err)
	}
	got, _ := os.ReadFile(target)
	if string(got) != "old" {
		t.Fatalf("target changed: %q", got)
	}
}

func TestListWithPolicyFiltersSources(t *testing.T) {
	manifest := Manifest{Schema: 1, Channels: map[string]ChannelPolicy{"stable": {
		Latest: "1.26.0729.2000", MinSupported: "1.26.0701.0000", RolloutPercent: 100, Sources: []string{"azure"},
	}}}
	body, _ := json.Marshal(manifest)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(body) }))
	defer srv.Close()
	eng := Engine{Channel: "stable", ManifestURL: srv.URL, Sources: []Source{
		fakeSource{name: "github", rels: []Release{rel("1.26.0729.2000", "github")}},
		fakeSource{name: "azure", rels: []Release{rel("1.26.0729.2000", "azure")}},
	}}
	releases, failures, err := eng.ListWithPolicy(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(failures) != 0 || len(releases) != 1 || releases[0].Source != "azure" {
		t.Fatalf("releases=%+v failures=%+v", releases, failures)
	}
}
