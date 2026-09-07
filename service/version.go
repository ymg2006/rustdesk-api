package service

import (
	"errors"
	"fmt"

	"github.com/ymg2006/rustdesk-api/v2/model"
	"gorm.io/gorm"
)

const appReleaseCacheKeyPrefix = "rustdesk-api:v1:app-release:latest:"

var appReleaseCacheKeys = []string{
	appReleaseCacheKeyPrefix + "all",
	appReleaseCacheKeyPrefix + "windows",
	appReleaseCacheKeyPrefix + "linux",
	appReleaseCacheKeyPrefix + "macos",
	appReleaseCacheKeyPrefix + "android",
}

var ErrAppReleaseNotFound = errors.New("app release not found")

func appReleaseCacheKey(platform string) (string, bool) {
	switch platform {
	case "":
		return appReleaseCacheKeyPrefix + "all", true
	case "windows", "android":
		return appReleaseCacheKeyPrefix + platform, true
	case "ubuntu", "linux":
		return appReleaseCacheKeyPrefix + "linux", true
	case "macos", "mac":
		return appReleaseCacheKeyPrefix + "macos", true
	default:
		// Preserve the existing exact-match behavior for arbitrary platform values
		// without creating an unbounded cache-key space.
		return "", false
	}
}

func invalidateLatestAppReleases() {
	cacheDelete(appReleaseCacheKeys...)
}

// AppReleaseService version release management
type AppReleaseService struct {
}

// Latest returns the latest release when available.
// Deprecated: use LatestWithError when the caller can surface database failures.
func (s *AppReleaseService) Latest(platform string) *model.AppRelease {
	v, err := s.LatestWithError(platform)
	if err != nil && Logger != nil {
		Logger.Warnf("get latest app release: %v", err)
	}
	return v
}

// LatestWithError returns the latest release and distinguishes a normal
// no-release result from a database failure. Database failures are never cached.
func (s *AppReleaseService) LatestWithError(platform string) (*model.AppRelease, error) {
	key, cacheable := appReleaseCacheKey(platform)
	if cacheable {
		var cached model.AppRelease
		if cacheGet(key, &cached) && cached.Id > 0 {
			return &cached, nil
		}
	}

	var v model.AppRelease
	db := DB.Where("status = ?", model.COMMON_STATUS_ENABLE)
	if platform != "" {
		// platform alias compatibility: the client may send ubuntu/linux or macos/mac.
		// Stored values may differ, so aliases are matched as synonymous groups.
		aliasMap := map[string][]string{
			"ubuntu":  {"ubuntu", "linux"},
			"linux":   {"linux", "ubuntu"},
			"macos":   {"macos", "mac"},
			"mac":     {"mac", "macos"},
			"windows": {"windows"},
			"android": {"android"},
		}
		platforms, ok := aliasMap[platform]
		if !ok {
			platforms = []string{platform}
		}
		db = db.Where("platform IN ?", platforms)
	}
	err := db.Order("created_at desc").First(&v).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get latest app release: %w", err)
	}
	if cacheable {
		cacheSet(key, &v, appReleaseCacheTTL)
	}
	return &v, nil
}

// List returns the management release list.
// Deprecated: use ListWithError when the caller can surface database failures.
func (s *AppReleaseService) List(page, pageSize uint) ([]*model.AppRelease, int64) {
	list, total, err := s.ListWithError(page, pageSize)
	if err != nil && Logger != nil {
		Logger.Warnf("list app releases: %v", err)
	}
	return list, total
}

// ListWithError gets the management release list while preserving database failures.
func (s *AppReleaseService) ListWithError(page, pageSize uint) ([]*model.AppRelease, int64, error) {
	var list []*model.AppRelease
	var total int64
	if err := DB.Model(&model.AppRelease{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count app releases: %w", err)
	}
	if err := DB.Order("created_at desc").Scopes(Paginate(page, pageSize)).Find(&list).Error; err != nil {
		return nil, 0, fmt.Errorf("list app releases: %w", err)
	}
	return list, total, nil
}

func (s *AppReleaseService) Create(v *model.AppRelease) error {
	if err := DB.Create(v).Error; err != nil {
		return fmt.Errorf("create app release: %w", err)
	}
	// A new enabled release can become the latest for any platform alias.
	invalidateLatestAppReleases()
	return nil
}

func (s *AppReleaseService) Update(v *model.AppRelease) error {
	result := DB.Model(v).Where("id = ?", v.Id).Updates(v)
	if result.Error != nil {
		return fmt.Errorf("update app release: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		exists, err := appReleaseExists(v.Id)
		if err != nil {
			return err
		}
		if !exists {
			return ErrAppReleaseNotFound
		}
	}
	// Update may change platform, status, or returned metadata.
	invalidateLatestAppReleases()
	return nil
}

func (s *AppReleaseService) Delete(id uint) error {
	result := DB.Delete(&model.AppRelease{}, id)
	if result.Error != nil {
		return fmt.Errorf("delete app release: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrAppReleaseNotFound
	}
	// Delete can expose the previous release as latest.
	invalidateLatestAppReleases()
	return nil
}

// FindById finds a release by ID.
// Deprecated: use FindByIdWithError when the caller can surface database failures.
func (s *AppReleaseService) FindById(id uint) *model.AppRelease {
	v, err := s.FindByIdWithError(id)
	if err != nil && !errors.Is(err, ErrAppReleaseNotFound) && Logger != nil {
		Logger.Warnf("find app release by id: %v", err)
	}
	return v
}

// FindByIdWithError finds a release and distinguishes not-found from database failures.
func (s *AppReleaseService) FindByIdWithError(id uint) (*model.AppRelease, error) {
	var v model.AppRelease
	err := DB.Where("id = ?", id).First(&v).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrAppReleaseNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find app release: %w", err)
	}
	return &v, nil
}

func appReleaseExists(id uint) (bool, error) {
	var count int64
	if err := DB.Model(&model.AppRelease{}).Where("id = ?", id).Count(&count).Error; err != nil {
		return false, fmt.Errorf("check app release existence: %w", err)
	}
	return count > 0, nil
}
