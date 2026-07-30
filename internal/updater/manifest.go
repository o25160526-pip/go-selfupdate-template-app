package updater

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/your-org/go-selfupdate-template/internal/signing"
	appversion "github.com/your-org/go-selfupdate-template/internal/version"
)

var (
	ErrManifestSignature = errors.New("manifest signature verification failed")
	ErrManifestExpired   = errors.New("manifest expired")
	ErrBlockedVersion    = errors.New("version is blocked by policy")
	ErrBelowMinimum      = errors.New("version is below minimum supported")
	ErrOutsideRollout    = errors.New("machine is outside rollout percentage")
	ErrAbovePolicyLatest = errors.New("target is newer than manifest latest")
)

type ChannelPolicy struct {
	Latest         string   `json:"latest"`
	MinSupported   string   `json:"min_supported"`
	ForceUpdate    bool     `json:"force_update"`
	RolloutPercent int      `json:"rollout_percent"`
	Sources        []string `json:"sources"`
}
type Manifest struct {
	Schema    int                      `json:"schema"`
	ExpiresAt *time.Time               `json:"expires_at,omitempty"`
	Channels  map[string]ChannelPolicy `json:"channels"`
	Blocked   []string                 `json:"blocked"`
	KeyID     string                   `json:"key_id,omitempty"`
	Signature string                   `json:"signature"`
}

type PolicyDecision struct {
	Allowed       bool   `json:"allowed"`
	Forced        bool   `json:"forced"`
	Reason        string `json:"reason"`
	Latest        string `json:"latest"`
	MinSupported  string `json:"min_supported"`
	RolloutBucket int    `json:"rollout_bucket"`
}

func ParseManifest(data []byte) (Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return m, err
	}
	if err := m.Validate(time.Now().UTC()); err != nil {
		return m, err
	}
	return m, nil
}
func (m Manifest) Validate(now time.Time) error {
	if m.Schema != 1 {
		return fmt.Errorf("unsupported manifest schema %d", m.Schema)
	}
	if len(m.Channels) == 0 {
		return errors.New("manifest has no channels")
	}
	if m.ExpiresAt != nil && now.After(*m.ExpiresAt) {
		return ErrManifestExpired
	}
	for name, p := range m.Channels {
		if p.RolloutPercent < 0 || p.RolloutPercent > 100 {
			return fmt.Errorf("channel %s rollout_percent out of range", name)
		}
		if _, err := appversion.Parse(p.Latest); err != nil {
			return fmt.Errorf("channel %s latest: %w", name, err)
		}
		if _, err := appversion.Parse(p.MinSupported); err != nil {
			return fmt.Errorf("channel %s min_supported: %w", name, err)
		}
	}
	return nil
}
func (m Manifest) unsignedBytes() ([]byte, error) { m.Signature = ""; return json.Marshal(m) }
func (m Manifest) Verify(publicKeys []string) error {
	sig, err := base64.StdEncoding.DecodeString(m.Signature)
	if err != nil {
		return fmt.Errorf("%w: invalid base64", ErrManifestSignature)
	}
	payload, err := m.unsignedBytes()
	if err != nil {
		return err
	}
	for _, encoded := range publicKeys {
		publicKey, err := signing.ParsePublicKey(encoded)
		if err != nil {
			continue
		}
		if ed25519.Verify(publicKey.Key, payload, sig) {
			return nil
		}
	}
	return ErrManifestSignature
}
func (m *Manifest) Sign(privateKey ed25519.PrivateKey, keyID string) error {
	m.KeyID = keyID
	m.Signature = ""
	payload, err := m.unsignedBytes()
	if err != nil {
		return err
	}
	m.Signature = base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, payload))
	return nil
}
func (m Manifest) Evaluate(channel, current, target, machineID string) (PolicyDecision, error) {
	p, ok := m.Channels[channel]
	if !ok {
		return PolicyDecision{}, fmt.Errorf("unknown channel %q", channel)
	}
	d := PolicyDecision{Allowed: true, Latest: p.Latest, MinSupported: p.MinSupported}
	currentBlocked := false
	for _, b := range m.Blocked {
		if target == b {
			d.Allowed = false
			d.Reason = "target version is blocked"
			return d, ErrBlockedVersion
		}
		if current == b {
			currentBlocked = true
		}
	}
	cv, err := appversion.Parse(current)
	if err != nil {
		return d, err
	}
	minv, _ := appversion.Parse(p.MinSupported)
	tv, err := appversion.Parse(target)
	if err != nil {
		return d, err
	}
	latestv, err := appversion.Parse(p.Latest)
	if err != nil {
		return d, err
	}
	if channel != string(Internal) && tv.Compare(latestv) > 0 {
		d.Allowed = false
		d.Reason = "target newer than manifest latest"
		return d, ErrAbovePolicyLatest
	}
	if currentBlocked {
		d.Forced = true
		d.Reason = "current version is blocked"
	}
	if cv.Compare(minv) < 0 {
		d.Forced = true
		d.Reason = "current version below min_supported"
	}
	if tv.Compare(minv) < 0 {
		d.Allowed = false
		d.Reason = "target below min_supported"
		return d, ErrBelowMinimum
	}
	bucket := rolloutBucket(machineID)
	d.RolloutBucket = bucket
	if p.RolloutPercent < 100 && bucket >= p.RolloutPercent && !d.Forced {
		d.Allowed = false
		d.Reason = "outside staged rollout"
		return d, ErrOutsideRollout
	}
	if p.ForceUpdate {
		d.Forced = true
		if d.Reason == "" {
			d.Reason = "force_update policy"
		}
	}
	if d.Reason == "" {
		d.Reason = "allowed"
	}
	return d, nil
}
func rolloutBucket(machineID string) int {
	sum := sha256.Sum256([]byte(machineID))
	n, _ := strconv.ParseUint(fmt.Sprintf("%x", sum[:4]), 16, 32)
	return int(n % 100)
}
