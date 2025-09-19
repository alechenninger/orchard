package mem

import (
	"context"

	"github.com/alechenninger/orchard/internal/domain"
)

type Service struct{}

func New() *Service { return &Service{} }

func (s *Service) Prepare(ctx context.Context, vm *domain.VM) error {
	// No-op for tests
	return nil
}

var _ domain.VMArtifacts = (*Service)(nil)
