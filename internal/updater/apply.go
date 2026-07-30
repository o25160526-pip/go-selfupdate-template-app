package updater

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/minio/selfupdate"
)

var ErrUpdateLocked = errors.New("another update is already running")

func WithLock(path string, fn func() error) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if os.IsExist(err) {
		if st, statErr := os.Stat(path); statErr == nil && time.Since(st.ModTime()) > 15*time.Minute {
			if removeErr := os.Remove(path); removeErr == nil {
				f, err = os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
			}
		}
		if os.IsExist(err) {
			return ErrUpdateLocked
		}
	}
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(f, "pid=%d time=%s\n", os.Getpid(), time.Now().UTC().Format(time.RFC3339))
	_ = f.Close()
	defer os.Remove(path)
	return fn()
}
func ApplyBinary(newBinary, target, backup string) error {
	f, err := os.Open(newBinary)
	if err != nil {
		return err
	}
	defer f.Close()
	if backup == "" {
		backup = target + ".rollback"
	}
	return selfupdate.Apply(f, selfupdate.Options{TargetPath: target, TargetMode: 0o755, OldSavePath: backup})
}
func Rollback(target, backup string) error {
	if backup == "" {
		backup = target + ".rollback"
	}
	if _, err := os.Stat(backup); err != nil {
		return err
	}
	failed := target + ".failed"
	_ = os.Remove(failed)
	if err := os.Rename(target, failed); err != nil {
		return err
	}
	if err := os.Rename(backup, target); err != nil {
		_ = os.Rename(failed, target)
		return err
	}
	_ = os.Remove(failed)
	return nil
}
