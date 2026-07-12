package response

import (
	"github.com/gin-gonic/gin"
	apperrors "github.com/breadgarlicbigint/bread-golang-boilerplate/shared/errors"
	pkgi18n "github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/i18n"
)

// OKI18n translates the message key then calls OK.
func OKI18n(c *gin.Context, key string, data interface{}) {
	OK(c, pkgi18n.TC(c, key), data)
}

// OKWithMetaI18n translates the message key then calls OKWithMeta.
func OKWithMetaI18n(c *gin.Context, key string, data interface{}, meta *Meta) {
	OKWithMeta(c, pkgi18n.TC(c, key), data, meta)
}

// CreatedI18n translates the message key then calls Created.
func CreatedI18n(c *gin.Context, key string, data interface{}) {
	Created(c, pkgi18n.TC(c, key), data)
}

// ErrorI18n translates the message key then calls Error.
func ErrorI18n(c *gin.Context, status int, key string, errs ...ErrorDetail) {
	Error(c, status, pkgi18n.TC(c, key), errs...)
}

// UnauthorizedI18n translates and sends a 401.
func UnauthorizedI18n(c *gin.Context, key string) {
	Unauthorized(c, pkgi18n.TC(c, key))
}

// ForbiddenI18n translates and sends a 403.
func ForbiddenI18n(c *gin.Context, key string) {
	Forbidden(c, pkgi18n.TC(c, key))
}

// NotFoundI18n translates and sends a 404.
func NotFoundI18n(c *gin.Context, key string) {
	NotFound(c, pkgi18n.TC(c, key))
}

// HandleAppError is the single translation point from a service-layer error to
// an HTTP response. Domain errors (*errors.AppError) translate via their Key
// when set (falling back to the raw Message otherwise); anything else is
// logged and reported as a generic translated 500 — the client never sees a
// raw Go error string. Use this instead of a per-handler handleError copy.
func HandleAppError(c *gin.Context, err error) {
	if ae, ok := apperrors.As(err); ok {
		if ae.Key != "" {
			ErrorI18n(c, ae.Status, ae.Key)
		} else {
			Error(c, ae.Status, ae.Message)
		}
		return
	}
	LogInternal(err, "unexpected error")
	InternalServerError(c, pkgi18n.TC(c, "http.500"))
}

// ValidationErrorI18n maps validator.ValidationErrors to translated field messages.
func ValidationErrorI18n(c *gin.Context, fields []ErrorDetail) {
	tr, lang := pkgi18n.FromContext(c)
	if tr != nil {
		for i, f := range fields {
			translated := tr.T(lang, "validation."+f.Message, map[string]string{"Field": f.Field})
			if translated != "validation."+f.Message {
				fields[i].Message = translated
			}
		}
	}
	UnprocessableEntity(c, pkgi18n.TC(c, "http.422"), fields...)
}
