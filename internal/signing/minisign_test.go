package signing

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"strings"
	"testing"
	"time"
)

func TestDecodePrivateKeyAcceptsSeedAndExpandedKey(t *testing.T) {
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	values := []string{
		base64.StdEncoding.EncodeToString(privateKey.Seed()),
		base64.StdEncoding.EncodeToString(privateKey),
	}
	for _, value := range values {
		decoded, err := DecodePrivateKey(value)
		if err != nil {
			t.Fatal(err)
		}
		if !decoded.Equal(privateKey) {
			t.Fatal("decoded private key does not match")
		}
	}
	if _, err := DecodePrivateKey(base64.StdEncoding.EncodeToString([]byte("short"))); err == nil {
		t.Fatal("invalid key length accepted")
	}
}

func TestMinisignRoundTripAndRawCompatibility(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	message := []byte("release binary")
	sig, err := Sign(priv, message, time.Unix(123, 0))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(sig, "trusted comment: timestamp:123") {
		t.Fatalf("signature=%q", sig)
	}
	for _, key := range []string{base64.StdEncoding.EncodeToString(pub), EncodePublicKey(pub), PublicKeyText(pub)} {
		if !Verify(key, message, sig) {
			t.Fatalf("failed key format %q", key)
		}
	}
	raw := base64.StdEncoding.EncodeToString(ed25519.Sign(priv, message))
	if !Verify(EncodePublicKey(pub), message, raw) {
		t.Fatal("raw compatibility failed")
	}
	if Verify(EncodePublicKey(pub), []byte("tampered"), sig) {
		t.Fatal("tampered message verified")
	}
}
