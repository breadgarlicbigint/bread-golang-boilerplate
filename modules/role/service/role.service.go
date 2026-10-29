package service

import (
	"context"

	"github.com/breadgarlicbigint/bread-golang-boilerplate/modules/role/entity"
)

// RoleRepo is the data-access dependency this service needs from the
// repository package (kept minimal so it's easy to fake in unit tests).
type RoleRepo interface {
	FindAll(ctx context.Context) ([]entity.Role, error)
	ListAssignable(ctx context.Context) ([]entity.Role, error)
}

type RoleService struct {
	repo RoleRepo
}

func New(repo RoleRepo) *RoleService {
	return &RoleService{repo: repo}
}

// List returns every role, used to populate role-selection dropdowns.
func (s *RoleService) List(ctx context.Context) ([]entity.Role, error) {
	return s.repo.FindAll(ctx)
}

// ListAssignable returns only the roles a non-wildcard admin may assign (the
// create-user dropdown). The authoritative check is still server-side at user
// creation — this only shapes the UI.
func (s *RoleService) ListAssignable(ctx context.Context) ([]entity.Role, error) {
	return s.repo.ListAssignable(ctx)
}
