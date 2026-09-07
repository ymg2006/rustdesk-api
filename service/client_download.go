package service

import (
	"errors"
	"fmt"

	"github.com/ymg2006/rustdesk-api/v2/model"
	"gorm.io/gorm"
)

// ClientDownloadService client download link management
type ClientDownloadService struct{}

var ErrClientDownloadNotFound = errors.New("client download not found")

// ActiveList Gets the enabled download list (public).
// Deprecated: use ActiveListWithError when the caller can surface database failures.
func (s *ClientDownloadService) ActiveList() []*model.ClientDownload {
	list, err := s.ActiveListWithError()
	if err != nil && Logger != nil {
		Logger.Warnf("list active client downloads: %v", err)
	}
	return list
}

// ActiveListWithError gets the enabled download list and preserves database
// failures so callers do not mistake them for an empty successful response.
func (s *ClientDownloadService) ActiveListWithError() ([]*model.ClientDownload, error) {
	var list []*model.ClientDownload
	if cacheGet(clientDownloadActiveCacheKey, &list) {
		return list, nil
	}
	if err := DB.Where("status = ?", model.COMMON_STATUS_ENABLE).
		Order("sort_order asc, created_at desc").
		Find(&list).Error; err != nil {
		return nil, fmt.Errorf("list active client downloads: %w", err)
	}
	cacheSet(clientDownloadActiveCacheKey, list, clientDownloadCacheTTL)
	return list, nil
}

// List Get the download list (management side, paging).
// Deprecated: use ListWithError when the caller can surface database failures.
func (s *ClientDownloadService) List(page, pageSize uint) ([]*model.ClientDownload, int64) {
	list, total, err := s.ListWithError(page, pageSize)
	if err != nil && Logger != nil {
		Logger.Warnf("list client downloads: %v", err)
	}
	return list, total
}

// ListWithError gets the management list while preserving database failures.
func (s *ClientDownloadService) ListWithError(page, pageSize uint) ([]*model.ClientDownload, int64, error) {
	var list []*model.ClientDownload
	var total int64
	if err := DB.Model(&model.ClientDownload{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count client downloads: %w", err)
	}
	if err := DB.Order("sort_order asc, created_at desc").Scopes(Paginate(page, pageSize)).Find(&list).Error; err != nil {
		return nil, 0, fmt.Errorf("list client downloads: %w", err)
	}
	return list, total, nil
}

// Create Create download link.
func (s *ClientDownloadService) Create(v *model.ClientDownload) error {
	if err := DB.Create(v).Error; err != nil {
		return fmt.Errorf("create client download: %w", err)
	}
	// Create may add a row to the public enabled-download list.
	cacheDelete(clientDownloadActiveCacheKey)
	return nil
}

// Update update download link.
func (s *ClientDownloadService) Update(v *model.ClientDownload) error {
	result := DB.Model(v).Where("id = ?", v.Id).Updates(v)
	if result.Error != nil {
		return fmt.Errorf("update client download: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		exists, err := clientDownloadExists(v.Id)
		if err != nil {
			return err
		}
		if !exists {
			return ErrClientDownloadNotFound
		}
	}
	// Update may change public fields, ordering, or enabled status.
	cacheDelete(clientDownloadActiveCacheKey)
	return nil
}

// Delete delete download link.
func (s *ClientDownloadService) Delete(id uint) error {
	result := DB.Delete(&model.ClientDownload{}, id)
	if result.Error != nil {
		return fmt.Errorf("delete client download: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrClientDownloadNotFound
	}
	// Delete may remove a row from the public enabled-download list.
	cacheDelete(clientDownloadActiveCacheKey)
	return nil
}

// FindById Find by ID.
// Deprecated: use FindByIdWithError when the caller can surface database failures.
func (s *ClientDownloadService) FindById(id uint) *model.ClientDownload {
	v, err := s.FindByIdWithError(id)
	if err != nil && !errors.Is(err, ErrClientDownloadNotFound) && Logger != nil {
		Logger.Warnf("find client download by id: %v", err)
	}
	return v
}

// FindByIdWithError finds a download and distinguishes not-found from database failures.
func (s *ClientDownloadService) FindByIdWithError(id uint) (*model.ClientDownload, error) {
	var v model.ClientDownload
	err := DB.Where("id = ?", id).First(&v).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrClientDownloadNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find client download: %w", err)
	}
	return &v, nil
}

func clientDownloadExists(id uint) (bool, error) {
	var count int64
	if err := DB.Model(&model.ClientDownload{}).Where("id = ?", id).Count(&count).Error; err != nil {
		return false, fmt.Errorf("check client download existence: %w", err)
	}
	return count > 0, nil
}
