package proc

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/alechenninger/orchard/internal/domain"
)

type Manager struct{ store domain.VMStore }

func New(store domain.VMStore) *Manager { return &Manager{store: store} }

// StartDetached re-execs this binary with the hidden _shim subcommand.
func (m *Manager) StartDetached(ctx context.Context, vm domain.VM) (int, error) {
	// Clean up any stale runtime files from a previous crashed shim
	_ = m.cleanStaleRuntime(ctx, vm.Name)

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
	paths, err := m.store.RuntimePaths(ctx, vmName)
	if err != nil {
		return 0, err
	}
	deadline := time.Now().Add(15 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		if _, err := os.Stat(paths.ReadyFile); err == nil {
			// Read pid
			f, err := os.Open(paths.PIDFile)
			if err != nil {
				return 0, err
			}
			defer f.Close()
			var pid int
			if _, err := fmt.Fscan(bufio.NewReader(f), &pid); err != nil {
				return 0, err
			}
			return pid, nil
		}
		if time.Now().After(deadline) {
			return 0, fmt.Errorf("timeout waiting for readiness of %s", vmName)
		}
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-ticker.C:
		}
	}
}

func (m *Manager) GetPID(ctx context.Context, vmName string) (int, error) {
	paths, err := m.store.RuntimePaths(ctx, vmName)
	if err != nil {
		return 0, err
	}
	f, err := os.Open(paths.PIDFile)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	var pid int
	if _, err := fmt.Fscan(bufio.NewReader(f), &pid); err != nil {
		return 0, err
	}
	return pid, nil
}

// cleanStaleRuntime removes lock/ready/pid if the recorded PID is not running.
func (m *Manager) cleanStaleRuntime(ctx context.Context, vmName string) error {
	paths, err := m.store.RuntimePaths(ctx, vmName)
	if err != nil {
		return err
	}
	// Read pid if present
	f, err := os.Open(paths.PIDFile)
	if err != nil {
		// no pidfile, nothing to do
		return nil
	}
	defer f.Close()
	var pid int
	if _, err := fmt.Fscan(bufio.NewReader(f), &pid); err != nil {
		return nil
	}
	// Check if process is alive: kill(pid, 0)
	if err := syscall.Kill(pid, 0); err == nil {
		// still running
		return nil
	}
	// Not running → remove stale files
	_ = os.Remove(paths.PIDFile)
	_ = os.Remove(paths.ReadyFile)
	_ = os.RemoveAll(paths.LockDir)
	return nil
}
