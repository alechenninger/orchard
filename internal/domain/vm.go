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

// VMArtifacts prepares per-VM artifacts on the host filesystem.
type VMArtifacts interface {
	// Prepare ensures per-VM directory exists, clones/copies base image to disk.img,
	// creates nvram.bin placeholder, and sets SeedISOPath.
	Prepare(ctx context.Context, vm *VM) error
}

// RuntimeState abstracts ephemeral runtime coordination for a VM on the host.
type RuntimeState interface {
	AcquireLock(ctx context.Context, vmName string) (release func() error, err error)
	WritePID(ctx context.Context, vmName string, pid int) error
	ReadPID(ctx context.Context, vmName string) (int, error)
	MarkReady(ctx context.Context, vmName string) error
	Clear(ctx context.Context, vmName string) error
	CleanupIfStale(ctx context.Context, vmName string) error
	WaitReadyAndPID(ctx context.Context, vmName string) (int, error)
}
