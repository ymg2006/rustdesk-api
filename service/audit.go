package service

import (
	"fmt"
	"github.com/ymg2006/rustdesk-api/v2/global"
	"github.com/ymg2006/rustdesk-api/v2/model"
	"gorm.io/gorm"
	"strconv"
	"strings"
	"sync"
	"time"
)

type AuditService struct {
}

func (as *AuditService) AuditConnList(page, pageSize uint, where func(tx *gorm.DB)) (res *model.AuditConnList) {
	res = &model.AuditConnList{}
	res.Page = int64(page)
	res.PageSize = int64(pageSize)
	tx := DB.Model(&model.AuditConn{})
	if where != nil {
		where(tx)
	}
	tx.Count(&res.Total)
	tx.Scopes(Paginate(page, pageSize))
	tx.Find(&res.AuditConns)
	return
}

// Create
func (as *AuditService) CreateAuditConn(u *model.AuditConn) error {
	res := DB.Create(u).Error
	return res
}
func (as *AuditService) DeleteAuditConn(u *model.AuditConn) error {
	return DB.Delete(u).Error
}

// Update update
func (as *AuditService) UpdateAuditConn(u *model.AuditConn) error {
	return DB.Model(u).Updates(u).Error
}

// InfoByPeerIdAndConnId
func (as *AuditService) InfoByPeerIdAndConnId(peerId string, connId int64) (res *model.AuditConn) {
	res = &model.AuditConn{}
	DB.Where("peer_id = ? and conn_id = ?", peerId, connId).First(res)
	return
}

// UpsertByPeerIdAndConnId updates the non-zero field of an existing record by (peer_id, conn_id); creates it if it does not exist.
// When the client connection is established, new is sent first (without peer information), and then an update completion without action is sent after the authorization is successful.
// from_peer/from_name/session_id/type. The two are independent asynchronous HTTPs, and there is a race in arrival order:
// If the update arrives before new, the old logic discards the update because the record cannot be found, resulting in the final record missing the source name.
// After changing to upsert, regardless of the order of arrival, the two requests will hit the same record and merge their respective fields (Updates only writes non-zero fields,
// Existing information will not be overwritten with null values), completely eliminating race loss.
func (as *AuditService) UpsertByPeerIdAndConnId(u *model.AuditConn) error {
	ex := as.InfoByPeerIdAndConnId(u.PeerId, u.ConnId)
	if ex.Id != 0 {
		return DB.Model(&model.AuditConn{}).Where("id = ?", ex.Id).Updates(u).Error
	}
	return DB.Create(u).Error
}

// ConnInfoById
func (as *AuditService) ConnInfoById(id uint) (res *model.AuditConn) {
	res = &model.AuditConn{}
	DB.Where("id = ?", id).First(res)
	return
}

// FileInfoById
func (as *AuditService) FileInfoById(id uint) (res *model.AuditFile) {
	res = &model.AuditFile{}
	DB.Where("id = ?", id).First(res)
	return
}

func (as *AuditService) AuditFileList(page, pageSize uint, where func(tx *gorm.DB)) (res *model.AuditFileList) {
	res = &model.AuditFileList{}
	res.Page = int64(page)
	res.PageSize = int64(pageSize)
	tx := DB.Model(&model.AuditFile{})
	if where != nil {
		where(tx)
	}
	tx.Count(&res.Total)
	tx.Scopes(Paginate(page, pageSize))
	tx.Find(&res.AuditFiles)
	return
}

// CreateAuditFile
func (as *AuditService) CreateAuditFile(u *model.AuditFile) error {
	res := DB.Create(u).Error
	return res
}
func (as *AuditService) DeleteAuditFile(u *model.AuditFile) error {
	return DB.Delete(u).Error
}

// Update update
func (as *AuditService) UpdateAuditFile(u *model.AuditFile) error {
	return DB.Model(u).Updates(u).Error
}

func (as *AuditService) BatchDeleteAuditConn(ids []uint) error {
	return DB.Where("id in (?)", ids).Delete(&model.AuditConn{}).Error
}

// StartStaleConnCloseSweep periodically cleans up "in-progress" orphan connection audit records in the background.
// Conditions that trigger shutdown (just one of them is met):
//  1. The peer device has been offline for more than 5 minutes (crashes/network interruptions/processes are killed, etc.);
//  2. It has been more than 24 hours since the record was created (to cover the scenario where the controller is upgraded and replaced and the close event is not received).
//
// This prevents the old client from not sending a close audit when exiting, causing the "Recent Connection Record" on the home page to be permanently stuck in "In Progress".
// This operation is idempotent and is safe for multi-instance deployment.
func (as *AuditService) StartStaleConnCloseSweep() {
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()
	as.closeStaleConns()
	for range ticker.C {
		as.closeStaleConns()
	}
}

func (as *AuditService) closeStaleConns() {
	now := time.Now()
	offlineThreshold := now.Unix() - 300        // The peer is offline for more than 5 minutes
	maxAgeThreshold := now.Add(-24 * time.Hour) // Logging for more than 24 hours
	sub := DB.Model(&model.Peer{}).
		Select("id").
		Where("last_online_time <= ? OR last_online_time = 0", offlineThreshold)
	res := DB.Model(&model.AuditConn{}).
		Where("close_time = 0").
		Where("(peer_id IN (?)) OR (created_at < ?)", sub, maxAgeThreshold).
		Update("close_time", now.Unix())
	if res.Error != nil {
		global.Logger.Warn("closeStaleConns", res.Error)
		return
	}
	if res.RowsAffected > 0 {
		global.Logger.Infof("closeStaleConns: closed %d stale 'in-progress' audit_conn records", res.RowsAffected)
	}
}

// ========== Connect Heartbeat Tracking ==========
// connHeartbeats records the most recent heartbeat time for each active connection.
// key = "peerId:connId", value = time.Time
var connHeartbeats sync.Map

// RecordConnHeartbeat records the heartbeat time of a batch of connections.
// Called in the Heartbeat controller, passing in the list of active connections reported by the client's heartbeat.
func (as *AuditService) RecordConnHeartbeat(peerId string, connIds []int) {
	now := time.Now()
	for _, cid := range connIds {
		connHeartbeats.Store(fmt.Sprintf("%s:%d", peerId, cid), now)
	}
}

// StartConnHeartbeatSweep periodically clears the connection audit records of heartbeat timeouts in the background.
// If there is no heartbeat update for an active connection within 60 seconds (the client disconnects abnormally), audit logging is turned off.
// Coexisting with StartStaleConnCloseSweep: the former is faster (60s) and more granular (per connection);
// The latter is more conservative (5 minutes) and takes device-level offline.
func (as *AuditService) StartConnHeartbeatSweep() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	as.closeStaleConnsByHeartbeat()
	for range ticker.C {
		as.closeStaleConnsByHeartbeat()
	}
}

func (as *AuditService) closeStaleConnsByHeartbeat() {
	now := time.Now()
	threshold := now.Add(-60 * time.Second)
	var toClose []string // peerId:connId format
	connHeartbeats.Range(func(key, value interface{}) bool {
		if lastBeat, ok := value.(time.Time); ok {
			if lastBeat.Before(threshold) {
				toClose = append(toClose, key.(string))
			}
		}
		return true
	})
	if len(toClose) == 0 {
		return
	}
	// Traverse the records to be closed
	var closedCount int64
	for _, key := range toClose {
		parts := strings.SplitN(key, ":", 2)
		if len(parts) != 2 {
			connHeartbeats.Delete(key)
			continue
		}
		peerId, connIdStr := parts[0], parts[1]
		connId, err := strconv.ParseInt(connIdStr, 10, 64)
		if err != nil {
			connHeartbeats.Delete(key)
			continue
		}
		// Check both directions of peer_id (controlled terminal) and from_peer (control terminal) at the same time:
		// The controlled terminal disconnects the heartbeat → close; the controlled terminal disconnects the heartbeat and should be closed even if the controlled terminal is still alive.
		res := DB.Model(&model.AuditConn{}).
			Where("close_time = 0").
			Where("(peer_id = ? AND conn_id = ?) OR (from_peer = ? AND conn_id = ?)", peerId, connId, peerId, connId).
			Update("close_time", now.Unix())
		if res.Error == nil {
			closedCount += res.RowsAffected
		}
		connHeartbeats.Delete(key)
	}
	if closedCount > 0 {
		global.Logger.Infof("connHeartbeatSweep: closed %d stale connections (heartbeat timeout)", closedCount)
	}
}

func (as *AuditService) BatchDeleteAuditFile(ids []uint) error {
	return DB.Where("id in (?)", ids).Delete(&model.AuditFile{}).Error
}

// CloseInProgressByFromPeerAndPeer closes the audit record of the connection of the same user (FromPeer) to the same peer (PeerId) that is still "in progress" (close_time=0).
// Used to solve the problem of abnormal disconnection (the client did not send a close audit) causing the "Recent Connection Record" to always display "In Progress":
// When the user initiates a connection (new) to the same peer again, the previous record that is still in progress is set to closed, and the status is reset.
// This operation is idempotent and is safe for multi-instance deployment.
func (as *AuditService) CloseInProgressByFromPeerAndPeer(fromPeer, peerId string) {
	if fromPeer == "" || peerId == "" {
		return
	}
	now := time.Now().Unix()
	res := DB.Model(&model.AuditConn{}).
		Where("close_time = 0 AND from_peer = ? AND peer_id = ?", fromPeer, peerId).
		Update("close_time", now)
	if res.Error != nil {
		global.Logger.Warn("CloseInProgressByFromPeerAndPeer", res.Error)
		return
	}
	if res.RowsAffected > 0 {
		global.Logger.Infof("CloseInProgressByFromPeerAndPeer: closed %d in-progress audit_conn for %s -> %s", res.RowsAffected, fromPeer, peerId)
	}
}
