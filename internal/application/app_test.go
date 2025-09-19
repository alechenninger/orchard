package application

import (
	"context"
	"runtime"
	"testing"

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
