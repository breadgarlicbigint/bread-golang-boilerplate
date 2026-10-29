package pagination

import (
	"math"
	"strconv"

	"github.com/gin-gonic/gin"
)

const (
	DefaultPage    = 1
	DefaultPerPage = 20
	MaxPerPage     = 100
)

// Query binds pagination query params from the request.
type Query struct {
	Page    int    `form:"page"`
	PerPage int    `form:"perPage"`
	Search  string `form:"search"`
	SortBy  string `form:"sortBy"`
	SortDir string `form:"sortDir"` // asc | desc
	Cursor  string `form:"cursor"`

	// ── Grouping ─────────────────────────────────────────────────────────────
	// GroupBy holds the grouping levels in nesting order — client-facing keys
	// resolved through the module's GroupFields whitelist, never bson paths.
	// Empty ⇒ no grouping, and the response is byte-identical to an ungrouped
	// one. Paging stays over raw records either way.
	GroupBy []string `form:"groupBy"`

	// ── Group paging ─────────────────────────────────────────────────────────
	// A SECOND paging mode, engaged only by GroupPaging and only alongside
	// GroupBy. In it the pager counts VISIBLE rows — every group header plus the
	// rows of open groups — so collapsing a group shortens the list and the page
	// count drops. Total stays the RECORD count in both modes.
	//
	// A group is open iff `ExpandAll ? !Collapsed.has(path) : Expanded.has(path)`
	// AND every ancestor is open. Two flags rather than one set so "all open
	// except these" and "all closed except these" are each one flag, without the
	// client ever holding the full group list.
	GroupPaging bool `form:"groupPaging"`
	ExpandAll   bool `form:"expandAll"`
	// Expanded/Collapsed carry JSON-encoded group paths — the same shape
	// `_metadata.groups[].path` and `groupPath` use. Malformed or unknown
	// entries are IGNORED, not refused (see ExpansionSet).
	Expanded  []string `form:"expanded"`
	Collapsed []string `form:"collapsed"`
}

// IsGroupPaged reports whether this request asked for the group-paging mode.
// Grouping is a precondition: `groupPaging` with no `groupBy` has nothing to
// page over and is ignored rather than refused.
func (q *Query) IsGroupPaged() bool { return q.GroupPaging && q.IsGrouped() }

// Normalize enforces defaults and caps.
func (q *Query) Normalize() {
	if q.Page < 1 {
		q.Page = DefaultPage
	}
	if q.PerPage < 1 {
		q.PerPage = DefaultPerPage
	}
	if q.PerPage > MaxPerPage {
		q.PerPage = MaxPerPage
	}
	if q.SortDir != "asc" && q.SortDir != "desc" {
		q.SortDir = "desc"
	}
}

// Skip returns the MongoDB skip value for offset-based pagination.
func (q *Query) Skip() int64 {
	return int64((q.Page - 1) * q.PerPage)
}

// Limit returns the MongoDB limit value.
func (q *Query) Limit() int64 {
	return int64(q.PerPage)
}

// Meta builds the response metadata from total count.
func (q *Query) BuildMeta(total int64) *PaginationMeta {
	totalPage := int(math.Ceil(float64(total) / float64(q.PerPage)))
	return &PaginationMeta{
		Total:     total,
		Page:      q.Page,
		PerPage:   q.PerPage,
		TotalPage: totalPage,
		HasNext:   q.Page < totalPage,
		HasPrev:   q.Page > 1,
	}
}

// PaginationMeta mirrors response.Meta for internal usage.
type PaginationMeta struct {
	Total     int64  `json:"total"`
	Page      int    `json:"page"`
	PerPage   int    `json:"perPage"`
	TotalPage int    `json:"totalPage"`
	HasNext   bool   `json:"hasNext"`
	HasPrev   bool   `json:"hasPrev"`
	Cursor    string `json:"cursor,omitempty"`
	// VisibleTotal is the length of the visible sequence in group-paging mode —
	// headers plus the rows of open groups. A POINTER and omitempty: absent is
	// what every response outside group paging must stay, and 0 is a real length
	// (every group closed on an empty list), so a plain int64 couldn't tell the
	// two apart.
	VisibleTotal *int64 `json:"visibleTotal,omitempty"`
}

// FromContext binds and normalises a Query from a Gin context.
func FromContext(c *gin.Context) Query {
	q := Query{
		Page:    intParam(c, "page", DefaultPage),
		PerPage: intParam(c, "perPage", DefaultPerPage),
		Search:  c.Query("search"),
		SortBy:  c.Query("sortBy"),
		SortDir: c.Query("sortDir"),
		Cursor:  c.Query("cursor"),

		// QueryArray, not Query: groupBy repeats, one entry per nesting level.
		GroupBy: c.QueryArray("groupBy"),

		// Only the literal "true" engages the mode or flips the default: this is
		// a view preference, so an uninterpretable value leaves the read in its
		// default shape rather than blanking the list with a 400.
		GroupPaging: c.Query("groupPaging") == "true",
		ExpandAll:   c.Query("expandAll") == "true",
		Expanded:    c.QueryArray("expanded"),
		Collapsed:   c.QueryArray("collapsed"),
	}
	q.Normalize()
	return q
}

func intParam(c *gin.Context, key string, def int) int {
	s := c.Query(key)
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil || v < 1 {
		return def
	}
	return v
}
