package fs

import (
	"context"
	"fmt"
	"io"
	"path/filepath"

	"os"

	"github.com/alechenninger/orchard/internal/domain"
	fsstore "github.com/alechenninger/orchard/internal/vmstore/fs"
	"github.com/spf13/afero"
)

type Service struct {
	baseDir string
	fs      afero.Fs
}

func NewWithBaseDir(baseDir string) *Service           { return &Service{baseDir: baseDir, fs: afero.NewOsFs()} }
func NewDefault() *Service                             { return &Service{baseDir: fsstore.DefaultBaseDir(), fs: afero.NewOsFs()} }
func NewWithFS(baseDir string, fsys afero.Fs) *Service { return &Service{baseDir: baseDir, fs: fsys} }

func (s *Service) Prepare(ctx context.Context, vm *domain.VM) error {
	vmDir := filepath.Join(s.baseDir, "vms", vm.Name)
	af := &afero.Afero{Fs: s.fs}
	if err := af.MkdirAll(vmDir, 0o755); err != nil {
		return err
	}
	diskPath := filepath.Join(vmDir, "disk.img")
	efiPath := filepath.Join(vmDir, "nvram.bin")
	seedPath := filepath.Join(vmDir, "seed.iso")

	if err := copyFile(s.fs, vm.BaseImageRef, diskPath); err != nil {
		return fmt.Errorf("copy base image: %w", err)
	}
	if f, err := s.fs.OpenFile(efiPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644); err == nil {
		_ = f.Close()
	} else {
		return err
	}
	vm.DiskPath = diskPath
	vm.EFIVarsPath = efiPath
	vm.SeedISOPath = seedPath
	return nil
}

func copyFile(fsys afero.Fs, src, dst string) error {
	in, err := fsys.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := fsys.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	if s, ok := out.(interface{ Sync() error }); ok {
		return s.Sync()
	}
	return nil
}

var _ domain.VMArtifacts = (*Service)(nil)
