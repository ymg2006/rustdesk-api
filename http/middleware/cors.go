package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ymg2006/rustdesk-api/v2/global"
)

// Cors is the CORS middleware.
//
// SECURITY before the fix: the old implementation reflected the request Origin directly into
// Access-Control-Allow-Origin and always sent Access-Control-Allow-Credentials: true. That allowed
// arbitrary third-party sites to issue credentialed cross-origin requests (api-token / Cookie) in
// the user's browser and read responses, causing CSRF and sensitive-data exposure risks.
//
// SECURITY after the fix: reflect the Origin only when it matches the configured allowlist
// (cors.allow-origins), and allow credentials only on matches. An empty allowlist disables CORS
// completely by omitting ACAO, so browsers block cross-origin responses.
func Cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		// Same-origin requests or non-browser requests without Origin do not need CORS handling.
		if origin == "" {
			c.Next()
			return
		}

		// Origin not in the allowlist: do not set any CORS response headers.
		// Browsers block the actual response when Access-Control-Allow-Origin is missing.
		// Preflight OPTIONS requests are also rejected without ACAO, disabling CORS at the source.
		if !isOriginAllowed(origin) {
			c.Next()
			return
		}

		// Reflect the concrete origin and allow credentials only for allowlisted origins.
		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Headers", "api-token,content-type,authorization")
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")

		if c.Request.Method == http.MethodOptions {
			// Preflight request: return 204 directly for allowlisted origins without entering later routes.
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// isOriginAllowed checks whether origin is in the allowlist using exact, case-insensitive matching.
func isOriginAllowed(origin string) bool {
	for _, allowed := range global.Config.Cors.AllowOrigins {
		if allowed == "" {
			continue
		}
		if strings.EqualFold(allowed, origin) {
			return true
		}
	}
	return false
}
