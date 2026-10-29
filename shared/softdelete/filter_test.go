package softdelete_test

import (
	"testing"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/softdelete"
)

// The default must be the literal nil every repository wrote by hand before, or
// an untouched caller silently changes what it matches.
func TestClause_LiveRowsAreNil(t *testing.T) {
	if got := softdelete.Clause(false); got != nil {
		t.Errorf("Clause(false) = %v, want nil", got)
	}
}

func TestClause_DeletedRowsAreNotNil(t *testing.T) {
	got, ok := softdelete.Clause(true).(bson.M)
	if !ok {
		t.Fatalf("Clause(true) = %T, want bson.M", softdelete.Clause(true))
	}
	if v, present := got["$ne"]; !present || v != nil {
		t.Errorf("Clause(true) = %v, want {$ne: nil}", got)
	}
}

// Master data expresses soft delete as active:false , so the
// polarity is inverted. Getting this backwards would show the whole live
// catalogue under a heading that says "deleted".
func TestActive_IsInverted(t *testing.T) {
	if !softdelete.Active(false) {
		t.Error("Active(false) = false, want true — a live list matches active rows")
	}
	if softdelete.Active(true) {
		t.Error("Active(true) = true, want false — the reveal matches removed rows")
	}
}
