package api

import (
	"github.com/gin-gonic/gin"
	"github.com/ymg2006/rustdesk-api/v2/http/response"
	"github.com/ymg2006/rustdesk-api/v2/service"
)

type ClientDownload struct{}

// List Gets the enabled client download list (public)
func (v *ClientDownload) List(c *gin.Context) {
	list, err := service.AllService.ClientDownloadService.ActiveListWithError()
	if err != nil {
		response.ServerError(c)
		return
	}
	response.Success(c, gin.H{
		"list": list,
	})
}
