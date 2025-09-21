package fs

import (
	"bufio"
	"context"
	"fmt"
	"path/filepath"
	"syscall"
	"time"

	"os"

	"github.com/alechenninger/orchard/internal/domain"
	fsstore "github.com/alechenninger/orchard/internal/vmstore/fs"
	"github.com/spf13/afero"
)

type Service struct {
	baseDir string
	fs      afero.Fs
}

func New(baseDir string) *Service { return &Service{baseDir: baseDir, fs: afero.NewOsFs()} }

func NewDefault() *Service { return New(fsstore.DefaultBaseDir()) }

func NewWithFS(baseDir string, fsys afero.Fs) *Service { return &Service{baseDir: baseDir, fs: fsys} }

func (s *Service) vmDir(name string) string { return filepath.Join(s.baseDir, "vms", name) }

func (s *Service) paths(name string) (pid, ready, lock string) {
	d := s.vmDir(name)
	return filepath.Join(d, "vm.pid"), filepath.Join(d, "vm.ready"), filepath.Join(d, "vm.lock.d")
}

func (s *Service) AcquireLock(ctx context.Context, vmName string) (func() error, error) {
	_, _, lock := s.paths(vmName)
	af := &afero.Afero{Fs: s.fs}
	if err := af.MkdirAll(s.vmDir(vmName), 0o755); err != nil {
		return nil, err
	}
	if err := s.fs.Mkdir(lock, 0o755); err != nil {
		return nil, fmt.Errorf("lock in use: %w", err)
	}
	return func() error { return af.RemoveAll(lock) }, nil
}

func (s *Service) WritePID(ctx context.Context, vmName string, pid int) error {
	p, _, _ := s.paths(vmName)
	af := &afero.Afero{Fs: s.fs}
	if err := af.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	tmp := p + ".tmp"
	f, err := s.fs.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	w := bufio.NewWriter(f)
	if _, err := fmt.Fprintf(w, "%d\n", pid); err != nil {
		f.Close()
		return err
	}
	if err := w.Flush(); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := s.fs.Rename(tmp, p); err != nil {
		return err
	}
	if df, err := s.fs.Open(filepath.Dir(p)); err == nil {
		_ = df.Sync()
		_ = df.Close()
	}
	return nil
}

func (s *Service) ReadPID(ctx context.Context, vmName string) (int, error) {
	p, _, _ := s.paths(vmName)
	f, err := s.fs.Open(p)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	var pid int
	if _, err := fmt.Fscan(bufio.NewReader(f), &pid); err != nil {
		return 0, err
	}
	return pid, nil
}

func (s *Service) MarkReady(ctx context.Context, vmName string) error {
	_, r, _ := s.paths(vmName)
	af := &afero.Afero{Fs: s.fs}
	if err := af.MkdirAll(filepath.Dir(r), 0o755); err != nil {
		return err
	}
	tmp := r + ".tmp"
	if err := af.WriteFile(tmp, []byte(time.Now().Format(time.RFC3339Nano)), 0o644); err != nil {
		return err
	}
	if err := s.fs.Rename(tmp, r); err != nil {
		return err
	}
	if df, err := s.fs.Open(filepath.Dir(r)); err == nil {
		_ = df.Sync()
		_ = df.Close()
	}
	return nil
}

func (s *Service) Clear(ctx context.Context, vmName string) error {
	p, r, _ := s.paths(vmName)
	_ = s.fs.Remove(p)
	_ = s.fs.Remove(r)
	return nil
}

func (s *Service) CleanupIfStale(ctx context.Context, vmName string) error {
	p, r, l := s.paths(vmName)
	f, err := s.fs.Open(p)
	if err != nil {
		return nil
	}
	defer f.Close()
	var pid int
	if _, err := fmt.Fscan(bufio.NewReader(f), &pid); err != nil {
		return nil
	}
	if err := syscallKill(pid, 0); err == nil {
		return nil
	}
	_ = s.fs.Remove(p)
	_ = s.fs.Remove(r)
	_ = s.fs.RemoveAll(l)
	return nil
}

func (s *Service) WaitReadyAndPID(ctx context.Context, vmName string) (int, error) {
	p, r, _ := s.paths(vmName)
	deadline := time.Now().Add(15 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		if _, err := s.fs.Stat(r); err == nil {
			f, err := s.fs.Open(p)
			if err != nil {
				return 0, err
			}
			defer f.Close()
			var pid int
			if _, err := fmt.Fscan(bufio.NewReader(f), &pid); err != nil {
				return 0, err
			}
			return pid, nil
		}
		if time.Now().After(deadline) {
			return 0, fmt.Errorf("timeout waiting for readiness of %s", vmName)
		}
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-ticker.C:
		}
	}
}

// small indirection for testability
var syscallKill = func(pid int, sig int) error { return syscall.Kill(pid, syscall.Signal(sig)) }

var _ domain.RuntimeState = (*Service)(nil)
