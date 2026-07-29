package service

import (
	"net"
	"strconv"
	"time"

	"github.com/ymg2006/rustdesk-api/v2/model"
)

type ServerStatusService struct{}

// ListAll returns server probe entries shared by all administrators
func (s *ServerStatusService) ListAll() []model.ServerStatusMonitor {
	var list []model.ServerStatusMonitor
	DB.Where("enabled = ?", 1).Order("row_id").Find(&list)
	return list
}

func (s *ServerStatusService) Create(m *model.ServerStatusMonitor) error {
	return DB.Create(m).Error
}

func (s *ServerStatusService) Update(m *model.ServerStatusMonitor) error {
	updates := map[string]interface{}{
		"name":     m.Name,
		"host":     m.Host,
		"port":     m.Port,
		"protocol": m.Protocol,
		"enabled":  m.Enabled,
	}
	return DB.Model(&model.ServerStatusMonitor{}).Where("row_id = ?", m.RowId).Updates(updates).Error
}

func (s *ServerStatusService) Delete(id uint) error {
	DB.Where("row_id = ?", id).Delete(&model.ServerStatusMonitor{})
	return nil
}

// ProbeResult Single detection result
type ProbeResult struct {
	RowId     uint   `json:"row_id"`
	Name      string `json:"name"`
	Host      string `json:"host"`
	Port      int    `json:"port"`
	Addr      string `json:"addr"`
	Status    string `json:"status"` // up | down
	LatencyMs int64  `json:"latency_ms"`
	Error     string `json:"error"`
}

// Probe performs TCP connectivity detection on a single entry
func (s *ServerStatusService) Probe(m model.ServerStatusMonitor) ProbeResult {
	addr := m.Host
	if m.Port > 0 {
		addr = net.JoinHostPort(m.Host, strconv.Itoa(m.Port))
	}
	res := ProbeResult{RowId: m.RowId, Name: m.Name, Host: m.Host, Port: m.Port, Addr: addr}
	start := time.Now()
	conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil {
		res.Status = "down"
		res.Error = err.Error()
		return res
	}
	_ = conn.Close()
	res.Status = "up"
	res.LatencyMs = time.Since(start).Milliseconds()
	return res
}

// ProbeAll Probes all enabled entries (shared by administrator)
func (s *ServerStatusService) ProbeAll() []ProbeResult {
	list := s.ListAll()
	results := make([]ProbeResult, 0, len(list))
	for _, m := range list {
		results = append(results, s.Probe(m))
	}
	return results
}
