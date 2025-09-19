package fs

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/alechenninger/orchard/internal/domain"
	fsstore "github.com/alechenninger/orchard/internal/vmstore/fs"
)

type Service struct {
	baseDir string
}

func NewWithBaseDir(baseDir string) *Service { return &Service{baseDir: baseDir} }
func NewDefault() *Service                   { return &Service{baseDir: fsstore.DefaultBaseDir()} }

func (s *Service) Prepare(ctx context.Context, vm *domain.VM) error {
	vmDir := filepath.Join(s.baseDir, "vms", vm.Name)
	if err := os.MkdirAll(vmDir, 0o755); err != nil {
		return err
	}
	diskPath := filepath.Join(vmDir, "disk.img")
	efiPath := filepath.Join(vmDir, "nvram.bin")
	seedPath := filepath.Join(vmDir, "seed.iso")

	if err := copyFile(vm.BaseImageRef, diskPath); err != nil {
		return fmt.Errorf("copy base image: %w", err)
	}
	if f, err := os.OpenFile(efiPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644); err == nil {
		_ = f.Close()
	} else {
		return err
	}
	vm.DiskPath = diskPath
	vm.EFIVarsPath = efiPath
	vm.SeedISOPath = seedPath
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

var _ domain.VMArtifacts = (*Service)(nil)
