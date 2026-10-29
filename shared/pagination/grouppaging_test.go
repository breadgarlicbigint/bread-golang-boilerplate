package pagination

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
)

// Group paging's arithmetic. Everything here is pure,
// which is the point: three list endpoints cannot use the Mongo helper and must
// call this same planner, so the window calculation is tested once rather than
// trusted three times.

func q(page, perPage int, opts ...func(*Query)) Query {
	out := Query{Page: page, PerPage: perPage}
	for _, o := range opts {
		o(&out)
	}
	out.Normalize()
	return out
}

func expanded(paths ...string) func(*Query) {
	return func(x *Query) { x.Expanded = paths }
}

func expandAll() func(*Query) {
	return func(x *Query) { x.ExpandAll = true }
}

func collapsed(paths ...string) func(*Query) {
	return func(x *Query) { x.Collapsed = paths }
}

// leaves builds a one-level tree: APPROVED(12), PENDING(4), REJECTED(2).
func oneLevel() []*GroupNode {
	return BuildGroupTree([]GroupLeaf{
		{Path: []any{"APPROVED"}, Count: 12, Summary: 1200},
		{Path: []any{"PENDING"}, Count: 4, Summary: 400},
		{Path: []any{"REJECTED"}, Count: 2, Summary: 200},
	}, 1)
}

func kinds(items []any) string {
	var b strings.Builder
	for _, it := range items {
		if _, ok := it.(*GroupHeaderItem); ok {
			b.WriteByte('H')
		} else {
			b.WriteByte('r')
		}
	}
	return b.String()
}

func TestPlanWindow_EverythingClosedByDefault(t *testing.T) {
	plan := PlanWindow(oneLevel(), 1, NewExpansionSet(q(1, 10)), false, q(1, 10))

	// Collapsed is the default (group-paging), so the whole list is three headers.
	if plan.VisibleTotal != 3 {
		t.Fatalf("visibleTotal = %d, want 3", plan.VisibleTotal)
	}
	// The assertion that matters: NO row query is issued at all. A version that
	// fetched rows and hid them client-side would pass every render test and
	// still be the design this replaced.
	if plan.RowLimit != 0 {
		t.Fatalf("rowLimit = %d, want 0 when nothing is open", plan.RowLimit)
	}
	if len(plan.OpenLeaves) != 0 {
		t.Fatalf("openLeaves = %v, want none", plan.OpenLeaves)
	}
	if plan.AllOpen {
		t.Fatal("AllOpen must be false when every group is closed")
	}
	for _, slot := range plan.Slots {
		if slot.Header == nil {
			t.Fatal("a closed tree must yield header slots only")
		}
		if !slot.Header.Collapsed {
			t.Fatalf("header %v reported open", slot.Header.Path)
		}
	}
}

func TestPlanWindow_HeadersConsumePageSlots(t *testing.T) {
	// The DevExpress rule, and the one the whole mode is judged by: perPage=10
	// over three headers and one open group of 12 fills page 1 with 3 headers +
	// 7 rows, NOT 3 headers + 10 rows.
	query := q(1, 10, expanded(`["APPROVED"]`))
	plan := PlanWindow(oneLevel(), 1, NewExpansionSet(query), false, query)

	if got := len(plan.Slots); got != 10 {
		t.Fatalf("page holds %d slots, want exactly perPage=10", got)
	}
	items := AssembleWindow(plan, make([]string, plan.RowLimit))
	if got := kinds(items); got != "Hrrrrrrrrr" {
		// APPROVED's header takes the first slot, so only NINE of its twelve
		// rows fit — the tenth is what a header-free pager would have shown.
		t.Fatalf("page 1 sequence = %q, want %q", got, "Hrrrrrrrrr")
	}
	if plan.RowLimit != 9 {
		t.Fatalf("rowLimit = %d, want 9 (10 slots minus the header)", plan.RowLimit)
	}
	if plan.VisibleTotal != 15 {
		t.Fatalf("visibleTotal = %d, want 15 (3 headers + 12 rows)", plan.VisibleTotal)
	}
}

func TestPlanWindow_RowSkipCountsOpenGroupsOnly(t *testing.T) {
	// THE assertion that catches the plausible wrong version — the one that
	// windows RECORDS and hides the closed ones afterwards.
	//
	// APPROVED (12 rows) is CLOSED and sorts FIRST, which is the only
	// arrangement where the two implementations disagree: its records are absent
	// from the restricted row set, so page 2 must skip 2 rows, never 12 + 2. Had
	// the closed group sorted last, both versions would agree and this would
	// assert nothing.
	query := q(2, 4, expanded(`["PENDING"]`, `["REJECTED"]`))
	plan := PlanWindow(oneLevel(), 1, NewExpansionSet(query), false, query)

	if plan.RowSkip != 2 {
		t.Fatalf("rowSkip = %d, want 2 — a closed group's records are not in the row set", plan.RowSkip)
	}
	// Slots: H(APPROVED, closed) H(PENDING) p1 p2 | p3 p4 H(REJECTED) r1 | r2
	// Page 2 opens inside PENDING, so its header is REPEATED at the top — the
	// leading H below — without consuming one of the four slots.
	items := AssembleWindow(plan, make([]string, plan.RowLimit))
	if got := kinds(items); got != "HrrHr" {
		t.Fatalf("page 2 sequence = %q, want %q", got, "HrrHr")
	}
	if plan.RowLimit != 3 {
		t.Fatalf("rowLimit = %d, want 3", plan.RowLimit)
	}
	if plan.VisibleTotal != 9 {
		t.Fatalf("visibleTotal = %d, want 9 (3 headers + 4 + 2 rows)", plan.VisibleTotal)
	}
	// The closed group's 12 records are still counted as records — only the
	// pager stops seeing them.
	if len(plan.OpenLeaves) != 2 {
		t.Fatalf("openLeaves = %v, want PENDING and REJECTED only", plan.OpenLeaves)
	}
}

func TestPlanWindow_RepeatsTheHeaderOfAGroupContinuedFromTheLastPage(t *testing.T) {
	// A group of 12 rows at perPage 4 covers three pages. Without the repeat,
	// pages 2 and 3 show rows under no heading at all — and the control that
	// would collapse the group is two pages back, so the operator cannot close
	// the thing they are looking at.
	query := q(2, 4, expanded(`["APPROVED"]`))
	plan := PlanWindow(oneLevel(), 1, NewExpansionSet(query), false, query)

	if len(plan.Slots) == 0 || plan.Slots[0].Header == nil {
		t.Fatal("page 2 must open with the continued header")
	}
	head := plan.Slots[0].Header
	if !head.Continued {
		t.Error("a repeated header must be marked Continued — a client keys and styles it differently")
	}
	if PathID(head.Path) != `["APPROVED"]` || head.Count != 12 {
		t.Errorf("repeat describes %v (%d), want APPROVED (12)", head.Path, head.Count)
	}
	if head.Collapsed {
		t.Error("the repeat must carry the group's RESOLVED state, which is open here")
	}

	// It is CONTEXT, not content: the page still holds its four real slots and
	// the sequence length has not moved.
	if plan.RowLimit != 4 {
		t.Errorf("rowLimit = %d, want 4 — a repeat must not eat a slot", plan.RowLimit)
	}
	if plan.VisibleTotal != 15 {
		t.Errorf("visibleTotal = %d, want 15 — repeats are not part of the sequence", plan.VisibleTotal)
	}

	// A page that opens ON a header needs no repeat.
	first := q(1, 4, expanded(`["APPROVED"]`))
	if h := PlanWindow(oneLevel(), 1, NewExpansionSet(first), false, first).Slots[0].Header; h == nil || h.Continued {
		t.Error("page 1 opens on APPROVED's real header, which must not be marked Continued")
	}
}

func TestPlanWindow_RepeatsEveryAncestorOfANestedContinuation(t *testing.T) {
	// Two levels deep, the page must open with BOTH headings: the status and the
	// company inside it. Repeating only the innermost leaves the row set looking
	// like a top-level group.
	tree := BuildGroupTree([]GroupLeaf{
		{Path: []any{"APPROVED", "honda"}, Count: 6},
	}, 2)
	query := q(2, 3, expanded(`["APPROVED"]`, `["APPROVED","honda"]`))
	plan := PlanWindow(tree, 2, NewExpansionSet(query), false, query)

	if got := kinds(AssembleWindow(plan, make([]string, plan.RowLimit))); got != "HHrrr" {
		t.Fatalf("nested continuation = %q, want %q", got, "HHrrr")
	}
	if !plan.Slots[0].Header.Continued || !plan.Slots[1].Header.Continued {
		t.Error("both repeated ancestors must be marked Continued")
	}
	if plan.Slots[0].Header.Depth != 0 || plan.Slots[1].Header.Depth != 1 {
		t.Error("the repeats must carry their real depths, or the page indents wrongly")
	}
}

func TestPlanWindow_VisibleTotalIsIndependentOfThePageAsked(t *testing.T) {
	// The pager reads this off whichever page it happens to be on, so it must
	// describe the whole sequence rather than the window.
	for _, page := range []int{1, 2, 3, 99} {
		query := q(page, 5, expandAll())
		plan := PlanWindow(oneLevel(), 1, NewExpansionSet(query), false, query)
		if plan.VisibleTotal != 21 {
			t.Fatalf("page %d: visibleTotal = %d, want 21", page, plan.VisibleTotal)
		}
	}
}

func TestPlanWindow_PastTheEndIsEmptyNotNegative(t *testing.T) {
	query := q(99, 10, expandAll())
	plan := PlanWindow(oneLevel(), 1, NewExpansionSet(query), false, query)
	if len(plan.Slots) != 0 || plan.RowLimit != 0 || plan.RowSkip != 0 {
		t.Fatalf("past the end must be empty, got %d slots skip=%d limit=%d",
			len(plan.Slots), plan.RowSkip, plan.RowLimit)
	}
	if plan.VisibleTotal != 21 {
		t.Fatalf("visibleTotal = %d, want 21 even past the end", plan.VisibleTotal)
	}
}

func TestPlanWindow_ExpandAllOpensEverythingAndNeedsNoRestriction(t *testing.T) {
	query := q(1, 100, expandAll())
	plan := PlanWindow(oneLevel(), 1, NewExpansionSet(query), false, query)
	if !plan.AllOpen {
		t.Fatal("AllOpen must be true so the row query carries no $or at all")
	}
	if plan.VisibleTotal != 21 {
		t.Fatalf("visibleTotal = %d, want 21", plan.VisibleTotal)
	}
	if plan.RowLimit != 18 {
		t.Fatalf("rowLimit = %d, want 18", plan.RowLimit)
	}
}

func TestPlanWindow_ExpandAllWithAnException(t *testing.T) {
	query := q(1, 100, expandAll(), collapsed(`["PENDING"]`))
	plan := PlanWindow(oneLevel(), 1, NewExpansionSet(query), false, query)
	if plan.AllOpen {
		t.Fatal("one exception must switch the row query back to a restriction")
	}
	if plan.VisibleTotal != 17 {
		t.Fatalf("visibleTotal = %d, want 17 (3 headers + 12 + 2)", plan.VisibleTotal)
	}
	if len(plan.OpenLeaves) != 2 {
		t.Fatalf("openLeaves = %v, want APPROVED and REJECTED only", plan.OpenLeaves)
	}
}

func TestPlanWindow_NestedChildIsInvisibleWhileItsParentIsClosed(t *testing.T) {
	tree := BuildGroupTree([]GroupLeaf{
		{Path: []any{"APPROVED", "tata"}, Count: 3},
		{Path: []any{"APPROVED", "honda"}, Count: 2},
		{Path: []any{"PENDING", "tata"}, Count: 4},
	}, 2)

	// Parent closed: one header, nothing beneath it.
	closedParent := q(1, 50)
	plan := PlanWindow(tree, 2, NewExpansionSet(closedParent), false, closedParent)
	if plan.VisibleTotal != 2 {
		t.Fatalf("visibleTotal = %d, want 2 top-level headers", plan.VisibleTotal)
	}

	// Parent open, children closed: the two child headers appear, no rows.
	parentOnly := q(1, 50, expanded(`["APPROVED"]`))
	plan = PlanWindow(tree, 2, NewExpansionSet(parentOnly), false, parentOnly)
	if plan.VisibleTotal != 4 {
		t.Fatalf("visibleTotal = %d, want 4 (2 top + 2 children)", plan.VisibleTotal)
	}
	if plan.RowLimit != 0 {
		t.Fatalf("rowLimit = %d, want 0 — a leaf group is still closed", plan.RowLimit)
	}

	// A child expanded while its parent is NOT must stay hidden: an expansion
	// entry is not a licence to surface rows from inside a closed group.
	orphan := q(1, 50, expanded(`["APPROVED","tata"]`))
	plan = PlanWindow(tree, 2, NewExpansionSet(orphan), false, orphan)
	if plan.VisibleTotal != 2 || plan.RowLimit != 0 {
		t.Fatalf("an expanded child under a closed parent leaked: visible=%d rows=%d",
			plan.VisibleTotal, plan.RowLimit)
	}

	// Both open: header, child header, its rows, second child header, its rows.
	both := q(1, 50, expanded(`["APPROVED"]`, `["APPROVED","honda"]`, `["APPROVED","tata"]`))
	plan = PlanWindow(tree, 2, NewExpansionSet(both), false, both)
	items := AssembleWindow(plan, make([]string, plan.RowLimit))
	if got := kinds(items); got != "HHrrHrrrH" {
		// honda sorts before tata, so its 2 rows come first.
		t.Fatalf("nested sequence = %q, want %q", got, "HHrrHrrrH")
	}
}

func TestPlanWindow_SummaryRidesTheHeaderOnlyWhenDeclared(t *testing.T) {
	query := q(1, 10)
	with := PlanWindow(oneLevel(), 1, NewExpansionSet(query), true, query)
	if with.Slots[0].Header.Summary == nil || *with.Slots[0].Header.Summary != 1200 {
		t.Fatalf("summary = %v, want 1200", with.Slots[0].Header.Summary)
	}
	without := PlanWindow(oneLevel(), 1, NewExpansionSet(query), false, query)
	if without.Slots[0].Header.Summary != nil {
		t.Fatal("an endpoint with no summary must omit the field, not send 0")
	}
}

func TestBuildGroupTree_RollsPrefixesUp(t *testing.T) {
	tree := BuildGroupTree([]GroupLeaf{
		{Path: []any{"APPROVED", "tata"}, Count: 3, Summary: 300},
		{Path: []any{"APPROVED", "honda"}, Count: 2, Summary: 250},
	}, 2)
	if len(tree) != 1 {
		t.Fatalf("roots = %d, want 1", len(tree))
	}
	if tree[0].Count != 5 || tree[0].Summary != 550 {
		t.Fatalf("parent rolled up to %d/%d, want 5/550", tree[0].Count, tree[0].Summary)
	}
	if len(tree[0].Children) != 2 {
		t.Fatalf("children = %d, want 2", len(tree[0].Children))
	}
}

func TestBuildGroupTree_OrdersLikeMongosAscendingSort(t *testing.T) {
	// Getting this wrong does not fail loudly — the headers simply stop lining
	// up with the rows underneath them — which is why it is asserted directly.
	tree := BuildGroupTree([]GroupLeaf{
		{Path: []any{"beta"}, Count: 1},
		{Path: []any{nil}, Count: 1},
		{Path: []any{"alpha"}, Count: 1},
	}, 1)
	got := []any{tree[0].Key, tree[1].Key, tree[2].Key}
	if got[0] != nil || got[1] != "alpha" || got[2] != "beta" {
		t.Fatalf("order = %v, want [nil alpha beta] — null and missing sort first", got)
	}

	bools := BuildGroupTree([]GroupLeaf{
		{Path: []any{true}, Count: 1},
		{Path: []any{false}, Count: 1},
	}, 1)
	if bools[0].Key != false || bools[1].Key != true {
		t.Fatalf("bool order = %v, want [false true]", []any{bools[0].Key, bools[1].Key})
	}
}

func TestExpansionSet_IgnoresWhatItCannotUse(t *testing.T) {
	query := q(1, 10, expanded(
		`["APPROVED"]`,
		`not json`,
		`[]`,
		`["a","b","c","d"]`, // deeper than MaxGroupDepth
		`{"not":"an array"}`,
	))
	set := NewExpansionSet(query)

	if !set.Open([]any{"APPROVED"}) {
		t.Fatal("the one good path must still open its group")
	}
	// Refusing the request instead would turn a cosmetic client bug into a
	// blank work board (group-paging); an ignored path simply leaves its group closed.
	if set.Open([]any{"a", "b", "c"}) {
		t.Fatal("an over-deep path must be ignored, not honoured")
	}
	if len(set.exceptions) != 1 {
		t.Fatalf("exceptions = %d, want 1", len(set.exceptions))
	}
}

func TestExpansionSet_KeysCanonically(t *testing.T) {
	// `[ "A" ]` and `["A"]` are one path; a set keyed on the raw query string
	// would treat them as two and open neither reliably.
	set := NewExpansionSet(q(1, 10, expanded(`[ "APPROVED" ]`)))
	if !set.Open([]any{"APPROVED"}) {
		t.Fatal("whitespace in the encoded path must not change which group it names")
	}
}

func TestExpansionSet_CapsThePathList(t *testing.T) {
	paths := make([]string, MaxExpansionPaths+50)
	for i := range paths {
		b, _ := json.Marshal([]any{uuid.NewString()})
		paths[i] = string(b)
	}
	set := NewExpansionSet(q(1, 10, expanded(paths...)))
	if len(set.exceptions) != MaxExpansionPaths {
		t.Fatalf("honoured %d paths, want the cap of %d", len(set.exceptions), MaxExpansionPaths)
	}
}

func TestExpansionSet_ExpandAllInvertsTheExceptions(t *testing.T) {
	set := NewExpansionSet(q(1, 10, expandAll(), collapsed(`["PENDING"]`), expanded(`["APPROVED"]`)))
	if !set.Open([]any{"APPROVED"}) {
		t.Fatal("under expandAll an unlisted group is open")
	}
	if set.Open([]any{"PENDING"}) {
		t.Fatal("under expandAll a listed group is closed")
	}
	// `expanded` is ignored in this mode, so a group named in BOTH is closed —
	// the exception list is whichever one the flag selects, never a merge.
	if set.Open([]any{"PENDING"}) {
		t.Fatal("expanded[] must not override collapsed[] under expandAll")
	}
}

func TestAssembleWindow_PairsRowsWithTheirOwnHeaders(t *testing.T) {
	query := q(1, 10, expanded(`["APPROVED"]`))
	plan := PlanWindow(oneLevel(), 1, NewExpansionSet(query), false, query)
	items := AssembleWindow(plan, []string{"a", "b", "c", "d", "e", "f", "g"})

	row, ok := items[1].(GroupPagedRow)
	if !ok {
		t.Fatalf("item 1 = %T, want a row", items[1])
	}
	if row.Row != "a" {
		t.Fatalf("first row = %v, want the first fetched row", row.Row)
	}
	if PathID(row.GroupPath) != `["APPROVED"]` {
		t.Fatalf("row tagged %v, want APPROVED", row.GroupPath)
	}
}

func TestAssembleWindow_ShortRowSliceDropsSlotsRatherThanMispairing(t *testing.T) {
	// A record deleted between the count aggregation and the row read. The page
	// deliberately carries a header AFTER its rows: emitting the leftover row
	// slots would pair REJECTED's rows with PENDING's header, which is a page
	// that looks entirely plausible and is wrong.
	query := q(1, 10, expanded(`["PENDING"]`, `["REJECTED"]`))
	plan := PlanWindow(oneLevel(), 1, NewExpansionSet(query), false, query)
	if got := kinds(AssembleWindow(plan, []string{"a", "b", "c", "d", "e", "f"})); got != "HHrrrrHrr" {
		t.Fatalf("full page = %q, want %q", got, "HHrrrrHrr")
	}
	if got := kinds(AssembleWindow(plan, []string{"a", "b", "c"})); got != "HHrrrH" {
		t.Fatalf("short page = %q, want %q — headers stay, unfilled row slots go", got, "HHrrrH")
	}
}

func TestOpenLeafClause_RestrictsToOpenGroups(t *testing.T) {
	levels := []GroupField{{Path: "approvalStatus", Kind: GroupString}}

	all := PlanWindow(oneLevel(), 1, NewExpansionSet(q(1, 50, expandAll())), false, q(1, 50, expandAll()))
	if OpenLeafClause(levels, all) != nil {
		t.Fatal("everything open must carry no restriction at all")
	}

	none := PlanWindow(oneLevel(), 1, NewExpansionSet(q(1, 50)), false, q(1, 50))
	clause := OpenLeafClause(levels, none)
	if clause == nil {
		t.Fatal("nothing open must match NOTHING — omitting the clause returns every row")
	}
	if _, hasOr := clause["$or"]; hasOr {
		t.Fatalf("empty $or is a Mongo error, got %v", clause)
	}

	query := q(1, 50, expanded(`["APPROVED"]`))
	one := PlanWindow(oneLevel(), 1, NewExpansionSet(query), false, query)
	or, ok := OpenLeafClause(levels, one)["$or"].(bson.A)
	if !ok || len(or) != 1 {
		t.Fatalf("clause = %v, want one $or branch", OpenLeafClause(levels, one))
	}
}

func TestOpenLeafClause_ReEncodesAUUIDKey(t *testing.T) {
	// Group keys travel as canonical text and are STORED as BSON Binary
	// subtype 4; matching on the string would silently return no rows, which
	// reads as an empty group rather than a type error.
	id := uuid.New()
	other := uuid.New()
	levels := []GroupField{{Path: "fitment.carCompanyId", Kind: GroupUUID}}
	// A second, CLOSED group: with everything open the plan is AllOpen and
	// carries no clause at all, so a one-group fixture would assert nothing.
	tree := BuildGroupTree([]GroupLeaf{
		{Path: []any{id.String()}, Count: 2},
		{Path: []any{other.String()}, Count: 1},
	}, 1)
	query := q(1, 50, expanded(`["`+id.String()+`"]`))
	plan := PlanWindow(tree, 1, NewExpansionSet(query), false, query)

	or := OpenLeafClause(levels, plan)["$or"].(bson.A)
	got := or[0].(bson.M)["fitment.carCompanyId"]
	if _, isString := got.(string); isString {
		t.Fatal("a UUID group key must be re-encoded to uuid.UUID, not matched as text")
	}
	if got != id {
		t.Fatalf("re-encoded to %v, want %v", got, id)
	}
}

func TestWithClause_AppendsRatherThanReplacing(t *testing.T) {
	// Master data expresses its whole filter as one $and; assigning here drops
	// tenant scope, and every group then reports a cross-tenant count that looks
	// like a plausible number.
	filter := bson.M{"tenantId": "t1", "$and": bson.A{bson.M{"scope": "TENANT"}}}
	out := WithClause(filter, bson.M{"x": 1})

	and := out["$and"].(bson.A)
	if len(and) != 2 {
		t.Fatalf("$and = %v, want the original clause kept", and)
	}
	if out["tenantId"] != "t1" {
		t.Fatal("the rest of the filter must survive")
	}
	if len(filter["$and"].(bson.A)) != 1 {
		t.Fatal("the caller's filter must not be mutated")
	}
}

func TestBuildMetaVisible_TotalStaysRecords(t *testing.T) {
	// 's defect, pinned: `total` is the record count in BOTH modes, and the
	// page arithmetic comes from the visible sequence instead.
	query := q(1, 10)
	meta := query.BuildMetaVisible(1904, 47)

	if meta.Total != 1904 {
		t.Fatalf("total = %d, want the record count 1904", meta.Total)
	}
	if meta.VisibleTotal == nil || *meta.VisibleTotal != 47 {
		t.Fatalf("visibleTotal = %v, want 47", meta.VisibleTotal)
	}
	if meta.TotalPage != 5 {
		t.Fatalf("totalPage = %d, want 5 (ceil(47/10)) — not 191", meta.TotalPage)
	}
	if !meta.HasNext || meta.HasPrev {
		t.Fatalf("hasNext/hasPrev = %v/%v, want true/false", meta.HasNext, meta.HasPrev)
	}
}

func TestBuildMetaVisible_LeavesTheOrdinaryMetaAlone(t *testing.T) {
	query := q(1, 10)
	if query.BuildMeta(47).VisibleTotal != nil {
		t.Fatal("a non-group-paged response must omit visibleTotal entirely")
	}
}

func TestQuery_GroupPagingNeedsGrouping(t *testing.T) {
	// `groupPaging` with nothing to group by has no sequence to page over; it is
	// ignored rather than refused, like every other view preference here.
	ungrouped := Query{GroupPaging: true}
	if ungrouped.IsGroupPaged() {
		t.Fatal("groupPaging without groupBy must not engage the mode")
	}
	grouped := Query{GroupPaging: true, GroupBy: []string{"status"}}
	if !grouped.IsGroupPaged() {
		t.Fatal("groupPaging with groupBy must engage the mode")
	}
}
