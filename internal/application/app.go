package application

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/alechenninger/orchard/internal/domain"
	shimproc "github.com/alechenninger/orchard/internal/shim/proc"
	fsstore "github.com/alechenninger/orchard/internal/vmstore/fs"
)

type App struct {
	Store domain.VMStore
	Shim  domain.ShimProcessManager
}

func New(store domain.VMStore) *App {
	return &App{Store: store, Shim: shimproc.New()}
}

func NewDefault() *App {
	return &App{Store: fsstore.New(defaultBaseDir()), Shim: shimproc.New()}
}

type UpParams struct {
	ImagePath   string
	CPUs        int
	MemoryMiB   int
	DiskSizeGiB int
	SSHKeyPath  string
}

func (a *App) Up(ctx context.Context, p UpParams) (*domain.VM, error) {
	absImage, err := filepath.Abs(p.ImagePath)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(absImage); err != nil {
		return nil, fmt.Errorf("image path invalid: %w", err)
	}

	sshKeyPath := p.SSHKeyPath
	if sshKeyPath == "" {
		home, _ := os.UserHomeDir()
		candidates := []string{
			filepath.Join(home, ".ssh", "id_ed25519.pub"),
			filepath.Join(home, ".ssh", "id_rsa.pub"),
		}
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				sshKeyPath = c
				break
			}
		}
	}

	name, err := a.Store.NextName(ctx)
	if err != nil {
		return nil, err
	}

	vm := domain.VM{
		Name:         name,
		CPUs:         p.CPUs,
		MemoryMiB:    p.MemoryMiB,
		DiskSizeGiB:  p.DiskSizeGiB,
		BaseImageRef: absImage,
		Hostname:     name,
		Status:       "stopped",
	}
	_ = sshKeyPath // reserved for cloud-init later

	if err := a.Store.Save(ctx, vm); err != nil {
		return nil, err
	}
	return &vm, nil
}

func (a *App) ListVMs(ctx context.Context) ([]domain.VM, error) {
	return a.Store.List(ctx)
}

func defaultBaseDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "orchard")
	}
	return filepath.Join(home, ".orchard")
}
