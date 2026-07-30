package http

import (
	"time"

	"github.com/ymg2006/rustdesk-api/v2/global"
)

// orDefault returns the safe default def when v <= 0; otherwise it returns v.
func orDefault(v, def int) int {
	if v <= 0 {
		return def
	}
	return v
}

// serverTimeouts builds read/write/idle timeouts from configuration.
// Missing or invalid values (<=0) fall back to safe defaults (15/30/120 seconds)
// to mitigate slow attacks and connection exhaustion.
func serverTimeouts() (read, write, idle time.Duration) {
	read = time.Duration(orDefault(global.Config.Server.ReadTimeout, 15)) * time.Second
	write = time.Duration(orDefault(global.Config.Server.WriteTimeout, 30)) * time.Second
	idle = time.Duration(orDefault(global.Config.Server.IdleTimeout, 120)) * time.Second
	return
}
