package updater

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// MetadataCache stores immutable/revalidatable HTTP metadata separately from
// binary blobs. Entries are fresh for TTL and then revalidated with ETag.
type MetadataCache struct {
	Root string
	TTL  time.Duration
}

type metadataRecord struct {
	URL       string    `json:"url"`
	ETag      string    `json:"etag,omitempty"`
	FetchedAt time.Time `json:"fetched_at"`
	BodyPath  string    `json:"body_path"`
}

func (c MetadataCache) ttl() time.Duration {
	if c.TTL <= 0 {
		return 15 * time.Minute
	}
	return c.TTL
}

func (c MetadataCache) Get(ctx context.Context, client *http.Client, url, token string, headers http.Header, limit int64) ([]byte, error) {
	if client == nil {
		client = http.DefaultClient
	}
	if c.Root == "" {
		res, err := fetchMetadataResponse(ctx, client, url, token, headers, "", limit)
		return res.Body, err
	}
	keyInput := url + "\x00" + token
	sum := sha256.Sum256([]byte(keyInput))
	key := hex.EncodeToString(sum[:])
	dir := filepath.Join(c.Root, "metadata")
	metaPath := filepath.Join(dir, key+".json")
	bodyPath := filepath.Join(dir, key+".body")
	var rec metadataRecord
	var cached []byte
	if b, err := os.ReadFile(metaPath); err == nil && json.Unmarshal(b, &rec) == nil && rec.URL == url {
		cached, _ = os.ReadFile(bodyPath)
		if len(cached) > 0 && time.Since(rec.FetchedAt) < c.ttl() {
			return cached, nil
		}
	}
	response, err := fetchMetadataResponse(ctx, client, url, token, headers, rec.ETag, limit)
	if err == errNotModified && len(cached) > 0 {
		rec.FetchedAt = time.Now().UTC()
		rec.BodyPath = bodyPath
		_ = writeJSONAtomic(metaPath, rec, 0o600)
		return cached, nil
	}
	if err != nil {
		return nil, err
	}
	body, etag := response.Body, response.ETag
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	if err := writeAtomic(bodyPath, body, 0o600); err != nil {
		return nil, err
	}
	rec = metadataRecord{URL: url, ETag: etag, FetchedAt: time.Now().UTC(), BodyPath: bodyPath}
	if err := writeJSONAtomic(metaPath, rec, 0o600); err != nil {
		return nil, err
	}
	return body, nil
}

type metadataResponse struct {
	Body []byte
	ETag string
}

var errNotModified = fmt.Errorf("metadata not modified")

func fetchMetadataResponse(ctx context.Context, client *http.Client, url, token string, headers http.Header, etag string, limit int64) (metadataResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return metadataResponse{}, err
	}
	for k, values := range headers {
		for _, value := range values {
			req.Header.Add(k, value)
		}
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if strings.Contains(url, "/releases/assets/") {
		req.Header.Set("Accept", "application/octet-stream")
	}
	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}
	resp, err := client.Do(req)
	if err != nil {
		return metadataResponse{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotModified {
		return metadataResponse{}, errNotModified
	}
	if resp.StatusCode/100 != 2 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return metadataResponse{}, fmt.Errorf("metadata status %d: %s", resp.StatusCode, string(b))
	}
	if limit <= 0 {
		limit = 2 << 20
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return metadataResponse{}, err
	}
	if int64(len(b)) > limit {
		return metadataResponse{}, fmt.Errorf("metadata exceeds %d bytes", limit)
	}
	return metadataResponse{Body: b, ETag: resp.Header.Get("ETag")}, nil
}

func writeAtomic(path string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, mode); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func writeJSONAtomic(path string, value any, mode os.FileMode) error {
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return writeAtomic(path, b, mode)
}
