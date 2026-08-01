package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

const defaultCSP = "default-src 'self'; " +
	"script-src 'self' 'unsafe-inline'; " +
	"style-src 'self' 'unsafe-inline'; " +
	"img-src 'self' data:; " +
	"font-src 'self' data:; " +
	"connect-src 'self' ws: wss:; " +
	"form-action 'self'; " +
	"object-src 'none'; " +
	"base-uri 'self'; " +
	"frame-ancestors 'self'"

const webClientCSP = "default-src 'self'; " +
	"script-src 'self' 'unsafe-inline' 'unsafe-eval' blob:; " +
	"worker-src 'self' blob:; " +
	"connect-src 'self' https: wss: data: blob:; " +
	"img-src 'self' data: blob:; " +
	"style-src 'self' 'unsafe-inline'; " +
	"font-src 'self' data:; " +
	"media-src 'self' blob:; " +
	"manifest-src 'self'; " +
	"frame-src 'self'; " +
	"form-action 'self'; " +
	"object-src 'none'; " +
	"base-uri 'self'; " +
	"frame-ancestors 'self'"

// CSP applies a restrictive policy by default and enables the browser
// capabilities required by RustDesk Web Client V1 and V2 only.
func CSP() gin.HandlerFunc {
	return func(c *gin.Context) {
		policy := defaultCSP

		if isWebClientPath(c.Request.URL.Path) {
			policy = webClientCSP
		}

		c.Header("Content-Security-Policy", policy)
		c.Next()
	}
}

func isWebClientPath(path string) bool {
	return path == "/webclient2" ||
		strings.HasPrefix(path, "/webclient2/")
}
