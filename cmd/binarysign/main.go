package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/your-org/go-selfupdate-template/internal/signing"
)

func main() {
	file := flag.String("file", "", "binary to sign")
	out := flag.String("out", "", "signature output; defaults to <file>.sig")
	flag.Parse()
	if *file == "" {
		fatal("--file is required")
	}
	if *out == "" {
		*out = *file + ".sig"
	}
	raw := strings.TrimSpace(os.Getenv("APP_BINARY_PRIVATE_KEY"))
	if raw == "" {
		raw = strings.TrimSpace(os.Getenv("APP_MANIFEST_PRIVATE_KEY"))
	}
	key, err := base64.StdEncoding.DecodeString(raw)
	if err != nil || len(key) != ed25519.PrivateKeySize {
		fatal("APP_BINARY_PRIVATE_KEY or APP_MANIFEST_PRIVATE_KEY must be a base64 Ed25519 private key")
	}
	data, err := os.ReadFile(*file)
	if err != nil {
		fatal(err.Error())
	}
	sig, err := signing.Sign(ed25519.PrivateKey(key), data, time.Now().UTC())
	if err != nil {
		fatal(err.Error())
	}
	if err := os.WriteFile(*out, []byte(sig+"\n"), 0o600); err != nil {
		fatal(err.Error())
	}
	fmt.Println(*out)
}
func fatal(s string) { fmt.Fprintln(os.Stderr, s); os.Exit(1) }
