package app

import (
	"errors"

	"github.com/gin-gonic/gin"

	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/config"
)

// mountSwagger serves the Swagger UI at /docs/*any when cfg.Swagger says so.
// With SWAGGER_USER and SWAGGER_PASSWORD set, the whole /docs tree — the UI and
// the doc.json it loads — sits behind HTTP basic auth; the API calls "Try it
// out" makes are under /v1 and keep their own Bearer/API-key gates.
//
// Production may enable it only WITH credentials: the page lists every route,
// so an enabled production Swagger without them refuses to start rather than
// publishing the map anonymously.
func mountSwagger(root *gin.RouterGroup, cfg *config.Config, handler gin.HandlerFunc) error {
	sw := cfg.Swagger
	if !sw.Enabled {
		return nil
	}
	hasCreds := sw.User != "" && sw.Password != ""
	if cfg.App.Env == "production" && !hasCreds {
		return errors.New("SWAGGER_ENABLED in production requires SWAGGER_USER and SWAGGER_PASSWORD")
	}
	if hasCreds {
		root.GET("/docs/*any", gin.BasicAuthForRealm(gin.Accounts{sw.User: sw.Password}, "API docs"), handler)
		return nil
	}
	root.GET("/docs/*any", handler)
	return nil
}
