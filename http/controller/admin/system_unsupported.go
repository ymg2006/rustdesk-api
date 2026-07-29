//go:build !linux

package admin

import (
	"github.com/gin-gonic/gin"
	"github.com/ymg2006/rustdesk-api/v2/http/response"
)

// ServiceRestart non-Linux platforms do not support online restart and prompt manual restart.
func (co *Config) ServiceRestart(c *gin.Context) {
	response.Fail(c, 400, response.TranslateMsg(c, "UnsupportedOnlineRestart"))
}
