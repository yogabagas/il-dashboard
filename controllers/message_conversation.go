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

type MessageConversationController struct{}

func (o *MessageConversationController) Init() {
	gocom.GET("/api/v1/instant-link/conversations", a.Allow(a.Role(a.ADMIN, a.OWNER, a.STAFF)), o.search)
	gocom.POST("/api/v1/instant-link/conversations/reply", a.Allow(a.Role(a.ADMIN, a.OWNER, a.STAFF)), o.reply)
}

func (o *MessageConversationController) reply(ctx gocom.Context) error {
	authInfo := a.Get(ctx)

	req := dtos.ConversationReplyReq{}
	if err := ctx.Bind(&req); err != nil {
		return ctx.SendError(common.ERR_INVALID_REQUEST)
	}

	// Admin can override clientId; owner/staff restriction is enforced in service layer
	if authInfo.ClientId == common.PROVIDER_ID {
		if req.ClientId == "" {
			req.ClientId = ctx.Query("clientId")
		}
	}

	ret, err := services.GetMessageConversationService().Reply(req, authInfo)
	if err != nil {
		return ctx.SendError(err)
	}

	return ctx.SendResult(ret)
}

func (o *MessageConversationController) search(ctx gocom.Context) error {
	authInfo := a.Get(ctx)

	pageNo, _ := strconv.Atoi(ctx.Query("pageNo"))
	rowPerPage, _ := strconv.Atoi(ctx.Query("rowPerPage"))

	req := dtos.MessageConversationSearchReq{
		Filter:     ctx.Query("filter"),
		FromNumber: ctx.Query("fromNumber"),
		SessionId:  ctx.Query("sessionId"),
		DateFrom:   ctx.Query("dateFrom"),
		DateTo:     ctx.Query("dateTo"),
		PageNo:     pageNo,
		RowPerPage: rowPerPage,
	}

	// Admin can choose clientId freely; owner/staff is restricted in service layer
	if authInfo.ClientId == common.PROVIDER_ID {
		req.ClientId = ctx.Query("clientId")
	}

	ret, haveNext, count := services.GetMessageConversationService().Search(req, authInfo)

	return common.SendPaged(ctx, ret, req.PageNo, haveNext, count)
}

//---------------------------------------

var messageConversationController *MessageConversationController
var messageConversationControllerOnce sync.Once

func GetMessageConversationController() *MessageConversationController {
	if messageConversationController == nil {
		messageConversationControllerOnce.Do(func() {
			messageConversationController = &MessageConversationController{}
		})
	}
	return messageConversationController
}
