package pagination_test

import (
	"testing"

	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/pagination"
)

func TestQuery_Normalize(t *testing.T) {
	cases := []struct {
		name    string
		input   pagination.Query
		wantPage    int
		wantPerPage int
		wantDir string
	}{
		{"defaults", pagination.Query{}, 1, 20, "desc"},
		{"negative page", pagination.Query{Page: -1}, 1, 20, "desc"},
		{"over max perPage", pagination.Query{PerPage: 999}, 1, 100, "desc"},
		{"valid asc", pagination.Query{Page: 2, PerPage: 10, SortDir: "asc"}, 2, 10, "asc"},
		{"invalid sortDir", pagination.Query{SortDir: "random"}, 1, 20, "desc"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			q := c.input
			q.Normalize()
			if q.Page != c.wantPage {
				t.Errorf("Page: want %d got %d", c.wantPage, q.Page)
			}
			if q.PerPage != c.wantPerPage {
				t.Errorf("PerPage: want %d got %d", c.wantPerPage, q.PerPage)
			}
			if q.SortDir != c.wantDir {
				t.Errorf("SortDir: want %s got %s", c.wantDir, q.SortDir)
			}
		})
	}
}

func TestQuery_SkipAndLimit(t *testing.T) {
	q := pagination.Query{Page: 3, PerPage: 10}
	q.Normalize()
	if q.Skip() != 20 {
		t.Errorf("Skip: want 20 got %d", q.Skip())
	}
	if q.Limit() != 10 {
		t.Errorf("Limit: want 10 got %d", q.Limit())
	}
}

func TestQuery_BuildMeta(t *testing.T) {
	q := pagination.Query{Page: 2, PerPage: 10}
	q.Normalize()
	meta := q.BuildMeta(35)

	if meta.Total != 35 {
		t.Errorf("Total: want 35 got %d", meta.Total)
	}
	if meta.TotalPage != 4 {
		t.Errorf("TotalPage: want 4 got %d", meta.TotalPage)
	}
	if !meta.HasNext {
		t.Error("HasNext: want true")
	}
	if !meta.HasPrev {
		t.Error("HasPrev: want true")
	}
}

func TestQuery_FirstPageMeta(t *testing.T) {
	q := pagination.Query{Page: 1, PerPage: 10}
	q.Normalize()
	meta := q.BuildMeta(5)

	if meta.HasPrev {
		t.Error("first page should not have prev")
	}
	if meta.HasNext {
		t.Error("single page should not have next")
	}
}
