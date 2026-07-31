package updater

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/your-org/go-selfupdate-template/internal/signing"
)

func TestDownloadResumeVerify(t *testing.T) {
	data := []byte(strings.Repeat("abc", 1000))
	sum := sha256.Sum256(data)
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	sig, err := signing.Sign(priv, data, time.Unix(123, 0))
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := 0
		if h := r.Header.Get("Range"); h != "" {
			fmt.Sscanf(h, "bytes=%d-", &start)
			w.WriteHeader(http.StatusPartialContent)
		}
		_, _ = w.Write(data[start:])
	}))
	defer srv.Close()
	dst := filepath.Join(t.TempDir(), "blob")
	_ = os.WriteFile(dst+".part", data[:111], 0600)
	state, _ := json.Marshal(downloadState{URL: srv.URL, Bytes: 111})
	_ = os.WriteFile(dst+".download.state.json", state, 0600)
	d := Downloader{Client: srv.Client()}
	got, err := d.Download(context.Background(), Asset{URL: srv.URL, SHA256: hex.EncodeToString(sum[:]), Signature: sig}, dst, []string{signing.EncodePublicKey(pub)})
	if err != nil {
		t.Fatal(err)
	}
	if got != hex.EncodeToString(sum[:]) {
		t.Fatal(got)
	}
	b, _ := os.ReadFile(dst)
	if string(b) != string(data) {
		t.Fatal("data mismatch")
	}
}
func TestDownloadRestartsOnRangeNotSatisfiable(t *testing.T) {
	data := []byte(strings.Repeat("abc", 1000))
	sum := sha256.Sum256(data)
	var seenRanges []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenRanges = append(seenRanges, r.Header.Get("Range"))
		if r.Header.Get("Range") == "bytes=3000-" {
			w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
			return
		}
		_, _ = w.Write(data)
	}))
	defer srv.Close()
	dst := filepath.Join(t.TempDir(), "blob")
	_ = os.WriteFile(dst+".part", data[:3000], 0600)
	state, _ := json.Marshal(downloadState{URL: srv.URL, Bytes: 3000})
	_ = os.WriteFile(dst+".download.state.json", state, 0600)
	d := Downloader{Client: srv.Client(), Retries: 1}
	got, err := d.Download(context.Background(), Asset{URL: srv.URL, SHA256: hex.EncodeToString(sum[:])}, dst, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != hex.EncodeToString(sum[:]) {
		t.Fatalf("hash %s", got)
	}
	if len(seenRanges) != 2 || seenRanges[0] != "bytes=3000-" || seenRanges[1] != "" {
		t.Fatalf("expected resume-then-restart, got %v", seenRanges)
	}
}

func TestDownloadSendsOctetStreamAcceptWithoutToken(t *testing.T) {
	data := []byte("binary-content")
	sum := sha256.Sum256(data)
	var accept string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/releases/assets/") {
			accept = r.Header.Get("Accept")
		}
		_, _ = w.Write(data)
	}))
	defer srv.Close()
	d := Downloader{Client: srv.Client(), Retries: 1}
	_, err := d.Download(context.Background(), Asset{URL: srv.URL + "/releases/assets/123", SHA256: hex.EncodeToString(sum[:])}, filepath.Join(t.TempDir(), "x"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if accept != "application/octet-stream" {
		t.Fatalf("Accept header = %q, want application/octet-stream", accept)
	}
}

func TestDownloadRejectsChecksum(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("bad")) }))
	defer srv.Close()
	_, err := (Downloader{Client: srv.Client(), Retries: 1}).Download(context.Background(), Asset{URL: srv.URL, SHA256: strings.Repeat("0", 64)}, filepath.Join(t.TempDir(), "x"), nil)
	if err == nil {
		t.Fatal("expected verify error")
	}
}

func TestDownloadResetsPartialWhenURLChanges(t *testing.T) {
	data := []byte("replacement-content")
	sum := sha256.Sum256(data)
	var rangeSeen string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rangeSeen = r.Header.Get("Range")
		_, _ = w.Write(data)
	}))
	defer srv.Close()
	dst := filepath.Join(t.TempDir(), "blob")
	_ = os.WriteFile(dst+".part", []byte("stale"), 0600)
	state, _ := json.Marshal(downloadState{URL: "https://old.invalid/blob", Bytes: 5})
	_ = os.WriteFile(dst+".download.state.json", state, 0600)
	_, err := (Downloader{Client: srv.Client(), Retries: 1}).Download(context.Background(), Asset{URL: srv.URL, SHA256: hex.EncodeToString(sum[:])}, dst, nil)
	if err != nil {
		t.Fatal(err)
	}
	if rangeSeen != "" {
		t.Fatalf("stale partial was resumed with %q", rangeSeen)
	}
	got, _ := os.ReadFile(dst)
	if string(got) != string(data) {
		t.Fatalf("got %q", got)
	}
}
