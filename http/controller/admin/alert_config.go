package admin

import (
	"github.com/gin-gonic/gin"
	"github.com/ymg2006/rustdesk-api/v2/global"
	"github.com/ymg2006/rustdesk-api/v2/http/response"
	"github.com/ymg2006/rustdesk-api/v2/model"
	"github.com/ymg2006/rustdesk-api/v2/service"
)

type AlertConfig struct{}

func (c *AlertConfig) List(ctx *gin.Context) {
	var configs []model.AlertConfig
	service.DB.Find(&configs)
	response.Success(ctx, gin.H{"list": configs})
}

func (c *AlertConfig) Create(ctx *gin.Context) {
	f := &model.AlertConfig{}
	if err := ctx.ShouldBindJSON(f); err != nil {
		response.Fail(ctx, 101, response.TranslateMsg(ctx, "ParamError"))
		return
	}
	if f.ChannelId == 0 {
		response.Fail(ctx, 101, response.TranslateMsg(ctx, "SelectNotifyChannel"))
		return
	}
	f.UserId = 0 // 0 means administrator sharing
	// Get channel type from channel
	ch := &model.AlertChannel{}
	service.DB.Where("row_id = ?", f.ChannelId).First(ch)
	if ch.RowId == 0 {
		response.Fail(ctx, 101, response.TranslateMsg(ctx, "NotifyChannelNotFound"))
		return
	}
	f.Channel = ch.Channel
	f.Name = ch.Name
	if err := service.DB.Create(f).Error; err != nil {
		global.Logger.Error("AlertConfig Create failed: ", err)
		response.Fail(ctx, 500, response.TranslateMsg(ctx, "SaveFailed")+err.Error())
		return
	}
	response.Success(ctx, f)
}

func (c *AlertConfig) Update(ctx *gin.Context) {
	f := &model.AlertConfig{}
	if err := ctx.ShouldBindJSON(f); err != nil || f.RowId == 0 {
		response.Fail(ctx, 101, response.TranslateMsg(ctx, "ParamError"))
		return
	}
	// If the channel_id is updated, the channel name is updated simultaneously
	if f.ChannelId > 0 {
		ch := &model.AlertChannel{}
		service.DB.Where("row_id = ?", f.ChannelId).First(ch)
		if ch.RowId > 0 {
			f.Channel = ch.Channel
			f.Name = ch.Name
		}
	}
	if err := service.DB.Model(&model.AlertConfig{}).Where("row_id = ?", f.RowId).Updates(f).Error; err != nil {
		global.Logger.Error("AlertConfig Update failed: ", err)
		response.Fail(ctx, 500, response.TranslateMsg(ctx, "SaveFailed")+err.Error())
		return
	}
	response.Success(ctx, nil)
}

func (c *AlertConfig) Delete(ctx *gin.Context) {
	form := &struct {
		Id uint `json:"id"`
	}{}
	if err := ctx.ShouldBindJSON(form); err != nil || form.Id == 0 {
		response.Fail(ctx, 101, response.TranslateMsg(ctx, "IdRequired"))
		return
	}
	// Make sure the record exists first
	var cfg model.AlertConfig
	service.DB.Where("row_id = ?", form.Id).First(&cfg)
	if cfg.RowId == 0 {
		response.Fail(ctx, 101, response.TranslateMsg(ctx, "RecordNotFound"))
		return
	}
	// Cascade deletion of monitoring targets to avoid leaving orphan data
	service.DB.Where("alert_id = ?", cfg.RowId).Delete(&model.AlertTarget{})
	service.DB.Delete(&cfg)
	response.Success(ctx, nil)
}
