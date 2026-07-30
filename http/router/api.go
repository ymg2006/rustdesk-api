package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	_ "github.com/ymg2006/rustdesk-api/v2/docs/api"
	"github.com/ymg2006/rustdesk-api/v2/global"
	"github.com/ymg2006/rustdesk-api/v2/http/controller/api"
	"github.com/ymg2006/rustdesk-api/v2/http/middleware"
)

func ApiInit(g *gin.Engine) {

	//g.Use(middleware.Cors())
	//swagger
	if global.Config.App.ShowSwagger == 1 {
		g.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.InstanceName("api")))
	}
	// Load HTML templates.
	g.LoadHTMLGlob("resources/templates/*")

	frg := g.Group("/api")

	{
		i := &api.Index{}
		frg.GET("/", i.Index)
		frg.GET("/version", i.Version)
		frg.GET("/admin/server/info", i.ServerInfo)

		frg.POST("/heartbeat", i.Heartbeat)
	}

	{
		l := &api.Login{}
		// If OIDC is returned, users can log in with OIDC.
		frg.GET("/login-options", l.LoginOptions)
		frg.POST("/login", middleware.Limiter(), l.Login)

	}

	{
		o := &api.Oauth{}
		// [method:POST] [uri:/api/oidc/auth]
		frg.POST("/oidc/auth", o.OidcAuth)
		// [method:GET] [uri:/api/oidc/auth-query?code=abc&id=xxxxx&uuid=xxxxx]
		frg.GET("/oidc/auth-query", o.OidcAuthQuery)
		//api/oauth/callback
		frg.GET("/oauth/callback", o.OauthCallback)
		frg.GET("/oauth/login", o.OauthCallback)
		frg.GET("/oauth/msg", o.Message)

		frg.GET("/oidc/callback", o.OauthCallback)
		frg.GET("/oidc/login", o.OauthCallback)
		frg.GET("/oidc/msg", o.Message)
	}
	{
		pe := &api.Peer{}
		// Submit system information.
		frg.POST("/sysinfo", pe.SysInfo)
		frg.POST("/sysinfo_ver", pe.SysInfoVer)
	}

	{
		v := &api.Version{}
		frg.GET("/version/latest", v.LatestVersion)
	}

	{
		cd := &api.ClientDownload{}
		frg.GET("/client-downloads", cd.List)
	}

	// User-facing download page.
	g.GET("/downloads", func(c *gin.Context) {
		c.HTML(200, "client_downloads.html", gin.H{})
	})

	if global.Config.App.WebClient == 1 {
		WebClientRoutes(frg)
	}

	{
		au := &api.Audit{}
		//[method:POST] [uri:/api/audit/conn]
		frg.POST("/audit/conn", au.AuditConn)
		//[method:POST] [uri:/api/audit/file]
		frg.POST("/audit/file", au.AuditFile)
	}

	// Process/port monitoring: clients report status and fetch delivered configuration.
	// Unauthenticated devices without tokens can report; authenticated devices still verify tokens.
	{
		pm := &api.Process{}
		frg.POST("/process/status", middleware.ProcessMonitorAuth(), pm.ProcessStatus)
		frg.GET("/process/config", middleware.ProcessMonitorAuth(), pm.ProcessConfig)
	}

	// Subscription payment callbacks (public).
	{
		sc := &api.SubscribeController{}
		frg.POST("/subscribe/notify", sc.Notify)
		frg.POST("/subscribe/webhook", sc.Webhook)
		frg.POST("/subscribe/sms-webhook", sc.SmsWebhook)
		frg.GET("/subscribe/plans", sc.Plans)
	}

	{
		an := &api.Announcement{}
		// Announcements (no login required).
		frg.GET("/announcements", an.List)
	}

	frg.Use(middleware.RustAuth())
	{
		u := &api.User{}
		frg.GET("/user/info", u.Info)
		frg.POST("/currentUser", u.Info)
	}
	{
		l := &api.Login{}
		frg.POST("/logout", l.Logout)
	}
	{
		gr := &api.Group{}
		frg.GET("/users", gr.Users)
		frg.GET("/peers", gr.Peers)
		// /api/device-group/accessible?current=1&pageSize=100
		frg.GET("/device-group/accessible", gr.Device)
	}

	{
		ab := &api.Ab{}
		// Get address book.
		frg.GET("/ab", ab.Ab)
		// Update address book.
		frg.POST("/ab", ab.UpAb)
	}

	PersonalRoutes(frg)

	// Subscription payment endpoints (login required).
	{
		sc := &api.SubscribeController{}
		frg.POST("/subscribe/create-order", sc.CreateOrder)
		frg.GET("/subscribe/order/:out_trade_no", sc.QueryOrder)
		frg.POST("/subscribe/claim", sc.Claim)
		frg.POST("/subscribe/redeem", sc.Redeem)
		frg.GET("/subscribe/mine", sc.Mine)
	}

	// Process/port monitoring routes were moved before RustAuth to support unauthenticated device reports.

	// Serve static files.
	g.StaticFS("/upload", http.Dir(global.Config.Gin.ResourcesPath+"/public/upload"))
	// QR-code payment images (resources/static/qr/).
	g.StaticFS("/static/qr", http.Dir(global.Config.Gin.ResourcesPath+"/static/qr"))
}

func PersonalRoutes(frg *gin.RouterGroup) {
	{
		ab := &api.Ab{}
		frg.POST("/ab/personal", ab.Personal)
		//[method:POST] [uri:/api/ab/settings] Request
		frg.POST("/ab/settings", ab.Settings)
		// [method:POST] [uri:/api/ab/shared/profiles?current=1&pageSize=100]
		frg.POST("/ab/shared/profiles", ab.SharedProfiles)
		//[method:POST] [uri:/api/ab/peers?current=1&pageSize=100&ab=1]
		frg.POST("/ab/peers", ab.Peers)
		// [method:POST] [uri:/api/ab/tags/1]
		frg.POST("/ab/tags/:guid", ab.PTags)
		//[method:POST] api/ab/peer/add/1
		frg.POST("/ab/peer/add/:guid", ab.PeerAdd)
		//[method:DELETE] [uri:/api/ab/peer/1]
		frg.DELETE("/ab/peer/:guid", ab.PeerDel)
		//[method:PUT] [uri:/api/ab/peer/update/1]
		frg.PUT("/ab/peer/update/:guid", ab.PeerUpdate)
		//[method:POST] [uri:/api/ab/tag/add/1]
		frg.POST("/ab/tag/add/:guid", ab.TagAdd)
		//[method:PUT] [uri:/api/ab/tag/rename/1]
		frg.PUT("/ab/tag/rename/:guid", ab.TagRename)
		//[method:PUT] [uri:/api/ab/tag/update/1]
		frg.PUT("/ab/tag/update/:guid", ab.TagUpdate)
		//[method:DELETE] [uri:/api/ab/tag/1]
		frg.DELETE("/ab/tag/:guid", ab.TagDel)

	}

}

func WebClientRoutes(frg *gin.RouterGroup) {
	w := &api.WebClient{}
	{
		frg.POST("/shared-peer", w.SharedPeer)
	}
	{
		frg.POST("/server-config", middleware.RustAuth(), w.ServerConfig)
		frg.POST("/server-config-v2", middleware.RustAuth(), w.ServerConfigV2)
	}

}
