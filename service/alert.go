package service

import (
	"encoding/json"
	"fmt"
	"github.com/ymg2006/rustdesk-api/v2/model"
	"time"
)

// peerNotifyRecord Notification record for each peer (used in the NotifiedPeers JSON field)
type peerNotifyRecord struct {
	Count     int   `json:"c"` // Number of notifications on the day
	LastTime  int64 `json:"t"` // Last notification timestamp
	Weight    int   `json:"w"` // Offline weight: Detected every 5 minutes, +1 if offline
	WeightDay int   `json:"d"` // The date the weight belongs to (YYYYMMDD), used for daily reset
}

// parseNotifiedPeers is compatible with parsing NotifiedPeers in both old and new formats.
func parseNotifiedPeers(data string) map[string]*peerNotifyRecord {
	result := make(map[string]*peerNotifyRecord)
	if data == "" {
		return result
	}
	// Try new format map[string]*peerNotifyRecord
	m := make(map[string]*peerNotifyRecord)
	if err := json.Unmarshal([]byte(data), &m); err == nil {
		for k, v := range m {
			result[k] = v
		}
		return result
	}
	// Compatible with old format map[string]int64
	old := make(map[string]int64)
	if err := json.Unmarshal([]byte(data), &old); err == nil {
		for k, v := range old {
			result[k] = &peerNotifyRecord{Count: 1, LastTime: v}
		}
	}
	return result
}

// isSameDay determines whether two timestamps are on the same day (local time)
func isSameDay(a, b int64) bool {
	ta := time.Unix(a, 0)
	tb := time.Unix(b, 0)
	return ta.Year() == tb.Year() && ta.YearDay() == tb.YearDay()
}

// dayKey returns the integer representation of the local date (YYYYMMDD), used for daily reset weight and consecutive day calculations
func dayKey(t int64) int {
	tm := time.Unix(t, 0)
	return tm.Year()*10000 + int(tm.Month())*100 + tm.Day()
}

const (
	maxNotifyPerDay        = 3               // Maximum number of notifications per day for the same device
	notifyCooldown         = 600             // The minimum notification interval (seconds) of a single peer to avoid repeated triggering in a short period of time
	offlineWeightThreshold = 10              // The alarm will only be triggered when the offline weight reaches this value (10 times × 5 minutes = 50 minutes)
	checkInterval          = 5 * time.Minute // Offline detection interval
)

type AlertService struct{}

func (s *AlertService) StartChecker() {
	AllService.AlertService = s
	go func() {
		for {
			s.checkOfflineDevices()
			time.Sleep(checkInterval)
		}
	}()
	Logger.Info("Alert checker started")
}

// getMonitoredPeerIds returns the list of device IDs that this alarm configuration should monitor.
// MonitorAll=1: Monitor all devices in the user's address book
// MonitorAll=2: Only monitor the devices or collections selected in alert_targets
func (s *AlertService) getMonitoredPeerIds(cfg *model.AlertConfig) ([]string, bool) {
	if cfg.MonitorAll == 1 {
		// User's own address book
		var abEntries []model.AddressBook

		var ownColls []model.AddressBookCollection
		DB.Where("user_id = ?", cfg.UserId).Find(&ownColls)
		ownCollIds := []uint{0}
		for _, col := range ownColls {
			ownCollIds = append(ownCollIds, col.Id)
		}

		// Collections shared by others with this user
		user := &model.User{}
		DB.First(user, cfg.UserId)
		if user.Id > 0 {
			var rules []model.AddressBookCollectionRule
			ruleQuery := DB.Where("type = ? AND to_id = ?",
				model.ShareAddressBookRuleTypePersonal, user.Id)
			if user.GroupId > 0 {
				ruleQuery = DB.Where(
					"(type = ? AND to_id = ?) OR (type = ? AND to_id = ?)",
					model.ShareAddressBookRuleTypePersonal, user.Id,
					model.ShareAddressBookRuleTypeGroup, user.GroupId,
				)
			}
			ruleQuery.Find(&rules)
			for _, rule := range rules {
				ownCollIds = append(ownCollIds, rule.CollectionId)
			}
		}

		DB.Where("collection_id in (?)", ownCollIds).Find(&abEntries)
		if len(abEntries) == 0 {
			return nil, true
		}
		var peerIds []string
		for _, ab := range abEntries {
			peerIds = append(peerIds, ab.Id)
		}
		return peerIds, false
	}

	var targets []model.AlertTarget
	DB.Where("alert_id = ?", cfg.RowId).Find(&targets)
	if len(targets) == 0 {
		return s.getMonitoredPeerIds(&model.AlertConfig{
			MonitorAll: 1,
			UserId:     cfg.UserId,
		})
	}

	var peerIds []string
	for _, t := range targets {
		if t.TargetType == "peer" {
			peerIds = append(peerIds, t.TargetId)
		} else if t.TargetType == "collection" {
			var abEntries []model.AddressBook
			DB.Where("collection_id = ?", t.TargetId).Find(&abEntries)
			for _, ab := range abEntries {
				peerIds = append(peerIds, ab.Id)
			}
		}
	}
	return peerIds, false
}

func (s *AlertService) checkOfflineDevices() {
	var configs []model.AlertConfig
	DB.Where("enabled = 1 AND user_id > 0").Find(&configs)
	if len(configs) == 0 {
		return
	}

	now := time.Now().Unix()
	today := dayKey(now)
	prevDay := dayKey(now - 86400)

	// Processing by user group: each user’s in-site message configuration (if it exists)
	userStationCfg := make(map[uint]*model.AlertConfig)
	for i := range configs {
		if configs[i].Channel == "station" {
			userStationCfg[configs[i].UserId] = &configs[i]
		}
	}

	for _, cfg := range configs {
		if cfg.Channel == "station" {
			continue
		}
		threshold := int64(cfg.OfflineMin * 60)
		if threshold <= 0 {
			threshold = 300
		}

		peerIds, monitorAll := s.getMonitoredPeerIds(&cfg)

		// Load online information of monitored devices (excluding devices that have not been online for more than 30 days)
		var peers []model.Peer
		q := DB.Select("id, last_online_time, hostname, alias").
			Where("last_online_time > ?", now-30*86400)
		if !monitorAll && len(peerIds) > 0 {
			q = q.Where("id in (?)", peerIds)
		} else if !monitorAll {
			continue
		}
		q.Find(&peers)
		if len(peers) == 0 {
			continue
		}

		notifiedMap := parseNotifiedPeers(cfg.NotifiedPeers)

		// Re-online detection: If any monitored device comes online recently, the "3 consecutive days offline" limit will be reset.
		for _, peer := range peers {
			if peer.LastOnlineTime > now-300 {
				if cfg.ConsecutiveTriggerDays != 0 || cfg.LastTriggerDay != 0 {
					cfg.ConsecutiveTriggerDays = 0
					cfg.LastTriggerDay = 0
				}
				break
			}
		}

		// Update offline weight: detected every 5 minutes, +1 if offline; reset daily/online
		type candidate struct {
			peer   model.Peer
			weight int
		}
		var candidates []candidate
		for _, peer := range peers {
			rec, ok := notifiedMap[peer.Id]
			if !ok {
				rec = &peerNotifyRecord{}
			}
			// Daily weight reset
			if rec.WeightDay != today {
				rec.Weight = 0
				rec.WeightDay = today
			}
			switch {
			case peer.LastOnlineTime > now-300:
				// Already online: Reset weights
				rec.Weight = 0
				rec.WeightDay = today
			case peer.LastOnlineTime < now-threshold:
				// Offline (exceeds threshold time): Weight +1
				// If the device is offline before the alarm is created, this offline event will be skipped (the history will not be traced).
				if cfg.CreatedAt > 0 && peer.LastOnlineTime < cfg.CreatedAt {
					break
				}
				rec.Weight++
			default:
				// Just offline but not yet exceeded the threshold: not counted in the weight
				rec.Weight = 0
				rec.WeightDay = today
			}
			notifiedMap[peer.Id] = rec
			if rec.Weight >= offlineWeightThreshold {
				candidates = append(candidates, candidate{peer: peer, weight: rec.Weight})
			}
		}

		// Filter the devices that can be notified (exclude those that have reached the upper limit or are within the cooling period on that day)
		var alertPeers []candidate
		for _, c := range candidates {
			rec := notifiedMap[c.peer.Id]
			if isSameDay(rec.LastTime, now) && rec.Count >= maxNotifyPerDay {
				continue
			}
			if now-rec.LastTime < notifyCooldown {
				continue
			}
			alertPeers = append(alertPeers, c)
		}

		// If triggered for more than 3 consecutive days, no further email alerts will be sent.
		if cfg.ConsecutiveTriggerDays >= 3 {
			Logger.Infof("alert config %d: consecutive offline trigger days >= 3, skip email push", cfg.RowId)
			s.persistAlertCfg(cfg.RowId, notifiedMap, cfg.LastNotifiedAt, cfg.ConsecutiveTriggerDays, cfg.LastTriggerDay)
			continue
		}

		if len(alertPeers) == 0 {
			s.persistAlertCfg(cfg.RowId, notifiedMap, cfg.LastNotifiedAt, cfg.ConsecutiveTriggerDays, cfg.LastTriggerDay)
			continue
		}

		pushedAny := false
		for _, c := range alertPeers {
			peer := c.peer
			hostname := peer.Hostname
			if hostname == "" {
				hostname = peer.Id
			}
			alias := peer.Alias
			if alias == "" {
				alias = hostname
			}
			lastOnline := time.Unix(peer.LastOnlineTime, 0).Format("2006-01-02 15:04:05")
			title := "Device offline alarm"
			offlineMinutes := (now - peer.LastOnlineTime) / 60
			content := fmt.Sprintf("Device: %s\nAlias: %s\nID: %s\nOffline duration: %d minutes\nLast online: %s\nOffline weight: %d",
				hostname, alias, peer.Id, offlineMinutes, lastOnline, c.weight)

			// Send external channel notifications (email, etc.)
			AllService.NotifyService.SendByConfig(&cfg, title, content)

			// Does this user have in-site message configuration? If there is any, send a message on the site
			if stationCfg, ok := userStationCfg[cfg.UserId]; ok && stationCfg != nil {
				AllService.NotifyService.SendStationMessage(cfg.UserId, title, content, peer.Id)
			}

			// Update the peer's notification record
			rec := notifiedMap[peer.Id]
			if isSameDay(rec.LastTime, now) {
				rec.Count++
				rec.LastTime = now
			} else {
				rec.Count = 1
				rec.LastTime = now
			}
			notifiedMap[peer.Id] = rec
			pushedAny = true
		}

		// Clean up records that are not today (based on the date the weight belongs to) to avoid unlimited growth of the map.
		// At the same time, records of offline devices that are still accumulating weights on that day are retained.
		for k, v := range notifiedMap {
			if v.WeightDay != today {
				delete(notifiedMap, k)
			}
		}

		// Update the number of consecutive trigger days (only counted once per calendar day)
		if pushedAny {
			if cfg.LastTriggerDay == today {
				// Already counted on that day, no repeated accumulation
			} else if cfg.LastTriggerDay == prevDay {
				cfg.ConsecutiveTriggerDays++
			} else {
				cfg.ConsecutiveTriggerDays = 1
			}
			cfg.LastTriggerDay = today
		}

		s.persistAlertCfg(cfg.RowId, notifiedMap, now, cfg.ConsecutiveTriggerDays, cfg.LastTriggerDay)
	}
}

// persistAlertCfg persists the running status of the alarm configuration (notification records, number of consecutive trigger days, etc.)
func (s *AlertService) persistAlertCfg(rowId uint, notifiedMap map[string]*peerNotifyRecord, lastNotifiedAt int64, consecutive int, lastTriggerDay int) {
	encoded, _ := json.Marshal(notifiedMap)
	DB.Model(&model.AlertConfig{}).Where("row_id = ?", rowId).Updates(map[string]interface{}{
		"last_notified_at":         lastNotifiedAt,
		"notified_peers":           string(encoded),
		"consecutive_trigger_days": consecutive,
		"last_trigger_day":         lastTriggerDay,
	})
}
