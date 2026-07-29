package admin

import (
	"github.com/gin-gonic/gin"
	"github.com/ymg2006/rustdesk-api/v2/global"
	"github.com/ymg2006/rustdesk-api/v2/http/response"
	"github.com/ymg2006/rustdesk-api/v2/model"
	"github.com/ymg2006/rustdesk-api/v2/service"
)

type AlertChannel struct{}

func (ct *AlertChannel) List(c *gin.Context) {
	page, pageSize := parsePageParams(c)
	var total int64
	var list []model.AlertChannel
	query := service.DB.Model(&model.AlertChannel{})
	query.Count(&total)
	query.Order("row_id desc").Scopes(service.Paginate(page, pageSize)).Find(&list)
	response.Success(c, gin.H{"list": list, "total": total})
}

func (ct *AlertChannel) Create(c *gin.Context) {
	f := &model.AlertChannel{}
	if err := c.ShouldBindJSON(f); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamError"))
		return
	}
	if f.Name == "" {
		response.Fail(c, 101, response.TranslateMsg(c, "ChannelNameRequired"))
		return
	}
	if err := service.DB.Create(f).Error; err != nil {
		global.Logger.Error("AlertChannel Create failed: ", err)
		response.Fail(c, 500, response.TranslateMsg(c, "SaveFailed")+err.Error())
		return
	}
	response.Success(c, f)
}

func (ct *AlertChannel) Update(c *gin.Context) {
	f := &model.AlertChannel{}
	if err := c.ShouldBindJSON(f); err != nil || f.RowId == 0 {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamError"))
		return
	}
	old := &model.AlertChannel{}
	service.DB.Where("row_id = ?", f.RowId).First(old)
	if old.RowId == 0 {
		response.Fail(c, 101, response.TranslateMsg(c, "RecordNotFound"))
		return
	}
	if f.SmtpPass == "" {
		f.SmtpPass = old.SmtpPass // Leave the password blank and do not change it.
	}
	if err := service.DB.Model(old).Updates(f).Error; err != nil {
		global.Logger.Error("AlertChannel Update failed: ", err)
		response.Fail(c, 500, response.TranslateMsg(c, "SaveFailed")+err.Error())
		return
	}
	response.Success(c, nil)
}

func (ct *AlertChannel) Delete(c *gin.Context) {
	f := &struct {
		Id uint `json:"id"`
	}{}
	if err := c.ShouldBindJSON(f); err != nil || f.Id == 0 {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamError"))
		return
	}
	// Check whether any alarm rules are using this channel
	var usage int64
	service.DB.Model(&model.AlertConfig{}).Where("channel_id = ?", f.Id).Count(&usage)
	if usage > 0 {
		response.Fail(c, 101, response.TranslateMsg(c, "ChannelInUse"))
		return
	}
	ch := &model.AlertChannel{}
	service.DB.Where("row_id = ?", f.Id).Delete(ch)
	response.Success(c, nil)
}

// AllList returns all channels (without paging) for use by the selector
func (ct *AlertChannel) AllList(c *gin.Context) {
	var list []model.AlertChannel
	service.DB.Order("id desc").Find(&list)
	response.Success(c, gin.H{"list": list})
}

// Test tests sending a message to the specified channel to verify whether the channel configuration is correct.
// Request body: AlertChannel fields (name/channel/webhook_url/smtp_*）, optional row_id and test_recipients
func (ct *AlertChannel) Test(c *gin.Context) {
	f := &model.AlertChannel{}
	if err := c.ShouldBindJSON(f); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamError"))
		return
	}
	var extra struct {
		TestRecipients string `json:"test_recipients"`
	}
	_ = c.ShouldBindJSON(&extra)
	if f.Channel == "" {
		response.Fail(c, 101, response.TranslateMsg(c, "ChannelTypeMissing"))
		return
	}
	// If a saved channel is specified and no password is provided, it will be completed from the database (the channel password in the list is empty)
	if f.RowId > 0 && f.SmtpPass == "" {
		old := &model.AlertChannel{}
		service.DB.Where("row_id = ?", f.RowId).First(old)
		if old.RowId > 0 {
			f.SmtpPass = old.SmtpPass
		}
	}
	if err := service.AllService.NotifyService.TestChannel(f, extra.TestRecipients); err != nil {
		global.Logger.Warnf("[AlertChannel] Test send failed: %v", err)
		response.Fail(c, 500, response.TranslateMsg(c, "SendFailed")+err.Error())
		return
	}
	response.Success(c, nil)
}

func parsePageParams(c *gin.Context) (page, pageSize uint) {
	page = 1
	pageSize = 20
	// default values, can be extended
	return
}
