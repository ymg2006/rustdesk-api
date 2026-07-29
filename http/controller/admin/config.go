package admin

import (
	"github.com/gin-gonic/gin"
	"github.com/ymg2006/rustdesk-api/v2/global"
	"github.com/ymg2006/rustdesk-api/v2/http/response"
	"github.com/ymg2006/rustdesk-api/v2/model"
	"github.com/ymg2006/rustdesk-api/v2/service"
	"os"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
}

// ServerConfig RUSTDESK service configuration
// @Tags ADMIN
// @Summary RUSTDESK service configuration
// @Description service configuration, providing api-server to webclient
// @Accept  json
// @Produce  json
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /admin/config/server [get]
// @Security token
func (co *Config) ServerConfig(c *gin.Context) {
	cf := &response.ServerConfigResponse{
		IdServer:    global.Config.Rustdesk.IdServer,
		Key:         global.Config.Rustdesk.Key,
		RelayServer: global.Config.Rustdesk.RelayServer,
		ApiServer:   global.Config.Rustdesk.ApiServer,
	}
	response.Success(c, cf)
}

// AppConfig APP service configuration
// @Tags ADMIN
// @Summary APP service configuration
// @Description APP service configuration
// @Accept  json
// @Produce  json
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /admin/config/app [get]
// @Security token
func (co *Config) AppConfig(c *gin.Context) {
	response.Success(c, &gin.H{
		"web_client": global.Config.App.WebClient,
	})
}

// AdminConfig ADMIN service configuration
// @Tags ADMIN
// @Summary ADMIN service configuration
// @Description ADMIN service configuration
// @Accept  json
// @Produce  json
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /admin/config/admin [get]
// @Security token
func (co *Config) AdminConfig(c *gin.Context) {

	u := &model.User{}
	token, _ := c.Cookie("access_token")
	if token != "" {
		u, _ = service.AllService.UserService.InfoByAccessToken(token, "")
		if !service.AllService.UserService.CheckUserEnable(u) {
			u.Id = 0
		}
	}

	if u.Id == 0 {
		response.Success(c, &gin.H{
			"title": global.Config.Admin.Title,
		})
		return
	}

	hello := global.Config.Admin.Hello
	if hello == "" {
		helloFile := global.Config.Admin.HelloFile
		if helloFile != "" {
			b, err := os.ReadFile(helloFile)
			if err == nil && len(b) > 0 {
				hello = string(b)
			}
		}
	}

	//replace {{username}} to username
	hello = strings.Replace(hello, "{{username}}", u.Username, -1)
	response.Success(c, &gin.H{
		"title": global.Config.Admin.Title,
		"hello": hello,
	})
}

// ConfigFileGet reads the original content of the backend configuration file (config.yaml), only available to administrators
// @Tags ADMIN
// @Summary Read the backend configuration file
// @Description reads the original content of the configuration file for front-end editing
// @Produce json
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /admin/config/file/get [get]
// @Security token
func (co *Config) ConfigFileGet(c *gin.Context) {
	path := global.ConfigPath
	if path == "" {
		response.Fail(c, 500, response.TranslateMsg(c, "ConfigPathUnknown"))
		return
	}
	b, err := os.ReadFile(path)
	if err != nil {
		response.Fail(c, 500, response.TranslateMsg(c, "ReadConfigFailed")+err.Error())
		return
	}
	response.Success(c, gin.H{
		"path":    path,
		"content": string(b),
	})
}

// ConfigFileUpdate saves the original content of the backend configuration file (config.yaml) and is only available to administrators
// @Tags ADMIN
// @Summary Save the backend configuration file
// @Description Verifies YAML and writes back the configuration file. Modifications need to restart the service to take effect
// @Accept json
// @Produce json
// @Param body body object{content=string} true "Configuration file content"
// @Success 200 {object} response.Response
// @Failure 101 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /admin/config/file/update [post]
// @Security token
func (co *Config) ConfigFileUpdate(c *gin.Context) {
	type Req struct {
		Content string `json:"content" binding:"required"`
	}
	var req Req
	if err := c.ShouldBindJSON(&req); err != nil {
		global.Logger.Error("config update bind/parse error: " + err.Error())
		response.Fail(c, 101, response.TranslateMsg(c, "ConfigFormatInvalid"))
		return
	}
	path := global.ConfigPath
	if path == "" {
		response.Fail(c, 500, response.TranslateMsg(c, "ConfigPathUnknown"))
		return
	}
	// Verify the validity of YAML (not actually loaded into the running configuration, only parsed)
	v := viper.New()
	v.SetConfigType("yaml")
	if err := v.ReadConfig(strings.NewReader(req.Content)); err != nil {
		global.Logger.Error("config file YAML parse error: " + err.Error())
		response.Fail(c, 101, response.TranslateMsg(c, "ConfigYamlInvalid"))
		return
	}
	if err := os.WriteFile(path, []byte(req.Content), 0644); err != nil {
		response.Fail(c, 500, response.TranslateMsg(c, "WriteConfigFailed")+err.Error())
		return
	}
	response.Success(c, nil)
}
