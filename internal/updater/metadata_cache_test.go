package updater

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestMetadataCacheAssetFetchSendsOctetStreamAcceptWithoutToken(t *testing.T) {
	var accept string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/releases/assets/") {
			accept = r.Header.Get("Accept")
		}
		_, _ = w.Write([]byte("detached-signature"))
	}))
	defer srv.Close()
	cache := MetadataCache{Root: t.TempDir()}
	body, err := cache.Get(context.Background(), srv.Client(), srv.URL+"/releases/assets/456", "", nil, 1024)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "detached-signature" {
		t.Fatalf("body %q", body)
	}
	if accept != "application/octet-stream" {
		t.Fatalf("Accept header = %q, want application/octet-stream", accept)
	}
}

func TestMetadataCacheTTLAndETag(t *testing.T) {
	requests := 0
	conditional := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Header.Get("If-None-Match") == `"v1"` {
			conditional++
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Set("ETag", `"v1"`)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()
	cache := MetadataCache{Root: t.TempDir(), TTL: 20 * time.Millisecond}
	first, err := cache.Get(context.Background(), srv.Client(), srv.URL, "", nil, 1024)
	if err != nil {
		t.Fatal(err)
	}
	second, err := cache.Get(context.Background(), srv.Client(), srv.URL, "", nil, 1024)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) || requests != 1 {
		t.Fatalf("fresh cache requests=%d first=%s second=%s", requests, first, second)
	}
	time.Sleep(30 * time.Millisecond)
	third, err := cache.Get(context.Background(), srv.Client(), srv.URL, "", nil, 1024)
	if err != nil {
		t.Fatal(err)
	}
	if string(third) != string(first) || requests != 2 || conditional != 1 {
		t.Fatalf("etag revalidation requests=%d conditional=%d third=%s", requests, conditional, third)
	}
}
