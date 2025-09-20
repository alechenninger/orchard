package vfkit

import (
	"context"
	"os"
	"path/filepath"
	"sync"

	"github.com/alechenninger/orchard/internal/domain"
	"github.com/crc-org/vfkit/pkg/config"
	"github.com/crc-org/vfkit/pkg/vf"
)

type Provider struct {
	mu      sync.Mutex
	handles map[string]*vf.VirtualMachine
}

func New() *Provider { return &Provider{handles: make(map[string]*vf.VirtualMachine)} }

func (p *Provider) StartVM(ctx context.Context, vm domain.VM) (int, error) {
	// Build vfkit config
	// Decide whether to create/initialize the EFI variable store
	createVarStore := false
	if st, err := os.Stat(vm.EFIVarsPath); err != nil || st.Size() == 0 {
		createVarStore = true
	}
	boot := config.NewEFIBootloader(vm.EFIVarsPath, createVarStore)
	vmc := config.NewVirtualMachine(uint(vm.CPUs), uint64(vm.MemoryMiB), boot)

	// Disk
	if vm.DiskPath != "" {
		blk, err := config.VirtioBlkNew(vm.DiskPath)
		if err != nil {
			return 0, err
		}
		_ = vmc.AddDevice(blk)
	}

	// Serial log
	serialLog := filepath.Join(filepath.Dir(vm.DiskPath), "serial.log")
	serialDev, err := config.VirtioSerialNew(serialLog)
	if err != nil {
		return 0, err
	}
	_ = vmc.AddDevice(serialDev)

	// Network NAT
	netDev, err := config.VirtioNetNew("")
	if err != nil {
		return 0, err
	}
	_ = vmc.AddDevice(netDev)

	// RNG device (good practice for Linux guests)
	if rng, err := config.VirtioRngNew(); err == nil {
		_ = vmc.AddDevice(rng)
	}

	vfvm, err := vf.NewVirtualMachine(*vmc)
	if err != nil {
		return 0, err
	}
	if err := vfvm.Start(); err != nil {
		return 0, err
	}
	p.mu.Lock()
	p.handles[vm.Name] = vfvm
	p.mu.Unlock()
	return 0, nil
}

func (p *Provider) StopVM(ctx context.Context, vm domain.VM) error {
	p.mu.Lock()
	h := p.handles[vm.Name]
	delete(p.handles, vm.Name)
	p.mu.Unlock()
	if h == nil {
		return nil
	}
	return h.Stop()
}

func (p *Provider) IsRunning(ctx context.Context, vm domain.VM) (bool, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	_, ok := p.handles[vm.Name]
	return ok, nil
}

var _ domain.VirtualizationProvider = (*Provider)(nil)
