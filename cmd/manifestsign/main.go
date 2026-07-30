package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/your-org/go-selfupdate-template/internal/signing"
	"github.com/your-org/go-selfupdate-template/internal/updater"
)

func main() {
	in := flag.String("in", "configs/manifest.example.json", "input")
	out := flag.String("out", "manifest.signed.json", "output")
	keyID := flag.String("key-id", "current", "key ID")
	flag.Parse()
	raw := os.Getenv("APP_MANIFEST_PRIVATE_KEY")
	if raw == "" {
		fatal("APP_MANIFEST_PRIVATE_KEY is required (base64 Ed25519 seed or private key)")
	}
	key, err := signing.DecodePrivateKey(raw)
	if err != nil {
		fatal("invalid Ed25519 private key: " + err.Error())
	}
	b, err := os.ReadFile(*in)
	if err != nil {
		fatal(err.Error())
	}
	var m updater.Manifest
	if err := json.Unmarshal(b, &m); err != nil {
		fatal(err.Error())
	}
	if err := m.Sign(key, *keyID); err != nil {
		fatal(err.Error())
	}
	b, err = json.MarshalIndent(m, "", "  ")
	if err != nil {
		fatal(err.Error())
	}
	if err := os.WriteFile(*out, b, 0o600); err != nil {
		fatal(err.Error())
	}
	fmt.Println(*out)
}
func fatal(s string) { fmt.Fprintln(os.Stderr, s); os.Exit(1) }
