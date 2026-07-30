package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ymg2006/rustdesk-api/v2/http/response"
	"github.com/ymg2006/rustdesk-api/v2/service"
	"github.com/ymg2006/rustdesk-api/v2/utils"
)

// BackendUserAuth is the backend authorization middleware.
func BackendUserAuth() gin.HandlerFunc {
	return func(c *gin.Context) {

		// Read the session token from an HttpOnly cookie; frontend JavaScript cannot read it, which limits XSS token theft.
		token, err := c.Cookie("access_token")
		if err != nil || token == "" {
			response.Fail(c, 403, response.TranslateMsg(c, "NeedLogin"))
			c.Abort()
			return
		}
		// Compute the source fingerprint using User-Agent only and compare it with the fingerprint stored at issuance.
		// IP is excluded because reverse proxies or dual-stack networks can make c.ClientIP() change within one session.
		fingerprint := utils.Md5(c.GetHeader("User-Agent"))
		user, _ := service.AllService.UserService.InfoByAccessToken(token, fingerprint)
		if user.Id == 0 {
			// Invalid, expired, or source-mismatched token: clear any leftover cookie and reject.
			secure := false
			if c.Request.TLS != nil {
				secure = true
			} else if hp := c.GetHeader("X-Forwarded-Proto"); strings.EqualFold(hp, "https") {
				secure = true
			}
			c.SetSameSite(http.SameSiteLaxMode)
			c.SetCookie("access_token", "", -1, "/", "", secure, true)
			response.Fail(c, 403, response.TranslateMsg(c, "NeedLogin"))
			c.Abort()
			return
		}

		if !service.AllService.UserService.CheckUserEnable(user) || service.AllService.UserService.IsUserExpired(user) {
			c.JSON(401, gin.H{
				"error": response.TranslateMsg(c, "Unauthorized"),
			})
			c.Abort()
			return
		}

		c.Set("curUser", user)
		c.Set("token", token)
		// Note: the web backend does not auto-renew sessions. When the token expires (default 2h), users must log in again.
		// Together with source fingerprint binding, a leaked token can only be used in the original environment and expires quickly.

		c.Next()
	}
}
