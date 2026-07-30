package admin

import (
	"errors"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	adminReq "github.com/ymg2006/rustdesk-api/v2/http/request/admin"
	"github.com/ymg2006/rustdesk-api/v2/http/response"
	"github.com/ymg2006/rustdesk-api/v2/model"
	"github.com/ymg2006/rustdesk-api/v2/service"
)

type Announcement struct {
}

// List announcement list
func (a *Announcement) List(c *gin.Context) {
	announcements, err := service.AllService.AnnouncementService.ListAdmin()
	if err != nil {
		response.ServerError(c)
		return
	}
	response.Success(c, gin.H{
		"announcements": announcements,
	})
}

// Info announcement details
func (a *Announcement) Info(c *gin.Context) {
	id, err := strconv.ParseUint(c.Query("id"), 10, 64)
	if err != nil || id == 0 {
		response.Fail(c, 400, response.TranslateMsg(c, "ParamsError"))
		return
	}
	announcement, err := service.AllService.AnnouncementService.Info(uint(id))
	if errors.Is(err, service.ErrAnnouncementNotFound) {
		response.Fail(c, 404, response.TranslateMsg(c, "AnnouncementNotFound"))
		return
	}
	if err != nil {
		response.ServerError(c)
		return
	}
	response.Success(c, announcement)
}

// Create Create announcement
func (a *Announcement) Create(c *gin.Context) {
	req := &adminReq.AnnouncementCreateReq{}
	if err := c.ShouldBindJSON(req); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	req.Content = strings.TrimSpace(req.Content)
	if req.Title == "" {
		response.Fail(c, 401, response.TranslateMsg(c, "TitleRequired"))
		return
	}
	if req.Content == "" {
		response.Fail(c, 400, response.TranslateMsg(c, "ContentRequired"))
		return
	}
	status := 1
	if req.Status != nil {
		status = *req.Status
	}
	if status != 0 && status != 1 {
		response.Fail(c, 400, response.TranslateMsg(c, "InvalidAnnouncementStatus"))
		return
	}
	announcement := &model.Announcement{
		Title: req.Title, Content: req.Content, Status: status,
	}
	if err := service.AllService.AnnouncementService.Create(announcement); err != nil {
		response.ServerError(c)
		return
	}
	response.Success(c, announcement)
}

// Update update announcement
func (a *Announcement) Update(c *gin.Context) {
	req := &adminReq.AnnouncementUpdateReq{}
	if err := c.ShouldBindJSON(req); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	if req.ID == 0 {
		response.Fail(c, 401, response.TranslateMsg(c, "IdRequired"))
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	req.Content = strings.TrimSpace(req.Content)
	if req.Title == "" {
		response.Fail(c, 401, response.TranslateMsg(c, "TitleRequired"))
		return
	}
	if req.Content == "" {
		response.Fail(c, 400, response.TranslateMsg(c, "ContentRequired"))
		return
	}
	if req.Status != 0 && req.Status != 1 {
		response.Fail(c, 400, response.TranslateMsg(c, "InvalidAnnouncementStatus"))
		return
	}
	announcement := &model.Announcement{
		IdModel: model.IdModel{Id: req.ID},
		Title:   req.Title, Content: req.Content, Status: req.Status,
	}
	err := service.AllService.AnnouncementService.Update(announcement)
	if errors.Is(err, service.ErrAnnouncementNotFound) {
		response.Fail(c, 404, response.TranslateMsg(c, "AnnouncementNotFound"))
		return
	}
	if err != nil {
		response.ServerError(c)
		return
	}
	response.Success(c, nil)
}

// Delete delete announcement
func (a *Announcement) Delete(c *gin.Context) {
	id, parseErr := strconv.ParseUint(c.Query("id"), 10, 64)
	if parseErr != nil || id == 0 {
		response.Fail(c, 400, response.TranslateMsg(c, "ParamsError"))
		return
	}
	err := service.AllService.AnnouncementService.Delete(uint(id))
	if errors.Is(err, service.ErrAnnouncementNotFound) {
		response.Fail(c, 404, response.TranslateMsg(c, "AnnouncementNotFound"))
		return
	}
	if err != nil {
		response.ServerError(c)
		return
	}
	response.Success(c, nil)
}
