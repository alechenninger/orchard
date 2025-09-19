package application

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	artfs "github.com/alechenninger/orchard/internal/artifacts/fs"
	"github.com/alechenninger/orchard/internal/domain"
	runfs "github.com/alechenninger/orchard/internal/runstate/fs"
	shimproc "github.com/alechenninger/orchard/internal/shim/proc"
	fsstore "github.com/alechenninger/orchard/internal/vmstore/fs"
)

type App struct {
	Store     domain.VMStore
	Shim      domain.ShimProcessManager
	Artifacts domain.VMArtifacts
	Clock     domain.Clock
}

func New(store domain.VMStore, shim domain.ShimProcessManager, art domain.VMArtifacts) *App {
	return &App{Store: store, Shim: shim, Artifacts: art, Clock: domain.RealClock{}}
}

func NewDefault() *App {
	store := fsstore.NewDefault()
	run := runfs.NewDefault()
	shim := domain.ShimProcessManager(shimproc.New(store, run))
	art := artfs.NewDefault()
	return &App{Store: store, Shim: shim, Artifacts: art, Clock: domain.RealClock{}}
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

	// Ensure deterministic CreatedAt via injected clock if not set yet
	if vm.CreatedAt == 0 && a.Clock != nil {
		vm.CreatedAt = a.Clock.Now().UnixNano()
	}
	if err := a.Artifacts.Prepare(ctx, &vm); err != nil {
		return nil, err
	}
	if err := a.Store.Save(ctx, vm); err != nil { // persist updated paths
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

// Delete removes VM resources and metadata. If the VM is running and force is false,
// it returns an error. With force=true, it will attempt a Stop first.
func (a *App) Delete(ctx context.Context, nameOrID string, force bool) error {
	vm, err := a.Store.Load(ctx, nameOrID)
	if err != nil {
		return err
	}
	// Determine if running
	running := false
	if p, err := a.Shim.GetPID(ctx, vm.Name); err == nil && p > 0 {
		running = true
		vm.PID = p
	}
	if running {
		if !force {
			return fmt.Errorf("vm %s is running; use --force to stop and delete", vm.Name)
		}
		if err := a.Stop(ctx, vm.Name); err != nil {
			return err
		}
	}
	return a.Store.Delete(ctx, vm.Name)
}

// defaultBaseDir is now provided by vmstore/fs as DefaultBaseDir
