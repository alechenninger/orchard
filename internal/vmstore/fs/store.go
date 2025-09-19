package fs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/alechenninger/orchard/internal/domain"
)

type Store struct {
	baseDir string
	mu      sync.Mutex
}

func New(baseDir string) *Store {
	return &Store{baseDir: baseDir}
}

// NewDefault constructs a Store rooted at the default base directory.
func NewDefault() *Store {
	return New(DefaultBaseDir())
}

func (s *Store) ensureDirs() error {
	return os.MkdirAll(filepath.Join(s.baseDir, "vms"), 0o755)
}

func (s *Store) vmDir(name string) string {
	return filepath.Join(s.baseDir, "vms", name)
}

func (s *Store) NextName(ctx context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureDirs(); err != nil {
		return "", err
	}
	seqFile := filepath.Join(s.baseDir, "state", "names.json")
	if err := os.MkdirAll(filepath.Dir(seqFile), 0o755); err != nil {
		return "", err
	}
	type namesState struct{ Next int }
	st := namesState{Next: 1}
	if b, err := os.ReadFile(seqFile); err == nil {
		_ = json.Unmarshal(b, &st)
	}
	name := fmt.Sprintf("vm-%03d", st.Next)
	st.Next++
	b, _ := json.MarshalIndent(st, "", "  ")
	if err := os.WriteFile(seqFile, b, 0o644); err != nil {
		return "", err
	}
	return name, nil
}

func (s *Store) Save(ctx context.Context, vm domain.VM) error {
	if err := s.ensureDirs(); err != nil {
		return err
	}
	d := s.vmDir(vm.Name)
	if err := os.MkdirAll(d, 0o755); err != nil {
		return err
	}
	if vm.CreatedAt == 0 {
		vm.CreatedAt = time.Now().UnixNano()
	}
	b, _ := json.MarshalIndent(vm, "", "  ")
	return os.WriteFile(filepath.Join(d, "config.json"), b, 0o644)
}

func (s *Store) Load(ctx context.Context, nameOrID string) (*domain.VM, error) {
	if err := s.ensureDirs(); err != nil {
		return nil, err
	}
	d := s.vmDir(nameOrID)
	b, err := os.ReadFile(filepath.Join(d, "config.json"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("vm %s not found", nameOrID)
		}
		return nil, err
	}
	var vm domain.VM
	if err := json.Unmarshal(b, &vm); err != nil {
		return nil, err
	}
	return &vm, nil
}

func (s *Store) Delete(ctx context.Context, nameOrID string) error {
	if err := s.ensureDirs(); err != nil {
		return err
	}
	d := s.vmDir(nameOrID)
	return os.RemoveAll(d)
}

func (s *Store) List(ctx context.Context) ([]domain.VM, error) {
	if err := s.ensureDirs(); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(filepath.Join(s.baseDir, "vms"))
	if err != nil {
		return nil, err
	}
	var vms []domain.VM
	for _, e := range entries {
		if !e.IsDir() || !strings.HasPrefix(e.Name(), "vm-") {
			continue
		}
		vm, err := s.Load(ctx, e.Name())
		if err == nil && vm != nil {
			vms = append(vms, *vm)
		}
	}
	sort.Slice(vms, func(i, j int) bool { return vms[i].CreatedAt < vms[j].CreatedAt })
	return vms, nil
}

var _ domain.VMStore = (*Store)(nil)

func (s *Store) RuntimePaths(ctx context.Context, name string) (domain.VMRuntimePaths, error) {
	if err := s.ensureDirs(); err != nil {
		return domain.VMRuntimePaths{}, err
	}
	d := s.vmDir(name)
	return domain.VMRuntimePaths{
		Dir:         d,
		PIDFile:     filepath.Join(d, "vm.pid"),
		ReadyFile:   filepath.Join(d, "vm.ready"),
		LockDir:     filepath.Join(d, "vm.lock.d"),
		ConsoleSock: filepath.Join(d, "console.sock"),
	}, nil
}

// DefaultBaseDir returns the default base directory for VM state.
func DefaultBaseDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "orchard")
	}
	return filepath.Join(home, ".orchard")
}
