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

type MessageLogsController struct {
}

func (o *MessageLogsController) Init() {
	gocom.GET("/api/v1/instant-link/message-logs/:id", a.Allow(a.Role(a.ADMIN, a.OWNER, a.STAFF)), o.getById)
	gocom.GET("/api/v1/instant-link/message-logs", a.Allow(a.Role(a.ADMIN, a.OWNER, a.STAFF)), o.search)
}

func (o *MessageLogsController) getById(ctx gocom.Context) error {
	ret, err := services.GetMessageLogsService().GetById(ctx.Param("id"), a.Get(ctx))

	if err != nil {
		return ctx.SendError(err)
	}

	return ctx.SendResult(ret)
}

func (o *MessageLogsController) search(ctx gocom.Context) error {
	pageNo, _ := strconv.Atoi(ctx.Query("pageNo"))
	rowPerPage, _ := strconv.Atoi(ctx.Query("rowPerPage"))
	sortBy := ctx.Query("sortBy")
	sortOrder := ctx.Query("sortOrder")

	req := dtos.MessageLogSearchReq{
		Filter:       ctx.Query("filter"),
		ClientId:     ctx.Query("clientId"),
		CampaignId:   ctx.Query("campaignId"),
		Type:         ctx.Query("type"),
		Direction:    ctx.Query("direction"),
		Status:       ctx.Query("status"),
		RecipientVal: ctx.Query("recipientVal"),
		DateFrom:     ctx.Query("dateFrom"),
		DateTo:       ctx.Query("dateTo"),
		PageNo:       pageNo,
		RowPerPage:   rowPerPage,
	}

	ret, haveNext, count := services.GetMessageLogsService().Search(req, sortBy, sortOrder, a.Get(ctx))

	return common.SendPaged(ctx, ret, req.PageNo, haveNext, int64(count))
}

//---------------------------------------

var messageLogsController *MessageLogsController
var messageLogsControllerOnce sync.Once

func GetMessageLogsController() *MessageLogsController {

	if messageLogsController == nil {

		messageLogsControllerOnce.Do(func() {
			messageLogsController = &MessageLogsController{}
		})
	}

	return messageLogsController
}
