package updater

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/your-org/go-selfupdate-template/internal/signing"
)

var ErrVerification = errors.New("update verification failed")
var ErrDownload = errors.New("update download failed")

type Downloader struct {
	Client  *http.Client
	Retries int
	Backoff time.Duration
	Token   string
}
type downloadState struct {
	URL       string    `json:"url"`
	Bytes     int64     `json:"bytes"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (d Downloader) client() *http.Client {
	if d.Client != nil {
		return d.Client
	}
	return http.DefaultClient
}
func (d Downloader) Download(ctx context.Context, a Asset, dst string, publicKeys []string) (string, error) {
	retries := d.Retries
	if retries <= 0 {
		retries = 3
	}
	backoff := d.Backoff
	if backoff <= 0 {
		backoff = 200 * time.Millisecond
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return "", err
	}
	part := dst + ".part"
	statePath := dst + ".download.state.json"
	var last error
	for attempt := 0; attempt < retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(backoff * time.Duration(1<<uint(attempt-1))):
			}
		}
		if err := d.downloadOnce(ctx, a.URL, part, statePath); err != nil {
			last = err
			continue
		}
		sum, err := verifyFile(part, a.SHA256, a.Signature, publicKeys)
		if err != nil {
			return "", err
		}
		if err := os.Rename(part, dst); err != nil {
			return "", err
		}
		_ = os.Remove(statePath)
		return sum, nil
	}
	if last == nil {
		last = errors.New("download failed without a reported cause")
	}
	return "", fmt.Errorf("%w: %v", ErrDownload, last)
}
func (d Downloader) downloadOnce(ctx context.Context, url, part, statePath string) error {
	var offset int64
	if b, err := os.ReadFile(statePath); err == nil {
		var state downloadState
		if json.Unmarshal(b, &state) != nil || state.URL != url {
			_ = os.Remove(part)
			_ = os.Remove(statePath)
		}
	} else if _, err := os.Stat(part); err == nil {
		// A partial file without state cannot safely be resumed.
		_ = os.Remove(part)
	}
	if st, err := os.Stat(part); err == nil {
		offset = st.Size()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	if offset > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
	}
	if d.Token != "" {
		req.Header.Set("Authorization", "Bearer "+d.Token)
	}
	resp, err := d.client().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("download status %d", resp.StatusCode)
	}
	flags := os.O_CREATE | os.O_WRONLY
	if resp.StatusCode == http.StatusPartialContent && offset > 0 {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
		offset = 0
	}
	f, err := os.OpenFile(part, flags, 0o600)
	if err != nil {
		return err
	}
	n, copyErr := io.Copy(f, resp.Body)
	syncErr := f.Sync()
	closeErr := f.Close()
	st := downloadState{URL: url, Bytes: offset + n, UpdatedAt: time.Now().UTC()}
	if b, e := json.Marshal(st); e == nil {
		_ = os.WriteFile(statePath, b, 0o600)
	}
	if copyErr != nil {
		return copyErr
	}
	if syncErr != nil {
		return syncErr
	}
	return closeErr
}
func verifyFile(path, expectedSHA, signature string, keys []string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	hexsum := hex.EncodeToString(sum[:])
	if expectedSHA != "" && !strings.EqualFold(expectedSHA, hexsum) {
		return "", fmt.Errorf("%w: sha256 expected %s got %s", ErrVerification, expectedSHA, hexsum)
	}
	if signature != "" {
		ok := false
		for _, key := range keys {
			if signing.Verify(key, b, signature) {
				ok = true
				break
			}
		}
		if !ok {
			return "", fmt.Errorf("%w: minisign/Ed25519 signature mismatch", ErrVerification)
		}
	}
	return hexsum, nil
}
