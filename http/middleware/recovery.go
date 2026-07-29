package middleware

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/ymg2006/rustdesk-api/v2/global"
)

// Recovery is a panic recovery middleware used instead of gin.Recovery().
//
// Security point: any panic in handlers or later middleware is logged only on the server side.
// Panic content, which may include SQL, stack traces, internal paths, or other sensitive details,
// is never returned to clients. Clients always receive a generic 5xx envelope:
//
//	{"code":500,"message":"Internal server error."}
//
// SECURITY: the default gin.Recovery() returns plain text "500 Internal Server Error".
// This middleware standardizes the response as generic JSON (HTTP 200 + business code 500)
// and can be registered as the outermost middleware to catch every panic.
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				// Log only on the server side and do not leak any detail to clients.
				// Note: global.Logger can be nil in tests or uninitialized scenarios, so fall back to stdlib log.
				// Otherwise the recovery handler itself could panic again on a nil pointer.
				if global.Logger != nil {
					global.Logger.Error("panic recovered: " + fmt.Sprintf("%v", r))
				} else {
					log.Printf("[Recovery] panic recovered: %v", r)
				}
				// Return the generic 5xx envelope (HTTP 200 + code 500) only if headers have not been written yet.
				if !c.Writer.Written() {
					c.Abort()
					c.JSON(http.StatusOK, gin.H{"code": 500, "message": recoveryMessage(c)})
				}
			}
		}()
		c.Next()
	}
}

// AbortServerError aborts the request with a generic 500 response to avoid leaking internal error details
// such as SQL or stack traces. Controllers should use this for unexpected server-side errors instead of
// returning err.Error() directly to clients.
func AbortServerError(c *gin.Context) {
	if !c.Writer.Written() {
		c.Abort()

		c.JSON(http.StatusOK, gin.H{

			"code": 500,

			"message": recoveryMessage(c),
		})
	}
}

func recoveryMessage(c *gin.Context) string {
	if global.Localizer == nil {
		return "Internal server error."
	}
	msg, err := global.Localizer(c.GetHeader("Accept-Language")).LocalizeMessage(&i18n.Message{ID: "ServerInternalError"})
	if err != nil {
		return "Internal server error."
	}
	return msg
}
