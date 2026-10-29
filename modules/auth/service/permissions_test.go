package service

import (
	"strings"
	"testing"

	roleentity "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/role/entity"
)

func TestUnionPermissions_DropsDuplicatesKeepsOrder(t *testing.T) {
	got := unionPermissions([]string{"a", "b"}, []string{"b", "c"})
	if strings.Join(got, ",") != "a,b,c" {
		t.Errorf("union = %v, want [a b c]", got)
	}
	// No extras → the role list is returned unchanged.
	role := []string{"x"}
	if got := unionPermissions(role, nil); strings.Join(got, ",") != "x" {
		t.Errorf("union with no extras = %v", got)
	}
}

func TestEffectivePermissions_UnionMinusRevoked(t *testing.T) {
	got := effectivePermissions([]string{"a", "b"}, []string{"c"}, []string{"b"})
	if strings.Join(got, ",") != "a,c" {
		t.Errorf("effective = %v, want [a c]", got)
	}
}

func TestEffectivePermissions_WildcardIsNeverSubtracted(t *testing.T) {
	// "*" is a claim that every gate passes, not an enumerable set — revoking a
	// slug from a wildcard holder must leave the wildcard intact.
	got := effectivePermissions([]string{string(roleentity.PermAll)}, nil, []string{"users:write"})
	if len(got) != 1 || got[0] != string(roleentity.PermAll) {
		t.Errorf("wildcard was subtracted from: %v", got)
	}
}
