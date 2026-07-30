package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	requstform "github.com/ymg2006/rustdesk-api/v2/http/request/api"
	"github.com/ymg2006/rustdesk-api/v2/http/response"
	"github.com/ymg2006/rustdesk-api/v2/model"
	"github.com/ymg2006/rustdesk-api/v2/service"
)

type Index struct {
}

// Index Home Page
// @Tags Home Page
// @Summary Home Page
// @Description Home
// @Accept  json
// @Produce  json
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router / [get]
func (i *Index) Index(c *gin.Context) {
	response.Success(
		c,
		"Hello Gwen",
	)
}

// Heartbeat
// @Tags Home Page
// @Summary heartbeat
// @Description heartbeat
// @Accept  json
// @Produce  json
// @Success 200 {object} nil
// @Failure 500 {object} response.Response
// @Router /heartbeat [post]
func (i *Index) Heartbeat(c *gin.Context) {
	info := &requstform.PeerInfoInHeartbeat{}
	err := c.ShouldBindJSON(info)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{})
		return
	}
	if info.Uuid == "" {
		c.JSON(http.StatusOK, gin.H{})
		return
	}
	peer := service.AllService.PeerService.FindById(info.Id)
	if peer == nil || peer.RowId == 0 {
		c.JSON(http.StatusOK, gin.H{})
		return
	}
	//If it is within 40s, it will not be updated.
	if time.Now().Unix()-peer.LastOnlineTime >= 30 {
		upp := &model.Peer{RowId: peer.RowId, LastOnlineTime: time.Now().Unix(), LastOnlineIp: c.ClientIP()}
		service.AllService.PeerService.Update(upp)
	}

	// Record the heartbeat of active connections (conns list reported by the client)
	if len(info.Conns) > 0 {
		service.AllService.AuditService.RecordConnHeartbeat(info.Id, info.Conns)
	}

	resp := gin.H{}

	// Policy delivery: search based on user-specified priority
	// Binding priority: user (highest) > group > tag > global (bottom)
	// Multiple policies of the same type have the highest numerical priority.
	var foundStrategy *model.Strategy

	// Query all enabled policies, in descending order of numerical priority
	var allEnabled []model.Strategy
	service.DB.Where("status = 1").Order("priority desc").Find(&allEnabled)

	// Collect the tag ID associated with the device (through the address book, the tag name is stored)
	var peerTagIds []uint
	var abs []model.AddressBook
	service.DB.Where("id = ?", peer.Id).Find(&abs)
	for _, ab := range abs {
		for _, tagName := range ab.Tags {
			var tag model.Tag
			service.DB.Where("name = ?", tagName).Limit(1).Find(&tag)
			if tag.Id > 0 {
				peerTagIds = append(peerTagIds, tag.Id)
			}
		}
	}

	// Search by binding type priority: user → group → tag → global
	// For each type, take the first one with the highest numerical priority.
	bindOrder := []string{"user", "group", "tag", "global"}
	for _, bindType := range bindOrder {
		if foundStrategy != nil {
			break
		}
		for _, s := range allEnabled {
			if s.BindType != bindType {
				continue
			}
			match := false
			switch bindType {
			case "user":
				// User binding: matches the user to whom the device belongs (excluding policies shared by others)
				if peer.UserId > 0 && s.BindId == peer.UserId {
					match = true
				}
			case "group":
				// Device group binding
				if peer.GroupId > 0 && s.BindId == peer.GroupId {
					match = true
				}
			case "tag":
				// Label binding: The label is in the device’s address book
				for _, tid := range peerTagIds {
					if tid == s.BindId {
						match = true
						break
					}
				}
			case "global":
				// Global binding: works on all devices
				match = true
			}
			if match {
				foundStrategy = &s
				break
			}
		}
	}

	if foundStrategy != nil {
		// Parse ConfigItems (key=value per row) into a map
		configMap := make(map[string]string)
		for _, line := range strings.Split(foundStrategy.ConfigItems, "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				val := strings.TrimSpace(parts[1])
				if key != "" {
					configMap[key] = val
				}
			}
		}
		if len(configMap) > 0 {
			resp["strategy"] = model.StrategyOptions{ConfigOptions: configMap}
			resp["modified_at"] = time.Now().Unix()
		}
	}

	c.JSON(http.StatusOK, resp)
}

// Version version
// @Tags Home Page
// @Summary version
// @Description version
// @Accept  json
// @Produce  json
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /version [get]
func (i *Index) Version(c *gin.Context) {
	//Read resources/version file
	v := service.AllService.AppService.GetAppVersion()
	response.Success(
		c,
		v,
	)
}

// ServerInfo backend version information
// @Tags Home Page
// @Summary backend version information
// @Description returns the backend version number
// @Accept  json
// @Produce  json
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /server/info [get]
func (i *Index) ServerInfo(c *gin.Context) {
	v := service.AllService.AppService.GetAppVersion()
	response.Success(c, gin.H{
		"backend_version": v,
	})
}
