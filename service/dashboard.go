package service

import (
	"github.com/ymg2006/rustdesk-api/v2/model"
	"time"
)

type DashboardStats struct {
	TotalPeers       int64 `json:"total_peers"`
	OnlinePeers      int64 `json:"online_peers"`
	OfflinePeers     int64 `json:"offline_peers"`
	TotalUsers       int64 `json:"total_users"`
	TodayConnections int64 `json:"today_connections"`
}

type DashboardService struct{}

func (s *DashboardService) Stats() *DashboardStats {
	stats := &DashboardStats{}
	now := time.Now().Unix()

	DB.Model(&model.Peer{}).Count(&stats.TotalPeers)
	DB.Model(&model.Peer{}).Where("last_online_time > ?", now-300).Count(&stats.OnlinePeers)
	DB.Model(&model.Peer{}).Where("last_online_time <= ? OR last_online_time = 0", now-300).Count(&stats.OfflinePeers)
	DB.Model(&model.User{}).Count(&stats.TotalUsers)

	nowTime := time.Now()
	todayStart := time.Date(nowTime.Year(), nowTime.Month(), nowTime.Day(), 0, 0, 0, 0, nowTime.Location())
	DB.Model(&model.AuditConn{}).Where("action = 'new' and created_at > ?", todayStart).Count(&stats.TodayConnections)

	return stats
}

// UserStats returns device statistics for the specified user
func (s *DashboardService) UserStats(userId uint) *DashboardStats {
	stats := &DashboardStats{}
	now := time.Now().Unix()

	DB.Model(&model.Peer{}).Where("user_id = ?", userId).Count(&stats.TotalPeers)
	DB.Model(&model.Peer{}).Where("user_id = ? and last_online_time > ?", userId, now-300).Count(&stats.OnlinePeers)
	DB.Model(&model.Peer{}).Where("user_id = ? and (last_online_time <= ? OR last_online_time = 0)", userId, now-300).Count(&stats.OfflinePeers)
	// Ordinary users do not display the total number of users
	stats.TotalUsers = 0
	nowTime := time.Now()
	todayStart := time.Date(nowTime.Year(), nowTime.Month(), nowTime.Day(), 0, 0, 0, 0, nowTime.Location())
	DB.Model(&model.AuditConn{}).Where("action = 'new' and created_at > ?", todayStart).Count(&stats.TodayConnections)

	return stats
}
