package admin

import (
	"errors"

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
	list, total, err := service.AllService.AppReleaseService.ListWithError(uint(page), uint(pageSize))
	if err != nil {
		response.ServerError(c)
		return
	}
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
		response.ServerError(c)
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
	if err := service.AllService.AppReleaseService.Update(ver); err != nil {
		if errors.Is(err, service.ErrAppReleaseNotFound) {
			response.Fail(c, 101, response.TranslateMsg(c, "VersionNotFound"))
		} else {
			response.ServerError(c)
		}
		return
	}
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
	if err := service.AllService.AppReleaseService.Delete(form.Id); err != nil {
		if errors.Is(err, service.ErrAppReleaseNotFound) {
			response.Fail(c, 101, response.TranslateMsg(c, "VersionNotFound"))
		} else {
			response.ServerError(c)
		}
		return
	}
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
	ver, err := service.AllService.AppReleaseService.FindByIdWithError(form.Id)
	if errors.Is(err, service.ErrAppReleaseNotFound) {
		response.Fail(c, 101, response.TranslateMsg(c, "VersionNotFound"))
		return
	}
	if err != nil {
		response.ServerError(c)
		return
	}
	ver.Status = form.Status
	if err := service.AllService.AppReleaseService.Update(ver); err != nil {
		if errors.Is(err, service.ErrAppReleaseNotFound) {
			response.Fail(c, 101, response.TranslateMsg(c, "VersionNotFound"))
		} else {
			response.ServerError(c)
		}
		return
	}
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
