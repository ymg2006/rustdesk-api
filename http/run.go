//go:build !windows

package http

import (
	"net/http"

	"github.com/fvbock/endless"
	"github.com/gin-gonic/gin"
	"github.com/ymg2006/rustdesk-api/v2/global"
)

// Run starts the HTTP service on Unix-like platforms.
//
// It uses endless for graceful restarts (zero-downtime upgrades via SIGHUP/SIGUSR2, etc.)
// and explicitly sets read/write/idle timeouts to mitigate slow attacks. endless Serve()
// uses the embedded http.Server timeout settings, so these timeouts apply to real requests.
// If server.tls.enabled is configured, it listens with HTTPS directly.
func Run(g *gin.Engine, addr string) {
	srv := endless.NewServer(addr, g)
	read, write, idle := serverTimeouts()
	srv.ReadTimeout = read
	srv.WriteTimeout = write
	srv.IdleTimeout = idle

	var err error
	if global.Config.Server.TLS.Enabled {
		err = srv.ListenAndServeTLS(global.Config.Server.TLS.CertFile, global.Config.Server.TLS.KeyFile)
	} else {
		err = srv.ListenAndServe()
	}
	if err != nil && err != http.ErrServerClosed {
		global.Logger.Fatalf("server start failed: %v", err)
	}
}
