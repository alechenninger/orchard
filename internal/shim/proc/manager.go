package proc

import (
    "context"
    "os"
    "os/exec"
    "syscall"

    "github.com/alechenninger/orchard/internal/domain"
)

type Manager struct{}

func New() *Manager { return &Manager{} }

// StartDetached re-execs this binary with the hidden _shim subcommand.
func (m *Manager) StartDetached(ctx context.Context, vm domain.VM) (int, error) {
    exe, err := os.Executable()
    if err != nil {
        return 0, err
    }
    cmd := exec.CommandContext(ctx, exe, "_shim", "--vm", vm.Name)
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


