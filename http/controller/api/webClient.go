package api

import (
	"github.com/gin-gonic/gin"
	"github.com/ymg2006/rustdesk-api/v2/global"
	"github.com/ymg2006/rustdesk-api/v2/http/response"
	"github.com/ymg2006/rustdesk-api/v2/http/response/api"
	"github.com/ymg2006/rustdesk-api/v2/service"
	"time"
)

type WebClient struct {
}

// ServerConfig service configuration
// @Tags WEBCLIENT
// @Summary Service configuration
// @Description service configuration, providing api-server to webclient
// @Accept  json
// @Produce  json
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /server-config [get]
// @Security token
func (i *WebClient) ServerConfig(c *gin.Context) {
	u := service.AllService.UserService.CurUser(c)

	peers := map[string]*api.WebClientPeerPayload{}
	abs := service.AllService.AddressBookService.ListByUserIdAndCollectionId(u.Id, 0, 1, 100)
	for _, ab := range abs.AddressBooks {
		pp := &api.WebClientPeerPayload{}
		pp.FromAddressBook(ab)
		peers[ab.Id] = pp
	}
	response.Success(
		c,
		gin.H{
			"id_server": global.Config.Rustdesk.IdServer,
			"key":       global.Config.Rustdesk.Key,
			"peers":     peers,
		},
	)
}

// SharedPeer shared peer
// @Tags WEBCLIENT
// @Summary shared by peers
// @Description shared peer
// @Accept  json
// @Produce  json
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /shared-peer [post]
func (i *WebClient) SharedPeer(c *gin.Context) {
	j := &gin.H{}
	c.ShouldBindJSON(j)
	t := (*j)["share_token"].(string)
	if t == "" {
		response.Fail(c, 101, response.TranslateMsg(c, "ShareTokenRequired"))
		return
	}
	sr := service.AllService.AddressBookService.SharedPeer(t)
	if sr == nil || sr.Id == 0 {
		response.Fail(c, 101, response.TranslateMsg(c, "ShareNotFound"))
		return
	}
	if sr.Expire != 0 {
		//Determine whether it has expired, created_at + expire > now
		ca := time.Time(sr.CreatedAt)
		if ca.Add(time.Second * time.Duration(sr.Expire)).Before(time.Now()) {
			response.Fail(c, 101, response.TranslateMsg(c, "ShareExpired"))
			return
		}
	}

	ab := service.AllService.AddressBookService.InfoByUserIdAndId(sr.UserId, sr.PeerId)
	if ab.RowId == 0 {
		response.Fail(c, 101, response.TranslateMsg(c, "PeerNotFound"))
		return
	}
	pp := &api.WebClientPeerPayload{}
	pp.FromShareRecord(sr)
	pp.Info.Username = ab.Username
	pp.Info.Hostname = ab.Hostname
	response.Success(c, gin.H{
		"id_server": global.Config.Rustdesk.IdServer,
		"key":       global.Config.Rustdesk.Key,
		"peer":      pp,
	})
}

// ServerConfigV2 service configuration
// @Tags WEBCLIENT_V2
// @Summary Service configuration
// @Description service configuration, providing api-server to webclient
// @Accept  json
// @Produce  json
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /server-config-v2 [get]
// @Security token
func (i *WebClient) ServerConfigV2(c *gin.Context) {
	response.Success(
		c,
		gin.H{
			"id_server": global.Config.Rustdesk.IdServer,
			"key":       global.Config.Rustdesk.Key,
		},
	)
}
