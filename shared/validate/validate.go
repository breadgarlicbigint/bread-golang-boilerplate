// Package validate provides a single BindJSON helper that every handler uses.
// It ensures consistent error responses across the entire API:
//   - Malformed JSON           → 400 Bad Request   (message only, no field details)
//   - Struct validation errors → 422 Unprocessable Entity (errors[] with field+message)
package validate

import (
	"fmt"
	"reflect"
	"strings"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/response"
	pkgi18n "github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/i18n"
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
		response.BadRequest(c, pkgi18n.TC(c, "validation.invalidBody")+": "+sanitiseBindError(err))
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
		response.BadRequest(c, pkgi18n.TC(c, "validation.invalidQuery")+": "+sanitiseBindError(err))
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
// Field names use the JSON tag (registered above). Messages are i18n-translated
// when available, otherwise human-readable from the tag name.
func buildDetails(c *gin.Context, err error) []response.ErrorDetail {
	ve, ok := err.(validator.ValidationErrors)
	if !ok {
		return []response.ErrorDetail{{Message: err.Error()}}
	}

	tr, lang := pkgi18n.FromContext(c)

	details := make([]response.ErrorDetail, 0, len(ve))
	for _, fe := range ve {
		field := fe.Field() // already the JSON name thanks to RegisterTagNameFunc
		tag := fe.Tag()
		param := fe.Param()

		// Try i18n translation
		msg := humanMessage(tag, field, param)
		if tr != nil {
			translated := tr.T(lang, "validation."+tag, map[string]string{
				"Field": capitalize(field),
				"Param": param,
			})
			if translated != "validation."+tag {
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

// humanMessage produces a readable default message for known validator tags.
func humanMessage(tag, field, param string) string {
	field = capitalize(field)
	switch tag {
	case "required":
		return fmt.Sprintf("%s is required", field)
	case "email":
		return fmt.Sprintf("%s must be a valid email address", field)
	case "min":
		return fmt.Sprintf("%s must be at least %s characters long", field, param)
	case "max":
		return fmt.Sprintf("%s must not exceed %s characters", field, param)
	case "len":
		return fmt.Sprintf("%s must be exactly %s characters long", field, param)
	case "alphanum":
		return fmt.Sprintf("%s must contain only letters and numbers", field)
	case "alpha":
		return fmt.Sprintf("%s must contain only letters", field)
	case "numeric":
		return fmt.Sprintf("%s must be a number", field)
	case "oneof":
		return fmt.Sprintf("%s must be one of: %s", field, strings.ReplaceAll(param, " ", ", "))
	case "url":
		return fmt.Sprintf("%s must be a valid URL", field)
	case "e164":
		return fmt.Sprintf("%s must be a valid phone number in E.164 format (e.g. +15551234567)", field)
	case "gt":
		return fmt.Sprintf("%s must be greater than %s", field, param)
	case "gte":
		return fmt.Sprintf("%s must be greater than or equal to %s", field, param)
	case "lt":
		return fmt.Sprintf("%s must be less than %s", field, param)
	case "lte":
		return fmt.Sprintf("%s must be less than or equal to %s", field, param)
	case "unique":
		return fmt.Sprintf("%s must contain unique values", field)
	default:
		return fmt.Sprintf("%s failed validation: %s", field, tag)
	}
}

// sanitiseBindError removes the noisy "json: " prefix from binding errors.
func sanitiseBindError(err error) string {
	msg := err.Error()
	msg = strings.TrimPrefix(msg, "json: ")
	return msg
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}
