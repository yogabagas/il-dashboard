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

type CampaignsController struct {
}

//---------------------------------------

var campaignsController *CampaignsController
var campaignsControllerOnce sync.Once

func GetCampaignsController() *CampaignsController {

	if campaignsController == nil {

		campaignsControllerOnce.Do(func() {
			campaignsController = &CampaignsController{}
		})
	}

	return campaignsController
}

func (o *CampaignsController) Init() {
	gocom.POST("/api/v1/instant-link/campaigns", a.Allow(a.Role(a.ADMIN, a.OWNER)), o.create)
	gocom.PUT("/api/v1/instant-link/campaigns/:id", a.Allow(a.Role(a.ADMIN, a.OWNER)), o.update)
	gocom.GET("/api/v1/instant-link/campaigns/:id", a.Allow(a.Role(a.ADMIN, a.OWNER, a.STAFF)), o.getById)
	gocom.GET("/api/v1/instant-link/campaigns", a.Allow(a.Role(a.ADMIN, a.OWNER, a.STAFF)), o.search)
	gocom.DELETE("/api/v1/instant-link/campaigns/:id", a.Allow(a.Role(a.ADMIN, a.OWNER)), o.delete)
	gocom.GET("/api/v1/instant-link/campaigns/:id/recipients", a.Allow(a.Role(a.ADMIN, a.OWNER, a.STAFF)), o.getRecipients)
	gocom.GET("/api/v1/instant-link/campaigns/:id/stats", a.Allow(a.Role(a.ADMIN, a.OWNER, a.STAFF)), o.getStats)
	gocom.POST("/api/v1/instant-link/campaigns/:id/generate-recipients", a.Allow(a.Role(a.ADMIN, a.OWNER)), o.generateRecipients)
	gocom.POST("/api/v1/instant-link/campaigns/:id/start", a.Allow(a.Role(a.ADMIN, a.OWNER)), o.start)
	gocom.POST("/api/v1/instant-link/campaigns/:id/pause", a.Allow(a.Role(a.ADMIN, a.OWNER)), o.pause)
	gocom.POST("/api/v1/instant-link/campaigns/:id/resume", a.Allow(a.Role(a.ADMIN, a.OWNER)), o.resume)
}

func (o *CampaignsController) create(ctx gocom.Context) error {
	req := dtos.CampaignReq{}
	_ = ctx.Bind(&req)

	ret, err := services.GetCampaignsService().Create(req, a.Get(ctx))

	if err != nil {
		return ctx.SendError(err)
	}

	return ctx.SendResult(ret)
}

func (o *CampaignsController) update(ctx gocom.Context) error {
	req := dtos.CampaignUpdateReq{}
	_ = ctx.Bind(&req)

	ret, err := services.GetCampaignsService().Update(ctx.Param("id"), req, a.Get(ctx))

	if err != nil {
		return ctx.SendError(err)
	}

	return ctx.SendResult(ret)
}

func (o *CampaignsController) getById(ctx gocom.Context) error {
	ret, err := services.GetCampaignsService().GetById(ctx.Param("id"), a.Get(ctx))

	if err != nil {
		return ctx.SendError(err)
	}

	return ctx.SendResult(ret)
}

func (o *CampaignsController) search(ctx gocom.Context) error {
	pageNo, _ := strconv.ParseInt(ctx.Query("pageNo"), 10, 32)
	rowPerPage, _ := strconv.ParseInt(ctx.Query("rowPerPage"), 10, 32)

	filter := ctx.Query("filter")
	campaignType := ctx.Query("type")
	status := ctx.Query("status")
	senderId := ctx.Query("senderId")
	clientId := ctx.Query("clientId")
	dateFrom := ctx.Query("dateFrom")
	dateTo := ctx.Query("dateTo")

	ret, haveNext, count := services.GetCampaignsService().Search(filter, clientId, senderId, campaignType, status, dateFrom, dateTo, int(pageNo), int(rowPerPage), a.Get(ctx))

	return common.SendPaged(ctx, ret, int(pageNo), haveNext, int64(count))
}

func (o *CampaignsController) delete(ctx gocom.Context) error {
	err := services.GetCampaignsService().Delete(ctx.Param("id"), a.Get(ctx))

	if err != nil {
		return ctx.SendError(err)
	}

	return ctx.SendResult(true)
}

func (o *CampaignsController) getRecipients(ctx gocom.Context) error {
	pageNo, _ := strconv.ParseInt(ctx.Query("pageNo"), 10, 32)
	rowPerPage, _ := strconv.ParseInt(ctx.Query("rowPerPage"), 10, 32)

	status := ctx.Query("status")

	ret, haveNext, count := services.GetCampaignsService().GetRecipients(ctx.Param("id"), status, int(pageNo), int(rowPerPage), a.Get(ctx))

	return common.SendPaged(ctx, ret, int(pageNo), haveNext, int64(count))
}

func (o *CampaignsController) getStats(ctx gocom.Context) error {
	ret, err := services.GetCampaignsService().GetStats(ctx.Param("id"), a.Get(ctx))

	if err != nil {
		return ctx.SendError(err)
	}

	return ctx.SendResult(ret)
}

func (o *CampaignsController) generateRecipients(ctx gocom.Context) error {
	count, err := services.GetCampaignsService().GenerateRecipients(ctx.Param("id"), a.Get(ctx))

	if err != nil {
		return ctx.SendError(err)
	}

	return ctx.SendResult(map[string]interface{}{"success": true, "totalRecipients": count})
}

func (o *CampaignsController) start(ctx gocom.Context) error {
	ret, err := services.GetCampaignsService().Start(ctx.Param("id"), a.Get(ctx))

	if err != nil {
		return ctx.SendError(err)
	}

	return ctx.SendResult(ret)
}

func (o *CampaignsController) pause(ctx gocom.Context) error {
	ret, err := services.GetCampaignsService().Pause(ctx.Param("id"), a.Get(ctx))

	if err != nil {
		return ctx.SendError(err)
	}

	return ctx.SendResult(ret)
}

func (o *CampaignsController) resume(ctx gocom.Context) error {
	ret, err := services.GetCampaignsService().Resume(ctx.Param("id"), a.Get(ctx))

	if err != nil {
		return ctx.SendError(err)
	}

	return ctx.SendResult(ret)
}
