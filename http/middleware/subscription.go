package middleware

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ymg2006/rustdesk-api/v2/http/response"
	"github.com/ymg2006/rustdesk-api/v2/model"
	"github.com/ymg2006/rustdesk-api/v2/service"
)

// SubscriptionGuard validates subscription status.
// It is attached to user API groups and checks whether the current user's subscription is active.
// Allowlisted paths skip this check; see whitelist.
//
// Middleware order: CORS → Recovery → JwtAuth → SubscriptionGuard.
// JwtAuth has already injected curUser into the context.
type SubscriptionGuard struct {
	// whitelist contains allowed path prefixes without the /api/v1 prefix.
	// Public paths do not require JWT: login, register, notify, health, static.
	// JWT-required but subscription-exempt paths: create-order, order query, claim, redeem, mine.
	whitelist []string
}

// NewSubscriptionGuard creates a subscription guard.
func NewSubscriptionGuard() *SubscriptionGuard {
	return &SubscriptionGuard{
		whitelist: []string{
			// Authenticated but subscription-exempt; expired users can also access these endpoints.
			"subscribe/create-order",
			"subscribe/order/",
			"subscribe/claim",
			"subscribe/redeem",
			"subscribe/mine",
		},
	}
}

// Handle returns a gin.HandlerFunc.
func (g *SubscriptionGuard) Handle() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check whether the current path matches the allowlist.
		path := c.Request.URL.Path
		// Remove the /api/v1/ prefix.
		relativePath := strings.TrimPrefix(path, "/api/v1/")

		if g.isWhitelisted(relativePath) {
			c.Next()
			return
		}

		// Read the current user from context; JwtAuth injects it.
		userInterface, exists := c.Get("curUser")
		if !exists {
			response.Fail(c, 403, response.TranslateMsg(c, "NeedLogin"))
			c.Abort()
			return
		}
		user, ok := userInterface.(*model.User)
		if !ok {
			response.Fail(c, 403, response.TranslateMsg(c, "NeedLogin"))
			c.Abort()
			return
		}

		// Check subscription status.
		if user.SubscriptionExpireAt == nil || user.SubscriptionExpireAt.Before(time.Now()) {
			// Re-query to get fresh data and avoid stale cached values.
			freshUser := service.AllService.UserService.InfoById(user.Id)
			if freshUser.Id == 0 || freshUser.SubscriptionExpireAt == nil || freshUser.SubscriptionExpireAt.Before(time.Now()) {
				response.Fail(c, 4001, response.TranslateMsg(c, "SubscriptionExpired"))
				c.Abort()
				return
			}
			// Update the user stored in context.
			c.Set("curUser", freshUser)
			c.Next()
			return
		}

		c.Next()
	}
}

// isWhitelisted checks whether the path is allowlisted.
func (g *SubscriptionGuard) isWhitelisted(path string) bool {
	for _, wl := range g.whitelist {
		if strings.HasPrefix(path, wl) {
			return true
		}
	}
	return false
}
