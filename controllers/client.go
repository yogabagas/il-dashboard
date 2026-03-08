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

type ClientController struct {
}

func (o *ClientController) Init() {

	//gocom.POST("/client", a.Allow(a.Role(a.ADMIN, a.OWNER)), o.createClient)
	gocom.POST("/api/v1/instant-link/clients", a.Allow(a.Role(a.ADMIN, a.OWNER)), o.createClient)
	gocom.PUT("/api/v1/instant-link/clients/:id", a.Allow(a.Role(a.ADMIN, a.OWNER)), o.updateClient)
	gocom.GET("/api/v1/instant-link/clients/:id", a.Allow(a.Role(a.ADMIN)), o.getClient)
	gocom.GET("/api/v1/instant-link/clients", a.Allow(a.Role(a.ADMIN, a.OWNER), a.Access("SEARCH_CLIENT")), o.searchClient)
	gocom.DELETE("/api/v1/instant-link/clients/:id", a.Allow(a.Role(a.ADMIN, a.OWNER), a.Access("DELETE_CLIENT")), o.deleteClient)
	gocom.PUT("/api/v1/instant-link/clients/:id/onboard", a.Allow(a.Role(a.ADMIN, a.OWNER)), o.onboardMeta)
}

func (o *ClientController) createClient(ctx gocom.Context) error {

	req := dtos.ClientReq{}
	_ = ctx.Bind(&req)

	ret, err := services.GetClientSvc().Create(req, a.Get(ctx))

	if err != nil {
		return ctx.SendError(err)
	}

	return ctx.SendResult(ret)
}

func (o *ClientController) updateClient(ctx gocom.Context) error {

	req := dtos.ClientUpdateReq{}
	_ = ctx.Bind(&req)

	ret, err := services.GetClientSvc().Update(ctx.Param("id"), req, a.Get(ctx))

	if err != nil {
		return ctx.SendError(err)
	}

	return ctx.SendResult(ret)
}

func (o *ClientController) getClient(ctx gocom.Context) error {

	req := dtos.ClientReq{}
	_ = ctx.Bind(&req)

	auth := a.Get(ctx)
	ret, err := services.GetClientSvc().Get(ctx.Param("id"), auth)

	if err != nil {
		return ctx.SendError(err)
	}

	return ctx.SendResult(ret)
}

func (o *ClientController) searchClient(ctx gocom.Context) error {

	pageNo, _ := strconv.ParseInt(ctx.Query("pageNo"), 10, 32)
	rowPerPage, _ := strconv.ParseInt(ctx.Query("rowPerPage"), 10, 32)

	parentId := ctx.Query("parentId")
	clientId := ctx.Query("clientId")

	auth := a.Get(ctx)

	if auth.ClientId != common.PROVIDER_ID {
		parentId = common.PROVIDER_ID
	}

	if clientId != common.PROVIDER_ID {
		clientId = auth.ClientId
	}

	ret, haveNext, count := services.GetClientSvc().Search(ctx.Query("filter"), parentId, clientId, int(pageNo), int(rowPerPage))

	return common.SendPaged(ctx, ret, int(pageNo), haveNext, count)
}

func (o *ClientController) deleteClient(ctx gocom.Context) error {

	err := services.GetClientSvc().Delete(ctx.Param("id"), a.Get(ctx))

	if err != nil {
		return ctx.SendError(err)
	}

	return ctx.SendResult(true)
}

func (o *ClientController) onboardMeta(ctx gocom.Context) error {

	clientId := ctx.Param("id")

	ret, err := services.GetClientSvc().OnboardMeta(dtos.ClientMetaOnboardReq{
		ClientId: clientId,
	})
	if err != nil {
		return ctx.SendError(err)
	}

	return ctx.SendResult(ret)
}

//---------------------------------------

var clientController *ClientController
var clientControllerOnce sync.Once

func GetClientController() *ClientController {

	if clientController == nil {

		clientControllerOnce.Do(func() {
			clientController = &ClientController{}
		})
	}

	return clientController
}
