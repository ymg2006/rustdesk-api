package admin

import (
	"github.com/gin-gonic/gin"
	"github.com/ymg2006/rustdesk-api/v2/http/response"
	"github.com/ymg2006/rustdesk-api/v2/model"
	"github.com/ymg2006/rustdesk-api/v2/service"
)

type ClientDownload struct{}

// List Get list
func (v *ClientDownload) List(c *gin.Context) {
	page := 1
	pageSize := 10
	queryPage := c.Query("page")
	if queryPage != "" {
		if p, err := parseInt(queryPage); err == nil {
			page = p
		}
	}
	querySize := c.Query("page_size")
	if querySize != "" {
		if ps, err := parseInt(querySize); err == nil {
			pageSize = ps
		}
	}
	list, total := service.AllService.ClientDownloadService.List(uint(page), uint(pageSize))
	response.Success(c, gin.H{
		"list":  list,
		"total": total,
	})
}

// Create
func (v *ClientDownload) Create(c *gin.Context) {
	item := &model.ClientDownload{}
	if err := c.ShouldBindJSON(item); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamError"))
		return
	}
	if item.Name == "" || item.Url == "" {
		response.Fail(c, 101, response.TranslateMsg(c, "NameAndDownloadUrlRequired"))
		return
	}
	item.Status = int(model.COMMON_STATUS_ENABLE)
	service.AllService.ClientDownloadService.Create(item)
	response.Success(c, nil)
}

// Update update
func (v *ClientDownload) Update(c *gin.Context) {
	item := &model.ClientDownload{}
	if err := c.ShouldBindJSON(item); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamError"))
		return
	}
	if item.Id == 0 {
		response.Fail(c, 101, response.TranslateMsg(c, "IdRequired"))
		return
	}
	service.AllService.ClientDownloadService.Update(item)
	response.Success(c, nil)
}

// Delete Delete
func (v *ClientDownload) Delete(c *gin.Context) {
	form := &struct {
		Id uint `json:"id"`
	}{}
	if err := c.ShouldBindJSON(form); err != nil || form.Id == 0 {
		response.Fail(c, 101, response.TranslateMsg(c, "IdRequired"))
		return
	}
	service.AllService.ClientDownloadService.Delete(form.Id)
	response.Success(c, nil)
}

// SetEnable enable/disable
func (v *ClientDownload) SetEnable(c *gin.Context) {
	form := &struct {
		Id     uint `json:"id"`
		Status int  `json:"status"`
	}{Status: 1}
	if err := c.ShouldBindJSON(form); err != nil || form.Id == 0 {
		response.Fail(c, 101, response.TranslateMsg(c, "IdRequired"))
		return
	}
	item := service.AllService.ClientDownloadService.FindById(form.Id)
	if item == nil || item.Id == 0 {
		response.Fail(c, 101, response.TranslateMsg(c, "RecordNotFound"))
		return
	}
	item.Status = form.Status
	service.AllService.ClientDownloadService.Update(item)
	response.Success(c, nil)
}
