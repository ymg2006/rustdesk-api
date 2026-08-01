package admin

import (
	"github.com/gin-gonic/gin"
	"github.com/ymg2006/rustdesk-api/v2/global"
	"github.com/ymg2006/rustdesk-api/v2/http/request/admin"
	"github.com/ymg2006/rustdesk-api/v2/http/response"
	"github.com/ymg2006/rustdesk-api/v2/service"
	"gorm.io/gorm"
	"strconv"
)

type DeviceGroup struct {
}

// Detail device group
// @Tags device group
// @Summary Device Group Details
// @Description Device group details
// @Accept  json
// @Produce  json
// @Param id path int true "ID"
// @Success 200 {object} response.Response{data=model.Group}
// @Failure 500 {object} response.Response
// @Router /admin/device_group/detail/{id} [get]
// @Security token
func (ct *DeviceGroup) Detail(c *gin.Context) {
	id := c.Param("id")
	iid, _ := strconv.Atoi(id)
	u := service.AllService.GroupService.DeviceGroupInfoById(uint(iid))
	if u.Id > 0 {
		response.Success(c, u)
		return
	}
	response.Fail(c, 101, response.TranslateMsg(c, "ItemNotFound"))
}

// Create Create a device group
// @Tags device group
// @Summary Create device group
// @Description Create device group
// @Accept  json
// @Produce  json
// @Param body body admin.DeviceGroupForm true "Device group information"
// @Success 200 {object} response.Response{data=model.DeviceGroup}
// @Failure 500 {object} response.Response
// @Router /admin/device_group/create [post]
// @Security token
func (ct *DeviceGroup) Create(c *gin.Context) {
	f := &admin.DeviceGroupForm{}
	if err := c.ShouldBindJSON(f); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	errList := global.Validator.ValidStruct(c, f)
	if len(errList) > 0 {
		response.Fail(c, 101, errList[0])
		return
	}
	u := f.ToDeviceGroup()
	err := service.AllService.GroupService.DeviceGroupCreate(u)
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
// @Param name query string false "device group name"
// @Success 200 {object} response.Response{data=model.GroupList}
// @Failure 500 {object} response.Response
// @Router /admin/device_group/list [get]
// @Security token
func (ct *DeviceGroup) List(c *gin.Context) {
	query := &admin.PageQuery{}
	if err := c.ShouldBindQuery(query); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	name := c.Query("name")
	res := service.AllService.GroupService.DeviceGroupList(query.Page, query.PageSize, func(tx *gorm.DB) {
		if name != "" {
			tx.Where("name like ?", "%"+name+"%")
		}
	})
	response.Success(c, res)
}

// Update Edit
// @Tags device group
// @Summary Device Group Editing
// @Description Device Group Edit
// @Accept  json
// @Produce  json
// @Param body body admin.DeviceGroupForm true "Group information"
// @Success 200 {object} response.Response{data=model.Group}
// @Failure 500 {object} response.Response
// @Router /admin/device_group/update [post]
// @Security token
func (ct *DeviceGroup) Update(c *gin.Context) {
	f := &admin.DeviceGroupForm{}
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
	u := f.ToDeviceGroup()
	err := service.AllService.GroupService.DeviceGroupUpdate(u)
	if err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "OperationFailed")+err.Error())
		return
	}
	response.Success(c, nil)
}

// Delete Delete
// @Tags device group
// @Summary Device group deletion
// @Description Device group deletion
// @Accept  json
// @Produce  json
// @Param body body admin.DeviceGroupForm true "Group information"
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /admin/device_group/delete [post]
// @Security token
func (ct *DeviceGroup) Delete(c *gin.Context) {
	f := &admin.DeviceGroupForm{}
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
	u := service.AllService.GroupService.DeviceGroupInfoById(f.Id)
	if u.Id > 0 {
		err := service.AllService.GroupService.DeviceGroupDelete(u)
		if err == nil {
			response.Success(c, nil)
			return
		}
		response.Fail(c, 101, response.TranslateMsg(c, "OperationFailed")+err.Error())
		return
	}
	response.Fail(c, 101, response.TranslateMsg(c, "ItemNotFound"))
}
