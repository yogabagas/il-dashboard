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

type SendersController struct {
}

func (o *SendersController) Init() {
	gocom.POST("/api/v1/instant-link/senders", a.Allow(a.Role(a.ADMIN, a.OWNER)), o.create)
	gocom.PUT("/api/v1/instant-link/senders/:id", a.Allow(a.Role(a.ADMIN, a.OWNER)), o.update)
	gocom.GET("/api/v1/instant-link/senders/:id", a.Allow(a.Role(a.ADMIN, a.OWNER)), o.getById)
	gocom.GET("/api/v1/instant-link/senders", a.Allow(a.Role(a.ADMIN, a.OWNER)), o.search)
	gocom.DELETE("/api/v1/instant-link/senders/:id", a.Allow(a.Role(a.ADMIN, a.OWNER)), o.delete)
	gocom.POST("/api/v1/instant-link/senders/:id/verify", a.Allow(a.Role(a.ADMIN, a.OWNER)), o.verify)
}

func (o *SendersController) create(ctx gocom.Context) error {
	req := dtos.SenderReq{}
	_ = ctx.Bind(&req)

	ret, err := services.GetSendersService().Create(req, a.Get(ctx))

	if err != nil {
		return ctx.SendError(err)
	}

	return ctx.SendResult(ret)
}

func (o *SendersController) update(ctx gocom.Context) error {
	req := dtos.SenderUpdateReq{}
	_ = ctx.Bind(&req)

	ret, err := services.GetSendersService().Update(ctx.Param("id"), req, a.Get(ctx))

	if err != nil {
		return ctx.SendError(err)
	}

	return ctx.SendResult(ret)
}

func (o *SendersController) getById(ctx gocom.Context) error {
	ret, err := services.GetSendersService().GetById(ctx.Param("id"), a.Get(ctx))

	if err != nil {
		return ctx.SendError(err)
	}

	return ctx.SendResult(ret)
}

func (o *SendersController) search(ctx gocom.Context) error {
	pageNo, _ := strconv.ParseInt(ctx.Query("pageNo"), 10, 32)
	rowPerPage, _ := strconv.ParseInt(ctx.Query("rowPerPage"), 10, 32)

	filter := ctx.Query("filter")
	channelType := ctx.Query("type")
	status := ctx.Query("status")
	clientId := ctx.Query("clientId")

	ret, haveNext, count := services.GetSendersService().Search(
		filter,
		channelType,
		status,
		clientId,
		int(pageNo),
		int(rowPerPage),
		a.Get(ctx))

	return common.SendPaged(ctx, ret, int(pageNo), haveNext, int64(count))
}

func (o *SendersController) delete(ctx gocom.Context) error {
	err := services.GetSendersService().Delete(ctx.Param("id"), a.Get(ctx))

	if err != nil {
		return ctx.SendError(err)
	}

	return ctx.SendResult(true)
}

func (o *SendersController) verify(ctx gocom.Context) error {
	req := dtos.VerifyOTPCodeReq{}
	_ = ctx.Bind(&req)
	req.PhoneNumberId = ctx.Param("id")

	ret, err := services.GetSendersService().VerifySender(req.PhoneNumberId, req.Code, a.Get(ctx))
	if err != nil {
		return ctx.SendError(err)
	}

	return ctx.SendResult(ret)
}

//---------------------------------------

var sendersController *SendersController
var sendersControllerOnce sync.Once

func GetSendersController() *SendersController {

	if sendersController == nil {

		sendersControllerOnce.Do(func() {
			sendersController = &SendersController{}
		})
	}

	return sendersController
}
