package export_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"

	"github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/i18n"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/export"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/pagination"
)

func init() { gin.SetMode(gin.TestMode) }

type row struct {
	Name      string `json:"name"`
	AmountMin int64  `json:"amountMin"` // money, minor units
	Qty       int    `json:"qty"`
	Active    bool   `json:"active"`
	CreatedAt string `json:"createdAt"`
}

var cols = []export.Column{
	{Field: "name", Kind: export.Text},
	{Field: "amountMin", Label: "amount", Kind: export.Money},
	{Field: "qty", Kind: export.Number},
	{Field: "active", Kind: export.Bool},
	{Field: "createdAt", Label: "created", Kind: export.Date},
}

func newCtx() (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/x?format=xlsx", nil)
	tr, err := i18n.New("../../locales")
	if err == nil {
		c.Set(i18n.CtxTranslator, tr)
		c.Set(i18n.CtxLang, "en")
	}
	return c, w
}

func TestRequested(t *testing.T) {
	c, _ := newCtx()
	if !export.Requested(c) {
		t.Error("format=xlsx should be requested")
	}
}

// A money column arrives as integer minor units and must be written as a
// major-unit NUMBER (summable), converted exactly once.
func TestRespond_WritesTypedCells(t *testing.T) {
	c, w := newCtx()
	rows := []row{{Name: "Alice", AmountMin: 45000, Qty: 3, Active: true, CreatedAt: "2026-01-02T15:04:05Z"}}
	export.Respond(c, "things", cols, rows, int64(len(rows)))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", w.Code, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct == "" {
		t.Error("missing content-type")
	}
	f, err := excelize.OpenReader(bytes.NewReader(w.Body.Bytes()))
	if err != nil {
		t.Fatalf("open xlsx: %v", err)
	}
	sheet := f.GetSheetName(0)
	// Money cell: 45000 minor units → 450 (converted once), formatted "450.00".
	if got, _ := f.GetCellValue(sheet, "B2"); got != "450.00" && got != "450" {
		t.Errorf("money cell = %q, want 450(.00)", got)
	}
	// Bool → translated Yes.
	if got, _ := f.GetCellValue(sheet, "D2"); got != "Yes" {
		t.Errorf("bool cell = %q, want Yes", got)
	}
}

// Past MaxRows the whole request is refused (translated), never silently
// truncated — a short spreadsheet is indistinguishable from a complete one.
func TestRespond_RefusesTooLarge(t *testing.T) {
	c, w := newCtx()
	export.Respond(c, "things", cols, []row{}, export.MaxRows+1)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 for an over-cap export", w.Code)
	}
}

// Query widens the page to the whole result set and strips grouping.
func TestQuery_WidensAndStripsGrouping(t *testing.T) {
	q := export.Query(pagination.Query{Page: 3, PerPage: 20, GroupBy: []string{"x"}, GroupPaging: true})
	if q.Page != 1 || q.PerPage <= export.MaxRows {
		t.Errorf("Query did not widen: page=%d perPage=%d", q.Page, q.PerPage)
	}
	if len(q.GroupBy) != 0 || q.GroupPaging {
		t.Error("Query must strip grouping")
	}
}
