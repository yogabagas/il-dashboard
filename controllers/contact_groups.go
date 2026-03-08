package controllers

import (
	"strconv"
	"sync"

	"github.com/ariandi/gocom"
	common "gitlab.com/anti_metter/switching_common"
	a "gitlab.com/anti_metter/switching_common/auth"
	"gitlab.com/bot3342545/il-dashboard/dtos"
	"gitlab.com/bot3342545/il-dashboard/services"
)

type ContactGroupsController struct {
}

func (o *ContactGroupsController) Init() {
	gocom.POST("/api/v1/instant-link/contact-groups", a.Allow(a.Role(a.ADMIN, a.OWNER)), o.create)
	gocom.PUT("/api/v1/instant-link/contact-groups/:id", a.Allow(a.Role(a.ADMIN, a.OWNER)), o.update)
	gocom.GET("/api/v1/instant-link/contact-groups/:id", a.Allow(a.Role(a.ADMIN, a.OWNER, a.STAFF)), o.getById)
	gocom.GET("/api/v1/instant-link/contact-groups", a.Allow(a.Role(a.ADMIN, a.OWNER, a.STAFF)), o.search)
	gocom.DELETE("/api/v1/instant-link/contact-groups/:id", a.Allow(a.Role(a.ADMIN, a.OWNER)), o.delete)
	gocom.POST("/api/v1/instant-link/contact-groups/:id/members", a.Allow(a.Role(a.ADMIN, a.OWNER)), o.addMembers)
	gocom.DELETE("/api/v1/instant-link/contact-groups/:id/members", a.Allow(a.Role(a.ADMIN, a.OWNER)), o.removeMembers)
	gocom.GET("/api/v1/instant-link/contact-groups/:id/members", a.Allow(a.Role(a.ADMIN, a.OWNER, a.STAFF)), o.getMembers)
}

func (o *ContactGroupsController) create(ctx gocom.Context) error {
	req := dtos.ContactGroupReq{}
	_ = ctx.Bind(&req)

	ret, err := services.GetContactGroupsService().Create(req, a.Get(ctx))

	if err != nil {
		return ctx.SendError(err)
	}

	return ctx.SendResult(ret)
}

func (o *ContactGroupsController) update(ctx gocom.Context) error {
	req := dtos.ContactGroupUpdateReq{}
	_ = ctx.Bind(&req)

	ret, err := services.GetContactGroupsService().Update(ctx.Param("id"), req, a.Get(ctx))

	if err != nil {
		return ctx.SendError(err)
	}

	return ctx.SendResult(ret)
}

func (o *ContactGroupsController) getById(ctx gocom.Context) error {
	ret, err := services.GetContactGroupsService().GetById(ctx.Param("id"), a.Get(ctx))

	if err != nil {
		return ctx.SendError(err)
	}

	return ctx.SendResult(ret)
}

func (o *ContactGroupsController) search(ctx gocom.Context) error {
	pageNo, _ := strconv.ParseInt(ctx.Query("pageNo"), 10, 32)
	rowPerPage, _ := strconv.ParseInt(ctx.Query("rowPerPage"), 10, 32)

	filter := ctx.Query("filter")
	clientId := ctx.Query("clientId")

	ret, haveNext, count := services.GetContactGroupsService().Search(filter, clientId, int(pageNo), int(rowPerPage), a.Get(ctx))

	return common.SendPaged(ctx, ret, int(pageNo), haveNext, int64(count))
}

func (o *ContactGroupsController) delete(ctx gocom.Context) error {
	err := services.GetContactGroupsService().Delete(ctx.Param("id"), a.Get(ctx))

	if err != nil {
		return ctx.SendError(err)
	}

	return ctx.SendResult(map[string]interface{}{"success": true, "id": ctx.Param("id")})
}

func (o *ContactGroupsController) addMembers(ctx gocom.Context) error {
	var req struct {
		ContactIds []string `json:"contactIds"`
	}
	_ = ctx.Bind(&req)

	err := services.GetContactGroupsService().AddMembers(ctx.Param("id"), req.ContactIds, a.Get(ctx))

	if err != nil {
		return ctx.SendError(err)
	}

	return ctx.SendResult(map[string]interface{}{"success": true, "count": len(req.ContactIds)})
}

func (o *ContactGroupsController) removeMembers(ctx gocom.Context) error {
	var req struct {
		ContactIds []string `json:"contactIds"`
	}
	_ = ctx.Bind(&req)

	err := services.GetContactGroupsService().RemoveMembers(ctx.Param("id"), req.ContactIds, a.Get(ctx))

	if err != nil {
		return ctx.SendError(err)
	}

	return ctx.SendResult(map[string]interface{}{"success": true, "count": len(req.ContactIds)})
}

func (o *ContactGroupsController) getMembers(ctx gocom.Context) error {
	pageNo, _ := strconv.ParseInt(ctx.Query("pageNo"), 10, 32)
	rowPerPage, _ := strconv.ParseInt(ctx.Query("rowPerPage"), 10, 32)

	ret, haveNext, count := services.GetContactGroupsService().GetMembers(ctx.Param("id"), int(pageNo), int(rowPerPage), a.Get(ctx))

	return common.SendPaged(ctx, ret, int(pageNo), haveNext, int64(count))
}

//---------------------------------------

var contactGroupsController *ContactGroupsController
var contactGroupsControllerOnce sync.Once

func GetContactGroupsController() *ContactGroupsController {

	if contactGroupsController == nil {

		contactGroupsControllerOnce.Do(func() {
			contactGroupsController = &ContactGroupsController{}
		})
	}

	return contactGroupsController
}
