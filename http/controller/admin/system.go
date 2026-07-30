//go:build linux

package admin

import (
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ymg2006/rustdesk-api/v2/global"
	"github.com/ymg2006/rustdesk-api/v2/http/response"
)

// ServiceRestart restarts the backend service process, only available to administrators
// Restart strategy (automatic detection by priority):
//  1. The environment variable RUSTDESK_API_RESTART_CMD specifies custom commands (separated by spaces), which can be executed directly;
//  2. systemd environment (/run/systemd/systemexists or INVOCATION_ID is set): execute systemctl restart <service name>,
//     The service name is taken from RUSTDESK_API_SYSTEMD_SERVICE, default rustdesk-api.service;
//  3. s6 environment (/run/s6-rc/servicedirs/apiexists): execute s6-svc -r <service directory>;
//  4. Back to the bottom: When there is no daemon process, the current binary file will be re-executed by itself.
//
// Since restarting will interrupt the current process, the interface will first return success and then perform the restart action after a delay of 1 second.
//
// @Tags ADMIN
// @Summary Restart the backend service
// @Description Restart the api-server process to make the modified configuration take effect. Only available to administrators
// @Produce json
// @Success 200 {object} response.Response
// @Router /admin/config/restart [post]
// @Security token
func (co *Config) ServiceRestart(c *gin.Context) {
	go func() {
		time.Sleep(1 * time.Second)
		doRestart()
	}()
	response.Success(c, nil)
}

// doRestart performs the actual restart logic
func doRestart() {
	// 1. Customize restart command
	if cmdStr := os.Getenv("RUSTDESK_API_RESTART_CMD"); cmdStr != "" {
		fields := strings.Fields(cmdStr)
		if len(fields) > 0 {
			_ = exec.Command(fields[0], fields[1:]...).Run()
			return
		}
	}

	// 2. systemd
	if isSystemd() {
		svc := os.Getenv("RUSTDESK_API_SYSTEMD_SERVICE")
		if svc == "" {
			svc = "rustdesk-api.service"
		}
		// Give priority to systemctl, and fall back to the service command if it fails.
		if err := exec.Command("systemctl", "restart", svc).Run(); err == nil {
			return
		}
		_ = exec.Command("service", strings.TrimSuffix(svc, ".service"), "restart").Run()
		return
	}

	// 3. s6
	if _, err := os.Stat("/run/s6-rc/servicedirs/api"); err == nil {
		_ = exec.Command("s6-svc", "-r", "/run/s6-rc/servicedirs/api").Run()
		return
	}

	// 4. Back to the bottom: Self-re-execution
	_ = selfExecRestart()
}

// isSystemd determines whether the current process is running under systemd management
func isSystemd() bool {
	if _, err := os.Stat("/run/systemd/system"); err == nil {
		return true
	}
	if os.Getenv("INVOCATION_ID") != "" {
		return true
	}
	return false
}

// selfExecRestart re-executes the current binary file when there is no daemon process
func selfExecRestart() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	argv := os.Args
	if len(argv) == 0 {
		argv = []string{exe}
	}
	cmd := exec.Command(exe, argv[1:]...)
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if wd, werr := os.Getwd(); werr == nil {
		cmd.Dir = wd
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	// Give the child process some time to take over and then exit the current process (there will be no duplicate processes when there is no daemon)
	time.Sleep(500 * time.Millisecond)
	global.Logger.Info("self restart, exiting old process")
	os.Exit(0)
	return nil
}
