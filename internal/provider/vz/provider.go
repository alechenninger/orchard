package vfkit

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"path/filepath"
	"sync"

	"github.com/Code-Hex/vz/v3"
	"github.com/alechenninger/orchard/internal/domain"
	"github.com/spf13/afero"
)

type Provider struct {
	mu      sync.Mutex
	handles map[string]*vz.VirtualMachine
	fs      afero.Fs
}

func New() *Provider {
	return &Provider{handles: make(map[string]*vz.VirtualMachine), fs: afero.NewOsFs()}
}
func NewWithFS(fs afero.Fs) *Provider {
	return &Provider{handles: make(map[string]*vz.VirtualMachine), fs: fs}
}

func (p *Provider) StartVM(ctx context.Context, vm domain.VM) (int, error) {
	// Prepare EFI variable store
	createVarStore := false
	if st, err := p.fs.Stat(vm.EFIVarsPath); err != nil || st.Size() == 0 {
		createVarStore = true
	}
	var varStoreOpts []vz.NewEFIVariableStoreOption
	if createVarStore {
		varStoreOpts = append(varStoreOpts, vz.WithCreatingEFIVariableStore())
	}
	varStore, err := vz.NewEFIVariableStore(vm.EFIVarsPath, varStoreOpts...)
	if err != nil {
		return 0, err
	}

	bootLoader, err := vz.NewEFIBootLoader(vz.WithEFIVariableStore(varStore))
	if err != nil {
		return 0, err
	}

	memBytes := uint64(vm.MemoryMiB) * 1024 * 1024
	vmConfig, err := vz.NewVirtualMachineConfiguration(bootLoader, uint(vm.CPUs), memBytes)
	if err != nil {
		return 0, err
	}

	if vz.IsNestedVirtualizationSupported() {
		if platformCfg, err := vz.NewGenericPlatformConfiguration(); err == nil {
			if err := platformCfg.SetNestedVirtualizationEnabled(true); err != nil {
				slog.Warn("failed to enable nested virtualization", "error", err)
			} else {
				slog.Info("nested virtualization enabled for VM")
				vmConfig.SetPlatformVirtualMachineConfiguration(platformCfg)
			}
		} else {
			slog.Warn("failed to create generic platform config for nested virtualization", "error", err)
		}
	} else {
		slog.Info("nested virtualization not supported on this host; skipping")
	}

	if err := p.configureDevices(vmConfig, vm); err != nil {
		return 0, err
	}

	if valid, err := vmConfig.Validate(); err != nil {
		return 0, err
	} else if !valid {
		return 0, fmt.Errorf("virtual machine configuration invalid")
	}

	vzVM, err := vz.NewVirtualMachine(vmConfig)
	if err != nil {
		return 0, err
	}
	if err := vzVM.Start(); err != nil {
		return 0, err
	}

	p.mu.Lock()
	p.handles[vm.Name] = vzVM
	p.mu.Unlock()
	return 0, nil
}

func (p *Provider) configureDevices(vmConfig *vz.VirtualMachineConfiguration, vm domain.VM) error {
	var storage []vz.StorageDeviceConfiguration

	if vm.DiskPath != "" {
		attachment, err := vz.NewDiskImageStorageDeviceAttachmentWithCacheAndSync(vm.DiskPath, false, vz.DiskImageCachingModeCached, vz.DiskImageSynchronizationModeFsync)
		if err != nil {
			return err
		}
		blk, err := vz.NewVirtioBlockDeviceConfiguration(attachment)
		if err != nil {
			return err
		}
		storage = append(storage, blk)
	}

	if vm.SeedISOPath != "" {
		if st, err := p.fs.Stat(vm.SeedISOPath); err == nil && !st.IsDir() {
			attachment, err := vz.NewDiskImageStorageDeviceAttachmentWithCacheAndSync(vm.SeedISOPath, true, vz.DiskImageCachingModeCached, vz.DiskImageSynchronizationModeFsync)
			if err != nil {
				return err
			}
			isoCfg, err := vz.NewVirtioBlockDeviceConfiguration(attachment)
			if err != nil {
				return err
			}
			storage = append(storage, isoCfg)
		}
	}

	if len(storage) > 0 {
		vmConfig.SetStorageDevicesVirtualMachineConfiguration(storage)
	}

	serialLog := filepath.Join(filepath.Dir(vm.DiskPath), "serial.log")
	serialAttachment, err := vz.NewFileSerialPortAttachment(serialLog, true)
	if err != nil {
		return err
	}
	serialCfg, err := vz.NewVirtioConsoleDeviceSerialPortConfiguration(serialAttachment)
	if err != nil {
		return err
	}
	vmConfig.SetSerialPortsVirtualMachineConfiguration([]*vz.VirtioConsoleDeviceSerialPortConfiguration{serialCfg})

	natAttachment, err := vz.NewNATNetworkDeviceAttachment()
	if err != nil {
		return err
	}
	netCfg, err := vz.NewVirtioNetworkDeviceConfiguration(natAttachment)
	if err != nil {
		return err
	}
	if vm.MACAddress != "" {
		if hw, err := net.ParseMAC(vm.MACAddress); err == nil {
			if macObj, err := vz.NewMACAddress(hw); err == nil {
				netCfg.SetMACAddress(macObj)
			}
		}
	}
	vmConfig.SetNetworkDevicesVirtualMachineConfiguration([]*vz.VirtioNetworkDeviceConfiguration{netCfg})

	if rng, err := vz.NewVirtioEntropyDeviceConfiguration(); err == nil {
		vmConfig.SetEntropyDevicesVirtualMachineConfiguration([]*vz.VirtioEntropyDeviceConfiguration{rng})
	} else {
		slog.Warn("failed to configure entropy device", "error", err)
	}

	return nil
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
