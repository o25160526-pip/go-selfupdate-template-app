package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/your-org/go-selfupdate-template/internal/updater"
)

func main() {
	in := flag.String("in", "configs/manifest.example.json", "input")
	out := flag.String("out", "manifest.signed.json", "output")
	keyID := flag.String("key-id", "current", "key ID")
	flag.Parse()
	raw := os.Getenv("APP_MANIFEST_PRIVATE_KEY")
	if raw == "" {
		fatal("APP_MANIFEST_PRIVATE_KEY is required (base64 Ed25519 private key)")
	}
	key, err := base64.StdEncoding.DecodeString(raw)
	if err != nil || len(key) != ed25519.PrivateKeySize {
		fatal("invalid Ed25519 private key")
	}
	b, err := os.ReadFile(*in)
	if err != nil {
		fatal(err.Error())
	}
	var m updater.Manifest
	if err := json.Unmarshal(b, &m); err != nil {
		fatal(err.Error())
	}
	if err := m.Sign(ed25519.PrivateKey(key), *keyID); err != nil {
		fatal(err.Error())
	}
	b, err = json.MarshalIndent(m, "", "  ")
	if err != nil {
		fatal(err.Error())
	}
	if err := os.WriteFile(*out, b, 0600); err != nil {
		fatal(err.Error())
	}
	fmt.Println(*out)
}
func fatal(s string) { fmt.Fprintln(os.Stderr, s); os.Exit(1) }
