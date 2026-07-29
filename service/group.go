package service

import (
	"errors"

	"github.com/ymg2006/rustdesk-api/v2/model"
	"gorm.io/gorm"
)

type GroupService struct {
}

// InfoById gets user information based on user id
func (us *GroupService) InfoById(id uint) *model.Group {
	u := &model.Group{}
	DB.Where("id = ?", id).First(u)
	return u
}

// All takes all departments (used to build the tree)
func (us *GroupService) All() []*model.Group {
	var groups []*model.Group
	DB.Order("id asc").Find(&groups)
	return groups
}

func (us *GroupService) List(page, pageSize uint, where func(tx *gorm.DB)) (res *model.GroupList) {
	res = &model.GroupList{}
	res.Page = int64(page)
	res.PageSize = int64(pageSize)
	tx := DB.Model(&model.Group{})
	if where != nil {
		where(tx)
	}
	tx.Count(&res.Total)
	tx.Scopes(Paginate(page, pageSize))
	tx.Find(&res.Groups)
	return
}

// Tree Build department tree
func (us *GroupService) Tree() []*model.GroupTree {
	all := us.All()
	return us.buildTree(all, 0)
}

func (us *GroupService) buildTree(groups []*model.Group, parentId uint) []*model.GroupTree {
	nodes := make([]*model.GroupTree, 0)
	for _, g := range groups {
		if g.ParentId == parentId {
			node := &model.GroupTree{
				Group:    g,
				Children: us.buildTree(groups, g.Id),
			}
			node.UserCount = us.UserCountByGroupId(g.Id)
			nodes = append(nodes, node)
		}
	}
	return nodes
}

// Does HasChildren have sub-departments?
func (us *GroupService) HasChildren(id uint) bool {
	var c int64
	DB.Model(&model.Group{}).Where("parent_id = ?", id).Count(&c)
	return c > 0
}

// UserCountByGroupId Number of users under the department
func (us *GroupService) UserCountByGroupId(groupId uint) int64 {
	var c int64
	DB.Model(&model.User{}).Where("group_id = ?", groupId).Count(&c)
	return c
}

// DescendantIds gets all the descendant department IDs of a department (excluding itself)
func (us *GroupService) DescendantIds(parentId uint) []uint {
	ids := make([]uint, 0)
	all := us.All()
	var walk func(pid uint)
	walk = func(pid uint) {
		for _, g := range all {
			if g.ParentId == pid {
				ids = append(ids, g.Id)
				walk(g.Id)
			}
		}
	}
	walk(parentId)
	return ids
}

// IsDescendantOf determines whether maybeChild is a descendant of ancestor (including itself), used to prevent department level loops
func (us *GroupService) IsDescendantOf(maybeChild, ancestor uint) bool {
	all := us.All()
	cur := maybeChild
	for {
		if cur == ancestor {
			return true
		}
		if cur == 0 {
			return false
		}
		parent := uint(0)
		for _, g := range all {
			if g.Id == cur {
				parent = g.ParentId
				break
			}
		}
		if parent == cur {
			return false
		}
		cur = parent
	}
}

// Create
func (us *GroupService) Create(u *model.Group) error {
	res := DB.Create(u).Error
	return res
}

// Delete Delete (disabled when there are sub-departments or members to avoid the generation of orphan data)
func (us *GroupService) Delete(u *model.Group) error {
	if us.HasChildren(u.Id) {
		return errors.New("DeptHasChildren")
	}
	if us.UserCountByGroupId(u.Id) > 0 {
		return errors.New("DeptHasUsers")
	}
	return DB.Delete(u).Error
}

// Update update
func (us *GroupService) Update(u *model.Group) error {
	return DB.Model(u).Updates(u).Error
}

// DeviceGroupInfoById gets user information based on user id
func (us *GroupService) DeviceGroupInfoById(id uint) *model.DeviceGroup {
	u := &model.DeviceGroup{}
	DB.Where("id = ?", id).First(u)
	return u
}

func (us *GroupService) DeviceGroupList(page, pageSize uint, where func(tx *gorm.DB)) (res *model.DeviceGroupList) {
	res = &model.DeviceGroupList{}
	res.Page = int64(page)
	res.PageSize = int64(pageSize)
	tx := DB.Model(&model.DeviceGroup{})
	if where != nil {
		where(tx)
	}
	tx.Count(&res.Total)
	tx.Scopes(Paginate(page, pageSize))
	tx.Find(&res.DeviceGroups)
	return
}

func (us *GroupService) DeviceGroupCreate(u *model.DeviceGroup) error {
	res := DB.Create(u).Error
	return res
}
func (us *GroupService) DeviceGroupDelete(u *model.DeviceGroup) error {
	return DB.Delete(u).Error
}

func (us *GroupService) DeviceGroupUpdate(u *model.DeviceGroup) error {
	return DB.Model(u).Updates(u).Error
}
