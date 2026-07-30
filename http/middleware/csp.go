package middleware

import (
	"github.com/gin-gonic/gin"
)

// CSP sets Content-Security-Policy so scripts and styles can only come from same-origin ('self').
// This blocks external JavaScript from CDNs or attacker domains at the browser layer and mitigates XSS and supply-chain attacks.
//
// Policy notes:
//   - script-src 'self' 'unsafe-inline': allows same-origin scripts only. 'unsafe-inline' keeps compatibility
//     with legacy server-rendered templates that use inline scripts. The main web-admin SPA has no inline scripts,
//     so it effectively loads only same-origin bundled JavaScript and browsers reject all external scripts.
//   - style-src 'self' 'unsafe-inline': allows same-origin styles only; external CSS such as CDN fonts is blocked.
//   - img-src/font-src allow data: for inline SPA assets; object-src 'none' disables plugins;
//     base-uri 'self' prevents <base> injection; frame-ancestors 'self' mitigates clickjacking.
//   - connect-src permits self/ws/wss to avoid breaking web client connections.
func CSP() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Content-Security-Policy",
			"script-src 'self' 'unsafe-inline'; "+
				"style-src 'self' 'unsafe-inline'; "+
				"img-src 'self' data:; "+
				"font-src 'self' data:; "+
				"connect-src 'self' ws: wss:; "+
				"form-action 'self'; "+
				"object-src 'none'; "+
				"base-uri 'self'; "+
				"frame-ancestors 'self'")
		c.Next()
	}
}
