package http

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/ymg2006/rustdesk-api/v2/global"
	"github.com/ymg2006/rustdesk-api/v2/http/middleware"
	"github.com/ymg2006/rustdesk-api/v2/http/router"
)

func ApiInit() {
	gin.SetMode(global.Config.Gin.Mode)
	g := gin.New()

	//[WARNING] You trusted all proxies, this is NOT safe. We recommend you to set a value.
	//Please check https://pkg.go.dev/github.com/gin-gonic/gin#readme-don-t-trust-all-proxies for details.
	if global.Config.Gin.TrustProxy != "" {
		pro := strings.Split(global.Config.Gin.TrustProxy, ",")
		err := g.SetTrustedProxies(pro)
		if err != nil {
			panic(err)
		}
	}

	if global.Config.Gin.Mode == gin.ReleaseMode {
		// Redirect gin Recovery logs to the configured logger output.
		if global.Logger != nil {
			gin.DefaultErrorWriter = global.Logger.WriterLevel(logrus.ErrorLevel)
		}
	}
	g.NoRoute(func(c *gin.Context) {
		c.String(http.StatusNotFound, "404 not found")
	})
	g.Use(middleware.Recovery(), middleware.Logger(), middleware.Cors(), middleware.CSP())
	router.WebInit(g)
	router.Init(g)
	router.ApiInit(g)
	Run(g, global.Config.Gin.ApiAddr)
}
