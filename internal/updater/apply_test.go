package updater

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestApplyRollback(t *testing.T) {
	d := t.TempDir()
	target := filepath.Join(d, "app")
	next := filepath.Join(d, "next")
	backup := filepath.Join(d, "backup")
	_ = os.WriteFile(target, []byte("v1"), 0700)
	_ = os.WriteFile(next, []byte("v2"), 0700)
	if err := ApplyBinary(next, target, backup); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(target)
	if string(b) != "v2" {
		t.Fatal(string(b))
	}
	if err := Rollback(target, backup); err != nil {
		t.Fatal(err)
	}
	b, _ = os.ReadFile(target)
	if string(b) != "v1" {
		t.Fatal(string(b))
	}
}
func TestLock(t *testing.T) {
	p := filepath.Join(t.TempDir(), "lock")
	f, _ := os.Create(p)
	_ = f.Close()
	if err := WithLock(p, func() error { return nil }); err != ErrUpdateLocked {
		t.Fatalf("got %v", err)
	}
}

func TestWithLockRejectsActiveAndRecoversStale(t *testing.T) {
	lock := filepath.Join(t.TempDir(), "update.lock")
	if err := os.WriteFile(lock, []byte("active"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := WithLock(lock, func() error { return nil }); !errors.Is(err, ErrUpdateLocked) {
		t.Fatalf("active lock err=%v", err)
	}
	old := time.Now().Add(-20 * time.Minute)
	if err := os.Chtimes(lock, old, old); err != nil {
		t.Fatal(err)
	}
	called := false
	if err := WithLock(lock, func() error { called = true; return nil }); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("stale lock did not run callback")
	}
}
