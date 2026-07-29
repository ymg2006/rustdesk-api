package api

import (
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	request "github.com/ymg2006/rustdesk-api/v2/http/request/api"
	"github.com/ymg2006/rustdesk-api/v2/http/response"
	"github.com/ymg2006/rustdesk-api/v2/model"
	"github.com/ymg2006/rustdesk-api/v2/service"
	"time"
)

type Audit struct {
}

// AuditConn
// @Tags Audit
// @Summary Audit connection
// @Description audit connection
// @Accept  json
// @Produce  json
// @Param body body api.AuditConnForm true "Audit connection"
// @Success 200 {string} string ""
// @Failure 500 {object} response.Response
// @Router /audit/conn [post]
func (a *Audit) AuditConn(c *gin.Context) {
	af := &request.AuditConnForm{}
	err := c.ShouldBindBodyWith(af, binding.JSON)
	if err != nil {
		response.Error(c, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	/*ttt := &gin.H{}
	c.ShouldBindBodyWith(ttt, binding.JSON)
	fmt.Println(ttt)*/
	ac := af.ToAuditConn()
	if af.Action == model.AuditActionNew {
		// When the same user initiates a connection to the same peer again, the previous record that is still "in progress" will be closed first.
		// Avoid abnormal disconnection (no close audit received) causing the "Recent Connection Record" to always display "In Progress".
		service.AllService.AuditService.CloseInProgressByFromPeerAndPeer(ac.FromPeer, ac.PeerId)
		// If the new request does not carry session_id, ToAuditConn will format it as "0"; leave it blank to avoid upsert.
		// Overwrites the real session_id written by subsequent (or previous) update requests with "0".
		ac.SessionId = ""
		// Use upsert instead of Create directly: If the update request without action arrives first and the record has been created,
		// The same record should be hit here to completeip/actioninstead of inserting a duplicate record.
		service.AllService.AuditService.UpsertByPeerIdAndConnId(ac)
	} else if af.Action == model.AuditActionClose {
		ex := service.AllService.AuditService.InfoByPeerIdAndConnId(af.Id, af.ConnId)
		if ex.Id != 0 {
			ex.CloseTime = time.Now().Unix()
			service.AllService.AuditService.UpdateAuditConn(ex)
		}
	} else if af.Action == "" {
		// Completion update after successful authorization: changed to upsert. If the new request has not yet been dropped into the library (race condition), the record can also be created first.
		// This prevents information such as source names from being lost due to unavailable records.
		up := &model.AuditConn{
			ConnId:    ac.ConnId,
			PeerId:    ac.PeerId,
			FromPeer:  ac.FromPeer,
			FromName:  ac.FromName,
			SessionId: ac.SessionId,
			Type:      ac.Type,
		}
		service.AllService.AuditService.UpsertByPeerIdAndConnId(up)
		// When the completion update carries complete source information, it also tries to close the old in-progress record of the same (from_peer, peer_id).
		// Solve the problem that the "new" request lacks the peer field during reconnection, causing the old record to not be closed.
		if ac.FromPeer != "" && ac.PeerId != "" {
			service.AllService.AuditService.CloseInProgressByFromPeerAndPeer(ac.FromPeer, ac.PeerId)
		}
	}
	response.Success(c, "")
}

// AuditFile
// @Tags Audit
// @Summary audit file
// @Description audit file
// @Accept  json
// @Produce  json
// @Param body body api.AuditFileForm true "audit documents"
// @Success 200 {string} string ""
// @Failure 500 {object} response.Response
// @Router /audit/file [post]
func (a *Audit) AuditFile(c *gin.Context) {
	aff := &request.AuditFileForm{}
	err := c.ShouldBindBodyWith(aff, binding.JSON)
	if err != nil {
		response.Error(c, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	//ttt := &gin.H{}
	//c.ShouldBindBodyWith(ttt, binding.JSON)
	//fmt.Println(ttt)
	af := aff.ToAuditFile()
	service.AllService.AuditService.CreateAuditFile(af)
	response.Success(c, "")
}
