package pagination

import (
	"encoding/json"
	"fmt"
	"sort"
)

// Group paging — the second grouped mode.
//
// the grouping contract pages RAW RECORDS and expand/collapse is pure client-side visibility over
// a window that has already arrived. This mode pages the VISIBLE SEQUENCE
// instead: every group header at every level, plus the rows of open groups. So
// collapsing a group takes its rows out of the list and the page count drops,
// which is what DevExpress's remoteOperations.groupPaging does.
//
// It has to be server-side. Page boundaries depend on the expansion state, so
// collapsing a group on page 1 must pull rows forward from page 2 — arithmetic
// no client can do over a window somebody else chose.
//
// What did NOT change, and must not: `total` is still the record count. an earlier version
// redefined it to mean groups whenever grouping was on, which reported `total:1`
// for 12 records behind one group and collapsed the pager to a single page.
// `VisibleTotal` is a separate number, and the two never merge.
//
// The arithmetic here is PURE and lives in one place on purpose (rule 4a): three
// list endpoints cannot use the Mongo helper below — the services board and the
// stock report only exist after several $lookups, and /v1/platform/companies
// pages in Go — so they supply their own aggregations and call the same planner.
// Two copies of a window calculation would not fail loudly; they would drift,
// and one list's headers would stop matching its rows.

// MaxExpansionPaths caps how many expansion exceptions one request may carry.
// The row query turns them into an `$or`, so this is what keeps that bounded; a
// client with more open groups than this is past any screen that could show them.
const MaxExpansionPaths = 200

// GroupSummary declares an endpoint's optional per-group aggregate. One field,
// summed — a caller-declared aggregate SET would be a second user-supplied
// expression to whitelist, for a figure no screen has asked for.
//
// Path is a bson dotted path, module-owned and never client-supplied. An empty
// Path means the endpoint has no summary and headers carry counts alone.
type GroupSummary struct {
	Path string
}

// Declared reports whether the endpoint has a summary at all.
func (g GroupSummary) Declared() bool { return g.Path != "" }

// GroupNode is one group in the tree built from the count aggregation: its own
// key at its level, its FULL record count, its summed money, and its children.
type GroupNode struct {
	Key      any
	Count    int64
	Summary  int64
	Children []*GroupNode
}

// GroupHeaderItem is a header in the rendered sequence. `Collapsed` is the
// RESOLVED answer rather than an echo of the request, so a page whose expansion
// set was partly ignored still draws chevrons that match the rows beside them.
type GroupHeaderItem struct {
	Kind      string `json:"kind"` // always "header"
	Path      []any  `json:"path"`
	Depth     int    `json:"depth"`
	Count     int64  `json:"count"`
	Summary   *int64 `json:"summary,omitempty"`
	Collapsed bool   `json:"collapsed"`
	// Continued marks a header REPEATED at the top of a page because its group
	// began on an earlier one. It does NOT occupy a slot in the visible
	// sequence — it is context, not content — so a page still holds exactly
	// perPage real slots and `visibleTotal` does not move.
	//
	// Without it a group of 40 rows at perPage 10 leaves pages 2 onward showing
	// rows under no header at all, and — worse — the operator cannot collapse
	// the group they are looking at, because its only control is three pages
	// back.
	Continued bool `json:"continued,omitempty"`
}

// GroupPagedRow is a data row in the rendered sequence.
type GroupPagedRow struct {
	Kind      string `json:"kind"` // always "row"
	GroupPath []any  `json:"groupPath"`
	Row       any    `json:"row"`
}

// WindowPlan is the whole answer to "what is on this page" — computed from the
// group tree and the expansion state alone, before a single row is read.
//
// Slots are the sequence positions the page covers. RowSkip/RowLimit address the
// rows of OPEN groups in their global order, which is exactly the order the row
// query returns once it is restricted to those groups: the walk visits groups in
// the same ascending key order Mongo sorts them by, so a row's index in that
// restricted set is monotonic along the walk and the page's rows are contiguous
// in it. That is what keeps this to ONE row query rather than one per group.
type WindowPlan struct {
	// Slots, in order. A header slot carries its item; a row slot is a
	// placeholder to be filled from the row query, in order.
	Slots []WindowSlot
	// RowSkip/RowLimit address the restricted row set described above.
	RowSkip  int64
	RowLimit int64
	// VisibleTotal is the length of the whole visible sequence under this
	// expansion state — headers at every level plus the rows of open groups.
	VisibleTotal int64
	// OpenLeaves are the full paths of the leaf groups whose rows are visible.
	// The row query restricts to these; empty means no rows at all, which is a
	// legitimate page (everything closed) and not an error.
	OpenLeaves [][]any
	// AllOpen means nothing anywhere is closed, so the row query needs no
	// restriction at all. Not an optimisation to skip: `expandAll` with no
	// exceptions is the ordinary case after pressing Expand all, and enumerating
	// every leaf into an `$or` there would put one clause per distinct value of
	// the grouped column into the query — several hundred on a date grouping.
	AllOpen bool
}

// WindowSlot is one position on the page: either a header, or a row to be filled.
type WindowSlot struct {
	Header *GroupHeaderItem
	// GroupPath is set on a row slot, so a filled row can be tagged without
	// re-deriving which group it landed in.
	GroupPath []any
}

// ExpansionSet resolves the request's expansion flags into a decidable predicate.
//
// Malformed JSON, a path longer than the grouping, or a path naming a group that
// does not exist are all IGNORED — the group keeps its default state, which is
// the same outcome as never having been sent. Deliberately not a 400: this is a
// view preference, not a permission (contrast RequireDeletedView), and
// refusing a whole list over one turns a cosmetic client bug into a blank board.
type ExpansionSet struct {
	expandAll  bool
	exceptions map[string]struct{}
}

// NewExpansionSet reads the flags off a Query.
func NewExpansionSet(q Query) ExpansionSet {
	raw := q.Expanded
	if q.ExpandAll {
		raw = q.Collapsed
	}
	if len(raw) > MaxExpansionPaths {
		raw = raw[:MaxExpansionPaths]
	}
	set := ExpansionSet{expandAll: q.ExpandAll, exceptions: make(map[string]struct{}, len(raw))}
	for _, s := range raw {
		var path []any
		if err := json.Unmarshal([]byte(s), &path); err != nil {
			continue
		}
		if len(path) == 0 || len(path) > MaxGroupDepth {
			continue
		}
		// Re-key through the canonical encoder rather than trusting the client's
		// spacing: `["A"]` and `[ "A" ]` are one path, and a set keyed on the raw
		// string would treat them as two.
		set.exceptions[PathID(path)] = struct{}{}
	}
	return set
}

// Open reports whether one group path is open. Ancestors are NOT consulted here
// — the walk only ever asks about a group it has already descended into, so an
// ancestor check would be asked and answered twice per level.
func (e ExpansionSet) Open(path []any) bool {
	_, exception := e.exceptions[PathID(path)]
	if e.expandAll {
		return !exception
	}
	return exception
}

// OpenLeafSet is the plan's own answer to "which leaf groups are showing",
// keyed for lookup. The in-memory endpoints filter their rows through it rather
// than re-deriving the predicate — an ancestor check written a second time is a
// second place for it to disagree with the walk that produced the headers.
func (p WindowPlan) OpenLeafSet() map[string]struct{} {
	out := make(map[string]struct{}, len(p.OpenLeaves))
	for _, path := range p.OpenLeaves {
		out[PathID(path)] = struct{}{}
	}
	return out
}

// PathID is the canonical string form of a group path — JSON, so it matches the
// client's own JSON.stringify and a path round-trips through the query string
// unchanged. Used as a map key here and as a React key on the other side.
func PathID(path []any) string {
	b, err := json.Marshal(path)
	if err != nil {
		return fmt.Sprint(path)
	}
	return string(b)
}

// BuildGroupTree turns the flat leaf counts from a `$group` aggregation into the
// ordered tree the walk needs, rolling every prefix up as it goes.
//
// `leaves` holds one entry per distinct full path. Ordering must match the row
// query's `$sort`, which is ascending on each group field: nil (null AND a
// missing field, which Mongo sorts together and first) comes first, then values
// in natural order. Strings compare bytewise on both sides; a UUID decoded to
// its canonical lowercase-hex form compares in the same order as the stored
// bytes, since the dashes sit at fixed positions in every such string.
func BuildGroupTree(leaves []GroupLeaf, depth int) []*GroupNode {
	roots := make([]*GroupNode, 0)
	index := map[string]*GroupNode{}

	for _, leaf := range leaves {
		if len(leaf.Path) < depth {
			continue
		}
		siblings := &roots
		for level := 0; level < depth; level++ {
			prefix := leaf.Path[:level+1]
			id := PathID(prefix)
			node, seen := index[id]
			if !seen {
				node = &GroupNode{Key: leaf.Path[level]}
				index[id] = node
				*siblings = append(*siblings, node)
			}
			// Every prefix carries the roll-up of the leaves beneath it, summed
			// here rather than costing one aggregation per level.
			node.Count += leaf.Count
			node.Summary += leaf.Summary
			siblings = &node.Children
		}
	}
	sortNodes(roots)
	return roots
}

// GroupLeaf is one row of the count aggregation: a full group path with its
// record count and summed money.
type GroupLeaf struct {
	Path    []any
	Count   int64
	Summary int64
}

func sortNodes(nodes []*GroupNode) {
	sort.SliceStable(nodes, func(i, j int) bool { return lessKey(nodes[i].Key, nodes[j].Key) })
	for _, n := range nodes {
		sortNodes(n.Children)
	}
}

// lessKey orders group keys the way Mongo's ascending `$sort` does for the three
// kinds a group level can hold. Getting this wrong does not fail loudly — the
// headers and the rows simply stop lining up — which is why it is one function
// with a test rather than an inline comparator.
func lessKey(a, b any) bool {
	if a == nil {
		return b != nil // null and missing sort first
	}
	if b == nil {
		return false
	}
	switch av := a.(type) {
	case bool:
		bv, ok := b.(bool)
		return ok && !av && bv // false < true
	case string:
		bv, ok := b.(string)
		return ok && av < bv
	default:
		return fmt.Sprint(a) < fmt.Sprint(b)
	}
}

// PlanWindow walks the group tree under the given expansion state and returns
// what the requested page covers.
//
// The walk emits, in order: a header for every group at every level, and — for
// an open LEAF group — one slot per row it holds. Row slots are counted into
// `rowIndex` as they are emitted, which is the index they will have in the
// restricted row query, so the page's `RowSkip` is simply the rowIndex of its
// first row slot.
func PlanWindow(tree []*GroupNode, depth int, exp ExpansionSet, summary bool, q Query) WindowPlan {
	from := q.Skip()
	to := from + q.Limit()

	plan := WindowPlan{Slots: make([]WindowSlot, 0, q.PerPage), OpenLeaves: make([][]any, 0)}
	var visible int64
	var rowIndex int64
	firstRow := int64(-1)
	var rowsOnPage int64
	var closed int64

	// take records one visible slot. It is called for every slot in the whole
	// sequence — not only the page's — because `VisibleTotal` is the sequence's
	// full length and the pager needs it.
	take := func(slot WindowSlot, isRow bool) {
		if visible >= from && visible < to {
			plan.Slots = append(plan.Slots, slot)
			if isRow {
				if firstRow < 0 {
					firstRow = rowIndex
				}
				rowsOnPage++
			}
		}
		visible++
	}

	var walk func(nodes []*GroupNode, prefix []any, level int)
	walk = func(nodes []*GroupNode, prefix []any, level int) {
		for _, node := range nodes {
			path := append(append([]any{}, prefix...), node.Key)
			open := exp.Open(path)

			header := &GroupHeaderItem{
				Kind: "header", Path: path, Depth: level, Count: node.Count, Collapsed: !open,
			}
			if summary {
				total := node.Summary
				header.Summary = &total
			}
			take(WindowSlot{Header: header}, false)

			if !open {
				closed++
				continue
			}
			if level == depth-1 {
				// A leaf: its rows are the next `node.Count` entries of the
				// restricted row set, in the same order the row query returns
				// them.
				plan.OpenLeaves = append(plan.OpenLeaves, path)
				for i := int64(0); i < node.Count; i++ {
					take(WindowSlot{GroupPath: path}, true)
					rowIndex++
				}
				continue
			}
			walk(node.Children, path, level+1)
		}
	}
	walk(tree, nil, 0)

	plan.VisibleTotal = visible
	plan.AllOpen = closed == 0
	if firstRow >= 0 {
		plan.RowSkip = firstRow
		plan.RowLimit = rowsOnPage
	}
	prependContinuedHeaders(&plan, tree, depth, exp, summary)
	return plan
}

// prependContinuedHeaders repeats the headers a page opened inside of.
//
// A page whose first slot is a ROW began part-way through an open group, so
// every ancestor of that row belongs at the top of the page — otherwise its rows
// hang under no heading, and the control that would collapse the group is on a
// page the operator has already left.
//
// They are prepended OUTSIDE the window arithmetic: the row range and
// VisibleTotal are already fixed, and counting a repeat would make one group's
// rows cost more slots than another's for no reason the user could see.
func prependContinuedHeaders(plan *WindowPlan, tree []*GroupNode, depth int, exp ExpansionSet, summary bool) {
	if len(plan.Slots) == 0 || plan.Slots[0].Header != nil {
		return
	}
	path := plan.Slots[0].GroupPath
	nodes := tree
	repeats := make([]WindowSlot, 0, depth)
	for level := 0; level < len(path); level++ {
		var node *GroupNode
		for _, n := range nodes {
			if PathID([]any{n.Key}) == PathID([]any{path[level]}) {
				node = n
				break
			}
		}
		if node == nil {
			return
		}
		prefix := append([]any{}, path[:level+1]...)
		header := &GroupHeaderItem{
			Kind: "header", Path: prefix, Depth: level, Count: node.Count,
			Collapsed: !exp.Open(prefix), Continued: true,
		}
		if summary {
			total := node.Summary
			header.Summary = &total
		}
		repeats = append(repeats, WindowSlot{Header: header})
		nodes = node.Children
	}
	plan.Slots = append(repeats, plan.Slots...)
}

// BuildMetaVisible is BuildMeta for the group-paging mode: `total` stays the RECORD count while
// the page arithmetic is computed from the visible sequence, which is what the
// pager the user is operating actually counts.
func (q *Query) BuildMetaVisible(total, visibleTotal int64) *PaginationMeta {
	m := *q
	meta := m.BuildMeta(visibleTotal)
	meta.Total = total
	visible := visibleTotal
	meta.VisibleTotal = &visible
	return meta
}
