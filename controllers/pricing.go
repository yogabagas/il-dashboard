package controllers

import (
	"strconv"
	"sync"

	"github.com/ariandi/gocom"
	"github.com/ariandi/gocom/logger"
	common "gitlab.com/anti_metter/switching_common"
	a "gitlab.com/anti_metter/switching_common/auth"
	"gitlab.com/bot3342545/il-dashboard/dtos"
	"gitlab.com/bot3342545/il-dashboard/services"
)

type PricingController struct {
	gocom.Controller
}

var pricingController *PricingController
var oncePricingController sync.Once

func GetPricingController() *PricingController {
	oncePricingController.Do(func() {
		pricingController = &PricingController{}
	})
	return pricingController
}

func (o *PricingController) Init() {
	gocom.GET("/api/v1/instant-link/pricing", a.Allow(a.Role(a.ADMIN, a.OWNER)), o.search)
	gocom.GET("/api/v1/instant-link/pricing/:id", a.Allow(a.Role(a.ADMIN, a.OWNER)), o.getById)
	gocom.POST("/api/v1/instant-link/pricing", a.Allow(a.Role(a.ADMIN)), o.create)
	gocom.PUT("/api/v1/instant-link/pricing/:id", a.Allow(a.Role(a.ADMIN)), o.update)
	gocom.DELETE("/api/v1/instant-link/pricing/:id", a.Allow(a.Role(a.ADMIN)), o.delete)
}

func (o *PricingController) search(ctx gocom.Context) error {
	logger.Debug("PricingController.search - Start")

	pageNo, _ := strconv.ParseInt(ctx.Query("pageNo"), 10, 32)
	rowPerPage, _ := strconv.ParseInt(ctx.Query("rowPerPage"), 10, 32)

	ret, haveNext, count := services.GetPricingService().Search(
		ctx.Query("filter"),
		ctx.Query("serviceType"),
		ctx.Query("category"),
		ctx.Query("clientId"),
		ctx.Query("status"),
		int(pageNo),
		int(rowPerPage),
		a.Get(ctx),
	)

	return common.SendPaged(ctx, ret, int(pageNo), haveNext, int64(count))
}

func (o *PricingController) getById(ctx gocom.Context) error {
	logger.Debug("PricingController.getById - Start")

	ret, err := services.GetPricingService().GetById(ctx.Param("id"), a.Get(ctx))

	if err != nil {
		return ctx.SendError(err)
	}

	return ctx.SendResult(ret)
}

func (o *PricingController) create(ctx gocom.Context) error {
	logger.Debug("PricingController.create - Start")

	req := dtos.PricingReq{}
	_ = ctx.Bind(&req)

	ret, err := services.GetPricingService().Create(req, a.Get(ctx))

	if err != nil {
		return ctx.SendError(err)
	}

	return ctx.SendResult(ret)
}

func (o *PricingController) update(ctx gocom.Context) error {
	logger.Debug("PricingController.update - Start")

	req := dtos.PricingReq{}
	_ = ctx.Bind(&req)

	ret, err := services.GetPricingService().Update(ctx.Param("id"), req, a.Get(ctx))

	if err != nil {
		return ctx.SendError(err)
	}

	return ctx.SendResult(ret)
}

func (o *PricingController) delete(ctx gocom.Context) error {
	logger.Debug("PricingController.delete - Start")

	err := services.GetPricingService().Delete(ctx.Param("id"), a.Get(ctx))

	if err != nil {
		return ctx.SendError(err)
	}

	return ctx.SendResult(true)
}
