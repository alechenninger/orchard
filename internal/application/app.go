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

func New(store domain.VMStore, shim domain.ShimProcessManager) *App {
	return &App{Store: store, Shim: shim}
}

func NewDefault() *App {
	store := fsstore.NewDefault()
	shim := procShim(store)
	return &App{Store: store, Shim: shim}
}

func procShim(store domain.VMStore) domain.ShimProcessManager {
	return shimproc.New(store)
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

func (a *App) Start(ctx context.Context, nameOrID string) (*domain.VM, error) {
	vm, err := a.Store.Load(ctx, nameOrID)
	if err != nil {
		return nil, err
	}
	_, err = a.Shim.StartDetached(ctx, *vm)
	if err != nil {
		return nil, err
	}
	pid, err := a.Shim.WaitReadyAndPID(ctx, vm.Name)
	if err != nil {
		return nil, err
	}
	vm.PID = pid
	vm.Status = "running"
	_ = a.Store.Save(ctx, *vm)
	return vm, nil
}

func (a *App) Stop(ctx context.Context, nameOrID string) error {
	vm, err := a.Store.Load(ctx, nameOrID)
	if err != nil {
		return err
	}
	if vm.PID == 0 {
		if p, err := a.Shim.GetPID(ctx, vm.Name); err == nil {
			vm.PID = p
		} else {
			return nil
		}
	}
	if err := a.Shim.Stop(ctx, vm.PID); err != nil {
		return err
	}
	vm.PID = 0
	vm.Status = "stopped"
	return a.Store.Save(ctx, *vm)
}

// defaultBaseDir is now provided by vmstore/fs as DefaultBaseDir
