package service

import (
	"github.com/ymg2006/rustdesk-api/v2/model"
)

// ClientDownloadService client download link management
type ClientDownloadService struct{}

// ActiveList Gets the enabled download list (public)
func (s *ClientDownloadService) ActiveList() []*model.ClientDownload {
	var list []*model.ClientDownload
	DB.Where("status = ?", model.COMMON_STATUS_ENABLE).
		Order("sort_order asc, created_at desc").
		Find(&list)
	return list
}

// List Get the download list (management side, paging)
func (s *ClientDownloadService) List(page, pageSize uint) ([]*model.ClientDownload, int64) {
	var list []*model.ClientDownload
	var total int64
	DB.Model(&model.ClientDownload{}).Count(&total)
	DB.Order("sort_order asc, created_at desc").Scopes(Paginate(page, pageSize)).Find(&list)
	return list, total
}

// Create Create download link
func (s *ClientDownloadService) Create(v *model.ClientDownload) {
	DB.Create(v)
}

// Update update download link
func (s *ClientDownloadService) Update(v *model.ClientDownload) {
	DB.Model(v).Where("id = ?", v.Id).Updates(v)
}

// Delete delete download link
func (s *ClientDownloadService) Delete(id uint) {
	DB.Delete(&model.ClientDownload{}, id)
}

// FindById Find by ID
func (s *ClientDownloadService) FindById(id uint) *model.ClientDownload {
	var v model.ClientDownload
	DB.Where("id = ?", id).First(&v)
	if v.Id > 0 {
		return &v
	}
	return nil
}
