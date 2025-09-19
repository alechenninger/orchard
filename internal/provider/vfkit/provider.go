package vfkit

import (
	"context"
	"sync"

	"github.com/alechenninger/orchard/internal/domain"
)

type Provider struct {
	mu sync.Mutex
	// later: hold vfkit VM handles by name
	running map[string]struct{}
}

func New() *Provider { return &Provider{running: make(map[string]struct{})} }

func (p *Provider) StartVM(ctx context.Context, vm domain.VM) (int, error) {
	p.mu.Lock()
	p.running[vm.Name] = struct{}{}
	p.mu.Unlock()
	// TODO: create vfkit VM using library and start it
	// For now, return current shim pid as process owner
	return 0, nil
}

func (p *Provider) StopVM(ctx context.Context, vm domain.VM) error {
	p.mu.Lock()
	delete(p.running, vm.Name)
	p.mu.Unlock()
	// TODO: stop vfkit VM instance
	return nil
}

func (p *Provider) IsRunning(ctx context.Context, vm domain.VM) (bool, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	_, ok := p.running[vm.Name]
	return ok, nil
}

var _ domain.VirtualizationProvider = (*Provider)(nil)
