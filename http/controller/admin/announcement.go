package admin

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ymg2006/rustdesk-api/v2/http/response"
	"github.com/ymg2006/rustdesk-api/v2/model"
	"github.com/ymg2006/rustdesk-api/v2/service"
)

type Announcement struct {
}

// List announcement list
func (a *Announcement) List(c *gin.Context) {
	announcements := service.AllService.AnnouncementService.List()
	response.Success(c, gin.H{
		"announcements": announcements,
	})
}

// Info announcement details
func (a *Announcement) Info(c *gin.Context) {
	id, _ := strconv.Atoi(c.Query("id"))
	announcement := service.AllService.AnnouncementService.Info(id)
	if announcement.Id == 0 {
		response.Fail(c, 404, response.TranslateMsg(c, "AnnouncementNotFound"))
		return
	}
	response.Success(c, announcement)
}

// Create Create announcement
func (a *Announcement) Create(c *gin.Context) {
	announcement := &model.Announcement{}
	if err := c.ShouldBindJSON(announcement); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	if announcement.Title == "" {
		response.Fail(c, 401, response.TranslateMsg(c, "TitleRequired"))
		return
	}
	service.AllService.AnnouncementService.Create(announcement)
	response.Success(c, announcement)
}

// Update update announcement
func (a *Announcement) Update(c *gin.Context) {
	announcement := &model.Announcement{}
	if err := c.ShouldBindJSON(announcement); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	if announcement.Id == 0 {
		response.Fail(c, 401, response.TranslateMsg(c, "IdRequired"))
		return
	}
	service.AllService.AnnouncementService.Update(announcement)
	response.Success(c, nil)
}

// Delete delete announcement
func (a *Announcement) Delete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Query("id"))
	announcement := service.AllService.AnnouncementService.Info(id)
	if announcement.Id == 0 {
		response.Fail(c, 404, response.TranslateMsg(c, "AnnouncementNotFound"))
		return
	}
	service.AllService.AnnouncementService.Delete(announcement)
	response.Success(c, nil)
}
