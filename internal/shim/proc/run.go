package proc

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/alechenninger/orchard/internal/domain"
)

// RunChild is invoked in the _shim process to own the VM lifecycle.
func RunChild(ctx context.Context, store domain.VMStore, run domain.RuntimeState, provider domain.VirtualizationProvider, name string) error {
	// Virtualization.framework APIs require running on the main thread
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	vm, err := store.Load(ctx, name)
	if err != nil {
		return err
	}
	// Acquire lock
	release, err := run.AcquireLock(ctx, vm.Name)
	if err != nil {
		return err
	}
	defer release()

	// Write PID atomically early so Stop can find us even if provider errors
	if err := run.WritePID(ctx, vm.Name, os.Getpid()); err != nil {
		return err
	}
	// Start the VM via provider
	if _, err := provider.StartVM(ctx, *vm); err != nil {
		// Do not mark ready; ensure we exit with error so parent fails fast
		return err
	}

	// Mark ready only after successful provider start
	if err := run.MarkReady(ctx, vm.Name); err != nil {
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

	// Stop VM
	_ = provider.StopVM(ctx, *vm)

	// Cleanup readiness and pid on exit
	_ = run.Clear(ctx, vm.Name)
	return nil
}

// note: base dir selection is performed by the caller constructing the store
