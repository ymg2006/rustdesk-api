package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/sirupsen/logrus"
	cachelib "github.com/ymg2006/rustdesk-api/v2/lib/cache"
	"github.com/ymg2006/rustdesk-api/v2/model"
	"gorm.io/gorm"
)

type failingCache struct{}

func (failingCache) Get(context.Context, string, interface{}) error {
	return errors.New("cache unavailable")
}
func (failingCache) Set(context.Context, string, interface{}, time.Duration) error {
	return errors.New("cache unavailable")
}
func (failingCache) Delete(context.Context, string) error {
	return errors.New("cache unavailable")
}

type timeoutCache struct{ failingCache }

func (timeoutCache) Get(context.Context, string, interface{}) error {
	return context.DeadlineExceeded
}

type invalidJSONCache struct{}

func (invalidJSONCache) Get(_ context.Context, _ string, dest interface{}) error {
	return cachelib.DecodeValue("not-json", dest)
}
func (invalidJSONCache) Set(context.Context, string, interface{}, time.Duration) error { return nil }
func (invalidJSONCache) Delete(context.Context, string) error                          { return nil }

type userCacheMustNotBeUsed struct{}

func (userCacheMustNotBeUsed) Get(context.Context, string, interface{}) error {
	panic("user/auth reads must not use the application cache")
}
func (userCacheMustNotBeUsed) Set(context.Context, string, interface{}, time.Duration) error {
	return nil
}
func (userCacheMustNotBeUsed) Delete(context.Context, string) error { return nil }

func setupApplicationCacheTest(t *testing.T) {
	t.Helper()
	oldDB, oldLogger, oldCache := DB, Logger, Cache
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.AppRelease{}, &model.ClientDownload{}, &model.Announcement{}, &model.User{}); err != nil {
		t.Fatal(err)
	}
	DB = db
	Logger = logrus.New()
	t.Cleanup(func() {
		if closer, ok := Cache.(interface{ Close() error }); ok {
			_ = closer.Close()
		}
		DB, Logger, Cache = oldDB, oldLogger, oldCache
	})
}

func TestPublicApplicationCachesInvalidateAfterMutation(t *testing.T) {
	setupApplicationCacheTest(t)
	SetCache(cachelib.NewMemoryCache(0))

	releases := &AppReleaseService{}
	v1 := &model.AppRelease{Version: "1", Platform: "linux", Status: int(model.COMMON_STATUS_ENABLE)}
	if err := releases.Create(v1); err != nil {
		t.Fatal(err)
	}
	if got := releases.Latest("ubuntu"); got == nil || got.Version != "1" {
		t.Fatalf("first latest release = %#v", got)
	}
	v2 := &model.AppRelease{Version: "2", Platform: "ubuntu", Status: int(model.COMMON_STATUS_ENABLE)}
	if err := DB.Create(v2).Error; err != nil {
		t.Fatal(err)
	}
	if got := releases.Latest("linux"); got == nil || got.Version != "1" {
		t.Fatalf("linux/ubuntu aliases should share cached value; got %#v", got)
	}
	v1.Status = int(model.COMMON_STATUS_DISABLED)
	releases.Update(v1)
	if got := releases.Latest("linux"); got == nil || got.Version != "2" {
		t.Fatalf("latest release after invalidation = %#v", got)
	}

	downloads := &ClientDownloadService{}
	d1 := &model.ClientDownload{Name: "one", Status: int(model.COMMON_STATUS_ENABLE)}
	downloads.Create(d1)
	if got := downloads.ActiveList(); len(got) != 1 {
		t.Fatalf("active downloads = %d, want 1", len(got))
	}
	d2 := &model.ClientDownload{Name: "two", Status: int(model.COMMON_STATUS_ENABLE)}
	if err := DB.Create(d2).Error; err != nil {
		t.Fatal(err)
	}
	if got := downloads.ActiveList(); len(got) != 1 {
		t.Fatalf("cached active downloads = %d, want 1", len(got))
	}
	d1.Name = "updated"
	downloads.Update(d1)
	if got := downloads.ActiveList(); len(got) != 2 {
		t.Fatalf("active downloads after invalidation = %d, want 2", len(got))
	}

	announcements := &AnnouncementService{}
	a1 := &model.Announcement{Title: "one", Content: "first", Status: 1}
	if err := announcements.Create(a1); err != nil {
		t.Fatal(err)
	}
	if got, err := announcements.ListActiveForClient(); err != nil || len(got) != 1 {
		t.Fatalf("active announcements = %#v, %v", got, err)
	}
	a2 := &model.Announcement{Title: "two", Content: "second", Status: 1}
	if err := DB.Create(a2).Error; err != nil {
		t.Fatal(err)
	}
	if got, err := announcements.ListActiveForClient(); err != nil || len(got) != 1 {
		t.Fatalf("cached active announcements = %#v, %v", got, err)
	}
	a1.Status = 0
	if err := announcements.Update(a1); err != nil {
		t.Fatal(err)
	}
	if got, err := announcements.ListActiveForClient(); err != nil || len(got) != 1 || got[0]["title"] != "two" {
		t.Fatalf("active announcements after invalidation = %#v, %v", got, err)
	}
}

func TestPublicReadFallsBackToDatabaseWhenCacheFails(t *testing.T) {
	setupApplicationCacheTest(t)
	SetCache(failingCache{})

	download := &model.ClientDownload{Name: "available", Status: int(model.COMMON_STATUS_ENABLE)}
	if err := DB.Create(download).Error; err != nil {
		t.Fatal(err)
	}
	if got := (&ClientDownloadService{}).ActiveList(); len(got) != 1 || got[0].Name != "available" {
		t.Fatalf("database fallback returned %#v", got)
	}

	a := &model.Announcement{Title: "notice", Content: "content", Status: 1}
	if err := DB.Create(a).Error; err != nil {
		t.Fatal(err)
	}
	got, err := (&AnnouncementService{}).ListActiveForClient()
	if err != nil || len(got) != 1 || got[0]["title"] != "notice" {
		t.Fatalf("announcement database fallback = %#v, %v", got, err)
	}
}

func TestExpiredPublicCacheReloadsFromDatabase(t *testing.T) {
	setupApplicationCacheTest(t)
	memoryCache := cachelib.NewMemoryCache(0)
	SetCache(memoryCache)

	stale := []*model.ClientDownload{{Name: "stale", Status: int(model.COMMON_STATUS_ENABLE)}}
	if err := memoryCache.Set(context.Background(), clientDownloadActiveCacheKey, stale, time.Millisecond); err != nil {
		t.Fatal(err)
	}
	current := &model.ClientDownload{Name: "database", Status: int(model.COMMON_STATUS_ENABLE)}
	if err := DB.Create(current).Error; err != nil {
		t.Fatal(err)
	}
	time.Sleep(5 * time.Millisecond)
	got := (&ClientDownloadService{}).ActiveList()
	if len(got) != 1 || got[0].Name != "database" {
		t.Fatalf("expired cache reload returned %#v", got)
	}
}

func TestInvalidJSONFallsBackToDatabase(t *testing.T) {
	setupApplicationCacheTest(t)
	SetCache(invalidJSONCache{})
	current := &model.ClientDownload{Name: "database", Status: int(model.COMMON_STATUS_ENABLE)}
	if err := DB.Create(current).Error; err != nil {
		t.Fatal(err)
	}
	got := (&ClientDownloadService{}).ActiveList()
	if len(got) != 1 || got[0].Name != "database" {
		t.Fatalf("invalid JSON fallback returned %#v", got)
	}
}

func TestCacheTimeoutFallsBackToDatabase(t *testing.T) {
	setupApplicationCacheTest(t)
	SetCache(timeoutCache{})
	current := &model.ClientDownload{Name: "database", Status: int(model.COMMON_STATUS_ENABLE)}
	if err := DB.Create(current).Error; err != nil {
		t.Fatal(err)
	}
	got := (&ClientDownloadService{}).ActiveList()
	if len(got) != 1 || got[0].Name != "database" {
		t.Fatalf("timeout fallback returned %#v", got)
	}
}

func TestUserReadsRemainDatabaseIsolated(t *testing.T) {
	setupApplicationCacheTest(t)
	SetCache(userCacheMustNotBeUsed{})
	first := &model.User{Username: "first", Email: "first@example.test"}
	second := &model.User{Username: "second", Email: "second@example.test"}
	if err := DB.Create(first).Error; err != nil {
		t.Fatal(err)
	}
	if err := DB.Create(second).Error; err != nil {
		t.Fatal(err)
	}
	users := &UserService{}
	if got := users.InfoById(first.Id); got.Id != first.Id || got.Username != "first" {
		t.Fatalf("first user lookup returned %#v", got)
	}
	if got := users.InfoById(second.Id); got.Id != second.Id || got.Username != "second" {
		t.Fatalf("second user lookup returned %#v", got)
	}
}

type cacheActivityRecorder struct {
	setCalls    int
	deleteCalls int
}

func (c *cacheActivityRecorder) Get(context.Context, string, interface{}) error {
	return cachelib.ErrCacheMiss
}

func (c *cacheActivityRecorder) Set(context.Context, string, interface{}, time.Duration) error {
	c.setCalls++
	return nil
}

func (c *cacheActivityRecorder) Delete(context.Context, string) error {
	c.deleteCalls++
	return nil
}

func TestPublicDatabaseErrorsAreReturnedAndNotCached(t *testing.T) {
	setupApplicationCacheTest(t)
	recorder := &cacheActivityRecorder{}
	SetCache(recorder)

	if err := DB.Migrator().DropTable(&model.ClientDownload{}); err != nil {
		t.Fatal(err)
	}
	if _, err := (&ClientDownloadService{}).ActiveListWithError(); err == nil {
		t.Fatal("ActiveListWithError returned nil error after client_downloads table was dropped")
	}
	if recorder.setCalls != 0 {
		t.Fatalf("database failure populated client-download cache %d times", recorder.setCalls)
	}

	if err := DB.Migrator().DropTable(&model.AppRelease{}); err != nil {
		t.Fatal(err)
	}
	if _, err := (&AppReleaseService{}).LatestWithError("windows"); err == nil {
		t.Fatal("LatestWithError returned nil error after app_releases table was dropped")
	}
	if recorder.setCalls != 0 {
		t.Fatalf("database failure populated public cache %d times", recorder.setCalls)
	}
}

func TestClientDownloadDatabaseErrorsPropagate(t *testing.T) {
	setupApplicationCacheTest(t)
	recorder := &cacheActivityRecorder{}
	SetCache(recorder)
	if err := DB.Migrator().DropTable(&model.ClientDownload{}); err != nil {
		t.Fatal(err)
	}

	svc := &ClientDownloadService{}
	if _, _, err := svc.ListWithError(1, 10); err == nil {
		t.Fatal("ListWithError returned nil error after table was dropped")
	}
	if err := svc.Create(&model.ClientDownload{Name: "x"}); err == nil {
		t.Fatal("Create returned nil error after table was dropped")
	}
	if err := svc.Update(&model.ClientDownload{IdModel: model.IdModel{Id: 1}, Name: "x"}); err == nil {
		t.Fatal("Update returned nil error after table was dropped")
	}
	if err := svc.Delete(1); err == nil {
		t.Fatal("Delete returned nil error after table was dropped")
	}
	if _, err := svc.FindByIdWithError(1); err == nil {
		t.Fatal("FindByIdWithError returned nil error after table was dropped")
	}
	if recorder.deleteCalls != 0 {
		t.Fatalf("failed client-download mutations invalidated cache %d times", recorder.deleteCalls)
	}
}

func TestAppReleaseDatabaseErrorsPropagate(t *testing.T) {
	setupApplicationCacheTest(t)
	recorder := &cacheActivityRecorder{}
	SetCache(recorder)
	if err := DB.Migrator().DropTable(&model.AppRelease{}); err != nil {
		t.Fatal(err)
	}

	svc := &AppReleaseService{}
	if _, _, err := svc.ListWithError(1, 10); err == nil {
		t.Fatal("ListWithError returned nil error after table was dropped")
	}
	if err := svc.Create(&model.AppRelease{Version: "1", Platform: "windows"}); err == nil {
		t.Fatal("Create returned nil error after table was dropped")
	}
	if err := svc.Update(&model.AppRelease{IdModel: model.IdModel{Id: 1}, Version: "1"}); err == nil {
		t.Fatal("Update returned nil error after table was dropped")
	}
	if err := svc.Delete(1); err == nil {
		t.Fatal("Delete returned nil error after table was dropped")
	}
	if _, err := svc.FindByIdWithError(1); err == nil {
		t.Fatal("FindByIdWithError returned nil error after table was dropped")
	}
	if recorder.deleteCalls != 0 {
		t.Fatalf("failed app-release mutations invalidated cache %d times", recorder.deleteCalls)
	}
}

func TestMissingApplicationMutationsReportNotFound(t *testing.T) {
	setupApplicationCacheTest(t)
	SetCache(cachelib.NewMemoryCache(0))

	downloads := &ClientDownloadService{}
	missingDownload := &model.ClientDownload{IdModel: model.IdModel{Id: 999}, Name: "missing"}
	if err := downloads.Update(missingDownload); !errors.Is(err, ErrClientDownloadNotFound) {
		t.Fatalf("client download Update error = %v, want ErrClientDownloadNotFound", err)
	}
	if err := downloads.Delete(999); !errors.Is(err, ErrClientDownloadNotFound) {
		t.Fatalf("client download Delete error = %v, want ErrClientDownloadNotFound", err)
	}
	if _, err := downloads.FindByIdWithError(999); !errors.Is(err, ErrClientDownloadNotFound) {
		t.Fatalf("client download FindByIdWithError error = %v, want ErrClientDownloadNotFound", err)
	}

	releases := &AppReleaseService{}
	missingRelease := &model.AppRelease{IdModel: model.IdModel{Id: 999}, Version: "missing"}
	if err := releases.Update(missingRelease); !errors.Is(err, ErrAppReleaseNotFound) {
		t.Fatalf("app release Update error = %v, want ErrAppReleaseNotFound", err)
	}
	if err := releases.Delete(999); !errors.Is(err, ErrAppReleaseNotFound) {
		t.Fatalf("app release Delete error = %v, want ErrAppReleaseNotFound", err)
	}
	if _, err := releases.FindByIdWithError(999); !errors.Is(err, ErrAppReleaseNotFound) {
		t.Fatalf("app release FindByIdWithError error = %v, want ErrAppReleaseNotFound", err)
	}
}
