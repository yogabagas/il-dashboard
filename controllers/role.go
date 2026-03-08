package controllers

import (
	"github.com/ariandi/gocom"
	"github.com/ariandi/gocom/logger"
	common "gitlab.com/anti_metter/switching_common"
	a "gitlab.com/anti_metter/switching_common/auth"
	"gitlab.com/bot3342545/il-dashboard/dtos"
	"gitlab.com/bot3342545/il-dashboard/services"

	"strconv"
	"sync"
)

type RoleController struct {
}

func (o *RoleController) Init() {
	// redeploy
	gocom.POST("/api/v1/instant-link/roles/:id/menu", a.Allow(a.Role(a.ADMIN), a.Access("CREATE_ROLE")), o.setRoleMenu)
	gocom.GET("/api/v1/instant-link/roles/:id/menu", a.Allow(a.Role(a.ADMIN), a.Access("GET_ROLE")), o.getRoleMenu)
	gocom.GET("/api/v1/instant-link/roles/:id/menu/tree", a.Allow(a.Role(a.ADMIN), a.Access("GET_ROLE")), o.getTreeMenu)

	gocom.POST("/api/v1/instant-link/roles/:id/permission", a.Allow(a.Role(a.ADMIN), a.Access("CREATE_ROLE")), o.setRolePermission)
	gocom.GET("/api/v1/instant-link/roles/:id/permission", a.Allow(a.Role(a.ADMIN), a.Access("GET_ROLE")), o.getRolePermission)

	gocom.POST("/api/v1/instant-link/roles", a.Allow(a.Role(a.ADMIN), a.Access("CREATE_ROLE")), o.createRole)
	//gocom.POST("/api/v1/instant-link/roles", o.createRole)
	gocom.PUT("/api/v1/instant-link/roles/:id", a.Allow(a.Role(a.ADMIN), a.Access("UPDATE_ROLE")), o.updateRole)
	gocom.GET("/api/v1/instant-link/roles", a.Allow(a.Logged, a.Access("SEARCH_ROLE")), o.searchRole)
	gocom.DELETE("/api/v1/instant-link/roles/:id", a.Allow(a.Role(a.ADMIN), a.Access("DELETE_ROLE")), o.deleteRole) //
}

func (o *RoleController) createRole(ctx gocom.Context) error {

	req := dtos.RoleReq{}
	_ = ctx.Bind(&req)

	ret, err := services.GetRoleService().Create(req, a.Get(ctx))

	if err != nil {
		return ctx.SendError(err)
	}

	return ctx.SendResult(ret)
}

func (o *RoleController) updateRole(ctx gocom.Context) error {

	req := dtos.RoleReq{}
	_ = ctx.Bind(&req)

	ret, err := services.GetRoleService().Update(ctx.Param("id"), req, a.Get(ctx))

	if err != nil {
		return ctx.SendError(err)
	}

	return ctx.SendResult(ret)
}

func (o *RoleController) setRoleMenu(ctx gocom.Context) error {

	req := dtos.RoleMenuReq{}
	_ = ctx.Bind(&req)

	ret, err := services.GetRoleService().SetMenu(ctx.Param("id"), req, a.Get(ctx))

	if err != nil {
		return ctx.SendError(err)
	}

	return ctx.SendResult(ret)
}

func (o *RoleController) getRoleMenu(ctx gocom.Context) error {

	ret := services.GetRoleService().ListMenu(ctx.Param("id"), ctx.Query("parentId"))

	return ctx.SendResult(ret)
}

func (o *RoleController) getTreeMenu(ctx gocom.Context) error {

	ret := services.GetRoleService().TreeMenu(ctx.Param("id"))

	return ctx.SendResult(ret)
}

func (o *RoleController) setRolePermission(ctx gocom.Context) error {

	req := dtos.RolePermissionReq{}
	_ = ctx.Bind(&req)

	ret, err := services.GetRoleService().SetPermission(ctx.Param("id"), req, a.Get(ctx))

	if err != nil {
		return ctx.SendError(err)
	}

	return ctx.SendResult(ret)
}

func (o *RoleController) getRolePermission(ctx gocom.Context) error {

	ret := services.GetRoleService().ListPermission(ctx.Param("id"), ctx.Query("parentId"))

	return ctx.SendResult(ret)
}

func (o *RoleController) searchRole(ctx gocom.Context) error {
	pageNo, _ := strconv.ParseInt(ctx.Query("pageNo"), 10, 32)
	rowPerPage, _ := strconv.ParseInt(ctx.Query("rowPerPage"), 10, 32)

	auth := a.Get(ctx)
	clientId := ctx.Query("clientId")

	//if auth.ClientId != common.PROVIDER_ID {
	//	clientId = auth.ClientId
	//}

	logger.Info("[searchRole] auth.ClientId: ", auth.ClientId)

	ret, haveNext, count := services.GetRoleService().Search(ctx.Query("filter"), clientId, int(pageNo), int(rowPerPage))
	return common.SendPaged(ctx, ret, int(pageNo), haveNext, count)
}

func (o *RoleController) deleteRole(ctx gocom.Context) error {

	err := services.GetRoleService().Delete(ctx.Param("id"), a.Get(ctx))

	if err != nil {
		return ctx.SendError(err)
	}

	return ctx.SendResult(true)
}

//---------------------------------------

var roleController *RoleController
var roleControllerOnce sync.Once

func GetRoleController() *RoleController {

	if roleController == nil {

		roleControllerOnce.Do(func() {
			roleController = &RoleController{}
		})
	}

	return roleController
}
