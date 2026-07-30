package service

import (
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/ymg2006/rustdesk-api/v2/lib/lock"
	"github.com/ymg2006/rustdesk-api/v2/model"
	"gorm.io/gorm"
)

func setupInviteAnnouncementDB(t *testing.T) *gorm.DB {
	t.Helper()
	oldDB, oldLock := DB, Lock
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "service.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.InviteCode{}, &model.User{}, &model.Announcement{}); err != nil {
		t.Fatal(err)
	}
	DB = db
	Lock = lock.NewLocal()
	t.Cleanup(func() {
		DB, Lock = oldDB, oldLock
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

func TestInviteCodeLookupsDistinguishNotFoundAndDatabaseErrors(t *testing.T) {
	db := setupInviteAnnouncementDB(t)
	s := &InviteCodeService{}

	if _, err := s.InfoByCode("missing"); !errors.Is(err, ErrInviteCodeNotFound) {
		t.Fatalf("InfoByCode error = %v, want ErrInviteCodeNotFound", err)
	}
	if _, err := s.InfoByOrderID("missing"); !errors.Is(err, ErrInviteCodeNotFound) {
		t.Fatalf("InfoByOrderID error = %v, want ErrInviteCodeNotFound", err)
	}

	if err := db.Migrator().DropTable(&model.InviteCode{}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.List(InviteCodeFilter{}); err == nil {
		t.Fatal("List returned nil error after invite_codes table was dropped")
	}
	if _, err := s.InfoByCode("missing"); err == nil || errors.Is(err, ErrInviteCodeNotFound) {
		t.Fatalf("database error = %v, must not be mapped to not-found", err)
	}
}

func TestInviteCodeActivateConcurrentRedemptions(t *testing.T) {
	db := setupInviteAnnouncementDB(t)
	base := time.Now().Add(24 * time.Hour).Truncate(time.Second)
	user := &model.User{Username: "same-code-user", SubscriptionExpireAt: &base}
	if err := db.Create(user).Error; err != nil {
		t.Fatal(err)
	}
	code := &model.InviteCode{
		Code: "same-code", Plan: "pro", ExpireDays: 30,
		ExpireAt: time.Now().Add(24 * time.Hour), Status: "unused",
	}
	if err := db.Create(code).Error; err != nil {
		t.Fatal(err)
	}

	start := make(chan struct{})
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := (&InviteCodeService{}).Activate(code.Code, user.Id)
			errs <- err
		}()
	}
	close(start)
	wg.Wait()
	close(errs)

	successes, conflicts := 0, 0
	for err := range errs {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, ErrInviteCodeAlreadyConsumed):
			conflicts++
		default:
			t.Fatalf("unexpected activation error: %v", err)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("successes=%d conflicts=%d, want 1 and 1", successes, conflicts)
	}
	if err := db.First(user, user.Id).Error; err != nil {
		t.Fatal(err)
	}
	want := base.Add(30 * 24 * time.Hour)
	if user.SubscriptionExpireAt == nil || !user.SubscriptionExpireAt.Equal(want) {
		t.Fatalf("subscription expiry = %v, want %v", user.SubscriptionExpireAt, want)
	}
}

func TestInviteCodeActivateTwoCodesForOneUser(t *testing.T) {
	db := setupInviteAnnouncementDB(t)
	base := time.Now().Add(24 * time.Hour).Truncate(time.Second)
	user := &model.User{Username: "two-code-user", SubscriptionExpireAt: &base}
	if err := db.Create(user).Error; err != nil {
		t.Fatal(err)
	}
	codes := []model.InviteCode{
		{Code: "first-code", Plan: "pro", ExpireDays: 30, ExpireAt: time.Now().Add(24 * time.Hour), Status: "unused"},
		{Code: "second-code", Plan: "pro", ExpireDays: 30, ExpireAt: time.Now().Add(24 * time.Hour), Status: "unused"},
	}
	if err := db.Create(&codes).Error; err != nil {
		t.Fatal(err)
	}

	start := make(chan struct{})
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for _, code := range codes {
		wg.Add(1)
		go func(code string) {
			defer wg.Done()
			<-start
			_, err := (&InviteCodeService{}).Activate(code, user.Id)
			errs <- err
		}(code.Code)
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("activation failed: %v", err)
		}
	}
	if err := db.First(user, user.Id).Error; err != nil {
		t.Fatal(err)
	}
	want := base.Add(60 * 24 * time.Hour)
	if user.SubscriptionExpireAt == nil || !user.SubscriptionExpireAt.Equal(want) {
		t.Fatalf("subscription expiry = %v, want %v", user.SubscriptionExpireAt, want)
	}
}

func TestAnnouncementAdminClientAndZeroValueUpdate(t *testing.T) {
	db := setupInviteAnnouncementDB(t)
	s := &AnnouncementService{}
	active := &model.Announcement{Title: "active", Content: "content", Status: 1}
	inactive := &model.Announcement{Title: "inactive", Content: "content", Status: 0}
	if err := s.Create(active); err != nil {
		t.Fatal(err)
	}
	if err := s.Create(inactive); err != nil {
		t.Fatal(err)
	}
	adminList, err := s.ListAdmin()
	if err != nil || len(adminList) != 2 {
		t.Fatalf("admin list length=%d error=%v, want 2", len(adminList), err)
	}
	clientList, err := s.ListActiveForClient()
	if err != nil || len(clientList) != 1 {
		t.Fatalf("client list length=%d error=%v, want 1", len(clientList), err)
	}
	active.Status = 0
	if err := s.Update(active); err != nil {
		t.Fatal(err)
	}
	var got model.Announcement
	if err := db.First(&got, active.Id).Error; err != nil {
		t.Fatal(err)
	}
	if got.Status != 0 {
		t.Fatalf("updated status=%d, want 0", got.Status)
	}
	if err := s.Delete(999999); !errors.Is(err, ErrAnnouncementNotFound) {
		t.Fatalf("delete missing error=%v, want ErrAnnouncementNotFound", err)
	}
}
