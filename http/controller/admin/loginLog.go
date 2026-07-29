package admin

import (
	"github.com/gin-gonic/gin"
	"github.com/ymg2006/rustdesk-api/v2/global"
	"github.com/ymg2006/rustdesk-api/v2/http/request/admin"
	"github.com/ymg2006/rustdesk-api/v2/http/response"
	"github.com/ymg2006/rustdesk-api/v2/model"
	"github.com/ymg2006/rustdesk-api/v2/service"
	"gorm.io/gorm"
	"strconv"
)

type LoginLog struct {
}

// Detail login log
// @Tags login log
// @Summary Login log details
// @Description Login log details
// @Accept  json
// @Produce  json
// @Param id path int true "ID"
// @Success 200 {object} response.Response{data=model.LoginLog}
// @Failure 500 {object} response.Response
// @Router /admin/login_log/detail/{id} [get]
// @Security token
func (ct *LoginLog) Detail(c *gin.Context) {
	id := c.Param("id")
	iid, _ := strconv.Atoi(id)
	u := service.AllService.LoginLogService.InfoById(uint(iid))
	if u.Id > 0 {
		response.Success(c, u)
		return
	}
	response.Fail(c, 101, response.TranslateMsg(c, "ItemNotFound"))
}

// List list
// @Tags login log
// @Summary Login log list
// @Description Login log list
// @Accept  json
// @Produce  json
// @Param page query int false "page number"
// @Param page_size query int false "page size"
// @Param user_id query int false "User ID"
// @Success 200 {object} response.Response{data=model.LoginLogList}
// @Failure 500 {object} response.Response
// @Router /admin/login_log/list [get]
// @Security token
func (ct *LoginLog) List(c *gin.Context) {
	query := &admin.LoginLogQuery{}
	if err := c.ShouldBindQuery(query); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	res := service.AllService.LoginLogService.List(query.Page, query.PageSize, func(tx *gorm.DB) {
		if query.UserId > 0 {
			tx.Where("user_id = ?", query.UserId)
		}
		tx.Order("id desc")
	})
	// Enrich login logs with peer aliases
	type LogEntry struct {
		model.LoginLog
		PeerAlias string `json:"peer_alias"`
	}
	var enriched []*LogEntry
	var peerIds []string
	for _, log := range res.LoginLogs {
		if log.PeerId != "" {
			peerIds = append(peerIds, log.PeerId)
		}
	}
	aliasMap := make(map[string]string)
	if len(peerIds) > 0 {
		type Pa struct {
			Id    string `json:"id"`
			Alias string `json:"alias"`
		}
		var pa []Pa
		service.DB.Model(&model.Peer{}).Where("id in ?", peerIds).Find(&pa)
		for _, p := range pa {
			aliasMap[p.Id] = p.Alias
		}
	}
	for _, log := range res.LoginLogs {
		enriched = append(enriched, &LogEntry{
			LoginLog:  *log,
			PeerAlias: aliasMap[log.PeerId],
		})
	}
	if enriched == nil {
		enriched = []*LogEntry{}
	}
	response.Success(c, gin.H{"list": enriched, "total": res.Total})
}

// Delete Delete
// @Tags login log
// @Summary Login log deletion
// @Description Login log deletion
// @Accept  json
// @Produce  json
// @Param body body model.LoginLog true "Login log information"
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /admin/login_log/delete [post]
// @Security token
func (ct *LoginLog) Delete(c *gin.Context) {
	f := &model.LoginLog{}
	if err := c.ShouldBindJSON(f); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	id := f.Id
	errList := global.Validator.ValidVar(c, id, "required,gt=0")
	if len(errList) > 0 {
		response.Fail(c, 101, errList[0])
		return
	}
	l := service.AllService.LoginLogService.InfoById(f.Id)
	if l.Id == 0 {
		response.Fail(c, 101, response.TranslateMsg(c, "ItemNotFound"))
		return
	}
	err := service.AllService.LoginLogService.Delete(l)
	if err == nil {
		response.Success(c, nil)
		return
	}
	response.Fail(c, 101, err.Error())
}

// BatchDelete Delete
// @Tags login log
// @Summary Login log batch deletion
// @Description Batch deletion of login logs
// @Accept  json
// @Produce  json
// @Param body body admin.LoginLogIds true "Login log"
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /admin/login_log/batchDelete [post]
// @Security token
func (ct *LoginLog) BatchDelete(c *gin.Context) {
	f := &admin.LoginLogIds{}
	if err := c.ShouldBindJSON(f); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	if len(f.Ids) == 0 {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError"))
		return
	}

	err := service.AllService.LoginLogService.BatchDelete(f.Ids)
	if err == nil {
		response.Success(c, nil)
		return
	}
	response.Fail(c, 101, err.Error())
}
