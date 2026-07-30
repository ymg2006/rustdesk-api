package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/ymg2006/rustdesk-api/v2/global"
)

// Logger is the logging middleware.
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		global.Logger.WithFields(
			logrus.Fields{
				"uri":    c.Request.URL,
				"ip":     c.ClientIP(),
				"method": c.Request.Method,
			}).Debug("Request")
		c.Next()
	}
}
