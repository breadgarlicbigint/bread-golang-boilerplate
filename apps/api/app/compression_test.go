package app

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// gin-contrib/gzip deletes Content-Length; a client dividing by it to show
// download progress gets nothing. The raw file-serving prefix must be excluded
// so a served file keeps its length and is not compressed, while the JSON API
// (including /v1/files) stays compressed.
func TestResponseCompression(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := strings.Repeat("A", 4096)
	r := gin.New()
	r.Use(responseCompression())
	serve := func(c *gin.Context) {
		c.Header("Content-Length", "4096")
		c.String(http.StatusOK, body)
	}
	r.GET("/files/*key", serve)
	r.GET("/v1/files/:id", serve)
	r.GET("/metrics", serve)

	get := func(path string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Accept-Encoding", "gzip")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}

	t.Run("a served file keeps its length and is not compressed", func(t *testing.T) {
		w := get("/files/avatars/photo.png")
		if enc := w.Header().Get("Content-Encoding"); enc != "" {
			t.Fatalf("Content-Encoding = %q, want none", enc)
		}
		if cl := w.Header().Get("Content-Length"); cl != "4096" {
			t.Fatalf("Content-Length = %q, want 4096", cl)
		}
	})

	t.Run("/metrics is not double-compressed", func(t *testing.T) {
		if enc := get("/metrics").Header().Get("Content-Encoding"); enc != "" {
			t.Fatalf("Content-Encoding = %q, want none", enc)
		}
	})

	t.Run("the JSON API under /v1/files is still compressed", func(t *testing.T) {
		if enc := get("/v1/files/abc").Header().Get("Content-Encoding"); enc != "gzip" {
			t.Fatalf("Content-Encoding = %q, want gzip", enc)
		}
	})
}
