package application

import (
	"context"
	"runtime"
	"testing"

	"github.com/alechenninger/orchard/internal/domain"
	"github.com/alechenninger/orchard/internal/vmstore/mem"
)

func TestUpCreatesVMAndLists(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// Use a test image path that exists: use the current file as a stand-in
	_, file, _, _ := runtime.Caller(0)
	img := file

	store := mem.New()
	app := New(store)

	vm1, err := app.Up(ctx, UpParams{ImagePath: img, CPUs: 2, MemoryMiB: 1024, DiskSizeGiB: 10})
	if err != nil {
		t.Fatalf("Up failed: %v", err)
	}
	if vm1.Name == "" {
		t.Fatalf("expected name to be set")
	}

	vms, err := app.ListVMs(ctx)
	if err != nil {
		t.Fatalf("ListVMs failed: %v", err)
	}
	if len(vms) != 1 {
		t.Fatalf("expected 1 VM, got %d", len(vms))
	}
	if vms[0].Name != vm1.Name {
		t.Fatalf("unexpected VM name: %s", vms[0].Name)
	}

	// Create a second VM and ensure ordering by CreatedAt
	vm2, err := app.Up(ctx, UpParams{ImagePath: img, CPUs: 2, MemoryMiB: 1024, DiskSizeGiB: 10})
	if err != nil {
		t.Fatalf("Up 2 failed: %v", err)
	}
	vms, err = app.ListVMs(ctx)
	if err != nil {
		t.Fatalf("ListVMs 2 failed: %v", err)
	}
	if len(vms) != 2 {
		t.Fatalf("expected 2 VMs, got %d", len(vms))
	}
	if vms[0].Name != vm1.Name || vms[1].Name != vm2.Name {
		t.Fatalf("unexpected order: %v then %v", vms[0].Name, vms[1].Name)
	}
}

type fakeShim struct{ nextPID int }

func (f *fakeShim) StartDetached(ctx context.Context, vm domain.VM) (int, error) {
	f.nextPID++
	return f.nextPID, nil
}
func (f *fakeShim) Stop(ctx context.Context, pid int) error { return nil }

func TestStartStopUpdatesStore(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	_, file, _, _ := runtime.Caller(0)

	store := mem.New()
	app := New(store)
	app.Shim = &fakeShim{}

	vm, err := app.Up(ctx, UpParams{ImagePath: file})
	if err != nil {
		t.Fatalf("up failed: %v", err)
	}
	if vm.Status != "stopped" {
		t.Fatalf("expected stopped, got %s", vm.Status)
	}

	vm, err = app.Start(ctx, vm.Name)
	if err != nil {
		t.Fatalf("start failed: %v", err)
	}
	if vm.Status != "running" || vm.PID == 0 {
		t.Fatalf("expected running with pid, got %v pid=%d", vm.Status, vm.PID)
	}

	if err := app.Stop(ctx, vm.Name); err != nil {
		t.Fatalf("stop failed: %v", err)
	}
	vm2, err := app.Store.Load(ctx, vm.Name)
	if err != nil {
		t.Fatalf("load after stop failed: %v", err)
	}
	if vm2.Status != "stopped" || vm2.PID != 0 {
		t.Fatalf("expected stopped with pid=0, got %v pid=%d", vm2.Status, vm2.PID)
	}
}
