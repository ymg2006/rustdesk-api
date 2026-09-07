package service

import (
	"errors"
	"fmt"

	"github.com/ymg2006/rustdesk-api/v2/model"
	"gorm.io/gorm"
)

type AnnouncementService struct {
}

var ErrAnnouncementNotFound = errors.New("announcement not found")

func (as *AnnouncementService) ListAdmin() ([]model.Announcement, error) {
	var announcements []model.Announcement
	err := DB.Order("created_at desc").Find(&announcements).Error
	if err != nil {
		return nil, fmt.Errorf("list announcements for admin: %w", err)
	}
	return announcements, nil
}

func (as *AnnouncementService) ListActiveForClient() ([]map[string]interface{}, error) {
	var announcements []model.Announcement
	if !cacheGet(announcementActiveCacheKey, &announcements) {
		if err := DB.Where("status = ?", 1).Order("created_at desc").
			Find(&announcements).Error; err != nil {
			return nil, fmt.Errorf("list active announcements: %w", err)
		}
		cacheSet(announcementActiveCacheKey, announcements, announcementCacheTTL)
	}
	result := make([]map[string]interface{}, 0, len(announcements))
	for _, a := range announcements {
		result = append(result, map[string]interface{}{
			"id":         a.Id,
			"title":      a.Title,
			"content":    a.Content,
			"created_at": a.CreatedAt,
		})
	}
	return result, nil
}

func (as *AnnouncementService) Info(id uint) (*model.Announcement, error) {
	a := &model.Announcement{}
	err := DB.Where("id = ?", id).First(a).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrAnnouncementNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find announcement: %w", err)
	}
	return a, nil
}

func (as *AnnouncementService) Create(a *model.Announcement) error {
	err := DB.Transaction(func(tx *gorm.DB) error {
		status := a.Status
		if err := tx.Create(a).Error; err != nil {
			return fmt.Errorf("create announcement: %w", err)
		}
		if status == 0 {
			if err := tx.Model(&model.Announcement{}).Where("id = ?", a.Id).
				Update("status", 0).Error; err != nil {
				return fmt.Errorf("set announcement status: %w", err)
			}
			a.Status = 0
		}
		return nil
	})
	if err == nil {
		// Create changes the public active list, so invalidate only after the
		// database transaction has committed successfully.
		cacheDelete(announcementActiveCacheKey)
	}
	return err
}

func (as *AnnouncementService) Update(a *model.Announcement) error {
	result := DB.Model(&model.Announcement{}).Where("id = ?", a.Id).
		Updates(map[string]interface{}{
			"title": a.Title, "content": a.Content, "status": a.Status,
		})
	if result.Error != nil {
		return fmt.Errorf("update announcement: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrAnnouncementNotFound
	}
	// Update can change content or active status returned by the public list.
	cacheDelete(announcementActiveCacheKey)
	return nil
}

func (as *AnnouncementService) Delete(id uint) error {
	result := DB.Delete(&model.Announcement{}, id)
	if result.Error != nil {
		return fmt.Errorf("delete announcement: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrAnnouncementNotFound
	}
	// Delete removes an item from the public active list.
	cacheDelete(announcementActiveCacheKey)
	return nil
}
