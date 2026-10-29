package export

import (
	"bytes"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/i18n"
)

// contentType is the OOXML spreadsheet media type. Spelled out rather than
// guessed from the extension, because a browser that receives a generic
// octet-stream saves the file but Excel then refuses to open it without a
// warning the user has no way to interpret.
const contentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

// DefaultWidth is the column width used when a Column declares none. Wide
// enough for a date or a money amount without wrapping, which is what stops the
// first thing a user does with the file being a manual resize of every column.
const DefaultWidth = 18

// maxSheetName is Excel's own limit. A longer name is not a warning, it is a
// file Excel declines to open.
const maxSheetName = 31

// writeSheet builds the whole workbook in memory and returns its bytes.
//
// The STREAM writer rather than SetCellValue in a loop: the ordinary API holds
// every cell as an addressable object for the life of the file, which at
// MaxRows × twenty columns is two hundred thousand of them. The stream writer
// serialises each row as it is handed over. It has one rule — rows must be
// written in ascending order and never revisited — which is exactly what a
// export does anyway.
func writeSheet(
	title string,
	hdrs []string,
	cols []Column,
	records []map[string]any,
	tr *i18n.Translator,
	lang string,
) ([]byte, error) {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()

	sheet := sanitizeSheetName(title)
	if err := f.SetSheetName("Sheet1", sheet); err != nil {
		return nil, err
	}

	headerStyle, err := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "1F2933"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"EEF2F7"}},
		Alignment: &excelize.Alignment{Vertical: "center"},
	})
	if err != nil {
		return nil, err
	}
	// Two decimals and a thousands separator, because these cells are major units
	// converted out of integer minor units and an amount rendered as 400
	// where the ledger says 400.00 is the kind of difference that gets
	// reconciled by hand at the end of a month.
	moneyStyle, err := f.NewStyle(&excelize.Style{CustomNumFmt: strPtr("#,##0.00")})
	if err != nil {
		return nil, err
	}
	dateStyle, err := f.NewStyle(&excelize.Style{CustomNumFmt: strPtr("dd-mm-yyyy hh:mm")})
	if err != nil {
		return nil, err
	}

	sw, err := f.NewStreamWriter(sheet)
	if err != nil {
		return nil, err
	}

	for i, col := range cols {
		w := col.Width
		if w <= 0 {
			w = DefaultWidth
		}
		if err := sw.SetColWidth(i+1, i+1, w); err != nil {
			return nil, err
		}
	}

	header := make([]any, len(hdrs))
	for i, h := range hdrs {
		header[i] = excelize.Cell{StyleID: headerStyle, Value: h}
	}
	if err := sw.SetRow("A1", header); err != nil {
		return nil, err
	}

	yes, no := translate(tr, lang, "export.yes"), translate(tr, lang, "export.no")

	for n, rec := range records {
		row := make([]any, len(cols))
		for i, col := range cols {
			row[i] = cell(col, lookup(rec, col.Field), moneyStyle, dateStyle, tr, lang, yes, no)
		}
		axis, err := excelize.CoordinatesToCellName(1, n+2) // +2: 1-based, past the header
		if err != nil {
			return nil, err
		}
		if err := sw.SetRow(axis, row); err != nil {
			return nil, err
		}
	}

	if err := sw.Flush(); err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// cell converts one decoded JSON value into a typed spreadsheet cell.
//
// A value that does not fit its declared kind falls back to its text form
// rather than to an empty cell or an error. An export is a read of data that
// already exists: refusing the whole file because one row carries an
// unparseable timestamp would lose the other nine thousand rows to it, and a
// blank cell would silently claim the record has no value there.
func cell(
	col Column,
	v any,
	moneyStyle, dateStyle int,
	tr *i18n.Translator,
	lang, yes, no string,
) excelize.Cell {
	if v == nil {
		return excelize.Cell{}
	}

	switch col.Kind {
	case Money:
		if f, ok := numeric(v); ok {
			// Integer minor units → major units. The division is the ONLY place this
			// happens in an export, and it happens once: a second conversion
			// anywhere downstream is how a figure lands a paisa away from the
			// figure it was computed from.
			return excelize.Cell{StyleID: moneyStyle, Value: f / 100}
		}
	case Number:
		if f, ok := numeric(v); ok {
			return excelize.Cell{Value: f}
		}
	case Date:
		if s, ok := v.(string); ok && s != "" {
			if t, err := time.Parse(time.RFC3339, s); err == nil {
				return excelize.Cell{StyleID: dateStyle, Value: t}
			}
		}
	case Bool:
		if b, ok := v.(bool); ok {
			if b {
				return excelize.Cell{Value: yes}
			}
			return excelize.Cell{Value: no}
		}
	case Enum:
		s := text(v)
		if s == "" {
			return excelize.Cell{}
		}
		if got := lookupEnum(tr, lang, s); got != "" {
			return excelize.Cell{Value: got}
		}
		return excelize.Cell{Value: s}
	}

	return excelize.Cell{Value: text(v)}
}

// numeric normalises the number shapes encoding/json can produce.
func numeric(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int64:
		return float64(n), true
	case int:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	}
	return 0, false
}

// text renders any decoded JSON value as a string, keeping whole numbers whole
// — encoding/json decodes every number as a float64, and "12" reads very
// differently from "12.000000" in a quantity column.
func text(v any) string {
	switch s := v.(type) {
	case string:
		return s
	case bool:
		return strconv.FormatBool(s)
	case float64:
		if s == float64(int64(s)) {
			return strconv.FormatInt(int64(s), 10)
		}
		return strconv.FormatFloat(s, 'f', -1, 64)
	case []any:
		parts := make([]string, 0, len(s))
		for _, e := range s {
			parts = append(parts, text(e))
		}
		return strings.Join(parts, ", ")
	case nil:
		return ""
	default:
		return ""
	}
}

// sanitizeSheetName makes a translated title legal as an Excel sheet name.
// Non-Latin titles are fine — the characters Excel rejects are punctuation, not
// script — but a title long enough to trip the 31-character limit is not, and
// truncation must count RUNES rather than bytes or a multi-byte title is cut mid-character into an invalid name.
func sanitizeSheetName(s string) string {
	s = strings.Map(func(r rune) rune {
		if strings.ContainsRune(`[]:*?/\`, r) {
			return '-'
		}
		return r
	}, strings.TrimSpace(s))
	if s == "" {
		return "Sheet1"
	}
	if r := []rune(s); len(r) > maxSheetName {
		s = string(r[:maxSheetName])
	}
	return s
}

// lookupEnum resolves a wire value through the `export.values` block, returning
// "" when it is not there. Deliberately NOT `translate`, whose humanising
// fallback would turn SOME_ENUM into "SOME_ENUM" with a capital I and hide
// the fact that a value is unmapped; the caller falls back to the raw value so
// the sheet still says something true.
func lookupEnum(tr *i18n.Translator, lang, value string) string {
	if tr == nil {
		return ""
	}
	key := "export.values." + value
	if got := tr.T(lang, key); got != "" && got != key {
		return got
	}
	return ""
}

func strPtr(s string) *string { return &s }
