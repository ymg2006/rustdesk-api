package api

import (
	"github.com/gin-gonic/gin"
	"github.com/ymg2006/rustdesk-api/v2/http/response"
	"github.com/ymg2006/rustdesk-api/v2/service"
)

type Announcement struct {
}

// List client obtains the announcement list
// @Tags Announcement
// @Summary The client obtains the announcement list
// @Accept  json
// @Produce  json
// @Success 200 {object} response.Response
// @Router /api/announcements [get]
func (a *Announcement) List(c *gin.Context) {
	announcements, err := service.AllService.AnnouncementService.ListActiveForClient()
	if err != nil {
		response.ServerError(c)
		return
	}
	response.Success(c, gin.H{
		"announcements": announcements,
	})
}
