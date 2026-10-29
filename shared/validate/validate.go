// Package validate provides a single BindJSON helper that every handler uses.
// It ensures consistent error responses across the entire API:
//   - Malformed JSON           → 400 Bad Request   (message only, no field details)
//   - Struct validation errors → 422 Unprocessable Entity (errors[] with field+message)
//
// No message ever exposes an internal identifier: the prose names the field by
// its translated `fields.<jsonTag>` label, never the JSON tag, the Go struct
// field, the Go type or the validator tag. The machine-readable `field` key in
// each ErrorDetail is the exception — it is a wire identifier the clients key
// their per-field rendering off, and is never displayed.
package validate

import (
	"bytes"
	"log/slog"
	"reflect"
	"strings"
	"text/template"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	pkgi18n "github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/i18n"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/response"
)

var v = validator.New()

func init() {
	// Register tag name function so errors report JSON field names, not Go struct field names.
	// e.g. "email" instead of "Email"
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := fld.Tag.Get("json")
		if name == "" || name == "-" {
			return ""
		}
		// strip ",omitempty" etc.
		if idx := strings.Index(name, ","); idx != -1 {
			name = name[:idx]
		}
		return name
	})

	// `notblank` — a string that carries something once trimmed.
	//
	// `required` and `min=N` both measure the RAW wire value, so a field
	// documented as "a reason somebody typed" is satisfied by three spaces, which
	// the service then trims to the empty string. Use it ALONGSIDE `min`/`max`,
	// not instead of them: this rule says the value is not empty space, and says
	// nothing about its length. Custom rules live in a leaf package and register
	// in init() — go-playground's own `oneof` cannot express every constraint
	// (it PANICS on float fields, for one), so a rule is the escape hatch.
	_ = v.RegisterValidation("notblank", func(fl validator.FieldLevel) bool {
		return strings.TrimSpace(fl.Field().String()) != ""
	})
}

// BindJSON binds the request body and validates the struct in one call.
// Returns true if the request is valid; false if an error response has already been written.
//
// Usage in handlers (replaces the 2-step ShouldBindJSON + validate.Struct pattern):
//
//	var req dto.LoginRequest
//	if !validate.BindJSON(c, &req) {
//	    return
//	}
func BindJSON(c *gin.Context, req any) bool {
	// Step 1 — decode JSON
	if err := c.ShouldBindJSON(req); err != nil {
		logBindError(c, "json", err)
		response.BadRequest(c, pkgi18n.TC(c, "validation.invalidBody"))
		return false
	}

	// Step 2 — struct validation
	if err := v.Struct(req); err != nil {
		response.UnprocessableEntity(c,
			pkgi18n.TC(c, "validation.failed"),
			buildDetails(c, err)...,
		)
		return false
	}

	return true
}

// BindQuery binds query parameters and validates the struct.
func BindQuery(c *gin.Context, req any) bool {
	if err := c.ShouldBindQuery(req); err != nil {
		logBindError(c, "query", err)
		response.BadRequest(c, pkgi18n.TC(c, "validation.invalidQuery"))
		return false
	}
	if err := v.Struct(req); err != nil {
		response.UnprocessableEntity(c,
			pkgi18n.TC(c, "validation.failed"),
			buildDetails(c, err)...,
		)
		return false
	}
	return true
}

// ── helpers ───────────────────────────────────────────────────────────────────

// buildDetails maps validator.ValidationErrors to response.ErrorDetail slice.
// The `field` key uses the JSON tag (registered above) and stays untranslated —
// it is a wire identifier the clients key their per-field error rendering off,
// and is never rendered as text. The *message* names the field by its translated
// `fields.<jsonTag>` label instead, so no DTO/storage identifier reaches the UI.
// Every message goes through i18n: `validation.<messageKey>` in locales/*.json,
// with `validation.default` catching any tag that has no dedicated key. The
// English map below is only the no-translator fallback (unit tests, and any
// handler invoked outside the i18n middleware).
func buildDetails(c *gin.Context, err error) []response.ErrorDetail {
	ve, ok := err.(validator.ValidationErrors)
	if !ok {
		// Not a validation failure — an internal error string would leak Go
		// types, so report the generic translated message and log the detail.
		logBindError(c, "struct", err)
		return []response.ErrorDetail{{Message: pkgi18n.TC(c, "validation.failed")}}
	}

	tr, lang := pkgi18n.FromContext(c)

	details := make([]response.ErrorDetail, 0, len(ve))
	for _, fe := range ve {
		field := baseField(fe.Field()) // already the JSON name thanks to RegisterTagNameFunc
		tag := fe.Tag()
		key := messageKey(tag, fe.Kind())
		data := map[string]string{
			"Field": fieldLabel(tr, lang, field),
			"Param": friendlyParam(tag, fe.Param()),
		}

		msg := fallbackMessage(key, data)
		if tr != nil {
			if translated := tr.T(lang, "validation."+key, data); translated != "validation."+key {
				msg = translated
			} else if translated := tr.T(lang, "validation.default", data); translated != "validation.default" {
				msg = translated
			}
		}

		details = append(details, response.ErrorDetail{
			Field:   field,
			Message: msg,
		})
	}
	return details
}

// baseField strips the slice index go-playground appends to a `dive`d element,
// so `extraEmails[0]` is reported and labelled as `extraEmails`.
//
// Both halves of that matter:
//
//   - The LABEL lookup is `fields.<jsonTag>`, and `fields.extraEmails[0]` matches
//     nothing — so the message would fall through to humanize() and read
//     "Extra emails[0] must be a valid email address". An array index is not
//     human prose, and the whole point of the `fields` dictionary is that a
//     message never shows the reader how the data is shaped.
//   - The reported `errors[].field` is what clients key per-field rendering off,
//     and none match an indexed name — so the message would silently relocate to
//     the form's general area instead of appearing under the control it belongs to.
//
// Nested-struct dives are unaffected: go-playground already reports those by
// their LEAF field name, so this only ever bit slices of scalars.
func baseField(name string) string {
	if i := strings.IndexByte(name, '['); i > 0 && strings.HasSuffix(name, "]") {
		return name[:i]
	}
	return name
}

// fieldLabel turns a DTO JSON tag into the human, translated name shown to the
// user — `phoneNumber` → "Phone number". Labels live under the `fields` block of
// locales/*.json; shared/validate/field_labels_test.go fails the build when a
// validated DTO field has no label, because the fallback below is a safety net,
// not a licence to ship identifiers to the UI.
func fieldLabel(tr *pkgi18n.Translator, lang, jsonTag string) string {
	if tr != nil {
		key := "fields." + jsonTag
		if label := tr.T(lang, key); label != key {
			return label
		}
	}
	return humanize(jsonTag)
}

// humanize renders a camelCase identifier as a sentence-cased phrase
// ("phoneNumber" → "Phone number"). Used only when no label is available: a
// locale with no `fields` entry, or no translator bound at all (worker paths,
// unit tests). Never returns the raw identifier.
func humanize(s string) string {
	if s == "" {
		return s
	}
	var b strings.Builder
	for i, r := range s {
		switch {
		case i == 0:
			b.WriteRune(unicode.ToUpper(r))
		case unicode.IsUpper(r):
			b.WriteRune(' ')
			b.WriteRune(unicode.ToLower(r))
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// logBindError records the underlying decoder/binding error server-side. It is
// deliberately never returned to the client: Go's decoder text names the struct,
// the field and the Go type ("cannot unmarshal string into Go struct field
// CreateUserRequest.age of type int"), which is exactly the internal detail we
// keep out of the UI.
func logBindError(c *gin.Context, kind string, err error) {
	attrs := []any{"kind", kind, "error", err.Error()}
	if c != nil && c.Request != nil {
		attrs = append(attrs, "method", c.Request.Method, "path", c.Request.URL.Path)
	}
	slog.Debug("request binding failed", attrs...)
}

// messageKey resolves a validator tag to an i18n message key. Comparison tags
// mean different things per field kind — `min=1900` on an int is a lower bound,
// on a string a character count, on a slice an entry count — so they fan out into
// Number / Length / Items variants instead of one "characters" message.
func messageKey(tag string, kind reflect.Kind) string {
	switch tag {
	case "min", "max", "len", "gt", "gte", "lt", "lte":
		return tag + kindSuffix(kind)
	default:
		return tag
	}
}

func kindSuffix(kind reflect.Kind) string {
	switch kind {
	case reflect.String:
		return "Length"
	case reflect.Slice, reflect.Array, reflect.Map:
		return "Items"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return "Number"
	default:
		// Structs (time.Time), pointers, interfaces — a bare bound reads fine.
		return "Number"
	}
}

// friendlyParam renders a tag parameter for display. `oneof` packs its allowed
// values space-separated; everything else passes through untouched.
func friendlyParam(tag, param string) string {
	if tag == "oneof" {
		return strings.ReplaceAll(param, " ", ", ")
	}
	return param
}

// englishFallbacks mirrors the `validation` block of locales/en.json. It is used
// only when no translator is bound to the context; the locale files remain the
// source of truth for every served response.
var englishFallbacks = map[string]string{
	"required":  "{{.Field}} is required",
	"email":     "{{.Field}} must be a valid email address",
	"uuid":      "{{.Field}} must be a valid UUID",
	"url":       "{{.Field}} must be a valid URL",
	"e164":      "{{.Field}} must be a valid phone number in E.164 format (e.g. +15551234567)",
	"alphanum":  "{{.Field}} must contain only letters and numbers",
	"alpha":     "{{.Field}} must contain only letters",
	"numeric":   "{{.Field}} must be a number",
	"oneof":     "{{.Field}} must be one of: {{.Param}}",
	"contains":  "{{.Field}} must contain {{.Param}}",
	"unique":    "{{.Field}} must contain unique values",
	"notblank":  "{{.Field}} must not be blank",
	"minLength": "{{.Field}} must be at least {{.Param}} characters long",
	"maxLength": "{{.Field}} must not exceed {{.Param}} characters",
	"lenLength": "{{.Field}} must be exactly {{.Param}} characters long",
	"gtLength":  "{{.Field}} must be longer than {{.Param}} characters",
	"gteLength": "{{.Field}} must be at least {{.Param}} characters long",
	"ltLength":  "{{.Field}} must be shorter than {{.Param}} characters",
	"lteLength": "{{.Field}} must not exceed {{.Param}} characters",
	"minNumber": "{{.Field}} must be at least {{.Param}}",
	"maxNumber": "{{.Field}} must not exceed {{.Param}}",
	"lenNumber": "{{.Field}} must be exactly {{.Param}}",
	"gtNumber":  "{{.Field}} must be greater than {{.Param}}",
	"gteNumber": "{{.Field}} must be greater than or equal to {{.Param}}",
	"ltNumber":  "{{.Field}} must be less than {{.Param}}",
	"lteNumber": "{{.Field}} must be less than or equal to {{.Param}}",
	"minItems":  "{{.Field}} must contain at least {{.Param}} entries",
	"maxItems":  "{{.Field}} must not contain more than {{.Param}} entries",
	"lenItems":  "{{.Field}} must contain exactly {{.Param}} entries",
	"gtItems":   "{{.Field}} must contain more than {{.Param}} entries",
	"gteItems":  "{{.Field}} must contain at least {{.Param}} entries",
	"ltItems":   "{{.Field}} must contain fewer than {{.Param}} entries",
	"lteItems":  "{{.Field}} must not contain more than {{.Param}} entries",
	"default":   "{{.Field}} is not valid",
}

// fallbackMessage renders the English template for key, interpolating the same
// {{.Field}} / {{.Param}} placeholders the locale files use.
func fallbackMessage(key string, data map[string]string) string {
	tpl, ok := englishFallbacks[key]
	if !ok {
		tpl = englishFallbacks["default"]
	}
	return interpolate(tpl, data)
}

func interpolate(tpl string, data map[string]string) string {
	t, err := template.New("").Parse(tpl)
	if err != nil {
		return tpl
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return tpl
	}
	return buf.String()
}
