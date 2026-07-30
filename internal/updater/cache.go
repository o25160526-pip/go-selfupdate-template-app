package updater

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type CacheEntry struct {
	SHA256     string    `json:"sha256"`
	Version    string    `json:"version"`
	Asset      string    `json:"asset"`
	Path       string    `json:"path"`
	Size       int64     `json:"size"`
	LastAccess time.Time `json:"last_access"`
}
type Cache struct{ Root string }

func (c Cache) metaPath() string           { return filepath.Join(c.Root, "meta.json") }
func (c Cache) blobPath(sha string) string { return filepath.Join(c.Root, "blobs", sha) }
func (c Cache) Load() ([]CacheEntry, error) {
	b, err := os.ReadFile(c.metaPath())
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var v []CacheEntry
	return v, json.Unmarshal(b, &v)
}
func (c Cache) save(v []CacheEntry) error {
	if err := os.MkdirAll(c.Root, 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmp := c.metaPath() + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, c.metaPath())
}
func (c Cache) Import(src, sha, version, asset string) (string, error) {
	if sha == "" {
		return "", fmt.Errorf("sha256 required")
	}
	if err := os.MkdirAll(filepath.Join(c.Root, "blobs"), 0o755); err != nil {
		return "", err
	}
	dst := c.blobPath(sha)
	if err := copyFile(src, dst, 0o700); err != nil {
		return "", err
	}
	st, _ := os.Stat(dst)
	entries, _ := c.Load()
	now := time.Now().UTC()
	found := false
	for i := range entries {
		if entries[i].SHA256 == sha {
			entries[i].LastAccess = now
			entries[i].Path = dst
			found = true
		}
	}
	if !found {
		entries = append(entries, CacheEntry{SHA256: sha, Version: version, Asset: asset, Path: dst, Size: st.Size(), LastAccess: now})
	}
	return dst, c.save(entries)
}
func (c Cache) Prune(keep int) ([]CacheEntry, error) {
	if keep < 0 {
		keep = 0
	}
	v, err := c.Load()
	if err != nil {
		return nil, err
	}
	sort.Slice(v, func(i, j int) bool { return v[i].LastAccess.After(v[j].LastAccess) })
	var removed []CacheEntry
	if len(v) > keep {
		removed = append(removed, v[keep:]...)
		for _, e := range removed {
			_ = os.Remove(e.Path)
		}
		v = v[:keep]
	}
	return removed, c.save(v)
}
func copyFile(src, dst string, mode os.FileMode) error {
	b, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	tmp := dst + ".tmp"
	if err := os.WriteFile(tmp, b, mode); err != nil {
		return err
	}
	return os.Rename(tmp, dst)
}
