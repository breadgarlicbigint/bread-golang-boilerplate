package app

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/config"
)

func swaggerServe(t *testing.T, cfg *config.Config, user, pass string) (int, error) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	err := mountSwagger(r.Group(""), cfg, func(c *gin.Context) { c.String(http.StatusOK, "docs") })
	if err != nil {
		return 0, err
	}
	req := httptest.NewRequest(http.MethodGet, "/docs/doc.json", nil)
	if user != "" {
		req.SetBasicAuth(user, pass)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w.Code, nil
}

func swaggerCfg(env string, enabled bool, user, pass string) *config.Config {
	return &config.Config{
		App:     config.AppConfig{Env: env},
		Swagger: config.SwaggerConfig{Enabled: enabled, User: user, Password: pass},
	}
}

func TestMountSwagger(t *testing.T) {
	cases := []struct {
		name       string
		cfg        *config.Config
		user, pass string
		want       int
		wantErr    bool
	}{
		{"disabled is not mounted", swaggerCfg("development", false, "", ""), "", "", http.StatusNotFound, false},
		{"dev without creds is open", swaggerCfg("development", true, "", ""), "", "", http.StatusOK, false},
		{"creds demand basic auth", swaggerCfg("staging", true, "ops", "s3cret"), "", "", http.StatusUnauthorized, false},
		{"wrong password refused", swaggerCfg("production", true, "ops", "s3cret"), "ops", "nope", http.StatusUnauthorized, false},
		{"right password served", swaggerCfg("production", true, "ops", "s3cret"), "ops", "s3cret", http.StatusOK, false},
		{"production without creds refuses to start", swaggerCfg("production", true, "ops", ""), "", "", 0, true},
		{"production disabled needs no creds", swaggerCfg("production", false, "", ""), "", "", http.StatusNotFound, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			code, err := swaggerServe(t, tc.cfg, tc.user, tc.pass)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tc.wantErr)
			}
			if !tc.wantErr && code != tc.want {
				t.Errorf("GET /docs/doc.json = %d, want %d", code, tc.want)
			}
		})
	}
}
