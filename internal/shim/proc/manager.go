package proc

import (
	"context"
	"os"
	"os/exec"
	"syscall"

	"github.com/alechenninger/orchard/internal/domain"
)

type Manager struct {
	store domain.VMStore
	run   domain.RuntimeState
}

func New(store domain.VMStore, run domain.RuntimeState) *Manager {
	return &Manager{store: store, run: run}
}

// StartDetached re-execs this binary with the hidden _shim subcommand.
func (m *Manager) StartDetached(ctx context.Context, vm domain.VM) (int, error) {
	// Clean up any stale runtime files from a previous crashed shim
	_ = m.run.CleanupIfStale(ctx, vm.Name)

	exe, err := os.Executable()
	if err != nil {
		return 0, err
	}
	// Use exec.Command (not CommandContext) so child lifetime is not tied to parent ctx
	cmd := exec.Command(exe, "_shim", "--vm", vm.Name)
	// Detach from parent's process group
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		return 0, err
	}
	return cmd.Process.Pid, nil
}

func (m *Manager) Stop(ctx context.Context, pid int) error {
	p, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	// Best-effort SIGTERM
	return p.Signal(syscall.SIGTERM)
}

var _ domain.ShimProcessManager = (*Manager)(nil)

func (m *Manager) WaitReadyAndPID(ctx context.Context, vmName string) (int, error) {
	return m.run.WaitReadyAndPID(ctx, vmName)
}

func (m *Manager) GetPID(ctx context.Context, vmName string) (int, error) {
	return m.run.ReadPID(ctx, vmName)
}
