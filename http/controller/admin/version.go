package admin

import (
	"github.com/gin-gonic/gin"
	"github.com/ymg2006/rustdesk-api/v2/http/response"
	"github.com/ymg2006/rustdesk-api/v2/model"
	"github.com/ymg2006/rustdesk-api/v2/service"
)

type Version struct {
}

func (v *Version) List(c *gin.Context) {
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
	list, total := service.AllService.AppReleaseService.List(uint(page), uint(pageSize))
	response.Success(c, gin.H{
		"list":  list,
		"total": total,
	})
}

func (v *Version) Create(c *gin.Context) {
	ver := &model.AppRelease{}
	if err := c.ShouldBindJSON(ver); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamError"))
		return
	}
	if ver.Version == "" {
		response.Fail(c, 101, response.TranslateMsg(c, "VersionRequired"))
		return
	}
	if ver.Platform == "" {
		ver.Platform = "windows"
	}
	if ver.Url == "" {
		response.Fail(c, 101, response.TranslateMsg(c, "DownloadUrlRequired"))
		return
	}
	// It is enabled by default when it is not uploaded; if it has been uploaded, it respects the front-end settings.
	if ver.Status == 0 {
		ver.Status = int(model.COMMON_STATUS_ENABLE)
	}
	if err := service.AllService.AppReleaseService.Create(ver); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "SaveFailed")+err.Error())
		return
	}
	response.Success(c, nil)
}

func (v *Version) Update(c *gin.Context) {
	ver := &model.AppRelease{}
	if err := c.ShouldBindJSON(ver); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamError"))
		return
	}
	if ver.Id == 0 {
		response.Fail(c, 101, response.TranslateMsg(c, "IdRequired"))
		return
	}
	service.AllService.AppReleaseService.Update(ver)
	response.Success(c, nil)
}

func (v *Version) Delete(c *gin.Context) {
	form := &struct {
		Id uint `json:"id"`
	}{}
	if err := c.ShouldBindJSON(form); err != nil || form.Id == 0 {
		response.Fail(c, 101, response.TranslateMsg(c, "IdRequired"))
		return
	}
	service.AllService.AppReleaseService.Delete(form.Id)
	response.Success(c, nil)
}

func (v *Version) SetEnable(c *gin.Context) {
	form := &struct {
		Id     uint `json:"id"`
		Status int  `json:"status"`
	}{Status: 1}
	if err := c.ShouldBindJSON(form); err != nil || form.Id == 0 {
		response.Fail(c, 101, response.TranslateMsg(c, "IdRequired"))
		return
	}
	ver := service.AllService.AppReleaseService.FindById(form.Id)
	if ver == nil || ver.Id == 0 {
		response.Fail(c, 101, response.TranslateMsg(c, "VersionNotFound"))
		return
	}
	ver.Status = form.Status
	service.AllService.AppReleaseService.Update(ver)
	response.Success(c, nil)
}

func parseInt(s string) (int, error) {
	var n int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, nil
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
}
