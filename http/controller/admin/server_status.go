package admin

import (
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ymg2006/rustdesk-api/v2/global"
	"github.com/ymg2006/rustdesk-api/v2/http/response"
	"github.com/ymg2006/rustdesk-api/v2/model"
	"github.com/ymg2006/rustdesk-api/v2/service"
)

type ServerStatus struct {
}

// Status detects the connectivity of all server entries created by the current user and returns the hbbr payload (only available on the same machine)
// @Tags system
// @Summary Server status
// @Description Perform TCP detection on the server entries added by the user in the page, and return connectivity/delay; at the same time, try to query the hbbr load (optional)
// @Produce  json
// @Success 200 {object} response.Response
// @Router /server_status [get]
// @Security token
func (ct *ServerStatus) Status(c *gin.Context) {
	results := service.AllService.ServerStatusService.ProbeAll()
	response.Success(c, gin.H{
		"list":       results,
		"hbbr_stats": hbbrStats(),
	})
}

// List Lists server probe entries shared by all administrators
func (ct *ServerStatus) List(c *gin.Context) {
	list := service.AllService.ServerStatusService.ListAll()
	response.Success(c, gin.H{"list": list})
}

// Create a new server detection entry (shared by the administrator)
func (ct *ServerStatus) Create(c *gin.Context) {
	f := &model.ServerStatusMonitor{}
	if err := c.ShouldBindJSON(f); err != nil || f.Host == "" || f.Name == "" {
		response.Fail(c, 101, response.TranslateParamMsg(c, "RequiredFields", "name, host"))
		return
	}
	if f.Protocol != "tcp" {
		f.Protocol = "tcp"
	}
	if f.Port < 0 || f.Port > 65535 {
		f.Port = 0
	}
	if f.Enabled != 0 {
		f.Enabled = 1
	}
	f.UserId = 0 // 0 means administrator sharing
	if err := service.AllService.ServerStatusService.Create(f); err != nil {
		response.Fail(c, 500, "Save failed:"+err.Error())
		return
	}
	response.Success(c, f)
}

// Update updates server probe entries (shared by administrators)
func (ct *ServerStatus) Update(c *gin.Context) {
	f := &model.ServerStatusMonitor{}
	if err := c.ShouldBindJSON(f); err != nil || f.RowId == 0 {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamError"))
		return
	}
	if f.Protocol != "tcp" {
		f.Protocol = "tcp"
	}
	if f.Port < 0 || f.Port > 65535 {
		f.Port = 0
	}
	if f.Enabled != 0 {
		f.Enabled = 1
	}
	if err := service.AllService.ServerStatusService.Update(f); err != nil {
		response.Fail(c, 500, "Save failed:"+err.Error())
		return
	}
	response.Success(c, nil)
}

// Delete Delete server probe entry (shared by administrator)
func (ct *ServerStatus) Delete(c *gin.Context) {
	form := &struct {
		Id uint `json:"id"`
	}{}
	if err := c.ShouldBindJSON(form); err != nil || form.Id == 0 {
		response.Fail(c, 101, response.TranslateMsg(c, "IdRequired"))
		return
	}
	service.AllService.ServerStatusService.Delete(form.Id)
	response.Success(c, nil)
}

// usageEntry single hbbr relay connection statistics
type usageEntry struct {
	IP          string  `json:"ip"`
	Seconds     int64   `json:"seconds"`
	TrafficMB   float64 `json:"traffic_mb"`
	HighestKbps int64   `json:"highest_kbps"`
	AvgKbps     int64   `json:"avg_kbps"`
	SpeedKbps   int64   `json:"speed_kbps"`
}

// usageRe matches every line output by the hbbr`usage`command
var usageRe = regexp.MustCompile(`: (\d+)s ([\d.]+)MB (\d+)kb/s (\d+)kb/s (\d+)kb/s$`)

// hbbrStats connects to the hbbr loopback command port, sends the`u`(usage) command, and analyzes the number of relay connections and load.
// Note: hbbr only accepts command connections from the local machine (loopback), so api-server must be deployed on the same host as hbbr.
func hbbrStats() gin.H {
	host := global.Config.Admin.RelayStatsHost
	if host == "" {
		host = "127.0.0.1"
	}
	port := global.Config.Admin.RelayServerPort
	if port == 0 {
		port = 21117
	}
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil {
		return gin.H{
			"available": false,
			"host":      addr,
			"error":     err.Error(),
		}
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(3 * time.Second))
	if _, err := conn.Write([]byte("u\n")); err != nil {
		return gin.H{
			"available": false,
			"host":      addr,
			"error":     err.Error(),
		}
	}
	buf := make([]byte, 65536)
	var sb strings.Builder
	for {
		n, err := conn.Read(buf)
		if n > 0 {
			sb.Write(buf[:n])
		}
		if err != nil {
			break
		}
	}
	entries := parseUsage(sb.String())
	var totalMB float64
	var curSpeed, highestSpeed, connCount int64
	for _, e := range entries {
		totalMB += e.TrafficMB
		curSpeed += e.SpeedKbps
		if e.HighestKbps > highestSpeed {
			highestSpeed = e.HighestKbps
		}
		connCount++
	}
	return gin.H{
		"available":          true,
		"host":               addr,
		"connection_count":   connCount,
		"total_traffic_mb":   totalMB,
		"current_speed_kbps": curSpeed,
		"highest_speed_kbps": highestSpeed,
		"connections":        entries,
	}
}

// parseUsage parses hbbr usage command output text
func parseUsage(out string) []usageEntry {
	var entries []usageEntry
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		idx := strings.Index(line, ": ")
		if idx < 0 {
			continue
		}
		ip := line[:idx]
		rest := line[idx+2:]
		m := usageRe.FindStringSubmatch(rest)
		if m == nil {
			continue
		}
		secs, _ := strconv.ParseInt(m[1], 10, 64)
		mb, _ := strconv.ParseFloat(m[2], 64)
		highest, _ := strconv.ParseInt(m[3], 10, 64)
		avg, _ := strconv.ParseInt(m[4], 10, 64)
		speed, _ := strconv.ParseInt(m[5], 10, 64)
		entries = append(entries, usageEntry{
			IP:          ip,
			Seconds:     secs,
			TrafficMB:   mb,
			HighestKbps: highest,
			AvgKbps:     avg,
			SpeedKbps:   speed,
		})
	}
	return entries
}
