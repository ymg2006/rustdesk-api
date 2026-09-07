package api

import (
	"github.com/gin-gonic/gin"
	"github.com/ymg2006/rustdesk-api/v2/http/response"
	"github.com/ymg2006/rustdesk-api/v2/service"
)

type Version struct {
}

// LatestVersion Get the latest version
// @Tags version detection
// @Summary Get the latest version information
// @Description The client calls this interface to check whether there is a new version
// @Accept  json
// @Produce  json
// @Param platform query string false "Platform:windows/macos/linux/ubuntu/android"
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/version/latest [get]
func (v *Version) LatestVersion(c *gin.Context) {
	platform := c.DefaultQuery("platform", "")
	ver, err := service.AllService.AppReleaseService.LatestWithError(platform)
	if err != nil {
		response.ServerError(c)
		return
	}
	if ver == nil || ver.Id == 0 {
		response.Success(c, gin.H{
			"version":      "",
			"url":          "",
			"platform":     platform,
			"note":         "",
			"force_update": false,
		})
		return
	}
	response.Success(c, gin.H{
		"version":      ver.Version,
		"url":          ver.Url,
		"platform":     ver.Platform,
		"note":         ver.Note,
		"force_update": ver.ForceUpdate == 1,
	})
}
