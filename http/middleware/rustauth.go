package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/ymg2006/rustdesk-api/v2/global"
	"github.com/ymg2006/rustdesk-api/v2/http/response"
	"github.com/ymg2006/rustdesk-api/v2/service"
)

func RustAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Read the Authorization header.
		token := c.GetHeader("Authorization")
		if token == "" || len(token) <= 7 {
			c.JSON(401, gin.H{
				"error": response.TranslateMsg(c, "Unauthorized"),
			})
			c.Abort()
			return
		}
		// Extract the token in the format Bearer {token}.
		token = token[7:]

		// Prefer JWT verification when a JWT key is configured.
		// A valid JWT directly provides uid and skips database token lookup.
		if len(global.Jwt.Key) > 0 {
			uid, err := service.AllService.UserService.VerifyJWT(token)
			if err == nil && uid > 0 {
				user := service.AllService.UserService.InfoById(uid)
				if user.Id > 0 && service.AllService.UserService.CheckUserEnable(user) && !service.AllService.UserService.IsUserExpired(user) {
					c.Set("curUser", user)
					c.Set("token", token)
					c.Next()
					return
				}
			}
			// Fall back to database token lookup when JWT verification fails for compatibility with old clients.
		}

		user, ut := service.AllService.UserService.InfoByAccessToken(token, "")
		if user.Id == 0 {
			c.JSON(401, gin.H{
				"error": response.TranslateMsg(c, "Unauthorized"),
			})
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

		service.AllService.UserService.AutoRefreshAccessToken(ut)

		c.Next()
	}
}
