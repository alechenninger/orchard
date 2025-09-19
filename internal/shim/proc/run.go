package proc

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/alechenninger/orchard/internal/domain"
)

// RunChild is invoked in the _shim process to own the VM lifecycle.
func RunChild(ctx context.Context, store domain.VMStore, name string) error {
	vm, err := store.Load(ctx, name)
	if err != nil {
		return err
	}
	paths, err := store.RuntimePaths(ctx, vm.Name)
	if err != nil {
		return err
	}
	// Acquire lock directory
	if err := os.Mkdir(paths.LockDir, 0o755); err != nil {
		return fmt.Errorf("lock in use: %w", err)
	}
	defer os.RemoveAll(paths.LockDir)

	// Write PID atomically
	if err := writePIDAtomically(paths.PIDFile, os.Getpid()); err != nil {
		return err
	}
	// Write ready marker atomically
	if err := writeReadyAtomically(paths.ReadyFile); err != nil {
		return err
	}

	// Handle signals for graceful shutdown
	sigs := make(chan os.Signal, 2)
	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT)
	defer signal.Stop(sigs)

	slog.Info("shim child running", "vm", vm.Name, "pid", os.Getpid())

	// Placeholder for provider start; block until signal
	select {
	case <-ctx.Done():
		// context canceled
	case <-sigs:
		// got signal
	}

	// Cleanup readiness and pid on exit
	_ = os.Remove(paths.ReadyFile)
	_ = os.Remove(paths.PIDFile)
	return nil
}

func writePIDAtomically(path string, pid int) error {
	d := filepath.Dir(path)
	if err := os.MkdirAll(d, 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	w := bufio.NewWriter(f)
	if _, err := fmt.Fprintf(w, "%d\n", pid); err != nil {
		f.Close()
		return err
	}
	if err := w.Flush(); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	// fsync dir best-effort
	if df, err := os.Open(d); err == nil {
		_ = df.Sync()
		_ = df.Close()
	}
	return nil
}

func writeReadyAtomically(path string) error {
	d := filepath.Dir(path)
	if err := os.MkdirAll(d, 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(time.Now().Format(time.RFC3339Nano)), 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	if df, err := os.Open(d); err == nil {
		_ = df.Sync()
		_ = df.Close()
	}
	return nil
}

// note: base dir selection is performed by the caller constructing the store
