package admin

import (
	"github.com/gin-gonic/gin"
	"github.com/ymg2006/rustdesk-api/v2/global"
	"github.com/ymg2006/rustdesk-api/v2/http/request/admin"
	"github.com/ymg2006/rustdesk-api/v2/http/response"
	"github.com/ymg2006/rustdesk-api/v2/service"
	"strconv"
)

type Group struct {
}

// Detail group
// @Tags group
// @Summary Group details
// @Description Group details
// @Accept  json
// @Produce  json
// @Param id path int true "ID"
// @Success 200 {object} response.Response{data=model.Group}
// @Failure 500 {object} response.Response
// @Router /admin/group/detail/{id} [get]
// @Security token
func (ct *Group) Detail(c *gin.Context) {
	id := c.Param("id")
	iid, _ := strconv.Atoi(id)
	u := service.AllService.GroupService.InfoById(uint(iid))
	if u.Id > 0 {
		response.Success(c, u)
		return
	}
	response.Fail(c, 101, response.TranslateMsg(c, "ItemNotFound"))
}

// Create Create a group (department)
// @Tags group
// @Summary Create a group
// @Description Create a group
// @Accept  json
// @Produce  json
// @Param body body admin.GroupForm true "Group information"
// @Success 200 {object} response.Response{data=model.Group}
// @Failure 500 {object} response.Response
// @Router /admin/group/create [post]
// @Security token
func (ct *Group) Create(c *gin.Context) {
	f := &admin.GroupForm{}
	if err := c.ShouldBindJSON(f); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	errList := global.Validator.ValidStruct(c, f)
	if len(errList) > 0 {
		response.Fail(c, 101, errList[0])
		return
	}
	u := f.ToGroup()
	if u.ParentId > 0 {
		p := service.AllService.GroupService.InfoById(u.ParentId)
		if p.Id == 0 {
			response.Fail(c, 101, response.TranslateMsg(c, "ParentDeptNotFound"))
			return
		}
	}
	err := service.AllService.GroupService.Create(u)
	if err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "OperationFailed")+err.Error())
		return
	}
	response.Success(c, nil)
}

// List list
// @Tags group
// @Summary Group List
// @Description group list
// @Accept  json
// @Produce  json
// @Param page query int false "page number"
// @Param page_size query int false "page size"
// @Success 200 {object} response.Response{data=model.GroupList}
// @Failure 500 {object} response.Response
// @Router /admin/group/list [get]
// @Security token
func (ct *Group) List(c *gin.Context) {
	query := &admin.PageQuery{}
	if err := c.ShouldBindQuery(query); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	res := service.AllService.GroupService.List(query.Page, query.PageSize, nil)
	response.Success(c, res)
}

// Update Edit
// @Tags group
// @Summary Group Editor
// @Description Group Edit
// @Accept  json
// @Produce  json
// @Param body body admin.GroupForm true "Group information"
// @Success 200 {object} response.Response{data=model.Group}
// @Failure 500 {object} response.Response
// @Router /admin/group/update [post]
// @Security token
func (ct *Group) Update(c *gin.Context) {
	f := &admin.GroupForm{}
	if err := c.ShouldBindJSON(f); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	if f.Id == 0 {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError"))
		return
	}
	errList := global.Validator.ValidStruct(c, f)
	if len(errList) > 0 {
		response.Fail(c, 101, errList[0])
		return
	}
	u := f.ToGroup()
	// Prevent departments from being linked to themselves or their descendants, causing a hierarchical environment.
	if u.ParentId > 0 {
		if u.ParentId == u.Id {
			response.Fail(c, 101, response.TranslateMsg(c, "DeptCycleError"))
			return
		}
		p := service.AllService.GroupService.InfoById(u.ParentId)
		if p.Id == 0 {
			response.Fail(c, 101, response.TranslateMsg(c, "ParentDeptNotFound"))
			return
		}
		if service.AllService.GroupService.IsDescendantOf(u.ParentId, u.Id) {
			response.Fail(c, 101, response.TranslateMsg(c, "DeptCycleError"))
			return
		}
	}
	err := service.AllService.GroupService.Update(u)
	if err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "OperationFailed")+err.Error())
		return
	}
	response.Success(c, nil)
}

// Delete Delete
// @Tags group
// @Summary Group deletion
// @Description group deletion
// @Accept  json
// @Produce  json
// @Param body body admin.GroupForm true "Group information"
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /admin/group/delete [post]
// @Security token
func (ct *Group) Delete(c *gin.Context) {
	f := &admin.GroupForm{}
	if err := c.ShouldBindJSON(f); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	id := f.Id
	errList := global.Validator.ValidVar(c, id, "required,gt=0")
	if len(errList) > 0 {
		response.Fail(c, 101, errList[0])
		return
	}
	u := service.AllService.GroupService.InfoById(f.Id)
	if u.Id > 0 {
		err := service.AllService.GroupService.Delete(u)
		if err == nil {
			response.Success(c, nil)
			return
		}
		// Deletion is prohibited when there are sub-departments or members under the department.
		if err.Error() == "DeptHasChildren" {
			response.Fail(c, 101, response.TranslateMsg(c, "DeptHasChildren"))
			return
		}
		if err.Error() == "DeptHasUsers" {
			response.Fail(c, 101, response.TranslateMsg(c, "DeptHasUsers"))
			return
		}
		response.Fail(c, 101, response.TranslateMsg(c, "OperationFailed")+err.Error())
		return
	}
	response.Fail(c, 101, response.TranslateMsg(c, "ItemNotFound"))
}

// Tree department tree (organizational structure)
// @Tags group
// @Summary department tree
// @Description returns a nested department tree, including the number of members
// @Accept  json
// @Produce  json
// @Success 200 {object} response.Response{data=[]model.GroupTree}
// @Failure 500 {object} response.Response
// @Router /admin/group/tree [get]
// @Security token
func (ct *Group) Tree(c *gin.Context) {
	tree := service.AllService.GroupService.Tree()
	response.Success(c, tree)
}
