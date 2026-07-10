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
}

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
