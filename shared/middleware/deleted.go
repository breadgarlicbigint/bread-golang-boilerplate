package middleware

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	apperrors "github.com/breadgarlicbigint/bread-golang-boilerplate/shared/errors"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/response"
)

// Revealing soft-deleted records.
//
// A soft-deleted row is invisible in every list and by-id read by default.
// `deleted=true` on a list route that mounts RequireDeletedView flips the read
// to the deleted rows ALONE, behind the audit:read permission.
//
// The permission check and the flag live in ONE place on purpose. The query
// string is never consulted downstream: a repository asks its handler, which
// asks IncludeDeletedFromCtx, which reads a context value only this middleware
// ever sets. So a route that does not mount it CANNOT have the flag set, and a
// module that has not been taught to honour it falls back to showing active
// rows. The failure mode is "the toggle does nothing", never "an unauthorized
// user is shown removed records".

// DeletedQueryParam is the client-facing name of the reveal flag. A wire
// identifier, never rendered as text.
const DeletedQueryParam = "deleted"

// CtxIncludeDeleted is set ONLY by RequireDeletedView, only after the permission
// check has passed. Read it through IncludeDeletedFromCtx.
const CtxIncludeDeleted = "includeDeleted"

// permAuditRead mirrors modules/role/entity.PermAuditRead. Repeated here rather
// than imported so shared/ keeps depending on nothing in modules/. audit:read is
// the gate because a removed record is an audit surface.
const permAuditRead = "audit:read"

// ErrInvalidDeletedFilter reports a `deleted` value that is not a boolean. It
// reuses the generic query-parameter message rather than adding a locale key
// that says the same thing; the offending value is a wire identifier, not prose.
var ErrInvalidDeletedFilter = apperrors.NewI18n(
	http.StatusBadRequest,
	"INVALID_DELETED_FILTER",
	"validation.invalidQuery",
	"Invalid query parameters",
)

// RequireDeletedView authorises the reveal flag. Mount it BESIDE a list route's
// own permission middleware, never in place of it: seeing deleted rows still
// requires whatever seeing the rows requires.
//
// Absent, empty or false ⇒ today's behaviour, byte for byte, and the context
// flag stays unset. A non-boolean value is a 400 rather than a silent "no".
func RequireDeletedView() gin.HandlerFunc {
	return func(c *gin.Context) {
		raw, ok := c.GetQuery(DeletedQueryParam)
		if !ok || raw == "" {
			c.Next()
			return
		}
		want, err := strconv.ParseBool(raw)
		if err != nil {
			response.HandleAppError(c, ErrInvalidDeletedFilter)
			return
		}
		if !want {
			c.Next()
			return
		}
		if !HasPermission(c, permAuditRead) {
			response.ForbiddenI18n(c, "auth.insufficientPermission")
			return
		}
		c.Set(CtxIncludeDeleted, true)
		c.Next()
	}
}

// IncludeDeletedFromCtx reports whether this request was authorised to read the
// deleted rows instead of the live ones. The ONLY reader of CtxIncludeDeleted.
//
// Handlers pass the result explicitly into their module's list filter, the way
// they already pass the tenant and the user id — a repository must not reach
// into a request context for it.
func IncludeDeletedFromCtx(c *gin.Context) bool {
	v, _ := c.Get(CtxIncludeDeleted)
	b, _ := v.(bool)
	return b
}
