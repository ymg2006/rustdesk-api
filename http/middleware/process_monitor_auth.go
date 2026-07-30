package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ymg2006/rustdesk-api/v2/global"
	"github.com/ymg2006/rustdesk-api/v2/http/response"
	"github.com/ymg2006/rustdesk-api/v2/service"
)

// ProcessMonitorAuth authenticates process/port monitoring report and configuration-delivery endpoints.
//
// Design goals: allow devices that are not logged into api-server accounts to report their own monitoring status,
// while still providing device identity checks based on the rustdesk key shared by clients and the server.
//  1. Logged-in clients with a valid access_token / JWT are verified normally for backward compatibility.
//  2. Unauthenticated devices are trusted when they provide the server-configured rustdesk key in X-Rustdesk-Key.
//  3. When no rustdesk key is configured on the server, requests are allowed for backward compatibility.
//  4. Requests with neither a valid token nor the correct key are rejected with 401.
//
// Note: the rustdesk key is the hbbs server public key. This check confirms that the reporter has configured
// this server's public key. It is a soft identity check; security still depends on private/internal deployment isolation.
func ProcessMonitorAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1) Logged in: prefer JWT, then fall back to DB token.
		if auth := c.GetHeader("Authorization"); auth != "" && strings.HasPrefix(auth, "Bearer ") {
			token := strings.TrimSpace(auth[7:])
			if len(token) > 0 {
				if len(global.Jwt.Key) > 0 {
					if uid, err := service.AllService.UserService.VerifyJWT(token); err == nil && uid > 0 {
						if user := service.AllService.UserService.InfoById(uid); user.Id > 0 && service.AllService.UserService.CheckUserEnable(user) && !service.AllService.UserService.IsUserExpired(user) {
							c.Set("curUser", user)
							c.Set("token", token)
							c.Next()
							return
						}
					}
				}
				if user, _ := service.AllService.UserService.InfoByAccessToken(token, ""); user.Id > 0 && service.AllService.UserService.CheckUserEnable(user) && !service.AllService.UserService.IsUserExpired(user) {
					c.Set("curUser", user)
					c.Set("token", token)
					c.Next()
					return
				}
			}
		}

		// 2) Unauthenticated device: verify device identity with the shared rustdesk key.
		serverKey := normalizeRustdeskKey(global.Config.Rustdesk.Key)
		if serverKey != "" {
			clientKey := normalizeRustdeskKey(c.GetHeader("X-Rustdesk-Key"))
			if clientKey != "" && clientKey == serverKey {
				c.Next()
				return
			}
			// Device key mismatch: reject.
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": response.TranslateMsg(c, "InvalidRustdeskKey")})
			return
		}

		// 3) Server key is not configured: allow for backward compatibility.
		c.Next()
	}
}

// normalizeRustdeskKey removes all whitespace, including newlines and spaces, to compare PEM and raw base64 forms consistently.
func normalizeRustdeskKey(k string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(k)), "")
}
