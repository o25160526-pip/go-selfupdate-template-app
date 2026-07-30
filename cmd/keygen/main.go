package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"

	"github.com/your-org/go-selfupdate-template/internal/signing"
)

func main() {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	privateValue := base64.StdEncoding.EncodeToString(priv)
	fmt.Println("APP_BINARY_PRIVATE_KEY=" + privateValue)
	fmt.Println("APP_MANIFEST_PRIVATE_KEY=" + privateValue)
	fmt.Println("APP_CURRENT_PUBLIC_KEY=" + signing.EncodePublicKey(pub))
	fmt.Println("MINISIGN_PUBLIC_KEY_FILE:")
	fmt.Print(signing.PublicKeyText(pub))
}
