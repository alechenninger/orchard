package domain

import "context"

// VM represents a virtual machine's desired and runtime state.
type VM struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	CreatedAt    int64  `json:"createdAt"`
	CPUs         int    `json:"cpus"`
	MemoryMiB    int    `json:"memoryMiB"`
	DiskPath     string `json:"diskPath"`
	DiskSizeGiB  int    `json:"diskSizeGiB"`
	EFIVarsPath  string `json:"efiVarsPath"`
	SeedISOPath  string `json:"seedIsoPath"`
	MACAddress   string `json:"macAddress"`
	Hostname     string `json:"hostname"`
	BaseImageRef string `json:"baseImageRef"`

	// Runtime
	PID         int    `json:"pid"`
	ConsoleSock string `json:"consoleSock"`
	Status      string `json:"status"`
}

// VMService defines high-level VM operations.
type VMService interface {
	// Create prepares VM resources but does not start it.
	Create(ctx context.Context, params CreateParams) (*VM, error)
	// Start launches the VM (possibly via shim) and returns when ready or detached.
	Start(ctx context.Context, nameOrID string, opts StartOptions) (*VM, error)
	// Stop gracefully stops the VM.
	Stop(ctx context.Context, nameOrID string) error
	// Delete removes VM resources.
	Delete(ctx context.Context, nameOrID string) error
	// List returns known VMs.
	List(ctx context.Context) ([]VM, error)
	// Get returns a single VM by name or id.
	Get(ctx context.Context, nameOrID string) (*VM, error)
}

type CreateParams struct {
	Name        string
	BaseImage   string // path to base image provided by user
	CPUs        int
	MemoryMiB   int
	DiskSizeGiB int
	SSHKeyPath  string
}

type StartOptions struct {
	Detach bool
}

// VMStore persists VM metadata and provides name allocation.
type VMStore interface {
	NextName(ctx context.Context) (string, error)
	Save(ctx context.Context, vm VM) error
	Load(ctx context.Context, nameOrID string) (*VM, error)
	Delete(ctx context.Context, nameOrID string) error
	List(ctx context.Context) ([]VM, error)
	RuntimePaths(ctx context.Context, name string) (VMRuntimePaths, error)
}

// VirtualizationProvider abstracts vfkit usage.
type VirtualizationProvider interface {
	StartVM(ctx context.Context, vm VM) (pid int, err error)
	StopVM(ctx context.Context, vm VM) error
	IsRunning(ctx context.Context, vm VM) (bool, error)
}

// ShimProcessManager abstracts re-exec shim lifecycle.
type ShimProcessManager interface {
	StartDetached(ctx context.Context, vm VM) (pid int, err error)
	Stop(ctx context.Context, pid int) error
	WaitReadyAndPID(ctx context.Context, vmName string) (pid int, err error)
	GetPID(ctx context.Context, vmName string) (pid int, err error)
}

// VMRuntimePaths describes ephemeral runtime file locations for a VM.
type VMRuntimePaths struct {
	Dir         string
	PIDFile     string
	ReadyFile   string
	LockDir     string
	ConsoleSock string
}
