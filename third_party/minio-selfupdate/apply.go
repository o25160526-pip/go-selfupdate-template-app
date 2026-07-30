// Package selfupdate contains the atomic replacement flow used by MinIO's
// github.com/minio/selfupdate. This vendored subset intentionally keeps only
// the full-file replacement API used by this template.
package selfupdate

import (
	"bytes"
	"crypto"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/minio/selfupdate/internal/osext"
)

type Patcher interface {
	Patch(old io.Reader, new io.Writer, patch io.Reader) error
}

type Verifier struct{ VerifyFunc func([]byte) error }

func (v *Verifier) Verify(b []byte) error {
	if v == nil || v.VerifyFunc == nil {
		return nil
	}
	return v.VerifyFunc(b)
}

type Options struct {
	TargetPath  string
	TargetMode  os.FileMode
	Checksum    []byte
	Verifier    *Verifier
	Hash        crypto.Hash
	Patcher     Patcher
	OldSavePath string
}

func Apply(update io.Reader, opts Options) error {
	if err := PrepareAndCheckBinary(update, opts); err != nil {
		return err
	}
	return CommitBinary(opts)
}

func PrepareAndCheckBinary(update io.Reader, opts Options) error {
	targetPath, err := opts.getPath()
	if err != nil {
		return err
	}
	var newBytes []byte
	if opts.Patcher != nil {
		old, err := os.Open(targetPath)
		if err != nil {
			return err
		}
		defer old.Close()
		var out bytes.Buffer
		if err := opts.Patcher.Patch(old, &out, update); err != nil {
			return err
		}
		newBytes = out.Bytes()
	} else {
		newBytes, err = io.ReadAll(update)
		if err != nil {
			return err
		}
	}
	if opts.Checksum != nil {
		sum, err := checksumFor(opts.getHash(), newBytes)
		if err != nil {
			return err
		}
		if !bytes.Equal(sum, opts.Checksum) {
			return fmt.Errorf("updated file has wrong checksum: expected %x, got %x", opts.Checksum, sum)
		}
	}
	if opts.Verifier != nil {
		if err := opts.Verifier.Verify(newBytes); err != nil {
			return err
		}
	}
	dir, name := filepath.Dir(targetPath), filepath.Base(targetPath)
	newPath := filepath.Join(dir, "."+name+".new")
	fp, err := os.OpenFile(newPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, opts.getMode())
	if err != nil {
		return err
	}
	if _, err := fp.Write(newBytes); err != nil {
		fp.Close()
		return err
	}
	if err := fp.Sync(); err != nil {
		fp.Close()
		return err
	}
	return fp.Close()
}

func CommitBinary(opts Options) error {
	targetPath, err := opts.getPath()
	if err != nil {
		return err
	}
	dir, name := filepath.Dir(targetPath), filepath.Base(targetPath)
	newPath := filepath.Join(dir, "."+name+".new")
	oldPath := opts.OldSavePath
	removeOld := oldPath == ""
	if removeOld {
		oldPath = filepath.Join(dir, "."+name+".old")
	}
	_ = os.Remove(oldPath)
	if err := os.Rename(targetPath, oldPath); err != nil {
		return err
	}
	if err := os.Rename(newPath, targetPath); err != nil {
		if rerr := os.Rename(oldPath, targetPath); rerr != nil {
			return &rollbackErr{error: err, rollbackErr: rerr}
		}
		return err
	}
	if removeOld {
		_ = os.Remove(oldPath)
	}
	return nil
}

func RollbackError(err error) error {
	var re *rollbackErr
	if errors.As(err, &re) {
		return re.rollbackErr
	}
	return nil
}

type rollbackErr struct {
	error
	rollbackErr error
}

func (o *Options) CheckPermissions() error {
	path, err := o.getPath()
	if err != nil {
		return err
	}
	p := filepath.Join(filepath.Dir(path), "."+filepath.Base(path)+".check-perm")
	f, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, o.getMode())
	if err != nil {
		return err
	}
	_ = f.Close()
	return os.Remove(p)
}
func (o *Options) getPath() (string, error) {
	if o.TargetPath != "" {
		return o.TargetPath, nil
	}
	return osext.Executable()
}
func (o *Options) getMode() os.FileMode {
	if o.TargetMode == 0 {
		return 0o755
	}
	return o.TargetMode
}
func (o *Options) getHash() crypto.Hash {
	if o.Hash == 0 {
		return crypto.SHA256
	}
	return o.Hash
}
func checksumFor(h crypto.Hash, payload []byte) ([]byte, error) {
	if !h.Available() {
		return nil, errors.New("requested hash function not available")
	}
	x := h.New()
	_, _ = x.Write(payload)
	return x.Sum(nil), nil
}
