package application

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	artfs "github.com/alechenninger/orchard/internal/artifacts/fs"
	"github.com/alechenninger/orchard/internal/domain"
	fsstore "github.com/alechenninger/orchard/internal/vmstore/fs"
	"github.com/spf13/afero"
)

type noopBuilder struct{}

func (noopBuilder) Build(ctx context.Context, fs afero.Fs, srcDir string, dstPath string) error {
	a := &afero.Afero{Fs: fs}
	_ = a.MkdirAll(filepath.Dir(dstPath), 0o755)
	return a.WriteFile(dstPath, []byte("iso"), 0o644)
}

func TestUpCreatesVMAndLists(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// Use in-memory FS-backed services
	memfs := afero.NewMemMapFs()
	store := fsstore.NewWithFS("/testroot", memfs)
	art := artfs.NewWithFS("/testroot", memfs)
	app := New(store, &fakeShim{}, art, memfs, noopBuilder{})

	// Create fake base image and SSH key in memfs
	img := "/testroot/image.img"
	key := "/testroot/id_ed25519.pub"
	_ = afero.WriteFile(memfs, img, []byte("base"), 0o644)
	_ = afero.WriteFile(memfs, key, []byte("ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQ test"), 0o644)
	app.Clock = fixedClock{t: time.Unix(0, 1)}

	vm1, err := app.Up(ctx, UpParams{ImagePath: img, CPUs: 2, MemoryMiB: 1024, DiskSizeGiB: 10, SSHKeyPath: key})
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
	app.Clock = fixedClock{t: time.Unix(0, 2)}
	vm2, err := app.Up(ctx, UpParams{ImagePath: img, CPUs: 2, MemoryMiB: 1024, DiskSizeGiB: 10, SSHKeyPath: key})
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
func (f *fakeShim) WaitReadyAndPID(ctx context.Context, vmName string) (int, error) {
	return f.nextPID, nil
}
func (f *fakeShim) GetPID(ctx context.Context, vmName string) (int, error) { return f.nextPID, nil }

type fixedClock struct{ t time.Time }

func (f fixedClock) Now() time.Time { return f.t }

func TestStartStopUpdatesStore(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	memfs := afero.NewMemMapFs()
	store := fsstore.NewWithFS("/testroot", memfs)
	art := artfs.NewWithFS("/testroot", memfs)
	shim := &fakeShim{}
	app := New(store, shim, art, memfs, noopBuilder{})

	img := "/testroot/image.img"
	key := "/testroot/id_ed25519.pub"
	_ = afero.WriteFile(memfs, img, []byte("base"), 0o644)
	_ = afero.WriteFile(memfs, key, []byte("ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQ test"), 0o644)

	vm, err := app.Up(ctx, UpParams{ImagePath: img, SSHKeyPath: key})
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

func TestDeleteNonRunning(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	memfs := afero.NewMemMapFs()
	store := fsstore.NewWithFS("/testroot", memfs)
	art := artfs.NewWithFS("/testroot", memfs)
	app := New(store, &fakeShim{}, art, memfs, noopBuilder{})

	img := "/testroot/image.img"
	key := "/testroot/id_ed25519.pub"
	_ = afero.WriteFile(memfs, img, []byte("base"), 0o644)
	_ = afero.WriteFile(memfs, key, []byte("ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQ test"), 0o644)

	vm, err := app.Up(ctx, UpParams{ImagePath: img, SSHKeyPath: key})
	if err != nil {
		t.Fatalf("up failed: %v", err)
	}

	if err := app.Delete(ctx, vm.Name, false); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	if _, err := app.Store.Load(ctx, vm.Name); err == nil {
		t.Fatalf("expected not found after delete")
	}
}

func TestDeleteRunningRequiresForce(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	memfs := afero.NewMemMapFs()
	store := fsstore.NewWithFS("/testroot", memfs)
	art := artfs.NewWithFS("/testroot", memfs)
	shim := &fakeShim{}
	app := New(store, shim, art, memfs, noopBuilder{})

	img := "/testroot/image.img"
	key := "/testroot/id_ed25519.pub"
	_ = afero.WriteFile(memfs, img, []byte("base"), 0o644)
	_ = afero.WriteFile(memfs, key, []byte("ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQ test"), 0o644)

	vm, err := app.Up(ctx, UpParams{ImagePath: img, SSHKeyPath: key})
	if err != nil {
		t.Fatalf("up failed: %v", err)
	}
	// Simulate running by making shim report a PID
	shim.nextPID = 1234

	if err := app.Delete(ctx, vm.Name, false); err == nil {
		t.Fatalf("expected error when deleting running VM without force")
	}
	// With force, delete should succeed
	if err := app.Delete(ctx, vm.Name, true); err != nil {
		t.Fatalf("force delete failed: %v", err)
	}
}
