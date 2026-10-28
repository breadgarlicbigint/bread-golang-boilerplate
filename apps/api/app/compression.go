package app

import (
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

// responseCompression returns the global gzip middleware with the paths that
// must NOT be compressed excluded.
//
//   - /metrics: promhttp.Handler() already gzips its own response when the
//     client sends Accept-Encoding: gzip (Prometheus always does). Without the
//     exclusion this middleware gzips that already-gzipped body a second time;
//     Prometheus strips only one Content-Encoding: gzip layer and is then left
//     decoding the inner layer's raw gzip magic byte as plaintext, failing every
//     scrape ("expected a valid start token, got \x1f"). Verified against
//     Prometheus v2.55.1.
//   - /files/: gin-contrib/gzip deletes Content-Length, so any client showing
//     download progress (e.g. an in-app updater) gets no total; binary files
//     also don't compress. The JSON API under /v1/files stays compressed — only
//     the raw file-serving prefix is excluded.
func responseCompression() gin.HandlerFunc {
	return gzip.Gzip(gzip.DefaultCompression, gzip.WithExcludedPaths([]string{"/metrics", "/files/"}))
}
