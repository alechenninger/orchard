package mem

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/alechenninger/orchard/internal/domain"
)

type Store struct {
	mu   sync.Mutex
	vms  map[string]domain.VM // key: name
	next int
}

func New() *Store {
	return &Store{vms: make(map[string]domain.VM), next: 1}
}

func (s *Store) NextName(ctx context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	name := fmt.Sprintf("vm-%03d", s.next)
	s.next++
	return name, nil
}

func (s *Store) Save(ctx context.Context, vm domain.VM) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if vm.CreatedAt == 0 {
		vm.CreatedAt = time.Now().UnixNano()
	}
	s.vms[vm.Name] = vm
	return nil
}

func (s *Store) Load(ctx context.Context, nameOrID string) (*domain.VM, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	vm, ok := s.vms[nameOrID]
	if !ok {
		return nil, fmt.Errorf("vm %s not found", nameOrID)
	}
	v := vm
	return &v, nil
}

func (s *Store) Delete(ctx context.Context, nameOrID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.vms, nameOrID)
	return nil
}

func (s *Store) List(ctx context.Context) ([]domain.VM, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	vms := make([]domain.VM, 0, len(s.vms))
	for _, vm := range s.vms {
		vms = append(vms, vm)
	}
	sort.Slice(vms, func(i, j int) bool { return vms[i].CreatedAt < vms[j].CreatedAt })
	return vms, nil
}

var _ domain.VMStore = (*Store)(nil)

func (s *Store) RuntimePaths(ctx context.Context, name string) (domain.VMRuntimePaths, error) {
	return domain.VMRuntimePaths{
		Dir:         "/mem/" + name,
		PIDFile:     "/mem/" + name + "/vm.pid",
		ReadyFile:   "/mem/" + name + "/vm.ready",
		LockDir:     "/mem/" + name + "/vm.lock.d",
		ConsoleSock: "/mem/" + name + "/console.sock",
	}, nil
}
