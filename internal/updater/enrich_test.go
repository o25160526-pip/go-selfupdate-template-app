package updater

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEnrichAssetFromChecksumsAndSignature(t *testing.T) {
	data := []byte("binary")
	sum := sha256.Sum256(data)
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	sig := base64.StdEncoding.EncodeToString(ed25519.Sign(priv, data))
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	defer srv.Close()
	mux.HandleFunc("/checksums", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(hex.EncodeToString(sum[:]) + "  app_linux_amd64\n"))
	})
	mux.HandleFunc("/sig", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(sig + "\n")) })
	release := Release{Version: "1.26.0729.1930", Assets: []Asset{{Name: "app_linux_amd64", URL: srv.URL + "/asset"}, {Name: "checksums.txt", URL: srv.URL + "/checksums"}, {Name: "app_linux_amd64.sig", URL: srv.URL + "/sig"}}}
	eng := Engine{Client: srv.Client(), PublicKeys: []string{base64.StdEncoding.EncodeToString(pub)}}
	asset, err := eng.enrichAsset(context.Background(), release, release.Assets[0])
	if err != nil {
		t.Fatal(err)
	}
	if asset.SHA256 != hex.EncodeToString(sum[:]) || asset.Signature != sig {
		t.Fatalf("bad metadata %+v", asset)
	}
}

func TestEnrichAssetRequiresChecksum(t *testing.T) {
	eng := Engine{}
	_, err := eng.enrichAsset(context.Background(), Release{}, Asset{Name: "app_linux_amd64"})
	if err == nil {
		t.Fatal("expected checksum error")
	}
}
