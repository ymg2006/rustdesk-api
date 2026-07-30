package service

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/sirupsen/logrus"
	"github.com/ymg2006/rustdesk-api/v2/model"
	"gorm.io/gorm"
)

// setupAlertTestDB Initializes in-memory SQLite with global dependencies and returns db.
// Each test builds a database independently without interfering with each other; after the test is completed, global variables are restored to avoid contaminating other tests in the same package.
func setupAlertTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.SetMaxOpenConns(1) // The memory library needs a single connection to avoid each connection becoming an independent library.
	}
	if err := db.AutoMigrate(&model.Peer{}, &model.AlertConfig{}, &model.AlertTarget{}, &model.StationMessage{}, &model.AlertChannel{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	oldDB, oldLog, oldAll := DB, Logger, AllService
	DB = db
	Logger = logrus.New()
	AllService = &Service{}
	AllService.AlertService = &AlertService{}
	AllService.NotifyService = &NotifyService{}
	t.Cleanup(func() {
		DB, Logger, AllService = oldDB, oldLog, oldAll
	})
	return db
}

type seedOpts struct {
	offlineMin     int
	peerLastOnline int64 // Absolute timestamp; 0 means "offline exceeded the threshold" calculated by offlineMin
	consecutive    int   // Number of consecutive trigger days (preset)
	lastTriggerDay int   // Last trigger date (preset)
	notifiedPeers  string
	enabled        int
}

// seedScenario structure: an offline device + an smtp alarm rule (pointing to the device) + a station rule (same as user, used to count pushes).
// Returns the primary key and device id of the smtp configuration.
func seedScenario(t *testing.T, db *gorm.DB, o seedOpts) (uint, string) {
	peerId := "peer-1"
	lastOnline := o.peerLastOnline
	if lastOnline == 0 {
		lastOnline = time.Now().Unix() - int64(o.offlineMin)*60 - 600 // Far exceeds the threshold and is determined to be offline.
	}
	if o.enabled == 0 {
		o.enabled = 1
	}
	peer := model.Peer{Id: peerId, Hostname: "host-1", Alias: "alias-1", UserId: 1, LastOnlineTime: lastOnline}
	if err := db.Create(&peer).Error; err != nil {
		t.Fatalf("create peer: %v", err)
	}

	cfg := model.AlertConfig{
		UserId:                 1,
		Channel:                "smtp",
		ChannelId:              0, // No channel line -> SendByConfig safe no-op, no network will be initiated
		Name:                   "smtp-cfg",
		OfflineMin:             o.offlineMin,
		Enabled:                o.enabled,
		MonitorAll:             2,
		ConsecutiveTriggerDays: o.consecutive,
		LastTriggerDay:         o.lastTriggerDay,
		NotifiedPeers:          o.notifiedPeers,
	}
	if err := db.Create(&cfg).Error; err != nil {
		t.Fatalf("create cfg: %v", err)
	}
	if err := db.Create(&model.AlertTarget{AlertId: cfg.RowId, TargetType: "peer", TargetId: peerId, TargetName: peerId}).Error; err != nil {
		t.Fatalf("create target: %v", err)
	}
	// Same as user station rules: only used for populate userStationCfg, so that station_messages will be written to count when pushing
	if err := db.Create(&model.AlertConfig{UserId: 1, Channel: "station", ChannelId: 0, Name: "station-cfg", OfflineMin: o.offlineMin, Enabled: 1, MonitorAll: 2}).Error; err != nil {
		t.Fatalf("create station cfg: %v", err)
	}
	return cfg.RowId, peerId
}

func npJSON(weight, weightDay, count int, lastTime int64) string {
	m := map[string]*peerNotifyRecord{
		"peer-1": {Count: count, LastTime: lastTime, Weight: weight, WeightDay: weightDay},
	}
	b, _ := json.Marshal(m)
	return string(b)
}

func countStationMessages(t *testing.T, db *gorm.DB) int64 {
	var n int64
	if err := db.Model(&model.StationMessage{}).Count(&n).Error; err != nil {
		t.Fatalf("count station messages: %v", err)
	}
	return n
}

func getCfg(t *testing.T, db *gorm.DB, rowId uint) model.AlertConfig {
	var c model.AlertConfig
	if err := db.Where("row_id = ?", rowId).First(&c).Error; err != nil {
		t.Fatalf("get cfg: %v", err)
	}
	return c
}

// TestAlert_WeightAccumulation verification: detected every 5 minutes, if offline, the weight will be +1.
// A push is triggered only when the weight accumulates to 10 (then a cool-down period prevents repeated push).
func TestAlert_WeightAccumulation(t *testing.T) {
	db := setupAlertTestDB(t)
	cfgRowId, _ := seedScenario(t, db, seedOpts{offlineMin: 5})

	for i := 0; i < 11; i++ {
		AllService.AlertService.checkOfflineDevices()
	}

	if got := countStationMessages(t, db); got != 1 {
		t.Fatalf("When the weight reaches 10, it should be pushed once, but actually%dtimes", got)
	}
	rec := parseNotifiedPeers(getCfg(t, db, cfgRowId).NotifiedPeers)["peer-1"]
	if rec == nil {
		t.Fatalf("No notification record found for peer-1")
	}
	if rec.Weight < offlineWeightThreshold {
		t.Fatalf("Weight should be >=%d, actual%d", offlineWeightThreshold, rec.Weight)
	}
}

// TestAlert_ThresholdNotReachedNoPush Verification: No pushing if the weight does not reach the threshold.
func TestAlert_ThresholdNotReachedNoPush(t *testing.T) {
	db := setupAlertTestDB(t)
	seedScenario(t, db, seedOpts{offlineMin: 5})
	// Only detected 9 times, the weight is up to 9, less than 10
	for i := 0; i < 9; i++ {
		AllService.AlertService.checkOfflineDevices()
	}
	if got := countStationMessages(t, db); got != 0 {
		t.Fatalf("The weight does not reach the threshold and should not be pushed. Actual%dtimes", got)
	}
}

// TestAlert_DailyWeightReset Verification: Reset weight across days (If WeightDay is different from today, it will be reset to zero and then accumulated again).
func TestAlert_DailyWeightReset(t *testing.T) {
	db := setupAlertTestDB(t)
	today := dayKey(time.Now().Unix())
	yesterday := dayKey(time.Now().Unix() - 86400)
	cfgRowId, _ := seedScenario(t, db, seedOpts{
		offlineMin:    5,
		notifiedPeers: npJSON(5, yesterday, 0, 0), // Yesterday's accumulated weight 5
	})
	AllService.AlertService.checkOfflineDevices()

	rec := parseNotifiedPeers(getCfg(t, db, cfgRowId).NotifiedPeers)["peer-1"]
	if rec == nil {
		t.Fatalf("Notification record not found")
	}
	if rec.Weight != 1 {
		t.Fatalf("The weight should be 1 after daily reset (+1 this time after zeroing), actual%d", rec.Weight)
	}
	if rec.WeightDay != today {
		t.Fatalf("WeightDay should be%dtoday, actual%d", today, rec.WeightDay)
	}
}

// TestAlert_OnlineResetsWeight verification: The device resets its offline weight when it goes online.
func TestAlert_OnlineResetsWeight(t *testing.T) {
	db := setupAlertTestDB(t)
	today := dayKey(time.Now().Unix())
	cfgRowId, _ := seedScenario(t, db, seedOpts{
		offlineMin:     5,
		peerLastOnline: time.Now().Unix(), // Currently online
		notifiedPeers:  npJSON(5, today, 0, 0),
	})
	AllService.AlertService.checkOfflineDevices()

	rec := parseNotifiedPeers(getCfg(t, db, cfgRowId).NotifiedPeers)["peer-1"]
	if rec == nil {
		t.Fatalf("Notification record not found")
	}
	if rec.Weight != 0 {
		t.Fatalf("The weight should be reset to 0 after going online, actual%d", rec.Weight)
	}
	if got := countStationMessages(t, db); got != 0 {
		t.Fatalf("Online device should not push, actual%dtimes", got)
	}
}

// TestAlert_WithinThresholdNoWeight Verification: It is just offline but has not exceeded the threshold duration, and will not be included in the weight.
func TestAlert_WithinThresholdNoWeight(t *testing.T) {
	db := setupAlertTestDB(t)
	today := dayKey(time.Now().Unix())
	// Offline only 2 minutes, threshold 5 minutes -> should not accumulate weight
	cfgRowId, _ := seedScenario(t, db, seedOpts{
		offlineMin:     5,
		peerLastOnline: time.Now().Unix() - 120,
		notifiedPeers:  npJSON(0, today, 0, 0),
	})
	AllService.AlertService.checkOfflineDevices()

	rec := parseNotifiedPeers(getCfg(t, db, cfgRowId).NotifiedPeers)["peer-1"]
	if rec == nil || rec.Weight != 0 {
		t.Fatalf("Offline within the threshold should not accumulate weight, actual%+v", rec)
	}
	if got := countStationMessages(t, db); got != 0 {
		t.Fatalf("Should not be pushed within the threshold, actual%dtimes", got)
	}
}

// TestAlert_ConsecutiveThreeDaysSuppress verification: no more emails will be pushed after 3 consecutive days of triggering.
func TestAlert_ConsecutiveThreeDaysSuppress(t *testing.T) {
	db := setupAlertTestDB(t)
	today := dayKey(time.Now().Unix())
	cfgRowId, _ := seedScenario(t, db, seedOpts{
		offlineMin:    5,
		consecutive:   3,                       // 3 days in a row
		notifiedPeers: npJSON(10, today, 0, 0), // The weight has reached the standard and should have been triggered.
	})
	AllService.AlertService.checkOfflineDevices()

	if got := countStationMessages(t, db); got != 0 {
		t.Fatalf("Pushing should be stopped for more than 3 consecutive days, actual%dtimes", got)
	}
	if c := getCfg(t, db, cfgRowId); c.ConsecutiveTriggerDays != 3 {
		t.Fatalf("Number of consecutive days during suppression should remain 3, actual%d", c.ConsecutiveTriggerDays)
	}
}

// TestAlert_ReconnectResetsConsecutive Verification: The device goes online again to reset the "3 consecutive days" limit.
func TestAlert_ReconnectResetsConsecutive(t *testing.T) {
	db := setupAlertTestDB(t)
	cfgRowId, _ := seedScenario(t, db, seedOpts{
		offlineMin:     5,
		consecutive:    3,
		peerLastOnline: time.Now().Unix(), // Online -> Trigger reset
	})
	AllService.AlertService.checkOfflineDevices()

	if c := getCfg(t, db, cfgRowId); c.ConsecutiveTriggerDays != 0 || c.LastTriggerDay != 0 {
		t.Fatalf("Coming back online should reset the continuity limit, actual days=%dlastDay=%d", c.ConsecutiveTriggerDays, c.LastTriggerDay)
	}
}

// TestAlert_MaxThreePerDay verification: The same device has reached the upper limit of 3 times on the same day and will no longer be pushed.
func TestAlert_MaxThreePerDay(t *testing.T) {
	db := setupAlertTestDB(t)
	today := dayKey(time.Now().Unix())
	cfgRowId, _ := seedScenario(t, db, seedOpts{
		offlineMin:    5,
		notifiedPeers: npJSON(10, today, maxNotifyPerDay, time.Now().Unix()), // Already 3 times that day
	})
	AllService.AlertService.checkOfflineDevices()

	if got := countStationMessages(t, db); got != 0 {
		t.Fatalf("The upper limit has been reached for the day and should not be pushed. Actual%dtimes", got)
	}
	rec := parseNotifiedPeers(getCfg(t, db, cfgRowId).NotifiedPeers)["peer-1"]
	if rec == nil || rec.Count != maxNotifyPerDay {
		t.Fatalf("The upper limit has been reached. Count should remain%dand the actual value is%+v", maxNotifyPerDay, rec)
	}
}

// TestAlert_Cooldown verification: no repeated push during the cooling period (interval with the last push < notifyCooldown).
func TestAlert_Cooldown(t *testing.T) {
	db := setupAlertTestDB(t)
	today := dayKey(time.Now().Unix())
	cfgRowId, _ := seedScenario(t, db, seedOpts{
		offlineMin:    5,
		notifiedPeers: npJSON(10, today, 0, time.Now().Unix()), // Just pushed
	})
	AllService.AlertService.checkOfflineDevices()

	if got := countStationMessages(t, db); got != 0 {
		t.Fatalf("Should not be pushed during the cooling period, actual%dtimes", got)
	}
	_ = cfgRowId
}
