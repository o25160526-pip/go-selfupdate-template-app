package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPrecedence(t *testing.T) {
	d := t.TempDir()
	p := filepath.Join(d, "config.yaml")
	if err := os.WriteFile(p, []byte("channel: beta\ntimeout: 2m\ncache_dir: /from-file\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("APP_CHANNEL", "stable")
	t.Setenv("APP_CACHE_DIR", "/from-env")
	c, err := Load("app", p)
	if err != nil {
		t.Fatal(err)
	}
	if c.Channel != "stable" || c.CacheDir != "/from-env" || c.Timeout.String() != "2m0s" {
		t.Fatalf("bad precedence: %+v", c)
	}
}
func TestInternalRequiresTokenForUpdate(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.yaml")
	_ = os.WriteFile(p, []byte("channel: internal\n"), 0600)
	t.Setenv("APP_UPDATE_TOKEN", "")
	c, err := Load("app", p)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.ValidateForUpdate(); err == nil {
		t.Fatal("expected token error")
	}
}
