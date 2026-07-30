package updater

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCacheImportAndLRUPrune(t *testing.T) {
	root := t.TempDir()
	cache := Cache{Root: filepath.Join(root, "cache")}
	for i, sha := range []string{"aa", "bb", "cc"} {
		src := filepath.Join(root, sha)
		if err := os.WriteFile(src, []byte(sha), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := cache.Import(src, sha, "1.26.0729.190"+string(rune('0'+i)), "app_linux_amd64"); err != nil {
			t.Fatal(err)
		}
		time.Sleep(2 * time.Millisecond)
	}
	removed, err := cache.Prune(2)
	if err != nil {
		t.Fatal(err)
	}
	if len(removed) != 1 || removed[0].SHA256 != "aa" {
		t.Fatalf("removed=%+v", removed)
	}
	entries, err := cache.Load()
	if err != nil || len(entries) != 2 {
		t.Fatalf("entries=%+v err=%v", entries, err)
	}
	if _, err := os.Stat(removed[0].Path); !os.IsNotExist(err) {
		t.Fatalf("removed blob still exists: %v", err)
	}
}
