package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	apperrors "github.com/breadgarlicbigint/bread-golang-boilerplate/shared/errors"
)

func TestValidatePermissionGrant(t *testing.T) {
	cases := []struct {
		name           string
		extra, revoked []string
		wantErr        bool
	}{
		{"valid disjoint", []string{"users:read"}, []string{"users:write"}, false},
		{"unknown slug", []string{"not:a:perm"}, nil, true},
		{"wildcard not grantable", []string{"*"}, nil, true},
		{"same slug in both", []string{"users:read"}, []string{"users:read"}, true},
		{"empty is fine", nil, nil, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := validatePermissionGrant(c.extra, c.revoked)
			if (err != nil) != c.wantErr {
				t.Errorf("err = %v, wantErr %v", err, c.wantErr)
			}
			if c.wantErr && err != apperrors.ErrInvalidPermissionGrant {
				t.Errorf("want ErrInvalidPermissionGrant, got %v", err)
			}
		})
	}
}

func TestGuardAssignableRole(t *testing.T) {
	s := &UserService{}
	// A wildcard actor may assign anything, even without a resolver (returns
	// before the id is looked up).
	if err := s.guardAssignableRole(context.Background(), Actor{HasWildcard: true}, uuid.Nil); err != nil {
		t.Errorf("wildcard actor should assign any role, got %v", err)
	}
	// A non-wildcard actor with no resolver wired fails closed.
	if err := s.guardAssignableRole(context.Background(), Actor{}, uuid.Nil); err != apperrors.ErrRoleNotAssignable {
		t.Errorf("no resolver should fail closed, got %v", err)
	}
}
