package service

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ymg2006/rustdesk-api/v2/model"
)

type ProcessMonitorService struct{}

// applyOverrides applies single device override configuration to parent rule copy
func applyOverrides(r model.ProcessMonitorRule, ov map[string]interface{}) model.ProcessMonitorRule {
	if ov == nil {
		return r
	}
	if v, ok := ov["name"].(string); ok && v != "" {
		r.Name = v
	}
	if v, ok := ov["type"].(string); ok && (v == "process" || v == "port") {
		r.Type = v
	}
	if v, ok := ov["target"].(string); ok && v != "" {
		r.Target = v
	}
	if v, ok := ov["interval"].(float64); ok && v > 0 {
		r.Interval = int(v)
	}
	if v, ok := ov["down_threshold"].(float64); ok && v >= 0 {
		r.DownThreshold = int(v)
	}
	if v, ok := ov["alert_config_id"].(float64); ok && v >= 0 {
		r.AlertConfigId = uint(v)
	}
	if v, ok := ov["enabled"].(bool); ok {
		if v {
			r.Enabled = 1
		} else {
			r.Enabled = 0
		}
	} else if v, ok := ov["enabled"].(float64); ok {
		r.Enabled = int(v)
	}
	return r
}

// peerMatchesSource determines whether a device is currently covered by the set rules (device_group / ab_tags).
// Membership dynamics are taken from the device group/address book label to which the device currently belongs, to prevent the monitoring snapshot from being updated and continuing to alarm after the device is removed from the device group.
func (s *ProcessMonitorService) peerMatchesSource(peer *model.Peer, ab *model.AddressBook, rule *model.ProcessMonitorRule) bool {
	switch rule.SourceType {
	case "device_group":
		gid, err := strconv.ParseUint(rule.SourceId, 10, 64)
		if err != nil {
			return false
		}
		return peer.RowId > 0 && peer.GroupId == uint(gid)
	case "ab_tags":
		ruleTags := splitTagSet(rule.SourceId)
		if len(ruleTags) == 0 {
			return false
		}
		if ab == nil || ab.RowId == 0 {
			return false
		}
		var peerTags []string
		if len(ab.Tags) > 0 {
			_ = json.Unmarshal(ab.Tags, &peerTags)
		}
		for _, t := range peerTags {
			if _, ok := ruleTags[t]; ok {
				return true
			}
		}
		return false
	}
	return false
}

// splitTagSet parses a comma-separated string of tags into a set
func splitTagSet(s string) map[string]struct{} {
	set := make(map[string]struct{})
	for _, t := range strings.Split(s, ",") {
		t = strings.TrimSpace(t)
		if t != "" {
			set[t] = struct{}{}
		}
	}
	return set
}

// loadOverride reads the override configuration of a certain device on a certain set of rules (returns nil if it does not exist)
func (s *ProcessMonitorService) loadOverride(ruleId uint, peerId string) map[string]interface{} {
	var rp model.ProcessMonitorRulePeer
	DB.Where("rule_id = ? AND peer_id = ?", ruleId, peerId).First(&rp)
	if rp.RowId == 0 || len(rp.Overrides) == 0 {
		return nil
	}
	var ov map[string]interface{}
	if err := json.Unmarshal(rp.Overrides, &ov); err != nil {
		return nil
	}
	return ov
}

// findMatchingRule finds rules matching peer+type+target (single device first, followed by collection of rules)
// Dynamic analysis of the membership of collection rules (device_group/ab_tags): after the device is removed from the device group/tag, it no longer matches and the alarm stops.
func (s *ProcessMonitorService) findMatchingRule(peerId, typ, target string) *model.ProcessMonitorRule {
	// 1. Single device rules
	var single model.ProcessMonitorRule
	DB.Where("peer_id = ? AND type = ? AND target = ? AND enabled = ?", peerId, typ, target, 1).
		Where("source_type = ? OR source_type = ?", "peers", "").
		First(&single)
	if single.RowId > 0 {
		return &single
	}

	// 2. Collection rules: dynamically resolve membership relationships (no longer relies on ProcessMonitorRulePeer snapshot)
	peer := &model.Peer{}
	DB.Where("id = ?", peerId).First(peer)
	if peer.RowId == 0 {
		return &model.ProcessMonitorRule{}
	}
	ab := &model.AddressBook{}
	DB.Where("id = ?", peerId).First(ab)

	var groupRules []model.ProcessMonitorRule
	DB.Where("user_id = ? AND type = ? AND target = ? AND enabled = ? AND (source_type = ? OR source_type = ?)",
		peer.UserId, typ, target, 1, "device_group", "ab_tags").Find(&groupRules)
	for i := range groupRules {
		r := &groupRules[i]
		if !s.peerMatchesSource(peer, ab, r) {
			continue
		}
		rr := applyOverrides(*r, s.loadOverride(r.RowId, peerId))
		if rr.Enabled != 1 {
			// The device is excluded individually (override configuration is turned off)
			continue
		}
		return &rr
	}
	return &model.ProcessMonitorRule{}
}

// RulesByPeer returns the monitoring rules enabled for a device (including the expanded results of single device rules and collective rules)
// Collection rule membership is dynamically resolved, and the corresponding monitoring configuration will no longer be delivered after the device is removed from the device group/label.
func (s *ProcessMonitorService) RulesByPeer(peerId string) []model.ProcessMonitorRule {
	var rules []model.ProcessMonitorRule
	DB.Where("peer_id = ? AND enabled = ? AND (source_type = ? OR source_type = ?)", peerId, 1, "peers", "").
		Find(&rules)

	peer := &model.Peer{}
	DB.Where("id = ?", peerId).First(peer)
	if peer.RowId == 0 {
		return rules
	}
	ab := &model.AddressBook{}
	DB.Where("id = ?", peerId).First(ab)

	var groupRules []model.ProcessMonitorRule
	DB.Where("user_id = ? AND enabled = ? AND (source_type = ? OR source_type = ?)", peer.UserId, 1, "device_group", "ab_tags").
		Find(&groupRules)
	for i := range groupRules {
		r := &groupRules[i]
		if !s.peerMatchesSource(peer, ab, r) {
			continue
		}
		rr := applyOverrides(*r, s.loadOverride(r.RowId, peerId))
		if rr.Enabled != 1 {
			continue
		}
		rules = append(rules, rr)
	}
	return rules
}

// UpsertAndCheck writes the reporting status and determines whether to trigger the alarm according to the rules
func (s *ProcessMonitorService) UpsertAndCheck(peerId, name, typ, target string, running bool, now int64) {
	runningInt := 0
	if running {
		runningInt = 1
	}
	rule := s.findMatchingRule(peerId, typ, target)

	st := &model.ProcessMonitorStatus{}
	DB.Where("peer_id = ? AND type = ? AND target = ?", peerId, typ, target).First(st)
	if st.RowId == 0 {
		st = &model.ProcessMonitorStatus{PeerId: peerId, Type: typ, Target: target}
	}
	st.Name = name
	st.Type = typ
	st.Target = target
	st.Running = runningInt
	st.RuleId = rule.RowId
	if runningInt == 1 {
		st.LastSeen = now
		st.DownSince = 0
		st.Alerted = 0
	} else {
		if st.DownSince == 0 {
			st.DownSince = now
		}
	}
	DB.Save(st)

	// Triggering an alarm: down continues to exceed the threshold and no alarm has been sent yet
	if runningInt == 0 && rule.RowId > 0 && rule.AlertConfigId > 0 && rule.Enabled == 1 {
		if now-st.DownSince >= int64(rule.DownThreshold) && st.Alerted == 0 {
			s.fireAlert(rule, st)
			DB.Model(&model.ProcessMonitorStatus{}).Where("row_id = ?", st.RowId).Update("alerted", 1)
		}
	}
}

// fireAlert reuses existing alarm channels to send notifications
func (s *ProcessMonitorService) fireAlert(rule *model.ProcessMonitorRule, st *model.ProcessMonitorStatus) {
	cfg := &model.AlertConfig{}
	DB.Where("row_id = ?", rule.AlertConfigId).First(cfg)
	if cfg.RowId == 0 {
		return
	}
	typName := "process"
	if rule.Type == "port" {
		typName = "port"
	}
	title := fmt.Sprintf("Process monitoring alarm:%s", rule.Name)
	content := fmt.Sprintf("Device: %s\nMonitor: %s\nType: %s\nTarget: %s\nStatus: not running\nStart time: %s",
		st.PeerId, rule.Name, typName, rule.Target,
		time.Unix(st.DownSince, 0).Format("2006-01-02 15:04:05"))
	AllService.NotifyService.SendByConfig(cfg, title, content)
}
