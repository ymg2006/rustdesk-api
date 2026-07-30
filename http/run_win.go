//go:build windows

package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ymg2006/rustdesk-api/v2/global"
)

// Run starts the HTTP service on Windows.
//
// Windows does not use endless because its signal handling depends on Unix signals.
// Instead, it starts a standard-library http.Server and explicitly sets read/write/idle
// timeouts to mitigate slow attacks. If server.tls.enabled is configured, it listens with HTTPS directly.
func Run(g *gin.Engine, addr string) {
	read, write, idle := serverTimeouts()
	srv := &http.Server{
		Addr:         addr,
		Handler:      g,
		ReadTimeout:  read,
		WriteTimeout: write,
		IdleTimeout:  idle,
	}

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
