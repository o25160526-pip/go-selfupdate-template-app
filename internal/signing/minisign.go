// Package signing implements the Ed25519 (non-prehashed) minisign wire format
// without pulling GUI or crypto toolchains into the updater binary.
package signing

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	EdDSA     uint16 = 0x6445
	HashEdDSA uint16 = 0x4445
)

var ErrInvalidSignature = errors.New("invalid minisign signature")

type PublicKey struct {
	Algorithm uint16
	ID        uint64
	Key       ed25519.PublicKey
}

func keyID(publicKey ed25519.PublicKey) uint64 {
	// Minisign IDs are identifiers, not security boundaries. A deterministic
	// ID lets a raw Ed25519 private key produce a stable minisign public key.
	sum := sha256.Sum256(publicKey)
	return binary.LittleEndian.Uint64(sum[:8])
}

func EncodePublicKey(publicKey ed25519.PublicKey) string {
	raw := make([]byte, 2+8+ed25519.PublicKeySize)
	binary.LittleEndian.PutUint16(raw[:2], EdDSA)
	binary.LittleEndian.PutUint64(raw[2:10], keyID(publicKey))
	copy(raw[10:], publicKey)
	return base64.StdEncoding.EncodeToString(raw)
}

func PublicKeyText(publicKey ed25519.PublicKey) string {
	id := keyID(publicKey)
	return fmt.Sprintf("untrusted comment: minisign public key %016X\n%s\n", id, EncodePublicKey(publicKey))
}

// ParsePublicKey accepts a raw 32-byte Ed25519 key in base64, a 42-byte
// minisign public-key payload in base64, or the two-line minisign public file.
func ParsePublicKey(value string) (PublicKey, error) {
	value = strings.TrimSpace(value)
	if strings.Contains(value, "\n") {
		for _, line := range strings.Split(value, "\n") {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "untrusted comment:") {
				value = line
				break
			}
		}
	}
	raw, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return PublicKey{}, err
	}
	switch len(raw) {
	case ed25519.PublicKeySize:
		key := append(ed25519.PublicKey(nil), raw...)
		return PublicKey{Algorithm: EdDSA, ID: keyID(key), Key: key}, nil
	case 2 + 8 + ed25519.PublicKeySize:
		algorithm := binary.LittleEndian.Uint16(raw[:2])
		if algorithm != EdDSA && algorithm != HashEdDSA {
			return PublicKey{}, fmt.Errorf("invalid minisign public-key algorithm %d", algorithm)
		}
		return PublicKey{Algorithm: algorithm, ID: binary.LittleEndian.Uint64(raw[2:10]), Key: append(ed25519.PublicKey(nil), raw[10:]...)}, nil
	default:
		return PublicKey{}, fmt.Errorf("invalid public key length %d", len(raw))
	}
}

func Sign(privateKey ed25519.PrivateKey, message []byte, now time.Time) (string, error) {
	if len(privateKey) != ed25519.PrivateKeySize {
		return "", fmt.Errorf("invalid Ed25519 private key length %d", len(privateKey))
	}
	publicKey := privateKey.Public().(ed25519.PublicKey)
	id := keyID(publicKey)
	messageSignature := ed25519.Sign(privateKey, message)
	trustedComment := "timestamp:" + strconv.FormatInt(now.UTC().Unix(), 10)
	commentSignature := ed25519.Sign(privateKey, append(append([]byte(nil), messageSignature...), []byte(trustedComment)...))
	raw := make([]byte, 2+8+ed25519.SignatureSize)
	binary.LittleEndian.PutUint16(raw[:2], EdDSA)
	binary.LittleEndian.PutUint64(raw[2:10], id)
	copy(raw[10:], messageSignature)
	return fmt.Sprintf(
		"untrusted comment: signature from minisign key %016X\n%s\ntrusted comment: %s\n%s",
		id,
		base64.StdEncoding.EncodeToString(raw),
		trustedComment,
		base64.StdEncoding.EncodeToString(commentSignature),
	), nil
}

func Verify(publicKeyValue string, message []byte, signatureText string) bool {
	publicKey, err := ParsePublicKey(publicKeyValue)
	if err != nil {
		return false
	}
	signatureText = strings.TrimSpace(signatureText)
	// Backward compatibility for already-published raw Ed25519 signatures.
	if !strings.Contains(signatureText, "\n") {
		raw, err := base64.StdEncoding.DecodeString(signatureText)
		return err == nil && len(raw) == ed25519.SignatureSize && ed25519.Verify(publicKey.Key, message, raw)
	}
	parts := strings.SplitN(strings.ReplaceAll(signatureText, "\r\n", "\n"), "\n", 4)
	if len(parts) != 4 || !strings.HasPrefix(parts[0], "untrusted comment: ") || !strings.HasPrefix(parts[2], "trusted comment: ") {
		return false
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(parts[1]))
	if err != nil || len(raw) != 2+8+ed25519.SignatureSize {
		return false
	}
	algorithm := binary.LittleEndian.Uint16(raw[:2])
	id := binary.LittleEndian.Uint64(raw[2:10])
	if algorithm != EdDSA || id != publicKey.ID {
		// The dependency-free template intentionally emits EdDSA signatures.
		// HashEdDSA requires the minisign BLAKE2b prehash dependency.
		return false
	}
	messageSignature := raw[10:]
	if !ed25519.Verify(publicKey.Key, message, messageSignature) {
		return false
	}
	commentSignature, err := base64.StdEncoding.DecodeString(strings.TrimSpace(parts[3]))
	if err != nil || len(commentSignature) != ed25519.SignatureSize {
		return false
	}
	trustedComment := strings.TrimPrefix(parts[2], "trusted comment: ")
	globalMessage := append(append([]byte(nil), messageSignature...), []byte(trustedComment)...)
	return ed25519.Verify(publicKey.Key, globalMessage, commentSignature)
}
