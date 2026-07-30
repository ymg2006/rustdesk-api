package api

import (
	"time"

	"github.com/gin-gonic/gin"
	requstform "github.com/ymg2006/rustdesk-api/v2/http/request/api"
	"github.com/ymg2006/rustdesk-api/v2/http/response"
	"github.com/ymg2006/rustdesk-api/v2/model"
	"github.com/ymg2006/rustdesk-api/v2/service"
)

type Process struct{}

// ProcessStatus client reports monitoring status (requires Bearer authentication, see RustAuth middleware)
// @Tags process monitoring
// @Summary reports process/port monitoring status
// @Description The client regularly reports whether each monitoring item is running; the response brings back the latest monitoring configuration of the device (centralized delivery in the background)
// @Accept  json
// @Produce  json
// @Param body body api.ProcessStatusForm true "Report status"
// @Success 200 {object} response.Response
// @Router /process/status [post]
// @Security token
func (p *Process) ProcessStatus(c *gin.Context) {
	f := &requstform.ProcessStatusForm{}
	if err := c.ShouldBindJSON(f); err != nil || f.PeerId == "" {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamError"))
		return
	}
	now := time.Now().Unix()
	for _, it := range f.Items {
		if it.Target == "" || (it.Type != "process" && it.Type != "port") {
			continue
		}
		service.AllService.ProcessMonitorService.UpsertAndCheck(f.PeerId, it.Name, it.Type, it.Target, it.Running, now)
	}
	response.Success(c, gin.H{"rules": toRuleOut(service.AllService.ProcessMonitorService.RulesByPeer(f.PeerId))})
}

// ProcessConfig client pulls its own monitoring configuration (Bearer authentication required)
// @Tags process monitoring
// @Summary Get device monitoring configuration
// @Produce  json
// @Param peer_id query string true "device peer id"
// @Success 200 {object} response.Response
// @Router /process/config [get]
// @Security token
func (p *Process) ProcessConfig(c *gin.Context) {
	peerId := c.Query("peer_id")
	if peerId == "" {
		response.Fail(c, 101, response.TranslateParamMsg(c, "FieldRequired", "peer_id"))
		return
	}
	response.Success(c, gin.H{"rules": toRuleOut(service.AllService.ProcessMonitorService.RulesByPeer(peerId))})
}

// toRuleOut is converted into a streamlined structure sent to the client.
func toRuleOut(rules []model.ProcessMonitorRule) []gin.H {
	out := make([]gin.H, 0, len(rules))
	for _, r := range rules {
		out = append(out, gin.H{
			"name":     r.Name,
			"type":     r.Type,
			"target":   r.Target,
			"interval": r.Interval,
		})
	}
	return out
}
