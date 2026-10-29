// Package export turns any list endpoint's own result into a downloadable
// spreadsheet, SERVER-SIDE.
//
// # Why a query parameter and not an /export route
//
// An export is the list the user is already looking at, in a different
// container. Expressing it as `?format=xlsx` on the endpoint that already
// answers that list means the export inherits — structurally, with no second
// declaration that could drift — the route's permission gate, its tenant
// scoping, its soft-delete filter, its search, its sort and every module-owned
// filter it understands. A sibling `/export` route would have to restate all
// six, and the first one restated wrongly is a route that hands a Coworker a
// spreadsheet of something they cannot open on screen.
//
// It also avoids a routing collision that is easy to walk into: `/stock/export`
// beside `/stock/:partId` is a static-versus-wildcard conflict at the same
// position, which is a router-dependent panic rather than a compile error.
//
// # What is exported
//
// The FILTERED, SORTED record set — every row the current filters match, not
// just the visible page, capped at MaxRows. GROUPING IS IGNORED on purpose:
// grouping is a presentational mode  that changes how rows
// are arranged on screen, never which rows exist, and a spreadsheet has its own
// grouping tools that are better than any header row we could bake in.
//
// # No identifiers in headers
//
// Every header is TRANSLATED, through the `fields` block of `locales/*.json` —
// the same label dictionary validation prose already uses to name a field. A column therefore costs no new key when the field already has a
// label, and `labels_test.go` fails the build for one that does not. The
// filename is deliberately NOT translated: it is a file identifier, the same
// category as an enum value on the wire.
package export

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/i18n"
	apperrors "github.com/breadgarlicbigint/bread-golang-boilerplate/shared/errors"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/pagination"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/response"
)

// MaxRows is the largest export the API will build.
//
// A cap rather than a stream without one, because everything downstream of the
// query has a limit somebody else chose: a single Mongo read, the memory this
// process holds while the sheet is assembled, and Excel's own patience opening
// the result. Ten thousand rows is comfortably inside all three and well past any list read in a day.
//
// Past it the request is REFUSED with a translated message telling the user to
// narrow the filter — not silently truncated. A truncated spreadsheet is
// indistinguishable from a complete one once it is off the server, and somebody will reconcile against it.
const MaxRows = 10000

// ErrTooLarge is the refusal past MaxRows.
var ErrTooLarge = apperrors.NewI18n(
	http.StatusBadRequest,
	"EXPORT_TOO_LARGE",
	"export.tooLarge",
	"This export is too large — narrow the filter and try again",
)

// Kind says how a value is written into the sheet. It exists because the whole
// point of .xlsx over .csv is that a cell has a TYPE: an amount that arrives as
// text cannot be summed, and a date that arrives as text sorts alphabetically.
type Kind string

const (
	// Text is the default: written verbatim as a string.
	Text Kind = "text"
	// Number is a plain numeric cell (a quantity, a count, a percentage).
	Number Kind = "number"
	// Money is an INTEGER in MINOR UNITS on the wire and is written as a major-
	// unit NUMBER with two decimals — 40000 becomes 400.00, formatted, summable.
	// Never as a string with a symbol in it: that is a label, not an amount.
	Money Kind = "money"
	// Date is an RFC3339 timestamp on the wire, written as a real date cell.
	Date Kind = "date"
	// Bool is written as the translated Yes/No, never as TRUE/FALSE — the sheet
	// is read by a person, and the no-identifiers rule does not stop at the API's own prose.
	Bool Kind = "bool"
	// Enum is a value the wire carries as an identifier and a reader must not
	// see as one (the no-identifiers rule). It resolves through `export.values.<VALUE>`, a
	// single flat block rather than a map per column: these are wire constants,
	// globally unique in practice, and one block is one place to look when a
	// spreadsheet shows a raw enum value. An unmapped value falls back to
	// itself rather than to a blank cell — a status nobody translated is still
	// more use than an empty column.
	Enum Kind = "enum"
)

// Column is one spreadsheet column, declared by the module that owns the list.
type Column struct {
	// Field is a dotted path into the row's JSON shape ("a.b").
	// The row is marshalled to JSON first, so this is the SAME name the client
	// already receives — a column cannot drift from the payload without the
	// cell going empty, which is visible immediately.
	Field string
	// Label is the key under the `fields` block of locales/*.json. Empty means
	// "the last segment of Field", which is what makes the common case a
	// one-line declaration. Resolved per request language.
	Label string
	Kind  Kind
	// Width is the column width in characters. Zero takes DefaultWidth.
	Width float64
}

// labelKey returns the `fields.` key this column's header resolves through.
func (c Column) labelKey() string {
	name := c.Label
	if name == "" {
		name = c.Field
		if i := strings.LastIndex(name, "."); i >= 0 {
			name = name[i+1:]
		}
	}
	return "fields." + name
}

// Requested reports whether this request asked for a spreadsheet instead of
// JSON. Only the literal "xlsx" engages it: an uninterpretable value leaves the
// endpoint answering exactly what it always answered, which is the same posture pagination takes on its own view flags.
func Requested(c *gin.Context) bool { return c.Query("format") == "xlsx" }

// Query widens a bound pagination query to cover the whole result set.
//
// It sets PerPage past pagination.MaxPerPage DELIBERATELY. That cap protects
// the JSON list endpoint, where a page is a screenful and 10,000 records would
// be a response nobody asked for; an export is the opposite request by
// definition. Grouping is stripped for the reason in the package comment, and
// Page is reset to 1 so exporting from page 3 of a table does not silently drop
// the first two pages.
func Query(q pagination.Query) pagination.Query {
	q.Page = 1
	q.PerPage = MaxRows + 1 // +1 so a set AT the cap is distinguishable from one past it
	q.GroupBy = nil
	q.GroupPaging = false
	q.Expanded, q.Collapsed = nil, nil
	return q
}

// Respond writes rows as an .xlsx download, or the translated refusal when the
// result set is past MaxRows.
//
// `name` is the file's stem and the sheet's i18n key suffix: "customers" gives
// customers-2026-09-04.xlsx and a sheet titled by `export.sheets.customers`.
//
// Errors are answered through the normal envelope BEFORE any byte of the file
// is written — which is why the sheet is built in memory first. Once a
// Content-Disposition has gone out there is no way back to a JSON error, and a
// half-written spreadsheet downloads as a corrupt file with no explanation.
func Respond[T any](c *gin.Context, name string, cols []Column, rows []T, total int64) {
	if total > MaxRows || len(rows) > MaxRows {
		response.HandleAppError(c, ErrTooLarge)
		return
	}

	records, err := toRecords(rows)
	if err != nil {
		response.HandleAppError(c, apperrors.Wrap(
			http.StatusInternalServerError, "EXPORT_FAILED", "Could not build the export", err))
		return
	}

	tr, lang := i18n.FromContext(c)
	buf, err := writeSheet(sheetTitle(tr, lang, name), headers(tr, lang, cols), cols, records, tr, lang)
	if err != nil {
		response.HandleAppError(c, apperrors.Wrap(
			http.StatusInternalServerError, "EXPORT_FAILED", "Could not build the export", err))
		return
	}

	filename := fmt.Sprintf("%s-%s.xlsx", name, time.Now().Format("2006-01-02"))
	c.Header("Content-Disposition", `attachment; filename="`+filename+`"`)
	c.Header("Content-Type", contentType)
	// The web client reads the filename off this header, and a cross-origin
	// reader cannot see a header that is not exposed. Without it the browser
	// falls back to naming the file after the endpoint's last path segment.
	c.Header("Access-Control-Expose-Headers", "Content-Disposition")
	c.Data(http.StatusOK, contentType, buf)
}

// toRecords turns typed rows into generic maps by way of their own JSON shape.
//
// Through JSON rather than reflection over the struct, deliberately: it means a
// column addresses a field by the SAME name the API already publishes, so the
// spreadsheet and the payload can never disagree about what a field is called,
// and `omitempty` behaves identically in both.
func toRecords[T any](rows []T) ([]map[string]any, error) {
	b, err := json.Marshal(rows)
	if err != nil {
		return nil, err
	}
	var out []map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// lookup walks a dotted path into a decoded JSON object.
func lookup(rec map[string]any, path string) any {
	cur := any(rec)
	for _, seg := range strings.Split(path, ".") {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur, ok = m[seg]
		if !ok {
			return nil
		}
	}
	return cur
}

// headers resolves every column's translated header once per request.
func headers(tr *i18n.Translator, lang string, cols []Column) []string {
	out := make([]string, len(cols))
	for i, col := range cols {
		out[i] = translate(tr, lang, col.labelKey())
	}
	return out
}

func sheetTitle(tr *i18n.Translator, lang, name string) string {
	return translate(tr, lang, "export.sheets."+name)
}

// translate resolves a key, falling back to a humanised form of the key's last
// segment rather than to the key itself. `humanize` is shared/validate's safety
// net applied here for the same reason: a missing label must never put a raw
// identifier in front of a user (the no-identifiers rule), even though labels_test.go is what
// actually stops one shipping.
func translate(tr *i18n.Translator, lang, key string) string {
	if tr == nil {
		return humanize(key)
	}
	if got := tr.T(lang, key); got != "" && got != key {
		return got
	}
	return humanize(key)
}

func humanize(key string) string {
	s := key
	if i := strings.LastIndex(s, "."); i >= 0 {
		s = s[i+1:]
	}
	var b strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			b.WriteByte(' ')
			b.WriteRune(r + 32)
			continue
		}
		if i == 0 && r >= 'a' && r <= 'z' {
			b.WriteRune(r - 32)
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
