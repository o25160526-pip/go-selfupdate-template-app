package updater

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"testing"
	"time"

	"github.com/your-org/go-selfupdate-template/internal/signing"
)

func signedManifest(t *testing.T) (Manifest, string) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	exp := time.Now().Add(time.Hour)
	m := Manifest{Schema: 1, ExpiresAt: &exp, Channels: map[string]ChannelPolicy{"stable": {Latest: "1.26.0730.0100", MinSupported: "1.26.0701.0000", RolloutPercent: 100}}, Blocked: []string{"1.26.0715.1200"}}
	if err := m.Sign(priv, "current"); err != nil {
		t.Fatal(err)
	}
	return m, signing.EncodePublicKey(pub)
}
func TestManifestVerifyAndRotation(t *testing.T) {
	m, key := signedManifest(t)
	_, _, _ = ed25519.GenerateKey(rand.Reader)
	bad := base64.StdEncoding.EncodeToString(make([]byte, 32))
	if err := m.Verify([]string{bad, key}); err != nil {
		t.Fatal(err)
	}
	m.Channels["stable"] = ChannelPolicy{Latest: "1.26.0731.0100", MinSupported: "1.26.0701.0000", RolloutPercent: 100}
	if !errors.Is(m.Verify([]string{key}), ErrManifestSignature) {
		t.Fatal("tamper should fail")
	}
}
func TestManifestExpired(t *testing.T) {
	m, _ := signedManifest(t)
	past := time.Now().Add(-time.Hour)
	m.ExpiresAt = &past
	if !errors.Is(m.Validate(time.Now()), ErrManifestExpired) {
		t.Fatal("expected expiry")
	}
}
func TestPolicy(t *testing.T) {
	m, _ := signedManifest(t)
	d, err := m.Evaluate("stable", "1.26.0601.0000", "1.26.0730.0100", "machine")
	if err != nil || !d.Forced {
		t.Fatalf("expected forced: %+v %v", d, err)
	}
	d, err = m.Evaluate("stable", "1.26.0715.1200", "1.26.0730.0100", "machine")
	if err != nil || !d.Forced {
		t.Fatalf("blocked current should force update: %+v %v", d, err)
	}
}

func TestBlockedCurrentForcesUpgradeButBlockedTargetFails(t *testing.T) {
	m, _ := signedManifest(t)
	d, err := m.Evaluate("stable", "1.26.0715.1200", "1.26.0730.0100", "machine")
	if err != nil || !d.Forced || !d.Allowed {
		t.Fatalf("blocked current should force an allowed upgrade: %+v %v", d, err)
	}
	_, err = m.Evaluate("stable", "1.26.0702.0000", "1.26.0715.1200", "machine")
	if !errors.Is(err, ErrBlockedVersion) {
		t.Fatalf("blocked target got %v", err)
	}
}

func TestStableCannotExceedManifestLatest(t *testing.T) {
	m, _ := signedManifest(t)
	_, err := m.Evaluate("stable", "1.26.0702.0000", "1.26.0731.0000", "machine")
	if !errors.Is(err, ErrAbovePolicyLatest) {
		t.Fatalf("got %v", err)
	}
	m.Channels["internal"] = m.Channels["stable"]
	if _, err := m.Evaluate("internal", "1.26.0702.0000", "1.26.0731.0000", "machine"); err != nil {
		t.Fatalf("internal should allow draft newer than policy latest: %v", err)
	}
}
