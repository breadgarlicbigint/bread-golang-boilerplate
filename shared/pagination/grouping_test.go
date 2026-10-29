package pagination_test

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/pagination"
)

var fields = pagination.GroupFields{
	"status":  {Path: "approvalStatus", Kind: pagination.GroupString},
	"flag":    {Path: "isImportant", Kind: pagination.GroupBool},
	"company": {Path: "fitment.carCompanyId", Kind: pagination.GroupUUID},
}

// queryFrom builds a Query the way a real request does, so repeated params go
// through the same path handlers use.
func queryFrom(t *testing.T, rawQuery string) pagination.Query {
	t.Helper()
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/v1/parts?"+rawQuery, nil)
	return pagination.FromContext(c)
}

func TestQuery_UngroupedRequestIsUnchanged(t *testing.T) {
	q := queryFrom(t, "page=2&perPage=10")
	if q.IsGrouped() {
		t.Fatalf("a request with no groupBy must not be grouped: %+v", q)
	}
	// Paging is untouched by grouping either way — this is the  guarantee
	// that a grouped response still pages records.
	if q.Page != 2 || q.PerPage != 10 {
		t.Errorf("paging = page %d perPage %d, want 2/10", q.Page, q.PerPage)
	}
}

// Paging must survive grouping.  paged GROUPS, which made 12 records at
// perPage 10 collapse to one page as soon as they fell into one group — the bug
// this contract version exists to fix.
func TestQuery_GroupingDoesNotDisturbPaging(t *testing.T) {
	q := queryFrom(t, "groupBy=status&groupBy=company&page=3&perPage=25")
	if !q.IsGrouped() {
		t.Fatal("IsGrouped = false, want true")
	}
	if q.Page != 3 || q.PerPage != 25 {
		t.Errorf("paging = page %d perPage %d, want 3/25", q.Page, q.PerPage)
	}
	if q.Skip() != 50 || q.Limit() != 25 {
		t.Errorf("skip/limit = %d/%d, want 50/25", q.Skip(), q.Limit())
	}
}

func TestQuery_GroupByRepeatsInNestingOrder(t *testing.T) {
	q := queryFrom(t, "groupBy=status&groupBy=company")
	if len(q.GroupBy) != 2 || q.GroupBy[0] != "status" || q.GroupBy[1] != "company" {
		t.Fatalf("GroupBy = %v, want [status company] in order", q.GroupBy)
	}
}

func TestGroupFields_ResolveRejectsUnknownKey(t *testing.T) {
	if _, err := fields.Resolve("status"); err != nil {
		t.Fatalf("known key rejected: %v", err)
	}
	// A raw bson path must not be usable as a group key — that is the whole
	// point of the whitelist, and it also guards the $sort this now feeds.
	for _, bad := range []string{"approvalStatus", "tenantId", "$where", ""} {
		if _, err := fields.Resolve(bad); err == nil {
			t.Errorf("Resolve(%q) succeeded, want rejection", bad)
		}
	}
}

func TestGroupFields_Levels(t *testing.T) {
	t.Run("ungrouped yields no levels and no error", func(t *testing.T) {
		levels, err := fields.Levels(queryFrom(t, "page=1"))
		if err != nil || len(levels) != 0 {
			t.Fatalf("levels = %v, err = %v", levels, err)
		}
	})

	t.Run("resolves every level in nesting order", func(t *testing.T) {
		levels, err := fields.Levels(queryFrom(t, "groupBy=status&groupBy=company"))
		if err != nil {
			t.Fatal(err)
		}
		if len(levels) != 2 {
			t.Fatalf("levels = %d, want 2", len(levels))
		}
		if levels[0].Path != "approvalStatus" || levels[1].Path != "fitment.carCompanyId" {
			t.Errorf("levels = %+v, want status then company", levels)
		}
	})

	t.Run("rejects malformed grouping", func(t *testing.T) {
		bad := []struct{ name, raw string }{
			{"unknown level", "groupBy=nope"},
			{"raw bson path as a level", "groupBy=approvalStatus"},
			{"same level twice", "groupBy=status&groupBy=status"},
			{"deeper than the cap", "groupBy=status&groupBy=flag&groupBy=company&groupBy=nope2"},
		}
		for _, tc := range bad {
			t.Run(tc.name, func(t *testing.T) {
				if _, err := fields.Levels(queryFrom(t, tc.raw)); err == nil {
					t.Errorf("Levels(%q) succeeded, want rejection", tc.raw)
				}
			})
		}
	})
}

// A UUID group value must reach the client as the id it already knows. The
// driver hands back a primitive.Binary (UUIDs are BSON Binary subtype 4), which
// would otherwise marshal as {"Subtype":4,"Data":"…"} — unmatchable by any
// client trying to label the group.
func TestDecodeGroupKey_UUID(t *testing.T) {
	id := uuid.New()
	stored := primitive.Binary{Subtype: 0x04, Data: id[:]}
	if got := pagination.DecodeGroupKey(stored, pagination.GroupUUID); got != id.String() {
		t.Fatalf("DecodeGroupKey = %#v, want %q", got, id.String())
	}
}

func TestDecodeGroupKey_KindsAndAbsence(t *testing.T) {
	if got := pagination.DecodeGroupKey("APPROVED", pagination.GroupString); got != "APPROVED" {
		t.Errorf("string key = %#v", got)
	}
	if got := pagination.DecodeGroupKey(true, pagination.GroupBool); got != true {
		t.Errorf("bool key = %#v", got)
	}
	// A null value and a field absent from the document both arrive as a nil
	// `any` — the "no value" group, which is a real group.
	if got := pagination.DecodeGroupKey(nil, pagination.GroupString); got != nil {
		t.Errorf("null key = %#v, want nil", got)
	}
	// A value of the wrong BSON type must not panic; it degrades to that group.
	if got := pagination.DecodeGroupKey("APPROVED", pagination.GroupBool); got != nil {
		t.Errorf("mistyped key = %#v, want nil", got)
	}
	if got := pagination.DecodeGroupKey(primitive.Binary{Data: []byte{1, 2}}, pagination.GroupUUID); got != nil {
		t.Errorf("short uuid binary = %#v, want nil", got)
	}
}

// $sum comes back int32 while counts are small and int64 once they grow —
// reading only one silently reports every group as empty.
func TestGroupCountOf(t *testing.T) {
	cases := []struct {
		in   any
		want int64
	}{
		{int32(42), 42},
		{int64(42), 42},
		{float64(42), 42},
		{nil, 0},
		{"42", 0},
	}
	for _, tc := range cases {
		if got := pagination.GroupCountOf(tc.in); got != tc.want {
			t.Errorf("GroupCountOf(%#v) = %d, want %d", tc.in, got, tc.want)
		}
	}
}
